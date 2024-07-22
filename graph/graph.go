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
	"github.com/Lexer747/AcciPing/utils/errors"
	"github.com/Lexer747/AcciPing/utils/numeric"
)

type Graph struct {
	Term        *terminal.Terminal
	data        *data.Data
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
		Term:           t,
		data:           data.NewData(),
		dataMutex:      &sync.Mutex{},
		dataChannel:    input,
		url:            url,
		pingsPerMinute: pingsPerMinute,
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
		toWrite := g.computeFrame(timeBetweenFrames)
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

func getTimeBetweenFrames(fps int, pingsPerMinute float64) time.Duration {
	if fps == 0 {
		return ping.PingsPerMinuteToDuration(pingsPerMinute)
	} else {
		return time.Duration(1000/fps) * time.Millisecond
	}
}

func (g *Graph) Summarize() string {
	g.dataMutex.Lock()
	defer g.dataMutex.Unlock()
	return g.data.Header.String()
}

// TODO compute the frame into an existing buffer instead of a string API
func (g *Graph) computeFrame(timeBetweenFrames time.Duration) string {
	s := g.Term.Size() // This is race-y so ensure a consistent size for rendering
	g.dataMutex.Lock()
	count := g.data.TotalCount
	if count == 0 {
		g.dataMutex.Unlock()
		return "" // no data yet
	}
	g.lastFrame.spinnerIndex++
	spinnerValue := spinner(s, g.lastFrame.spinnerIndex, timeBetweenFrames)
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

var spinnerArray = [...]string{"⏴", "⏵", "⏶", "⏷"}

func spinner(s terminal.Size, i int, timeBetweenFrames time.Duration) string {
	// TODO refactor into a generic only paint me every X fps.
	// We want 300ms between spinner updates
	a := i / int(300/timeBetweenFrames.Milliseconds())
	return ansi.CursorPosition(2, s.Width-3) + ansi.Cyan(spinnerArray[a%len(spinnerArray)])
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
	PacketCount  int
	yAxis        yAxis
	xAxis        xAxis
	insideFrame  string
	spinnerIndex int
}

func (f frame) Match(s terminal.Size) bool {
	return f.xAxis.size == s.Width && f.yAxis.size == s.Height
}

func gradientString(gradient float64, info *data.Header) string {
	normalized := numeric.Normalize(gradient, info.MinGradient, info.MaxGradient)
	return typography.Gradient(normalized)
}

func translate(s terminal.Size, p ping.PingResults, info *data.Header) (row, column int) {
	// Ok something is off here, we are using the column for a width based re-scaling and the row for height
	// based re-scaling. This is essentially mirrored about both axis compared to my mental model 🤔.
	column = getColumn(p.Timestamp, info, s)
	row = int(numeric.NormalizeToRange(
		float64(p.Duration),
		float64(info.Stats.Min),
		float64(info.Stats.Max),
		float64(s.Height-1),
		2,
	))
	return
}

func getColumn(t time.Time, info *data.Header, s terminal.Size) int {
	timestamp := info.Span.End.Sub(t)
	return int(numeric.NormalizeToRange(
		float64(timestamp),
		0,
		float64(info.Span.Duration),
		float64(s.Width-1),
		13,
	))
}

var plain = ansi.LightGray(typography.Diamond)
var drop = ansi.Red(typography.Block)

func computeInnerFrame(s terminal.Size, d *data.Data) string {
	centreRow := s.Height / 2
	centreColumn := s.Width / 2
	if d.TotalCount <= 1 {
		return ansi.CursorPosition(centreRow, centreColumn) + plain + " " + d.Blocks[0].Raw[0].Duration.String()
	}
	ret := ""
	droppedBar := ""
	if d.Header.Stats.PacketsDropped > 0 {
		// TODO more width when few points
		droppedBar = strings.Repeat(drop+ansi.CursorDown(1)+ansi.CursorBack(1), s.Height-1)
	}
	// TODO plot gradient when few points

	for _, block := range d.Blocks {
		for _, p := range block.Raw {
			if p.Dropped() {
				// dropped packet
				column := getColumn(p.Timestamp, d.Header, s)
				ret += ansi.CursorPosition(2, column) + droppedBar
			} else {
				row, column := translate(s, p, d.Header)
				switch {
				// TODO change text justification based on the column
				case p.Duration == d.Stats.Min:
					ret += ansi.CursorPosition(row, column) + ansi.Green(typography.Diamond+" "+p.Duration.String())
				case p.Duration == d.Stats.Max:
					ret += ansi.CursorPosition(row, column) + ansi.Red(typography.Diamond+" "+p.Duration.String())
				default:
					ret += ansi.CursorPosition(row, column) + plain
				}
			}
		}
	}

	return ret
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
	titleBegin := ansi.Cyan(url) + " ["
	titleEnd := "] " + ansi.Green(sizeStr)
	remaining := size.Width - 7 - len(url) - 4 - len(sizeStr)
	title := titleBegin + stats.PickString(remaining) + titleEnd
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
	spanBase *data.TimeSpan
	axis     string
}

func paint(size terminal.Size, x, y, lines, spinner string) string {
	ret := ansi.Clear
	ret += lines + y
	ret += ansi.CursorPosition(size.Height, 1)
	ret += x
	ret += spinner
	return ret
}
