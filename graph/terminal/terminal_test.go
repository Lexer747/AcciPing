// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package terminal_test

import (
	"context"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/graph/terminal/ansi"
	"github.com/Lexer747/AcciPing/utils/errors"
	"github.com/stretchr/testify/require"
)

func TestTerminalWrite(t *testing.T) {
	t.Parallel()
	_, stdout, term, _, err := newTestTerminal()
	require.NoError(t, err)
	ctx, cancelFunc := context.WithCancelCause(context.Background())
	defer cancelFunc(nil)
	_, err = term.StartRaw(ctx, cancelFunc)
	require.NoError(t, err)
	const hello = "Hello world"
	term.Print(hello)
	require.Equal(t, ansi.HideCursor+hello, stdout.readString(t))
}

func TestTerminalReading(t *testing.T) {
	t.Parallel()
	stdin, _, term, _, err := newTestTerminal()
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
	stdin, stdout, term, _, err := newTestTerminal()
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
	_ = stdout.readString(t)
	_, _ = stdin.Write([]byte("a"))
	a := stdout.readString(t)
	require.Equal(t, "a", a)
	_, _ = stdin.Write([]byte("b"))
	b := stdout.readString(t)
	require.Equal(t, "b", b)
	_, _ = stdin.Write([]byte("c"))
	c := stdout.readString(t)
	require.Equal(t, "c", c)
}

type testErr struct{}

func (testErr) Error() string {
	return "testErr"
}

func newTestTerminal() (
	*TestFile,
	*TestFile,
	*terminal.Terminal,
	func(newSize terminal.Size),
	error,
) {
	stdin := newTestFile("stdin")
	stdout := newTestFile("stdout")
	m := &sync.Mutex{}
	captured := &terminal.Size{Height: 5, Width: 5}
	setTermSize := func(newSize terminal.Size) {
		m.Lock()
		defer m.Unlock()
		*captured = newSize
	}
	callback := func() terminal.Size {
		m.Lock()
		defer m.Unlock()
		return *captured
	}
	t, err := terminal.NewTestTerminal(stdin, stdout, callback)
	return stdin, stdout, t, setTermSize, err
}

type TestFile struct {
	fileName   string
	m          *sync.Mutex
	buffer     []byte
	readIndex  atomic.Int32
	writeIndex atomic.Int32
}

func newTestFile(name string) *TestFile {
	return &TestFile{fileName: name, m: &sync.Mutex{}, buffer: []byte{}}
}

func (f *TestFile) readString(t *testing.T) string {
	t.Helper()
	buffer := make([]byte, 255)
	n, err := f.Read(buffer)
	require.NoError(t, err)
	return string(buffer[:n])
}

func (f *TestFile) Read(p []byte) (n int, err error) {
	for f.readIndex.Load() == f.writeIndex.Load() { // block until data appears
		runtime.Gosched()
	}
	f.m.Lock()
	defer f.m.Unlock()
	r := int(f.readIndex.Load())
	w := int(f.writeIndex.Load())
	if r > w {
		panic("fix the test file, writer was behind reader")
	}
	toRead := w - r
	for i := range toRead {
		if i >= len(p) {
			return i, errors.Errorf("Buffer too small had %d more bytes to read", toRead-i)
		}
		p[i] = f.buffer[r+i]
	}
	f.readIndex.Store(int32(r + toRead))
	return toRead, nil
}

func (f *TestFile) Write(p []byte) (n int, err error) {
	f.m.Lock()
	defer f.m.Unlock()
	// just grow infinitely
	toGrow := len(f.buffer) + len(p)
	f.buffer = slices.Grow(f.buffer, toGrow)
	f.buffer = append(f.buffer, p...)
	f.writeIndex.Store(int32(toGrow))
	return len(p), nil
}
