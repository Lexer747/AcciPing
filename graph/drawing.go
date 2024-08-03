// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/graphdata"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/graph/terminal/ansi"
	"github.com/Lexer747/AcciPing/graph/terminal/typography"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/check"
	"github.com/Lexer747/AcciPing/utils/numeric"
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

// TODO compute the frame into an existing buffer instead of a string API
func (g *Graph) computeFrame(timeBetweenFrames time.Duration, drawSpinner bool) string {
	s := g.Term.Size() // This is race-y so ensure a consistent size for rendering
	g.data.Lock()
	count := g.data.LockFreeTotalCount()
	if count == 0 {
		g.data.Unlock()
		return "" // no data yet
	}
	spinnerValue := ""
	if drawSpinner {
		g.lastFrame.spinnerIndex++
		spinnerValue = spinner(s, g.lastFrame.spinnerIndex, timeBetweenFrames)
	}
	if count == g.lastFrame.PacketCount && g.lastFrame.Match(s) {
		g.data.Unlock() // fast path the frame didn't change
		return spinnerValue
	}

	x := computeXAxis(s, g.data.LockFreeHeader().TimeSpan, g.data.LockFreeSpanInfos())
	y := computeYAxis(s, g.data.LockFreeHeader().Stats, g.data.LockFreeURL())
	innerFrame, key := computeInnerFrame(g.data.LockFreeIter(), g.data.LockFreeRuns(), x, y, s)
	// Everything we need is now cached we can unlock a bit early while we tidy up for the next frame
	g.data.Unlock()
	finished := paint(s, x.bars, x.axis, y.axis, key, innerFrame, spinnerValue)
	g.lastFrame = frame{
		PacketCount:  count,
		yAxis:        y,
		xAxis:        x,
		insideFrame:  innerFrame,
		key:          key,
		spinnerIndex: g.lastFrame.spinnerIndex,
	}
	return finished
}

