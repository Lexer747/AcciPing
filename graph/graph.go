// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Lexer747/acci-ping/draw"
	"github.com/Lexer747/acci-ping/graph/data"
	"github.com/Lexer747/acci-ping/graph/graphdata"
	"github.com/Lexer747/acci-ping/graph/terminal"
	"github.com/Lexer747/acci-ping/gui"
	"github.com/Lexer747/acci-ping/ping"
	"github.com/Lexer747/acci-ping/utils/check"
)

type Graph struct {
	Term *terminal.Terminal
	guiI gui.GUI

	sinkAlive   bool
	dataChannel chan ping.PingResults

	pingsPerMinute float64

	data *graphdata.GraphData

	frameMutex *sync.Mutex
	lastFrame  frame

	drawingBuffer *draw.Buffer
}

func NewGraph(
	ctx context.Context,
	input chan ping.PingResults,
	t *terminal.Terminal,
	gui gui.GUI,
	pingsPerMinute float64,
	URL string,
	drawingBuffer *draw.Buffer,
) *Graph {
	return NewGraphWithData(ctx, input, t, gui, pingsPerMinute, data.NewData(URL), drawingBuffer)
}

func NewGraphWithData(
	ctx context.Context,
	input chan ping.PingResults,
	t *terminal.Terminal,
	gui gui.GUI,
	pingsPerMinute float64,
	data *data.Data,
	drawingBuffer *draw.Buffer,
) *Graph {
	g := &Graph{
		Term:           t,
		sinkAlive:      true,
		dataChannel:    input,
		pingsPerMinute: pingsPerMinute,
		data:           graphdata.NewGraphData(data),
		frameMutex:     &sync.Mutex{},
		lastFrame:      frame{},
		drawingBuffer:  drawingBuffer,
		guiI:           gui,
	}
	if ctx != nil {
		// A nil context is valid: It means that no new data is expected and the input channel isn't active
		go g.sink(ctx)
	}
	return g
}

// Run holds the thread an listens on it's ping channel continuously, drawing a new graph every time a new
// packet comes in. It only returns a fatal error in which case it couldn't continue drawing (although it may
// still panic). It will return [terminal.UserControlCErr] if the thread was cancelled by the user.
//
// Since this runs in a concurrent sense any method is thread safe but therefore may also block if another
// thread is already holding the lock.
//
// Returns
//   - The graph main function
//   - the defer function which will restore the terminal to the correct state
//   - a channel containing all the terminal size updates
//   - an error if creating any of the above failed.
func (g *Graph) Run(
	ctx context.Context,
	stop context.CancelCauseFunc,
	fps int,
	listeners []terminal.ConditionalListener,
	fallbacks []terminal.Listener,
) (func() error, func(), chan terminal.Size, error) {
	timeBetweenFrames := getTimeBetweenFrames(fps, g.pingsPerMinute)
	frameRate := time.NewTicker(timeBetweenFrames)
	cleanup, err := g.Term.StartRaw(ctx, stop, listeners, fallbacks)
	if err != nil {
		return nil, cleanup, nil, err
	}
	terminalUpdates := make(chan terminal.Size)
	graph := func() error {
		size := g.Term.Size()
		defer close(terminalUpdates)
		for {
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case <-frameRate.C:
				if err = g.Term.UpdateCurrentTerminalSize(); err != nil {
					return err
				}
				if size != g.Term.Size() {
					slog.Debug("sending size update", "size", size)
					terminalUpdates <- size
					size = g.Term.Size()
				}
				toWrite := g.computeFrame(timeBetweenFrames, true)
				err = toWrite(g.Term)
				if err != nil {
					return err
				}
			}
		}
	}
	return graph, cleanup, terminalUpdates, err
}

// OneFrame doesn't run the graph but runs all the code to create and print a single frame to the terminal.
func (g *Graph) OneFrame() error {
	if err := g.Term.ClearScreen(terminal.MoveHome); err != nil {
		return err
	}
	if err := g.Term.UpdateCurrentTerminalSize(); err != nil {
		return err
	}
	toWrite := g.computeFrame(0, false)
	return toWrite(g.Term)
}

// LastFrame will return the last graphical frame printed to the terminal.
func (g *Graph) LastFrame() string {
	g.frameMutex.Lock()
	defer g.frameMutex.Unlock()
	var b strings.Builder
	err := g.lastFrame.framePainterNoGui(&b)
	check.NoErr(err, "While painting frame to string buffer")
	return b.String()
}

// Summarise will summarise the graph's backed data according to the [*graphdata.GraphData.String] function.
func (g *Graph) Summarise() string {
	g.frameMutex.Lock()
	defer g.frameMutex.Unlock()
	return strings.ReplaceAll(g.data.String(), "| ", "\n\t")
}

func (g *Graph) sink(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			g.sinkAlive = false
			return
		case p, ok := <-g.dataChannel:
			// TODO configure logging channels
			// slog.Debug("graph sink data received", "packet", p)
			if !ok {
				g.sinkAlive = false
				return
			}
			g.data.AddPoint(p)
		}
	}
}

type frame struct {
	PacketCount       int64
	yAxis             yAxis
	xAxis             xAxis
	framePainter      func(io.Writer) error
	framePainterNoGui func(io.Writer) error
	spinnerIndex      int
}

func (f frame) Match(s terminal.Size) bool {
	return f.xAxis.size == s.Width && f.yAxis.size == s.Height
}

func (f frame) Size() terminal.Size {
	return terminal.Size{Height: f.xAxis.size, Width: f.yAxis.size}
}
