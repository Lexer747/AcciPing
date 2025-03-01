// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/Lexer747/AcciPing/drawbuffer"
	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/graphdata"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/graph/terminal/ansi"
	"github.com/Lexer747/AcciPing/graph/terminal/typography"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils"
	"github.com/Lexer747/AcciPing/utils/check"
	"github.com/Lexer747/AcciPing/utils/numeric"
	"github.com/Lexer747/AcciPing/utils/sliceutils"
	"github.com/Lexer747/AcciPing/utils/timeutils"
)

const drawingDebug = false

func getTimeBetweenFrames(fps int, pingsPerMinute float64) time.Duration {
	if fps == 0 {
		return ping.PingsPerMinuteToDuration(pingsPerMinute)
	} else {
		return time.Duration(1000/fps) * time.Millisecond
	}
}

var noFrame = func(w io.Writer) error { return nil }

func (g *Graph) computeFrame(timeBetweenFrames time.Duration, drawSpinner bool) func(io.Writer) error {
	// This is race-y so ensure a consistent size for rendering, don't allow each sub-frame to re-compute the
	// size of the terminal.
	s := g.Term.Size()
	g.data.Lock()
	count := g.data.LockFreeTotalCount()
	if count == 0 {
		g.data.Unlock()
		return noFrame // no data yet
	}
	spinnerValue := ""
	if drawSpinner {
		g.lastFrame.spinnerIndex++
		spinnerValue = spinner(s, g.lastFrame.spinnerIndex, timeBetweenFrames)
	}
	if count == g.lastFrame.PacketCount && g.lastFrame.Match(s) {
		g.data.Unlock() // fast path the frame didn't change
		return func(w io.Writer) error {
			return utils.Err(w.Write([]byte(spinnerValue)))
		}
	}

	g.drawingBuffer.Reset()

	header := g.data.LockFreeHeader()
	x := computeXAxis(
		g.drawingBuffer.Get(xAxisIndex),
		g.drawingBuffer.Get(barIndex),
		s,
		header.TimeSpan,
		g.data.LockFreeSpanInfos(),
	)
	y := computeYAxis(g.drawingBuffer.Get(yAxisIndex), s, header.Stats, g.data.LockFreeURL())
	computeFrame(
		g.drawingBuffer.Get(gradientIndex),
		g.drawingBuffer.Get(dataIndex),
		g.drawingBuffer.Get(keyIndex),
		g.data.LockFreeIter(),
		g.data.LockFreeRuns(),
		x, y, s,
	)
	g.drawingBuffer.Get(spinnerIndex).WriteString(spinnerValue)
	finished := paint(g.drawingBuffer)
	// Everything we need is now cached we can unlock a bit early while we tidy up for the next frame
	g.data.Unlock()
	g.lastFrame = frame{
		PacketCount:  count,
		yAxis:        y,
		xAxis:        x,
		framePainter: finished,
		spinnerIndex: g.lastFrame.spinnerIndex,
	}

	return finished
}

func translate(p ping.PingDataPoint, x *XAxisSpanInfo, y yAxis, s terminal.Size) (yCord, xCord int) {
	xCord = getX(p.Timestamp, x, y, s)
	yCord = getY(p.Duration, y, s)
	check.Checkf(xCord <= s.Width && yCord <= s.Height, "Computed coord out of bounds (%d,%d) vs %q", xCord, yCord, s.String())
	return
}

func getY(dur time.Duration, yAxis yAxis, s terminal.Size) int {
	return int(numeric.NormalizeToRange(
		float64(dur),
		float64(yAxis.stats.Min),
		float64(yAxis.stats.Max),
		float64(s.Height-2),
		2,
	))
}

func getX(t time.Time, span *XAxisSpanInfo, y yAxis, s terminal.Size) int {
	timestamp := span.timeSpan.End.Sub(t)
	// These are inverted deliberately since the drawing reference is symmetric in the x
	newMin := min(s.Width-1, span.endX)
	newMax := max(y.labelSize, span.startX)
	if newMin < newMax {
		tmp := newMin
		newMin = newMax
		newMax = tmp
	}
	computed := int(numeric.NormalizeToRange(
		float64(timestamp),
		0,
		float64(span.timeSpan.Duration),
		float64(newMin),
		float64(newMax),
	))

	check.Checkf(computed <= s.Width, "Computed coord out of bounds (%d) vs %q", computed, s.String())
	return computed
}