func translate(p ping.PingDataPoint, x XAxisSpanInfo, y yAxis, s terminal.Size) (yCord, xCord int) {
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

func getX(t time.Time, span XAxisSpanInfo, yAxis yAxis, s terminal.Size) int {
	timestamp := span.SpanInfo.TimeSpan.End.Sub(t)
	x := min(s.Width, yAxis.labelSize+span.startX)
	x1 := min(s.Width, span.endX)
	x2 := int(numeric.NormalizeToRange(
		float64(timestamp),
		0,
		float64(span.SpanInfo.TimeSpan.Duration),
		float64(x1),
		float64(x),
	))

	check.Checkf(x2 <= s.Width, "Computed coord out of bounds (%d) vs %q", x2, s.String())
	return x2
}

type gradientState struct {
	lastGoodIndex          int64
	lastGoodTerminalWidth  int
	lastGoodTerminalHeight int
}

func (g gradientState) set(i int64, x, y int) gradientState {
	return gradientState{
		lastGoodIndex:          i,
		lastGoodTerminalWidth:  x,
		lastGoodTerminalHeight: y,
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

func computeInnerFrame(
	iter *graphdata.Iter,
	runs *data.Runs,
	xAxis xAxis,
	yAxis yAxis,
	s terminal.Size,
) (string, string) {
	if iter.Total < 1 {
		return "", ""
	}
	centreY := (s.Height - 2) / 2
	centreX := (s.Width - 2) / 2
	if iter.Total == 1 {
		point, _ := iter.Get(0)
		return ansi.CursorPosition(centreY, centreX) + single + " " + point.Duration.String(), ""
	}
	ret := ""
	droppedBar, droppedFiller := makeDroppedPacketIndicators(yAxis.stats.PacketsDropped, s)

	// Now iterate over all the individual data points and add them to the graph

	if shouldGradient(runs) {
		ret += drawGradients(iter, xAxis, yAxis, s)
	}

	lastWasDropped := false
	lastDroppedTerminalX := -1
	window := newDrawWindow(iter.Total)
	for i := range iter.Total {
		p, spanIndex := iter.Get(i)
		span := xAxis.spans[spanIndex]
		x := getX(p.Timestamp, span, yAxis, s)
		if p.Dropped() {
			ret += ansi.CursorPosition(2, x) + droppedBar
			if lastWasDropped {
				for i := min(lastDroppedTerminalX, x) + 1; i < max(lastDroppedTerminalX, x); i++ {
					ret += ansi.CursorPosition(2, i) + droppedFiller
				}
			}
			lastWasDropped = true
			lastDroppedTerminalX = x
			continue
		}
		lastWasDropped = false
		y := getY(p.Duration, yAxis, s)
		ret += window.drawPoint(p, span.PingStats, yAxis.stats, x, y, centreX)
	}
	key := window.getKey()
	key = ansi.CursorPosition(s.Height-1, yAxis.labelSize+1) + key

	return ret, key
}

func drawGradients(
	iter *graphdata.Iter,
	xAxis xAxis, yAxis yAxis,
	s terminal.Size,
) string {
	ret := ""
	g := gradientState{}
	for i := range iter.Total {
		p, spanIndex := iter.Get(i)
		if p.Dropped() {
			g = g.dropped()
			continue
		}
		span := xAxis.spans[spanIndex]
		y, x := translate(p, span, yAxis, s)
		if g.draw() && !iter.IsLast(i) {
			lastP, lastSpanIndex := iter.Get(g.lastGoodIndex)
			if lastSpanIndex == spanIndex {
				ret += drawGradient(
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
		g = g.set(i, x, y)
	}
	return ret
}

func makeDroppedPacketIndicators(droppedPackets uint64, s terminal.Size) (string, string) {
	droppedBar := ""
	droppedFillerBar := ""
	if droppedPackets > 0 {
		droppedBar = makeBar(drop, s)
		droppedFillerBar = makeBar(dropFiller, s)
	}
	return droppedBar, droppedFillerBar
}

// A bar requires you start at the top of the terminal, general to draw a bar at coord x, do
// [ansi.CursorPosition(2, x)] before writing the bar.
func makeBar(repeating string, s terminal.Size) string {
	return strings.Repeat(repeating+ansi.CursorDown(1)+ansi.CursorBack(1), s.Height-2)
}

func drawGradient(
	xAxis XAxisSpanInfo,
	yAxis yAxis,
	x, y int,
	current ping.PingDataPoint,
	lastGood ping.PingDataPoint,
	lastGoodTerminalWidth int,
	lastGoodTerminalHeight int,
	s terminal.Size,
) string {
	ret := ""
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
		ret += ansi.CursorPosition(pointsY[i], pointsX[i]) + ansi.Gray(g)
	}
	return ret
}

func shouldGradient(runs *data.Runs) bool {
	return runs.GoodPackets.Longest > 2
}

func computeYAxis(size terminal.Size, stats *data.Stats, url string) yAxis {
	var b strings.Builder
	b.Grow(size.Height)

	finalTitle := makeTitle(size, stats, url)
	fmt.Fprint(&b, finalTitle)

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
		fmt.Fprint(&b, ansi.CursorPosition(h, 1))
		if i%gapSize == 1 {
			scaledDuration := numeric.NormalizeToRange(float64(i), float64(size.Height-2), 0, float64(stats.Min), float64(stats.Max))
			toPrint := timeutils.HumanString(time.Duration(scaledDuration), durationSize)
			fmt.Fprint(&b, ansi.Yellow(toPrint))
		} else {
			fmt.Fprint(&b, ansi.White(typography.Vertical))
		}
	}
	// Last line is always a bar
	fmt.Fprint(&b, ansi.CursorPosition(size.Height-1, 1)+ansi.White(typography.Vertical))
	return yAxis{
		size:      size.Height,
		stats:     stats,
		axis:      b.String(),
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
	axis      string
	labelSize int
}

type XAxisSpanInfo struct {
	*graphdata.SpanInfo
	startX int
	endX   int
}

func computeXAxis(s terminal.Size, overall *data.TimeSpan, spans []*graphdata.SpanInfo) xAxis {
	padding := ansi.White(typography.Horizontal)
	origin := ansi.Magenta(typography.Bullet) + " "
	space := s.Width - 3
	remaining := space
	var b strings.Builder
	// First add the initial dot for A E S T H E T I C S
	fmt.Fprint(&b, origin+padding)
	total := graphdata.Spans(spans).Count()
	retSpans := make([]XAxisSpanInfo, len(spans))

	// Now we need to iterate every "span", where a span is some pre-determined gap in the pings which is
	// considered so large that we are reasonably confident that it was another recording session.
	//
	// In each iteration, we must determine the time in which the span lives and how much terminal space it
	// should take up. And then the actual values so that we actually plot against this axis accurately.
	for i, span := range spans {
		toAdd := XAxisSpanInfo{
			SpanInfo: span,
			startX:   space - remaining,
		}
		ratio := float64(span.Count) / (float64(total))
		// TODO this way of working out how much space to take up is very flawed
		start, times := span.TimeSpan.FormatDraw(int(float64(space)*ratio), 2)
		if remaining <= 0 {
			toAdd.endX = max(space, toAdd.startX)
			retSpans[i] = toAdd
			continue
		} else if len(start)+6 >= remaining {
			trimmed := start[:min(len(start)-1, remaining)]
			if len(trimmed) > 5 {
				trimmed = trimmed[:len(trimmed)-5]
				fmt.Fprintf(&b, "[ %s ]", ansi.Cyan(trimmed))
				remaining -= len(trimmed) + 4
			} else {
				remaining -= len(trimmed)
			}
		} else {
			remaining -= len(start) + 4 + 2
			fmt.Fprintf(&b, "[ %s ]", ansi.Cyan(start))
			b.WriteString(padding + padding)
			remaining = xAxisDrawTimes(&b, times, remaining, padding)
		}
		toAdd.endX = space - remaining
		retSpans[i] = toAdd
	}
	var bars strings.Builder
	// Finally we add these vertical bars to indicate that the axis is not continuous and a new graph is
	// starting.
	if len(retSpans) > 1 {
		addYAxisVerticalSpanIndicator(s, retSpans, &bars)
	}
	return xAxis{
		size:        s.Width,
		spans:       retSpans,
		overallSpan: overall,
		axis:        b.String(),
		bars:        bars.String(),
	}
}

func addYAxisVerticalSpanIndicator(s terminal.Size, spans []XAxisSpanInfo, bars *strings.Builder) {
	spanSeparator := makeBar(ansi.Cyan(typography.DoubleVertical), s)
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

func xAxisDrawTimes(b *strings.Builder, times []string, remaining int, padding string) int {
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
	spans       []XAxisSpanInfo
	overallSpan *data.TimeSpan
	axis        string
	bars        string
}

func (x xAxis) GetSpan(t time.Time) (XAxisSpanInfo, bool) {
	// TODO binary search instead
	for _, span := range x.spans {
		check.NotNilf(span.SpanInfo, "span.SpanInfo missing: x: %+v, t:%+v", x, t)
		check.NotNilf(span.SpanInfo.TimeSpan, "span.SpanInfo.TimeSpan: x: %+v, t:%+v", x, t)
		if span.SpanInfo.TimeSpan.Contains(t) {
			return span, true
		}
	}
	return XAxisSpanInfo{}, false
}

// paint knows how to composite the parts of a frame and the spinner
func paint(size terminal.Size, bars, x, y, key, frame, spinner string) string {
	ret := ansi.Clear

	// Z-order is top to bottom so the first item added to ret is at the back, the last item is at the front
	ret += bars // bars should be overwritten by data and axis
	ret += frame
	ret += y
	ret += ansi.CursorPosition(size.Height, 1)
	ret += x
	ret += key
	ret += spinner
	return ret
}
