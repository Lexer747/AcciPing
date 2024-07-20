// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"time"

	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/ping"
)

func main() {
	p := ping.NewPing()
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancelFunc()
	channel, err := p.CreateChannel(ctx, "www.google.com", 5, 0)
	if err != nil {
		panic(err.Error())
	}
	g, err := graph.NewGraph(channel, ctx)
	if err != nil {
		panic(err.Error())
	}
	g.Run(ctx, 1)
}