type gradientState struct {
	lastGoodIndex          int64
	lastGoodTerminalWidth  int
	lastGoodTerminalHeight int
	lastGoodSpan           *XAxisSpanInfo
}

func (g gradientState) set(i int64, x, y int, s *XAxisSpanInfo) gradientState {
	return gradientState{
		lastGoodIndex:          i,
		lastGoodTerminalWidth:  x,
		lastGoodTerminalHeight: y,
		lastGoodSpan:           s,
	}
}

func (g gradientState) dropped() gradientState {
	return gradientState{lastGoodIndex: -1}
}
func (g gradientState) draw() bool {
	return g.lastGoodIndex != -1
}

var drop = ansi.Red(typography.Block)
var dropFiller = ansi.Red(typography.LightBlock)

func computeFrame(
	toWriteGradientTo, toWriteTo, toWriteKeyTo *bytes.Buffer,
	iter *graphdata.Iter,
	runs *data.Runs,
	xAxis xAxis,
	yAxis yAxis,
	s terminal.Size,
) {
	if iter.Total < 1 {
		return
	}
	centreY := (s.Height - 2) / 2
	centreX := (s.Width - 2) / 2
	if iter.Total == 1 {
		point := iter.Get(0)
		toWriteTo.WriteString(ansi.CursorPosition(centreY, centreX) + single + " " + point.Duration.String())
		return
	}
	droppedBar, droppedFiller := makeDroppedPacketIndicators(yAxis.stats.PacketsDropped, s)

	// Now iterate over all the individual data points and add them to the graph

	if shouldGradient(runs) {
		drawGradients(toWriteGradientTo, iter, xAxis, yAxis, s)
	}

	lastWasDropped := false
	lastDroppedTerminalX := -1
	window := newDrawWindow(iter.Total, len(xAxis.spans))
	xAxisIter := xAxis.NewIter()
	for i := range iter.Total {
		p := iter.Get(i)
		span := xAxisIter.Get(p)
		x := getX(p.Timestamp, span, yAxis, s)
		// TODO move the bars into the [drawWindow] so that the labels are on-top, also so that we don't
		// re-draw more than we need to.
		if p.Dropped() {
			toWriteTo.WriteString(ansi.CursorPosition(2, x) + droppedBar)
			if lastWasDropped {
				for i := min(lastDroppedTerminalX, x) + 1; i < max(lastDroppedTerminalX, x); i++ {
					toWriteTo.WriteString(ansi.CursorPosition(2, i) + droppedFiller)
				}
			}
			lastWasDropped = true
			lastDroppedTerminalX = x
			continue
		}
		lastWasDropped = false
		y := getY(p.Duration, yAxis, s)
		window.addPoint(p, span.pingStats, yAxis.stats, span.width, x, y, centreX)
	}
	window.draw(toWriteTo)
	toWriteKeyTo.WriteString(ansi.CursorPosition(s.Height-1, yAxis.labelSize+1))
	window.getKey(toWriteKeyTo)
}

func drawGradients(toWriteTo *bytes.Buffer, iter *graphdata.Iter, xAxis xAxis, yAxis yAxis, s terminal.Size) {
	g := gradientState{}
	xAxisIter := xAxis.NewIter()

	for i := range iter.Total {
		p := iter.Get(i)
		if p.Dropped() {
			g = g.dropped()
			continue
		}
		span := xAxisIter.Get(p)
		y, x := translate(p, span, yAxis, s)
		if g.draw() && !iter.IsLast(i) {
			if span == g.lastGoodSpan {
				lastP := iter.Get(g.lastGoodIndex)
				drawGradient(
					toWriteTo,
					span, yAxis,
					x, y,
					p,
					lastP,
					g.lastGoodTerminalWidth,
					g.lastGoodTerminalHeight,
					s,
				)
			}
		}
		g = g.set(i, x, y, span)
	}
}

func makeDroppedPacketIndicators(droppedPackets uint64, s terminal.Size) (string, string) {
	droppedBar := ""
	droppedFillerBar := ""
	if droppedPackets > 0 {
		droppedBar = makeBar(drop, s, false)
		droppedFillerBar = makeBar(dropFiller, s, false)
	}
	return droppedBar, droppedFillerBar
}

