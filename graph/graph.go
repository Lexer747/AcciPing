// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/graph/terminal/ansi"
	"github.com/Lexer747/AcciPing/graph/terminal/typography"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/numeric"
)

type Graph struct {
	Term *terminal.Terminal

	sinkAlive   bool
	dataChannel chan ping.PingResults

	url            string
	pingsPerMinute float64

	data      *data.Data
	dataMutex *sync.Mutex
	lastFrame frame
}

func NewGraph(ctx context.Context, input chan ping.PingResults, t *terminal.Terminal, pingsPerMinute float64, url string) (*Graph, error) {
	g := &Graph{
		Term:           t,
		data:           data.NewData(),
		dataMutex:      &sync.Mutex{},
		dataChannel:    input,
		url:            url,
		pingsPerMinute: pingsPerMinute,
		sinkAlive:      true,
	}
	go g.sink(ctx)
	return g, nil
}

func (g *Graph) Run(ctx context.Context, stop context.CancelCauseFunc, fps int) error {
	timeBetweenFrames := getTimeBetweenFrames(fps, g.pingsPerMinute)
	frameRate := time.NewTicker(timeBetweenFrames)
	cleanup, err := g.Term.StartRaw(ctx, stop) // TODO add UI listeners, zooming, changing ping speed - etc
	defer cleanup()
	if err != nil {
		return err
	}
	for {
		if err = g.Term.UpdateCurrentTerminalSize(); err != nil {
			return err
		}
		toWrite := g.computeFrame(timeBetweenFrames, true)
		// Currently no strong opinions on dropped frames this is fine
		<-frameRate.C
		g.Term.Print(toWrite)
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		default:
		}
	}
}

func (g *Graph) AddPoint(p ping.PingResults) {
	g.dataMutex.Lock()
	defer g.dataMutex.Unlock()
	g.data.AddPoint(p)
}

func (g *Graph) LastFrame() string {
	g.dataMutex.Lock()
	defer g.dataMutex.Unlock()
	return paint(
		g.lastFrame.Size(),
		g.lastFrame.xAxis.axis,
		g.lastFrame.yAxis.axis,
		g.lastFrame.insideFrame,
		"",
	)
}
func (g *Graph) ComputeFrame() string {
	return g.computeFrame(0, false)
}

func (g *Graph) Summarize() string {
	g.dataMutex.Lock()
	defer g.dataMutex.Unlock()
	raw := ""
	for _, block := range g.data.Blocks {
		for _, p := range block.Raw {
			raw += p.String() + "\n"
		}
	}

	return raw + g.data.Header.String()
}

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
	innerFrame := computeInnerFrame(s, g.data)
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
	typography.LeftTriangle,
	typography.UpTriangle,
	typography.RightTriangle,
	typography.DownTriangle,
}

func spinner(s terminal.Size, i int, timeBetweenFrames time.Duration) string {
	// TODO refactor into a generic only paint me every X fps.
	// We want 300ms between spinner updates
	a := i
	x := timeBetweenFrames.Milliseconds()
	if x != 0 && int(300/x) != 0 {
		a = i / int(300/x)
	}
	return ansi.CursorPosition(1, s.Width-3) + ansi.Cyan(spinnerArray[a%len(spinnerArray)])
}

func (g *Graph) sink(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			g.sinkAlive = false
			return
		case p := <-g.dataChannel:
			g.dataMutex.Lock()
			g.data.AddPoint(p)
			g.dataMutex.Unlock()
		default:
		}
	}
}

type frame struct {
	PacketCount  int
	yAxis        yAxis
	xAxis        xAxis
	insideFrame  string
	spinnerIndex int
}

func (f frame) Match(s terminal.Size) bool {
	return f.xAxis.size == s.Width && f.yAxis.size == s.Height
}

func (f frame) Size() terminal.Size {
	return terminal.Size{Height: f.xAxis.size, Width: f.yAxis.size}
}

func gradientString(gradient float64, info *data.Header) string {
	normalized := numeric.Normalize(gradient, info.MinGradient, info.MaxGradient)
	return typography.Gradient(normalized)
}

func translate(s terminal.Size, p ping.PingResults, info *data.Header) (y, x int) {
	x = getX(p.Timestamp, info, s)
	y = int(numeric.NormalizeToRange(
		float64(p.Duration),
		float64(info.Stats.Min),
		float64(info.Stats.Max),
		float64(s.Height-1),
		2,
	))
	return
}

func getX(t time.Time, info *data.Header, s terminal.Size) int {
	timestamp := info.Span.End.Sub(t)
	return int(numeric.NormalizeToRange(
		float64(timestamp),
		0,
		float64(info.Span.Duration),
		float64(s.Width-1),
		13,
	))
}

var plain = ansi.LightGray(typography.Multiply)
var drop = ansi.Red(typography.Block)
var dropFiller = ansi.Red(typography.LightBlock)

