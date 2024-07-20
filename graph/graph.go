// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/errors"
)

type Graph struct {
	term        *terminal.Terminal
	data        *Data
	dataMutex   *sync.Mutex
	dataChannel chan ping.PingResults
}

func NewGraph(input chan ping.PingResults, ctx context.Context) (*Graph, error) {
	t, err := terminal.NewTerminal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to make graph")
	}
	g := &Graph{
		term:        t,
		data:        NewData(),
		dataMutex:   &sync.Mutex{},
		dataChannel: input,
	}
	go g.sink(ctx)
	return g, nil
}

func (g *Graph) Run(ctx context.Context, fps int) {
	// TODO setup UI thread to listen on the raw terminal
	frameRate := time.NewTicker(time.Duration(1000/fps) * time.Millisecond)
	// g.term.Run()
	for {
		<-frameRate.C // Currently no strong opinions on dropped frames this is fine
		g.dataMutex.Lock()
		fmt.Println(g.data.Header.String())
		g.dataMutex.Unlock()
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
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