// A bar requires you start at the top of the terminal, general to draw a bar at coord x, do
// [ansi.CursorPosition(2, x)] before writing the bar.
func makeBar(repeating string, s terminal.Size, touchAxis bool) string {
	offset := 3
	if touchAxis {
		offset = 2
	}
	return strings.Repeat(repeating+ansi.CursorDown(1)+ansi.CursorBack(1), s.Height-offset)
}

func drawGradient(
	toWriteTo *bytes.Buffer,
	xAxis *XAxisSpanInfo,
	yAxis yAxis,
	x, y int,
	current ping.PingDataPoint,
	lastGood ping.PingDataPoint,
	lastGoodTerminalWidth int,
	lastGoodTerminalHeight int,
	s terminal.Size,
) {
	gradientsToDrawX := float64(numeric.Abs(lastGoodTerminalWidth - x))
	gradientsToDrawY := float64(numeric.Abs(lastGoodTerminalHeight - y))
	gradientsToDraw := math.Sqrt(math.Pow(gradientsToDrawX, 2) + math.Pow(gradientsToDrawY, 2))
	stepSizeY := float64(current.Duration-lastGood.Duration) / gradientsToDraw
	stepSizeX := float64(current.Timestamp.Sub(lastGood.Timestamp)) / gradientsToDraw

	pointsX := make([]int, 0)
	pointsY := make([]int, 0)
	for toDraw := 1.5; toDraw < gradientsToDraw; toDraw++ {
		interpolatedDuration := lastGood.Duration + time.Duration(toDraw*stepSizeY)
		interpolatedStamp := lastGood.Timestamp.Add(time.Duration(toDraw * stepSizeX))
		p := ping.PingDataPoint{Duration: interpolatedDuration, Timestamp: interpolatedStamp}
		cursorY, cursorX := translate(p, xAxis, yAxis, s)
		pointsX = append(pointsX, cursorX)
		pointsY = append(pointsY, cursorY)
	}
	gradient := solve(pointsX, pointsY)
	for i, g := range gradient {
		toWriteTo.WriteString(ansi.CursorPosition(pointsY[i], pointsX[i]) + ansi.Gray(g))
	}
}

func shouldGradient(runs *data.Runs) bool {
	return runs.GoodPackets.Longest > 2
}

func computeYAxis(toWriteTo *bytes.Buffer, size terminal.Size, stats *data.Stats, url string) yAxis {
	toWriteTo.Grow(size.Height)

	finalTitle := makeTitle(size, stats, url)
	fmt.Fprint(toWriteTo, finalTitle)

	gapSize := 2
	if size.Height > 20 {
		gapSize++
	} else if size.Height < 12 {
		gapSize--
	}
	durationSize := (gapSize * 3) / 2

	// We skip the first and last two lines
	for i := range size.Height - 3 {
		h := i + 2
		fmt.Fprint(toWriteTo, ansi.CursorPosition(h, 1))
		if i%gapSize == 1 {
			scaledDuration := numeric.NormalizeToRange(float64(i), float64(size.Height-2), 0, float64(stats.Min), float64(stats.Max))
			toPrint := timeutils.HumanString(time.Duration(scaledDuration), durationSize)
			fmt.Fprint(toWriteTo, ansi.Yellow(toPrint))
		} else {
			fmt.Fprint(toWriteTo, ansi.White(typography.Vertical))
		}
	}
	// Last line is always a bar
	fmt.Fprint(toWriteTo, ansi.CursorPosition(size.Height-1, 1)+ansi.White(typography.Vertical))
	return yAxis{
		size:      size.Height,
		stats:     stats,
		labelSize: durationSize + 4,
	}
}

func makeTitle(size terminal.Size, stats *data.Stats, url string) string {
	// TODO string builder, or larger buffer impl
	const yAxisTitle = "Ping "
	sizeStr := size.String()
	titleBegin := ansi.Cyan(url)
	titleEnd := ansi.Green(sizeStr)
	remaining := size.Width - len(yAxisTitle) - len(url) - len(sizeStr)
	statsStr := stats.PickString(remaining)
	if len(statsStr) > 0 {
		statsStr = " [" + statsStr + "] "
	}
	title := titleBegin + statsStr + titleEnd
	titleIndent := (size.Width / 2) - (len(title) / 2)

	finalTitle := ansi.Home + ansi.Magenta(yAxisTitle) + ansi.CursorForward(titleIndent) + title
	if drawingDebug {
		finalTitle += ansi.CursorPosition(1, size.Width-1) + ansi.DarkRed(typography.LightBlock)
	}
	return finalTitle
}

