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
	"github.com/Lexer747/AcciPing/utils/numeric"
	"github.com/Lexer747/AcciPing/utils/timeutils"
)

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

	x := computeXAxis(s.Width, g.data.LockFreeHeader().TimeSpan, g.data.LockFreeSpanInfos())
	y := computeYAxis(s, g.data.LockFreeHeader().Stats, g.data.LockFreeURL())
	innerFrame := computeInnerFrame(g.data.LockFreeIter(), g.data.LockFreeRuns(), x, y, s)
	// Everything we need is now cached we can unlock a bit early while we tidy up for the next frame
	g.data.Unlock()
	finished := paint(s, x.axis, y.axis, innerFrame, spinnerValue)
	g.lastFrame = frame{
		PacketCount:  count,
		yAxis:        y,
		xAxis:        x,
		insideFrame:  innerFrame,
		spinnerIndex: g.lastFrame.spinnerIndex,
	}
	return finished
}

func translate(p ping.PingDataPoint, xAxis xAxis, yAxis yAxis, s terminal.Size) (y, x int) {
	x = getX(p.Timestamp, xAxis, yAxis, s)
	y = getY(p.Duration, yAxis, s)
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

func getX(t time.Time, xAxis xAxis, yAxis yAxis, s terminal.Size) int {
	// TODO this needs fixing
	timestamp := xAxis.overallSpan.End.Sub(t)
	return int(numeric.NormalizeToRange(
		float64(timestamp),
		0,
		float64(xAxis.overallSpan.Duration),
		float64(s.Width-1),
		float64(yAxis.labelSize),
	))
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

var plain = ansi.White(typography.Multiply)
var drop = ansi.Red(typography.Block)
var dropFiller = ansi.Red(typography.LightBlock)

func computeInnerFrame(
	iter *graphdata.Iter,
	runs *data.Runs,
	xAxis xAxis,
	yAxis yAxis,
	s terminal.Size,
) string {
	if iter.Total < 1 {
		return ""
	}
	centreY := (s.Height - 2) / 2
	centreX := (s.Width - 2) / 2
	if iter.Total == 1 {
		return ansi.CursorPosition(centreY, centreX) + plain + " " + iter.Get(0).Duration.String()
	}
	ret := ""
	droppedBar, droppedFiller := makeDroppedPacketIndicators(yAxis.stats.PacketsDropped, s)

	// Now iterate over all the individual data points and add them to the graph

	if shouldGradient(runs) {
		ret += drawGradients(iter, xAxis, yAxis, s)
	}

	lastWasDropped := false
	lastDroppedTerminalX := -1
	for i := range iter.Total {
		p := iter.Get(i)
		x := getX(p.Timestamp, xAxis, yAxis, s)
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
		ret += drawPoint(p, yAxis.stats, x, y, centreX)
	}

	return ret
}

func drawGradients(
	iter *graphdata.Iter,
	xAxis xAxis, yAxis yAxis,
	s terminal.Size,
) string {
	ret := ""
	g := gradientState{}
	for i := range iter.Total {
		p := iter.Get(i)
		if p.Dropped() {
			g = g.dropped()
			continue
		}
		y, x := translate(p, xAxis, yAxis, s)
		if g.draw() && !iter.IsLast(i) {
			ret += drawGradient(
				xAxis, yAxis,
				x, y,
				p,
				iter.Get(g.lastGoodIndex),
				g.lastGoodTerminalWidth,
				g.lastGoodTerminalHeight,
				s,
			)
		}
		g = g.set(i, x, y)
	}
	return ret
}

func makeDroppedPacketIndicators(droppedPackets uint64, s terminal.Size) (string, string) {
	droppedBar := ""
	droppedFiller := ""
	if droppedPackets > 0 {
		droppedBar = strings.Repeat(drop+ansi.CursorDown(1)+ansi.CursorBack(1), s.Height-3)
		droppedFiller = strings.Repeat(dropFiller+ansi.CursorDown(1)+ansi.CursorBack(1), s.Height-3)
	}
	return droppedBar, droppedFiller
}

func drawGradient(
	xAxis xAxis,
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

func drawPoint(p ping.PingDataPoint, stats *data.Stats, x, y, centreX int) string {
	leftJustify := x > centreX
	isMin := p.Duration == stats.Min
	isMax := p.Duration == stats.Max
	switch {
	case isMin && leftJustify:
		label := p.Duration.String()
		return ansi.CursorPosition(y, x-len(label)) + ansi.Green(label+" "+typography.UpTriangle)
	case isMin:
		return ansi.CursorPosition(y, x) + ansi.Green(typography.UpTriangle+" "+p.Duration.String())
	case isMax && leftJustify:
		label := p.Duration.String()
		return ansi.CursorPosition(y, x-len(label)) + ansi.Red(label+" "+typography.DownTriangle)
	case isMax:
		return ansi.CursorPosition(y, x) + ansi.Red(typography.DownTriangle+" "+p.Duration.String())
	default:
		return ansi.CursorPosition(y, x) + plain
	}
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
			x := time.Duration(scaledDuration)
			toPrint := timeutils.HumanString(x, durationSize)
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
	return finalTitle
}

type yAxis struct {
	size      int
	stats     *data.Stats
	axis      string
	labelSize int
}

func computeXAxis(size int, overall *data.TimeSpan, spans []*graphdata.SpanInfo) xAxis {
	padding := ansi.White(typography.Horizontal)
	origin := ansi.Magenta(typography.Bullet) + " "
	remaining := size - 2
	var b strings.Builder
	fmt.Fprintf(&b, origin+padding)
	for span := range spans {

	}
	return xAxis{
		size:        size,
		spans:       spans,
		overallSpan: overall,
		axis:        b.String(),
	}
}

type xAxis struct {
	size        int
	spans       []*graphdata.SpanInfo
	overallSpan *data.TimeSpan
	axis        string
}

// paint knows how to composite the parts of a frame and the spinner
func paint(size terminal.Size, x, y, lines, spinner string) string {
	ret := ansi.Clear
	ret += lines + y
	ret += ansi.CursorPosition(size.Height, 1)
	ret += x
	ret += spinner
	return ret
}
