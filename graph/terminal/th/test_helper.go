// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package th

import (
	"io"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Lexer747/acci-ping/graph/terminal"
	"gotest.tools/v3/assert"
)

func NewTestTerminal() (
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
	callback := func() terminal.Size {
		m.Lock()
		defer m.Unlock()
		return *captured
	}
	t, err := terminal.NewTestTerminal(stdin, stdout, callback)
	setTermSize := func(newSize terminal.Size) {
		m.Lock()
		*captured = newSize
		m.Unlock()
		_ = t.UpdateCurrentTerminalSize()
	}
	return stdin, stdout, t, setTermSize, err
}

type TestFile struct {
	fileName   string
	m          *sync.Mutex
	buffer     []byte
	readIndex  atomic.Int64
	writeIndex atomic.Int64
}

func newTestFile(name string) *TestFile {
	return &TestFile{fileName: name, m: &sync.Mutex{}, buffer: []byte{}}
}

func (f *TestFile) ReadString(t *testing.T) string {
	t.Helper()
	buffer := make([]byte, 255)
	n, err := f.Read(buffer)
	assert.NilError(t, err)
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
		panic("fix the test file impl, writer was behind reader")
	}
	toRead := w - r
	for i := range toRead {
		if i >= len(p) {
			return i, io.EOF
		}
		p[i] = f.buffer[r+i]
	}
	f.readIndex.Store(int64(r + toRead))
	return toRead, nil
}

func (f *TestFile) Write(p []byte) (n int, err error) {
	f.m.Lock()
	defer f.m.Unlock()
	// just grow infinitely
	toGrow := len(f.buffer) + len(p)
	f.buffer = slices.Grow(f.buffer, toGrow)
	f.buffer = append(f.buffer, p...)
	f.writeIndex.Store(int64(toGrow))
	return len(p), nil
}

func (f *TestFile) WriteCtrlC(t *testing.T) {
	t.Helper()
	_, err := f.Write([]byte("\x03"))
	assert.NilError(t, err)
}