type yAxis struct {
	size      int
	stats     *data.Stats
	labelSize int
}

type XAxisSpanInfo struct {
	spans     []*graphdata.SpanInfo
	spanStats *data.Stats
	pingStats *data.Stats
	timeSpan  *data.TimeSpan
	startX    int
	endX      int
	width     int
}

func computeXAxis(
	toWriteTo, toWriteSpanBars *bytes.Buffer,
	s terminal.Size,
	overall *data.TimeSpan,
	spans []*graphdata.SpanInfo,
) xAxis {
	padding := ansi.White(typography.Horizontal)
	origin := ansi.Magenta(typography.Bullet) + " "
	space := s.Width - 6
	remaining := space
	// First add the initial dot for A E S T H E T I C S
	fmt.Fprint(toWriteTo, ansi.CursorPosition(s.Height, 1)+origin+padding+padding+padding+padding)
	total := graphdata.Spans(spans).Count()
	xAxisSpans := combineSpansPixelWise(spans, space, total)

	// Now we need to iterate every "span", where a span is some pre-determined gap in the pings which is
	// considered so large that we are reasonably confident that it was another recording session.
	//
	// In each iteration, we must determine the time in which the span lives and how much terminal space it
	// should take up. And then the actual values so that we actually plot against this axis accurately.
	for _, span := range xAxisSpans {
		span.startX = s.Width - remaining

		start, times := span.timeSpan.FormatDraw(span.width, 2)
		if len(times) < 1 {
			toCrop := max(min(span.width-2, len(start)-1), 0)
			cropped := start[:toCrop]
			remaining -= len(cropped) + 2
			fmt.Fprintf(toWriteTo, "%s", ansi.Cyan(cropped))
			toWriteTo.WriteString(padding + padding)
		} else {
			remaining -= len(start) + 4 + 2
			fmt.Fprintf(toWriteTo, "[ %s ]", ansi.Cyan(start))
			toWriteTo.WriteString(padding + padding)
			remaining = xAxisDrawTimes(toWriteTo, times, remaining, padding)
		}

		span.endX = s.Width - remaining
	}
	// Finally we add these vertical bars to indicate that the axis is not continuous and a new graph is
	// starting.
	if len(xAxisSpans) > 1 {
		addYAxisVerticalSpanIndicator(toWriteSpanBars, s, xAxisSpans)
	}
	return xAxis{
		size:        s.Width,
		spans:       xAxisSpans,
		overallSpan: overall,
	}
}

