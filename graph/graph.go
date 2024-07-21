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

	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/graph/terminal/ansi"
	"github.com/Lexer747/AcciPing/graph/terminal/typography"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/errors"
	"github.com/Lexer747/AcciPing/utils/numeric"
)

type Graph struct {
	term        *terminal.Terminal
	data        *Data
	dataMutex   *sync.Mutex
	dataChannel chan ping.PingResults

	url string

	pingsPerMinute float64

	lastFrame frame
}

func NewGraph(ctx context.Context, input chan ping.PingResults, pingsPerMinute float64, url string) (*Graph, error) {
	t, err := terminal.NewTerminal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make graph")
	}
	g := &Graph{
		term:           t,
		data:           NewData(),
		dataMutex:      &sync.Mutex{},
		dataChannel:    input,
		url:            url,
		pingsPerMinute: pingsPerMinute,
	}
	go g.sink(ctx)
	return g, nil
}

func (g *Graph) Run(ctx context.Context, stop context.CancelCauseFunc, fps int) error {
	var frameRate *time.Ticker
	if fps == 0 {
		frameRate = time.NewTicker(ping.PingsPerMinuteToDuration(g.pingsPerMinute))
	} else {
		frameRate = time.NewTicker(time.Duration(1000/fps) * time.Millisecond)
	}
	cleanup, err := g.term.StartRaw(ctx, stop) // TODO add UI listeners, zooming, changing ping speed - etc
	defer cleanup()
	if err != nil {
		return err
	}
	for {
		if err = g.term.UpdateCurrentTerminalSize(); err != nil {
			return err
		}
		toWrite := g.computeFrame()
		// Currently no strong opinions on dropped frames this is fine
		<-frameRate.C
		g.term.Print(toWrite)
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		default:
		}
	}
}

func (g *Graph) Summarize() string {
	g.dataMutex.Lock()
	defer g.dataMutex.Unlock()
	return g.data.Header.String()
}

// TODO compute the frame into an existing buffer instead of a string API
func (g *Graph) computeFrame() string {
	s := g.term.Size() // This is race-y so ensure a consistent size for rendering
	g.dataMutex.Lock()
	count := g.data.TotalCount
	if count == 0 {
		g.dataMutex.Unlock()
		return "" // no data yet
	}
	// if count == g.lastFrame.PacketCount && g.lastFrame.Match(s) {
	// 	g.dataMutex.Unlock() // fast path the frame didn't change
	// 	return g.lastFrame.fullFrame
	// }

	x := computeXAxis(s.Width, g.data.Header.Span)
	y := computeYAxis(s, g.data.Header.Stats, g.url)
	innerFrame := computeInnerFrame(s, g.data)
	// Everything we need is now cached we can unlock a bit early while we tidy up for the next frame
	g.dataMutex.Unlock()
	finished := paint(s, x.axis, y.axis, innerFrame)
	g.lastFrame = frame{
		PacketCount: count,
		yAxis:       y,
		xAxis:       x,
		insideFrame: innerFrame,
		fullFrame:   finished,
	}
	return finished
}

func (g *Graph) sink(ctx context.Context) {
	for {
		p := <-g.dataChannel
		g.dataMutex.Lock()
		g.data.AddPoint(p)
		g.dataMutex.Unlock()
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

type frame struct {
	PacketCount int
	yAxis       yAxis
	xAxis       xAxis
	insideFrame string
	fullFrame   string
}

func (f frame) Match(s terminal.Size) bool {
	return f.xAxis.size == s.Width && f.yAxis.size == s.Height
}

var point = ansi.LightGray(typography.Diamond)

func gradientString(gradient float64, info *Header) string {
	normalized := numeric.Normalize(gradient, info.MinGradient, info.MaxGradient)
	return typography.Gradient(normalized)
}

func translate(s terminal.Size, p ping.PingResults, info *Header) (row, column int) {
	timestamp := info.Span.End.Sub(p.Timestamp)
	column = int(numeric.NormalizeToRange(
		float64(timestamp),
		0,
		float64(info.Span.Duration),
		float64(s.Width-1),
		13,
	))
	row = int(numeric.NormalizeToRange(
		float64(p.Duration),
		float64(info.Stats.Min),
		float64(info.Stats.Max),
		2,
		float64(s.Height-1),
	))
	// fmt.Printf("START %s | %s | row %d (%+v, %+v, %+v, %+v, %+v) | column %d (%+v, %+v, %+v, %+v, %+v) END ",
	// 	s.String(), p.String(),
	// 	row, timestamp, 0, info.Span.Duration, 1, s.Width,
	// 	column, p.Duration, info.Stats.Min, info.Stats.Max, 1, s.Height,
	// )
	return
}

func computeInnerFrame(s terminal.Size, d *Data) string {
	centreRow := s.Width / 2
	centreColumn := s.Height / 2
	if d.TotalCount <= 1 {
		return ansi.CursorPosition(centreRow, centreColumn) + point
	}
	ret := ""
	for _, block := range d.Blocks {
		for _, p := range block.Raw {
			if p.Error != nil {
				// dropped packet
				panic("implement dropped packets graphing")
			} else {
				row, column := translate(s, p, d.Header)
				ret += ansi.CursorPosition(row, column) + point
			}
		}
	}
	return ret
}

func computeYAxis(size terminal.Size, stats *Stats, url string) yAxis {
	var b strings.Builder
	// Making of a buffer of [size] will be too small because ansi + unicode will take up more bytes than the
	// character space they take up
	b.Grow(size.Height * 2)

	title := ansi.Cyan(url) + " [" + stats.String() + "] " + ansi.Green(size.String())
	titleIndent := (size.Width / 2) - (len(title) / 2)
	// TODO crop the title if it wont fit
	fmt.Fprint(&b, ansi.Home+ansi.Magenta("Latency")+ansi.CursorForward(titleIndent)+title)

	gapSize := 3
	if size.Height > 20 {
		gapSize++
	} else if size.Height < 12 {
		gapSize--
	}
	durationGap := (stats.Max - stats.Min) / time.Duration(size.Height/gapSize)
	for i := range size.Height - 2 {
		h := i + 2
		fmt.Fprint(&b, ansi.CursorPosition(h, 1))
		if i%gapSize == 1 {
			// TODO shorten these when low width terminal
			toPrint := stats.Max - (durationGap * time.Duration(i))
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

type yAxis struct {
	size  int
	stats *Stats
	axis  string
}

func computeXAxis(size int, span *TimeSpan) xAxis {
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
	toPrint := remaining / spacePerItem
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
	spanBase *TimeSpan
	axis     string
}

func paint(size terminal.Size, x string, y string, lines string) string {
	ret := ansi.Clear
	ret += lines + y
	ret += ansi.CursorPosition(size.Height, 1)
	ret += x
	return ret
}
