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

	"github.com/Lexer747/acci-ping/graph/terminal"
	"github.com/Lexer747/acci-ping/graph/terminal/ansi"
	"github.com/Lexer747/acci-ping/graph/terminal/th"
	"gotest.tools/v3/assert"
)

func TestTerminalWrite(t *testing.T) {
	t.Parallel()
	_, stdout, term, _, err := th.NewTestTerminal()
	assert.NilError(t, err)
	ctx, cancelFunc := context.WithCancelCause(context.Background())
	defer cancelFunc(nil)
	_, err = term.StartRaw(ctx, cancelFunc, nil, nil)
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
	_, err = term.StartRaw(ctx, cancelWithCause, nil, nil)
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
	testListener := terminal.ConditionalListener{
		Applicable: func(r rune) bool {
			lastRune = r
			return true
		},
		Listener: terminal.Listener{
			Action: func(r rune) error {
				assert.Equal(t, lastRune, r)
				err := term.Print(string(r))
				assert.NilError(t, err)
				return nil
			},
		},
	}
	_, err = term.StartRaw(ctx, cancelFunc, []terminal.ConditionalListener{testListener}, nil)
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

func TestTerminalFallbackListener(t *testing.T) {
	t.Parallel()
	stdin, _, term, _, err := th.NewTestTerminal()
	assert.NilError(t, err)
	ctx, cancelFunc := context.WithCancelCause(context.Background())
	defer cancelFunc(nil)
	lastRune := ' '
	m1 := make(chan struct{})
	m2 := make(chan struct{})
	defer close(m1)
	defer close(m2)
	testListener := terminal.ConditionalListener{
		Applicable: func(r rune) bool {
			return r == 'a'
		},
		Listener: terminal.Listener{
			Action: func(r rune) error {
				<-m1
				m2 <- struct{}{}
				return nil
			},
		},
	}
	fallback := terminal.Listener{
		Action: func(r rune) error {
			<-m1
			lastRune = r
			m2 <- struct{}{}
			return nil
		},
	}
	_, err = term.StartRaw(ctx, cancelFunc, []terminal.ConditionalListener{testListener}, []terminal.Listener{fallback})
	assert.NilError(t, err)
	_, _ = stdin.Write([]byte("a"))
	m1 <- struct{}{}
	<-m2
	assert.Equal(t, ' ', lastRune)
	_, _ = stdin.Write([]byte("b"))
	m1 <- struct{}{}
	<-m2
	assert.Equal(t, 'b', lastRune)
	_, _ = stdin.Write([]byte("c"))
	m1 <- struct{}{}
	<-m2
	assert.Equal(t, 'c', lastRune)
}