// combineSpansPixelWise is a very crucial pre-processing step we need to do before drawing a frame, the data
// storage part [graphdata.GraphData] of the program will have made fairly sensible decisions about which
// parts of the data were actually recorded together. However this part of the program doesn't have the
// context about how much pixel real estate we can grant per recording session. Therefore we must do this
// every frame to determine which of this recording sessions must be merged for the sake of drawing. I.e. we
// have 5 recording sessions [*graphdata.SpanInfo], but the middle two are so short they would only take up 1
// pixel in the x-axis. This function has the agency to combine those middle spans when creating the
// [XAxisSpanInfo].
func combineSpansPixelWise(spans []*graphdata.SpanInfo, startingWidth, total int) []*XAxisSpanInfo {
	retSpans := make([]*XAxisSpanInfo, 0, len(spans))
	// TODO make this configurable - right now we just use a percentage of the start width or 5 when the
	// screen is small.
	minPixels := max(int(float64(startingWidth)*0.05), 5)
	acc := 0.0
	idx := 0
	for _, span := range spans {
		ratio := float64(span.Count) / (float64(total))
		width := int(float64(startingWidth) * ratio)
		if width >= minPixels && acc == 0.0 {
			retSpans = append(retSpans, &XAxisSpanInfo{
				spans:     []*graphdata.SpanInfo{span},
				spanStats: span.SpanStats,
				pingStats: span.PingStats,
				timeSpan:  span.TimeSpan,
				width:     width,
			})
			idx++
			continue
		}
		width = int(float64(startingWidth) * (acc + ratio))
		if width >= minPixels {
			retSpans[idx].spans = append(retSpans[idx].spans, span)
			retSpans[idx].spanStats = retSpans[idx].spanStats.Merge(span.SpanStats)
			retSpans[idx].pingStats = retSpans[idx].pingStats.Merge(span.PingStats)
			retSpans[idx].timeSpan = retSpans[idx].timeSpan.Merge(span.TimeSpan)
			retSpans[idx].width = width
			acc = 0.0
			idx++
			continue
		}
		if acc == 0.0 {
			retSpans = append(retSpans, &XAxisSpanInfo{
				spans:     []*graphdata.SpanInfo{span},
				spanStats: span.SpanStats,
				pingStats: span.PingStats,
				timeSpan:  span.TimeSpan,
			})
		} else {
			retSpans[idx].spans = append(retSpans[idx].spans, span)
			retSpans[idx].spanStats = retSpans[idx].spanStats.Merge(span.SpanStats)
			retSpans[idx].pingStats = retSpans[idx].pingStats.Merge(span.PingStats)
			retSpans[idx].timeSpan = retSpans[idx].timeSpan.Merge(span.TimeSpan)
		}
		acc += ratio
	}
	// TODO this width expanding finalizing still leaves some of the terminal unfilled, fix that.
	totalWidth := sliceutils.Fold(retSpans, 0, func(x *XAxisSpanInfo, acc int) int { return x.width + acc })
	delta := startingWidth - totalWidth
	toAdd := delta / len(retSpans)
	for _, span := range retSpans {
		span.width += toAdd
		totalWidth += toAdd
	}
	delta = startingWidth - totalWidth
	retSpans[len(retSpans)-1].width += delta
	return retSpans
}

var spanBar = ansi.Cyan(typography.DoubleVertical)

func addYAxisVerticalSpanIndicator(bars *bytes.Buffer, s terminal.Size, spans []*XAxisSpanInfo) {
	spanSeparator := makeBar(spanBar, s, true)
	// Don't draw the last span since this is implied by the end of the terminal
	for _, span := range spans[:len(spans)-1] {
		if span.endX >= (s.Width - 1) {
			continue
			// Don't draw on-top of the y-axis
		}
		bars.WriteString(ansi.CursorPosition(2, span.endX+1) + spanSeparator)
	}
	// Reset the cursor back to the start of the axis
	bars.WriteString(ansi.CursorPosition(s.Height, 1))
}

func xAxisDrawTimes(b *bytes.Buffer, times []string, remaining int, padding string) int {
	for _, point := range times {
		if remaining <= len(point) {
			break
		}
		b.WriteString(ansi.Yellow(point))
		remaining -= len(point)
		if remaining <= 1 {
			break
		}
		b.WriteString(padding)
		remaining--
		if remaining <= 1 {
			break
		}
		b.WriteString(padding)
		remaining--
	}
	return remaining
}

type xAxis struct {
	size        int
	spans       []*XAxisSpanInfo
	overallSpan *data.TimeSpan
}

type xAxisIter struct {
	*xAxis
	spanIndex int
}

func (x xAxis) NewIter() *xAxisIter {
	return &xAxisIter{
		xAxis:     &x,
		spanIndex: 0,
	}
}

func (x *xAxisIter) Get(p ping.PingDataPoint) *XAxisSpanInfo {
	currentSpan := x.spans[x.spanIndex]
	if currentSpan.timeSpan.Contains(p.Timestamp) {
		return currentSpan
	}
	x.spanIndex++
	return x.Get(p)
}

// paint knows how to composite the parts of a frame and the spinner, returning a lambda which will draw the
// computed frame to the given writer.
func paint(toDraw *drawbuffer.Collection) func(toWriteTo io.Writer) error {
	return func(toWriteTo io.Writer) error {
		// First clear the screen from the last frame
		err := utils.Err(toWriteTo.Write([]byte(ansi.Clear)))
		if err != nil {
			return err
		}
		// Now in paint order, simply forward the bytes onto the writer
		for _, i := range paintOrder {
			err = utils.Err(toWriteTo.Write(toDraw.Get(i).Bytes()))
			if err != nil {
				return err
			}
		}
		return nil
	}
}
