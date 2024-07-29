// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"

	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/errors"
)

func main() {
	const demoURL = "www.google.com"
	p := ping.NewPing()
	ctx, cancelFunc := context.WithCancelCause(context.Background())
	defer cancelFunc(nil)
	term, err := terminal.NewTerminal()
	if err != nil {
		panic(err.Error())
	}
	pingsPerMinute := 60.0
	channel, err := p.CreateChannel(ctx, demoURL, pingsPerMinute, 10)
	if err != nil {
		panic(err.Error())
	}
	g, err := graph.NewGraph(ctx, channel, term, pingsPerMinute, demoURL)
	if err != nil {
		panic(err.Error())
	}
	// Very high FPS is good for responsiveness in the UI (since it's locked) and re-drawing on a re-size.
	err = g.Run(ctx, cancelFunc, 0)
	if err != nil && !errors.Is(err, terminal.UserCancelled) {
		panic(err.Error())
	} else {
		_ = g.Term.ClearScreen(true)
		g.Term.Print(g.LastFrame())
		g.Term.Print("Summary\n" + g.Summarize())
	}
}
