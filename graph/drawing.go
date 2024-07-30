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
	g.dataMutex.Lock()
	count := g.data.TotalCount
	if count == 0 {
		g.dataMutex.Unlock()
		return "" // no data yet
	}
	spinnerValue := ""
	if drawSpinner {
		g.lastFrame.spinnerIndex++
		spinnerValue = spinner(s, g.lastFrame.spinnerIndex, timeBetweenFrames)
	}
	if count == g.lastFrame.PacketCount && g.lastFrame.Match(s) {
		g.dataMutex.Unlock() // fast path the frame didn't change
		return spinnerValue
	}

	x := computeXAxis(s.Width, g.data.Header.Span)
	y := computeYAxis(s, g.data.Header.Stats, g.url)
	innerFrame := computeInnerFrame(s, g.data, y)
	// Everything we need is now cached we can unlock a bit early while we tidy up for the next frame
	g.dataMutex.Unlock()
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

var spinnerArray = [...]string{
	typography.UpperLeftQuadrantCircularArc,
	typography.UpperRightQuadrantCircularArc,
	typography.LowerRightQuadrantCircularArc,
	typography.LowerLeftQuadrantCircularArc,
}

func spinner(s terminal.Size, i int, timeBetweenFrames time.Duration) string {
	// TODO refactor into a generic only paint me every X fps.
	// We want 200ms between spinner updates
	a := i
	x := timeBetweenFrames.Milliseconds()
	if x != 0 && int(200/x) != 0 {
		a = i / int(200/x)
	}
	return ansi.CursorPosition(1, s.Width-3) + ansi.Cyan(spinnerArray[a%len(spinnerArray)])
}

func translate(s terminal.Size, p ping.PingResults, info *data.Header, labelSize int) (y, x int) {
	x = getX(p.Timestamp, info, s, labelSize)
	y = getY(p.Duration, info, s)
	return
}

func getY(dur time.Duration, info *data.Header, s terminal.Size) int {
	return int(numeric.NormalizeToRange(
		float64(dur),
		float64(info.Stats.Min),
		float64(info.Stats.Max),
		float64(s.Height-1),
		2,
	))
}

func getX(t time.Time, info *data.Header, s terminal.Size, labelSize int) int {
	timestamp := info.Span.End.Sub(t)
	return int(numeric.NormalizeToRange(
		float64(timestamp),
		0,
		float64(info.Span.Duration),
		float64(s.Width-1),
		float64(labelSize),
	))
}

type gradientState struct {
	lastGoodIndex          int
	lastGoodTerminalWidth  int
	lastGoodTerminalHeight int
}

func (g gradientState) set(i, x, y int) gradientState {
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

func computeInnerFrame(s terminal.Size, d *data.Data, yAxis yAxis) string {
	centreY := s.Height / 2
	centreX := s.Width / 2
	if d.TotalCount <= 1 {
		return ansi.CursorPosition(centreY, centreX) + plain + " " + d.Blocks[0].Raw[0].Duration.String()
	}
	ret := ""
	droppedBar, droppedFiller := makeDroppedPacketIndicators(d, s)

	// Now iterate over all the individual data points and add them to the graph

	if shouldGradient(s, d, yAxis.labelSize) {
		ret += drawGradients(d, s, yAxis)
	}

	lastWasDropped := false
	lastDroppedTerminalX := -1
	for _, block := range d.Blocks {
		for _, p := range block.Raw {
			x := getX(p.Timestamp, d.Header, s, yAxis.labelSize)
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
			y := getY(p.Duration, d.Header, s)
			ret += drawPoint(p, d, x, y, centreX)
		}
	}

	return ret
}

func drawGradients(d *data.Data, s terminal.Size, yAxis yAxis) string {
	ret := ""
	g := gradientState{}
	for bi, block := range d.Blocks {
		for i, p := range block.Raw {
			if p.Dropped() {
				g = g.dropped()
				continue
			}
			y, x := translate(s, p, d.Header, yAxis.labelSize)
			if g.draw() && !d.IsLast(bi, i-1) {
				ret += drawGradient(
					d.Header, x, y, p, s, yAxis.labelSize,
					block.Raw[g.lastGoodIndex], g.lastGoodTerminalWidth, g.lastGoodTerminalHeight,
				)
			}
			g = g.set(i, x, y)
		}
	}
	return ret
}

func makeDroppedPacketIndicators(d *data.Data, s terminal.Size) (string, string) {
	droppedBar := ""
	droppedFiller := ""
	if d.Header.Stats.PacketsDropped > 0 {
		droppedBar = strings.Repeat(drop+ansi.CursorDown(1)+ansi.CursorBack(1), s.Height-2)
		droppedFiller = strings.Repeat(dropFiller+ansi.CursorDown(1)+ansi.CursorBack(1), s.Height-2)
	}
	return droppedBar, droppedFiller
}

func drawGradient(
	header *data.Header,
	x, y int,
	current ping.PingResults,
	s terminal.Size,
	labelSize int,
	lastGood ping.PingResults,
	lastGoodTerminalWidth int,
	lastGoodTerminalHeight int,
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
		p := ping.PingResults{Duration: interpolatedDuration, Timestamp: interpolatedStamp}
		cursorY, cursorX := translate(s, p, header, labelSize)
		pointsX = append(pointsX, cursorX)
		pointsY = append(pointsY, cursorY)
	}
	gradient := solve(pointsX, pointsY)
	for i, g := range gradient {
		ret += ansi.CursorPosition(pointsY[i], pointsX[i]) + ansi.Gray(g)
	}
	return ret
}

