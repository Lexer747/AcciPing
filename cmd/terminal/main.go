// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"strings"

	"github.com/Lexer747/AcciPing/graph/terminal"
	"golang.org/x/net/context"
)

// A small demo of the terminal API, this program will emit a terminal sized line every time it hears a key,
// and exits on ctrl+c.
func main() {
	// First we need to check if we are running under a terminal
	t, err := terminal.NewTerminal()
	if err != nil {
		panic(err.Error())
	}
	// Now setup the cancelling context to give to the terminal once we start running
	ctx, cancelFunc := context.WithCancel(context.Background())

	// Create out example listener, we trigger on any detected input and always write a full terminal line
	writeLineListener := terminal.Listener{
		Name: "blankLine",
		Applicable: func(rune) bool {
			return true
		},
		Action: func() error {
			line := strings.Repeat(".", t.Size().Width)
			t.Write([]byte(line))
			return nil
		},
	}

	// Actually start the terminal program
	err = t.StartRaw(ctx, cancelFunc, writeLineListener)
	if err != nil {
		panic(err.Error())
	}
	// Hold the main thread until the context is cancelled by the terminal
	select {
	case <-ctx.Done():
	}
}
