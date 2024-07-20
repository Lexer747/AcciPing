// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package terminal

import (
	"context"
	"io"
	"os"
	"slices"
	"sync"

	"github.com/Lexer747/AcciPing/graph/terminal/ansi"
	"github.com/Lexer747/AcciPing/utils/bytes"
	"github.com/Lexer747/AcciPing/utils/errors"

	"golang.org/x/term"
)

type Size struct {
	Height int
	Width  int
}

type Terminal struct {
	size      Size
	listeners []Listener

	stdin                *stdin
	stdout               *stdout
	terminalSizeCallBack func() Size
	isTestTerminal       bool

	listenMutex *sync.Mutex
}

func NewTerminal() (*Terminal, error) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return nil, errors.Errorf("Not an expected terminal environment cannot get terminal size")
	}
	size, err := getCurrentTerminalSize(os.Stdout)
	if err != nil {
		return nil, err
	}
	return &Terminal{
		size:        size,
		listeners:   []Listener{},
		stdin:       &stdin{realFile: os.Stdin},
		stdout:      &stdout{realFile: os.Stdout},
		listenMutex: &sync.Mutex{},
	}, nil
}
func (t *Terminal) Size() Size {
	return t.size
}

type Listener struct {
	// Name is used for if a listener errors for easier identification, it may be omitted.
	Name string
	// Applicable is the applicability of this listen, i.e. for which input runes do you want this action to
	// be fired
	Applicable func(rune) bool
	// Action the callback which will be invoked when a user inputs the applicable rune, the rune passed is
	// the same rune passed to applicable. Note the terminal size will have been updated before this called,
	// but this is actually racey if the user is typing while changing size.
	Action func(rune) error
}

type UserControlCErr struct{}

func (UserControlCErr) Error() string {
	return "user cancelled"
}

// StartRaw takes ownership of the stdin/stdout and control of the incoming context. It will asynchronously
// block on the users input and forward characters to the relevant listener. By default a `ctrl+C` listener is
// added which will call the [stop] function when detected.
//
// To block a main thread until the `ctrl+C` listener is hit, simply wait on the input [ctx.Done()] channel.
//
// The `ctrl-c` listener will also provide the [terminal.UserControlCErr] cause when this happens for use with
// [error.Is].
func (t *Terminal) StartRaw(ctx context.Context, stop context.CancelCauseFunc, listeners ...Listener) error {
	closer := func() {}
	if !t.isTestTerminal {
		inFd := int(t.stdin.realFile.Fd())
		oldState, err := term.MakeRaw(inFd)
		if err != nil {
			return errors.Wrap(err, "failed to set terminal to raw mode")
		}
		closer = func() { _ = term.Restore(inFd, oldState) }
	}

	controlCListener := Listener{
		Name:       "ctrl+c",
		Applicable: func(r rune) bool { return r == '\x03' },
		Action: func(rune) error {
			closer()
			stop(UserControlCErr{})
			return nil
		},
	}
	t.listeners = slices.Concat(t.listeners, []Listener{controlCListener}, listeners)
	go t.beingListening(ctx)
	return nil
}

func (t *Terminal) ClearScreen(updateSize bool) error {
	if updateSize {
		if err := t.updateCurrentTerminalSize(); err != nil {
			return errors.Wrap(err, "while ctrl-f")
		}
	}
	err := t.Print(ansi.Clear + ansi.Home)
	return errors.Wrap(err, "while ctrl-f")
}

func (t *Terminal) Print(s string) error {
	err := t.Write([]byte(s))
	return err
}

func (t *Terminal) Write(b []byte) error {
	_, err := t.stdout.Write(b)
	return err
}

type listenResult struct {
	n   int
	err error
}

func (t *Terminal) beingListening(ctx context.Context) {
	buffer := make([]byte, 10)
	// TODO should be buffered? Are we ok dropping inputs?
	listenChannel := make(chan listenResult)
	processingChannel := make(chan struct{})
	// Create a go-routine which continuously reads from stdin
	go func() {
		// This is blocking hence why the go-routine wrapper exists, we still only free ourself when
		// the outer context is done which is racey.
		t.listen(ctx, listenChannel, processingChannel, buffer)
	}()

	for {
		// Spin forever, waiting on input from the context which has cancelled us from else where, or a new
		// input char.
		select {
		case <-ctx.Done():
			return
		case received := <-listenChannel:
			if received.err != nil {
				panic(errors.Wrap(received.err, "unexpected read failure in terminal"))
			}
			if err := t.updateCurrentTerminalSize(); err != nil {
				panic(errors.Wrap(err, "unexpected read failure in terminal"))
			}
			if received.n <= 0 {
				panic("unexpected 0 byte read")
			}
			r := rune(string(buffer[:received.n])[0])
			// TODO pre-sort and order the listeners, then create a lookup instead of a linear search
			// TODO document multiple valid listeners - especially ctrl-C interactions
			for _, l := range t.listeners {
				if !l.Applicable(r) {
					continue
				}
				err := l.Action(r)
				if err != nil {
					panic(errors.Wrapf(err, "unexpected failure Action %q in terminal", l.Name))
				}
			}
			// if we don't have the processing signal this clear would be racey against stdin.
			bytes.Clear(buffer, received.n)
			processingChannel <- struct{}{}
		}
	}
}

func (t *Terminal) listen(
	ctx context.Context,
	listenChannel chan listenResult,
	processingChannel chan struct{},
	buffer []byte,
) {
	defer close(listenChannel)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// We "listen" on the stdin waiting for user input.
			n, readErr := t.stdin.Read(buffer)
			listenChannel <- listenResult{n: n, err: readErr}
			// Now wait for the main loop to be done with that last read
			<-processingChannel
		}
	}
}

// getCurrentTerminalSize gets the current terminal size or error if the program doesn't have a terminal
// attached (e.g. go tests).
func getCurrentTerminalSize(file *os.File) (Size, error) {
	w, h, err := term.GetSize(int(file.Fd()))
	return Size{Height: h, Width: w}, errors.Wrap(err, "failed to get terminal size")
}

// updateCurrentTerminalSizes the terminals stored size.
func (t *Terminal) updateCurrentTerminalSize() error {
	if t.isTestTerminal {
		t.size = t.terminalSizeCallBack()
		return nil
	} else {
		var err error
		t.size, err = getCurrentTerminalSize(t.stdout.realFile)
		return err
	}
}

type stdout struct {
	realFile       *os.File
	stubFileWriter io.Writer
}

func (s *stdout) Write(b []byte) (int, error) {
	if s.realFile != nil {
		return s.realFile.Write(b)
	} else {
		return s.stubFileWriter.Write(b)
	}
}

type stdin struct {
	realFile       *os.File
	stubFileReader io.Reader
}

func (s *stdin) Read(b []byte) (int, error) {
	if s.realFile != nil {
		return s.realFile.Read(b)
	} else {
		return s.stubFileReader.Read(b)
	}
}

func NewTestTerminal(stdinReader io.Reader, stdoutWriter io.Writer, terminalSizeCallBack func() Size) (*Terminal, error) {
	size := terminalSizeCallBack()
	return &Terminal{
		size:                 size,
		listeners:            []Listener{},
		stdin:                &stdin{stubFileReader: stdinReader},
		stdout:               &stdout{stubFileWriter: stdoutWriter},
		terminalSizeCallBack: terminalSizeCallBack,
		isTestTerminal:       true,
		listenMutex:          &sync.Mutex{},
	}, nil
}
