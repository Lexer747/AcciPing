// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Lexer747/acci-ping/graph/terminal"
	"github.com/Lexer747/acci-ping/graph/terminal/ansi"
)

// A small demo of the terminal API, this program will emit a terminal sized line every time it hears a key,
// and exits on ctrl+c. TL:DR Debug program.
func main() {
	// First we need to check if we are running under a terminal
	t, err := terminal.NewTerminal()
	if err != nil {
		panic(err.Error())
	}
	// Now setup the cancelling context to give to the terminal once we start running
	ctx, cancelFunc := context.WithCancelCause(context.Background())

	// Create out example listener, we trigger on any detected input and always write a full terminal line
	writeLineListener := terminal.Listener{
		Name: "blankLine",
		Action: func(r rune) error {
			halfSize := (t.Size().Width - 21) / 2
			toPrint := fmt.Sprintf("W:%-5dH:%-5dR:%-5s", t.Size().Width, t.Size().Height, strconv.QuoteRune(r))
			line := strings.Repeat(".", halfSize) + ansi.Yellow(toPrint) + strings.Repeat(".", halfSize)
			if t.Size().Width%2 == 0 {
				line += "."
			} else {
				line += ""
			}
			return t.Print(line)
		},
	}
	// clear screen example:
	clearScreenListener := terminal.ConditionalListener{
		Applicable: func(r rune) bool {
			return r == 'l'
		},
		Listener: terminal.Listener{
			Name: "clear",
			Action: func(rune) error {
				return t.ClearScreen(terminal.UpdateSize)
			},
		},
	}
	// Actually start the terminal program. Note that the listeners are applied in order, so if more than one
	// is applicable then the last entry will happen last
	cleanup, err := t.StartRaw(ctx, cancelFunc, []terminal.ConditionalListener{clearScreenListener}, []terminal.Listener{writeLineListener})
	defer cleanup()
	if err != nil {
		panic(err.Error())
	}
	if err = t.ClearScreen(terminal.UpdateSize); err != nil {
		panic(err.Error())
	}
	t.Print("Press 'l' to clear the screen, any other char to print a line, ctrl-c to quit." + ansi.CursorPosition(2, 1))
	// Hold the main thread until the context is cancelled by the terminal
	<-ctx.Done()
}
