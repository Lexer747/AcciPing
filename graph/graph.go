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
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/ping"
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
func (g *Graph) Size() int {
	g.dataMutex.Lock()
	defer g.dataMutex.Unlock()
	return g.data.TotalCount
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
			g.dataMutex.Lock()
			g.data.AddPoint(p)
			g.dataMutex.Unlock()
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
