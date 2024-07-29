// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package terminal_test

import (
	"context"
	"testing"
	"time"

	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/graph/terminal/ansi"
	"github.com/Lexer747/AcciPing/graph/terminal/th"
	"github.com/stretchr/testify/require"
)

func TestTerminalWrite(t *testing.T) {
	t.Parallel()
	_, stdout, term, _, err := th.NewTestTerminal()
	require.NoError(t, err)
	ctx, cancelFunc := context.WithCancelCause(context.Background())
	defer cancelFunc(nil)
	_, err = term.StartRaw(ctx, cancelFunc)
	require.NoError(t, err)
	const hello = "Hello world"
	term.Print(hello)
	require.Equal(t, ansi.HideCursor+hello, stdout.ReadString(t))
}

func TestTerminalReading(t *testing.T) {
	t.Parallel()
	stdin, _, term, _, err := th.NewTestTerminal()
	require.NoError(t, err)
	timeout := testErr{}
	ctx, cancelFunc := context.WithTimeoutCause(context.Background(), time.Second, timeout)
	cancelWithCause := func(err error) { cancelFunc() }
	defer cancelWithCause(nil)
	_, err = term.StartRaw(ctx, cancelWithCause)
	require.NoError(t, err)
	_, _ = stdin.Write([]byte("\x03")) // ctrl-c will cause the terminal to cancel

	// Wait till the ctrl-c or timeout cancel the context
	<-ctx.Done()
	// if this is equal to our timeout error then the ctrl-c listener didn't work
	require.NotEqual(t, timeout, context.Cause(ctx))
}

func TestTerminalListener(t *testing.T) {
	t.Parallel()
	stdin, stdout, term, _, err := th.NewTestTerminal()
	require.NoError(t, err)
	ctx, cancelFunc := context.WithCancelCause(context.Background())
	defer cancelFunc(nil)
	lastRune := ' '
	testListener := terminal.Listener{
		Applicable: func(r rune) bool {
			lastRune = r
			return true
		},
		Action: func(r rune) error {
			require.Equal(t, lastRune, r)
			err := term.Print(string(r))
			require.NoError(t, err)
			return nil
		},
	}
	_, err = term.StartRaw(ctx, cancelFunc, testListener)
	require.NoError(t, err)
	_ = stdout.ReadString(t)
	_, _ = stdin.Write([]byte("a"))
	a := stdout.ReadString(t)
	require.Equal(t, "a", a)
	_, _ = stdin.Write([]byte("b"))
	b := stdout.ReadString(t)
	require.Equal(t, "b", b)
	_, _ = stdin.Write([]byte("c"))
	c := stdout.ReadString(t)
	require.Equal(t, "c", c)
}

type testErr struct{}

func (testErr) Error() string {
	return "testErr"
}
