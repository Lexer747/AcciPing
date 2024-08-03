// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import (
	"context"
	"sync"
	"time"

	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/graphdata"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/ping"
)

type Graph struct {
	Term *terminal.Terminal

	sinkAlive   bool
	dataChannel chan ping.PingResults

	pingsPerMinute float64

	data *graphdata.GraphData

	frameMutex *sync.Mutex
	lastFrame  frame
}

func NewGraph(ctx context.Context, input chan ping.PingResults, t *terminal.Terminal, pingsPerMinute float64, URL string) (*Graph, error) {
	return NewGraphWithData(ctx, input, t, pingsPerMinute, data.NewData(URL))
}

func NewGraphWithData(
	ctx context.Context,
	input chan ping.PingResults,
	t *terminal.Terminal,
	pingsPerMinute float64,
	data *data.Data,
) (*Graph, error) {
	g := &Graph{
		Term:           t,
		sinkAlive:      true,
		dataChannel:    input,
		pingsPerMinute: pingsPerMinute,
		data:           graphdata.NewGraphData(data),
		frameMutex:     &sync.Mutex{},
		lastFrame:      frame{},
	}
	if ctx != nil {
		// A nil context is valid: It means that no new data is expected and the input channel isn't active
		go g.sink(ctx)
	}
	return g, nil
}

// Run holds the thread an listens on it's ping channel continuously, drawing a new graph every time a new
// packet comes in. It only returns a fatal error in which case it couldn't continue drawing (although it may
// still panic). It will return [terminal.UserControlCErr] if the thread was cancelled by the user.
//
// Since this runs in a concurrent sense any method is thread safe but therefore may also block if another
// thread is already holding the lock.
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

// OneFrame doesn't run the graph but runs all the code to create and print a single frame to the terminal.
func (g *Graph) OneFrame() error {
	if err := g.Term.ClearScreen(true); err != nil {
		return err
	}
	if err := g.Term.UpdateCurrentTerminalSize(); err != nil {
		return err
	}
	toWrite := g.computeFrame(0, false)
	g.Term.Print(toWrite)
	return nil
}

// LastFrame will return the last graphical frame printed to the terminal.
func (g *Graph) LastFrame() string {
	g.frameMutex.Lock()
	defer g.frameMutex.Unlock()
	return paint(
		g.lastFrame.Size(),
		g.lastFrame.xAxis.bars,
		g.lastFrame.xAxis.axis,
		g.lastFrame.yAxis.axis,
		g.lastFrame.key,
		g.lastFrame.insideFrame,
		"",
	)
}

// Summarise will summarise the graph's backed data according to the [*graphdata.GraphData.String] function.
func (g *Graph) Summarise() string {
	g.frameMutex.Lock()
	defer g.frameMutex.Unlock()
	return g.data.String()
}

func (g *Graph) sink(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			g.sinkAlive = false
			return
		case p, ok := <-g.dataChannel:
			if !ok {
				g.sinkAlive = false
				return
			}
			g.data.AddPoint(p)
		}
	}
}

type frame struct {
	PacketCount  int64
	yAxis        yAxis
	xAxis        xAxis
	key          string
	insideFrame  string
	spinnerIndex int
}

func (f frame) Match(s terminal.Size) bool {
	return f.xAxis.size == s.Width && f.yAxis.size == s.Height
}

func (f frame) Size() terminal.Size {
	return terminal.Size{Height: f.xAxis.size, Width: f.yAxis.size}
}
