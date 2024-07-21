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
	frameRate := time.NewTicker(time.Duration(1000/fps) * time.Millisecond)
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
	if count == g.lastFrame.PacketCount && g.lastFrame.Match(s) {
		g.dataMutex.Unlock() // fast path the frame didn't change
		return g.lastFrame.fullFrame
	}

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

func computeInnerFrame(_ terminal.Size, _ *Data) string {
	return ""
}

func (f frame) Match(s terminal.Size) bool {
	return f.xAxis.size == s.Width && f.yAxis.size == s.Height
}

func computeYAxis(size terminal.Size, stats *Stats, url string) yAxis {
	var b strings.Builder
	// Making of a buffer of [size] will be too small because ansi + unicode will take up more bytes than the
	// character space they take up
	b.Grow(size.Height * 2)

	title := ansi.Cyan(url) + " [" + stats.String() + "] " + ansi.Green(size.String())
	titleIndent := (size.Width / 2) - (len(title) / 2)
	// TODO crop the title if it wont fit
	fmt.Fprint(&b, ansi.Magenta("Latency")+ansi.CursorForward(titleIndent)+title)

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
			fmt.Fprint(&b, ansi.White("|"))
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
	const spacePerItem = formatLen + 4
	padding := ansi.White("--")
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
		fmt.Fprint(&b, padding+ansi.Yellow(x)+padding)
		remaining -= spacePerItem
	}
	if remaining > 0 {
		// TODO also put some chars at the beginning of the axis
		final := strings.Repeat("-", remaining)
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
	ret := ansi.Clear + ansi.Home
	ret += y + lines
	ret += ansi.CursorPosition(size.Height, 1)
	ret += x
	return ret
}