func drawPoint(p ping.PingResults, d *data.Data, x, y, centreX int) string {
	leftJustify := x > centreX
	isMin := p.Duration == d.Stats.Min
	isMax := p.Duration == d.Stats.Max
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

func shouldGradient(s terminal.Size, d *data.Data, labelSize int) bool {
	// TODO account for dropped packets in these positions
	b := d.Blocks[0]
	first := getX(b.Raw[0].Timestamp, d.Header, s, labelSize)
	second := getX(b.Raw[1].Timestamp, d.Header, s, labelSize)
	return numeric.Abs(first-second) > 0
}

func computeYAxis(size terminal.Size, stats *data.Stats, url string) yAxis {
	var b strings.Builder
	// Making of a buffer of [size] will be too small because ansi + unicode will take up more bytes than the
	// character space they take up
	b.Grow(size.Height * 2)

	finalTitle := makeTitle(size, stats, url)
	fmt.Fprint(&b, finalTitle)

	gapSize := 3
	if size.Height > 20 {
		gapSize++
	} else if size.Height < 12 {
		gapSize--
	}
	durationSize := (gapSize * 3) / 2

	for i := range size.Height - 2 {
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
	return yAxis{
		size:      size.Height,
		stats:     stats,
		axis:      b.String(),
		labelSize: durationSize + 4,
	}
}

func makeTitle(size terminal.Size, stats *data.Stats, url string) string {
	// TODO string builder, or larger buffer impl
	const yAxisTitle = "Latency "
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

func computeXAxis(size int, span *data.TimeSpan) xAxis {
	const format = "15:04:05.99"
	const formatLen = 11
	const spacePerItem = formatLen + 6
	padding := ansi.White(typography.Horizontal + typography.Horizontal)
	var b strings.Builder
	// Making of a buffer of [size] will be too small because ansi + unicode will take up more bytes than the
	// character space they take up
	b.Grow(size * 2)
	fmt.Fprint(&b, ansi.Magenta(typography.Bullet)+" ")
	remaining := size - 2
	toPrint := max(remaining/spacePerItem, 1)
	durationGap := span.Duration / time.Duration(toPrint)
	// TODO don't repeat durations
	for i := range toPrint {
		t := span.Begin.Add(durationGap * time.Duration(i))
		timeStamp := t.Format(format)
		if len(timeStamp) < formatLen {
			if len(timeStamp) == 8 {
				timeStamp += ".00"
			} else if len(timeStamp) == 10 {
				timeStamp += "0"
			} else if len(timeStamp) == 9 {
				timeStamp += "00"
			}
		}
		fmt.Fprint(&b, padding+" "+ansi.Yellow(timeStamp)+" "+padding)
		remaining -= spacePerItem
	}
	if remaining > 1 {
		// TODO also put some chars at the beginning of the axis
		final := strings.Repeat(typography.Horizontal, remaining-1)
		fmt.Fprint(&b, ansi.White(final))
	}
	return xAxis{
		size:     size,
		spanBase: span,
		axis:     b.String(),
	}
}

type xAxis struct {
	size     int
	spanBase *data.TimeSpan
	axis     string
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