func computeInnerFrame(s terminal.Size, d *data.Data) string {
	centreY := s.Height / 2
	centreX := s.Width / 2
	if d.TotalCount <= 1 {
		return ansi.CursorPosition(centreY, centreX) + plain + " " + d.Blocks[0].Raw[0].Duration.String()
	}
	ret := ""
	droppedBar := ""
	droppedFiller := ""
	if d.Header.Stats.PacketsDropped > 0 {
		droppedBar = strings.Repeat(drop+ansi.CursorDown(1)+ansi.CursorBack(1), s.Height-1)
		droppedFiller = strings.Repeat(dropFiller+ansi.CursorDown(1)+ansi.CursorBack(1), s.Height-1)
	}
	drawGradient := shouldGradient(s, d)
	lastWasDropped := false
	lastDroppedTerminalX := -1
	var lastGood *ping.PingResults
	lastGoodTerminalWidth := -1

	// Now iterate over all the individual data points and add them to the graph
	for bi, block := range d.Blocks {
		for i, p := range block.Raw {
			if p.Dropped() {
				x := getX(p.Timestamp, d.Header, s)
				ret += ansi.CursorPosition(2, x) + droppedBar
				if lastWasDropped {
					for i := min(lastDroppedTerminalX, x) + 1; i < max(lastDroppedTerminalX, x); i++ {
						ret += ansi.CursorPosition(2, i) + droppedFiller
					}
				}
				lastWasDropped = true
				lastDroppedTerminalX = x
				lastGood = nil
				continue
			}
			y, x := translate(s, p, d.Header)
			if drawGradient && lastGood != nil && !lastWasDropped {
				if !d.IsLast(bi, i-1) {
					x1 := d.GetGradient(bi, i-1)
					gradient := numeric.Normalize(x1, d.MinGradient, d.MaxGradient)
					gradientsToDraw := float64(numeric.Abs(lastGoodTerminalWidth - x))
					// stepSizeY := float64(p.Duration-lastGood.Duration) / gradientsToDraw
					// stepSizeX := float64(p.Timestamp.Sub(lastGood.Timestamp)) / gradientsToDraw
					// fmt.Printf("Drawing gradient %f, %f | %f chars | stepX %f | stepY %f",
					// 	x1, gradient, gradientsToDraw, stepSizeX, stepSizeY)
					for gi := 1.0; gi < gradientsToDraw-1; gi++ {
						x1 := numeric.NormalizeToRange(gi, 0, gradientsToDraw, float64(lastGood.Duration), float64(p.Duration))
						x2 := numeric.NormalizeToRange(gi, 0, gradientsToDraw, 0.0, float64(p.Timestamp.Sub(lastGood.Timestamp)))
						gy, gx := translate(s, ping.PingResults{
							Duration:  time.Duration(x1),
							Timestamp: lastGood.Timestamp.Add(time.Duration(x2)),
						}, d.Header)
						ret += ansi.CursorPosition(gy, gx) + ansi.Gray(typography.Gradient(gradient))
					}
				}
			}
			ret += drawPoint(p, d, x, y, centreX)
			lastWasDropped = false
			lastGood = &p
			lastGoodTerminalWidth = x
		}
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
		return ansi.CursorPosition(y, x-(len(label)+2)) + ansi.Green(label+" "+typography.UpTriangle)
	case isMin:
		return ansi.CursorPosition(y, x) + ansi.Green(typography.UpTriangle+" "+p.Duration.String())
	case isMax && leftJustify:
		label := p.Duration.String()
		return ansi.CursorPosition(y, x-(len(label)+2)) + ansi.Red(label+" "+typography.DownTriangle)
	case isMax:
		return ansi.CursorPosition(y, x) + ansi.Red(typography.DownTriangle+" "+p.Duration.String())
	default:
		return ansi.CursorPosition(y, x) + plain
	}
}

func shouldGradient(s terminal.Size, d *data.Data) bool {
	return false
	// TODO account for dropped packets in these positions
	b := d.Blocks[0]
	first := getX(b.Raw[0].Timestamp, d.Header, s)
	second := getX(b.Raw[1].Timestamp, d.Header, s)
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

	for i := range size.Height - 2 {
		h := i + 2
		fmt.Fprint(&b, ansi.CursorPosition(h, 1))
		if i%gapSize == 1 {
			// TODO shorten these when low width terminal
			scaledDuration := numeric.NormalizeToRange(float64(i), float64(size.Height-2), 0, float64(stats.Min), float64(stats.Max))
			toPrint := time.Duration(scaledDuration)
			fmt.Fprint(&b, ansi.Yellow(toPrint.String()))
		} else {
			fmt.Fprint(&b, ansi.White(typography.Vertical))
		}
	}
	return yAxis{
		size:  size.Height,
		stats: stats,
		axis:  b.String(),
	}
}

func makeTitle(size terminal.Size, stats *data.Stats, url string) string {
	// TODO string builder, or larger buffer impl
	sizeStr := size.String()
	titleBegin := ansi.Cyan(url)
	titleEnd := ansi.Green(sizeStr)
	remaining := size.Width - 7 - len(url) - len(sizeStr)
	statsStr := stats.PickString(remaining)
	if len(statsStr) > 0 {
		statsStr = " [" + statsStr + "] "
	}
	title := titleBegin + statsStr + titleEnd
	titleIndent := (size.Width / 2) - (len(title) / 2)
	finalTitle := ansi.Home + ansi.Magenta("Latency") + ansi.CursorForward(titleIndent) + title
	return finalTitle
}

type yAxis struct {
	size  int
	stats *data.Stats
	axis  string
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
		x := t.Format(format)
		if len(x) < formatLen {
			x += "0"
		}
		fmt.Fprint(&b, padding+" "+ansi.Yellow(x)+" "+padding)
		remaining -= spacePerItem
	}
	if remaining > 0 {
		// TODO also put some chars at the beginning of the axis
		final := strings.Repeat(typography.Horizontal, remaining)
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
