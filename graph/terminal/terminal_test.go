// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package terminal_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/graph/terminal/ansi"
	"github.com/Lexer747/AcciPing/graph/terminal/th"
	"gotest.tools/v3/assert"
)

func TestTerminalWrite(t *testing.T) {
	t.Parallel()
	_, stdout, term, _, err := th.NewTestTerminal()
	assert.NilError(t, err)
	ctx, cancelFunc := context.WithCancelCause(context.Background())
	defer cancelFunc(nil)
	_, err = term.StartRaw(ctx, cancelFunc)
	assert.NilError(t, err)
	const hello = "Hello world"
	term.Print(hello)
	assert.Equal(t, ansi.HideCursor+hello, stdout.ReadString(t))
}

func TestTerminalReading(t *testing.T) {
	t.Parallel()
	stdin, _, term, _, err := th.NewTestTerminal()
	assert.NilError(t, err)
	timeout := testErr{}
	ctx, cancelFunc := context.WithTimeoutCause(context.Background(), time.Second, timeout)
	cancelWithCause := func(err error) { cancelFunc() }
	defer cancelWithCause(nil)
	_, err = term.StartRaw(ctx, cancelWithCause)
	assert.NilError(t, err)
	_, _ = stdin.Write([]byte("\x03")) // ctrl-c will cause the terminal to cancel

	// Wait till the ctrl-c or timeout cancel the context
	<-ctx.Done()
	// if this is equal to our timeout error then the ctrl-c listener didn't work
	assert.Assert(t, !errors.Is(timeout, context.Cause(ctx)))
}

func TestTerminalListener(t *testing.T) {
	t.Parallel()
	stdin, stdout, term, _, err := th.NewTestTerminal()
	assert.NilError(t, err)
	ctx, cancelFunc := context.WithCancelCause(context.Background())
	defer cancelFunc(nil)
	lastRune := ' '
	testListener := terminal.Listener{
		Applicable: func(r rune) bool {
			lastRune = r
			return true
		},
		Action: func(r rune) error {
			assert.Equal(t, lastRune, r)
			err := term.Print(string(r))
			assert.NilError(t, err)
			return nil
		},
	}
	_, err = term.StartRaw(ctx, cancelFunc, testListener)
	assert.NilError(t, err)
	_ = stdout.ReadString(t)
	_, _ = stdin.Write([]byte("a"))
	a := stdout.ReadString(t)
	assert.Equal(t, "a", a)
	_, _ = stdin.Write([]byte("b"))
	b := stdout.ReadString(t)
	assert.Equal(t, "b", b)
	_, _ = stdin.Write([]byte("c"))
	c := stdout.ReadString(t)
	assert.Equal(t, "c", c)
}

type testErr struct{}

func (testErr) Error() string {
	return "testErr"
}
