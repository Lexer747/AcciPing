// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package terminal

import (
	"context"
	"os"
	"strings"

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
}

func NewTerminal() (*Terminal, error) {
	size := Size{Height: 20, Width: 80}
	if isRunningUnderTerminal() {
		var err error
		size, err = GetCurrentTerminalSize()
		if err != nil {
			return nil, err
		}
	}
	return &Terminal{
		size:      size,
		listeners: []Listener{},
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
	// Action the callback which will be invoked when a user inputs the applicable rune.
	Action func() error
}

// StartRaw takes ownership of the stdin/stdout and control of the incoming context. It will asynchronously
// block on the users input and forward characters to the relevant listener. By default a `ctrl+C` listener is
// added which will call the [stop] function when detected.
//
// To block a main thread until the `ctrl+C` listener is hit, simply wait on the input [ctx.Done()] channel.
func (t *Terminal) StartRaw(ctx context.Context, stop context.CancelFunc, listeners ...Listener) error {
	inFd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(inFd)
	if err != nil {
		return errors.Wrap(err, "failed to set terminal to raw mode")
	}
	closer := func() { _ = term.Restore(inFd, oldState) }
	controlCListener := Listener{
		Name:       "ctrl+c",
		Applicable: func(r rune) bool { return r == '\u0003' },
		Action: func() error {
			closer()
			stop()
			return nil
		},
	}
	t.listeners = append(t.listeners, controlCListener)
	go t.beingListening(ctx)
	return nil
}

func (t *Terminal) Write(b []byte) error {
	_, err := os.Stdout.Write(b)
	return err
}

func (t *Terminal) beingListening(ctx context.Context) {
	buffer := make([]byte, 10)
	var err error
	var n int
	inputChannel := make(chan struct{})
	// Create a go-routine which continuously reads from stdin
	go func() {
		defer close(inputChannel)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// This is blocking hence why the go-routine wrapper exists, we still only free ourself when
				// the outer context is done which is racey.
				n, err = os.Stdin.Read(buffer)
				inputChannel <- struct{}{}
			}
		}
	}()

	for {
		// Spin forever, waiting on input from the context which has cancelled us from else where, or a new
		// input char.
		select {
		case <-ctx.Done():
			return
		case <-inputChannel:
			if err != nil {
				panic(errors.Wrap(err, "unexpected read failure in terminal"))
			}
			line := strings.Repeat(".", t.size.Width)
			t.Write([]byte(line))
			c := string(buffer[:n])
			r := []rune(c)[0]
			// TODO pre-sort and order the listeners, then create a lookup instead of a linear search
			// TODO document multiple valid listeners - especially ctrl-C interactions
			for _, l := range t.listeners {
				if l.Applicable(r) {
					err = l.Action()
					if err != nil {
						panic(errors.Wrapf(err, "unexpected failure Action %q in terminal", l.Name))
					}
				}
			}
			bytes.Clear(buffer, n)
		}
	}
}

// GetCurrentTerminalSize gets the current terminal size or error if the program doesn't have a terminal
// attached (e.g. go tests).q
func GetCurrentTerminalSize() (Size, error) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	return Size{Height: h, Width: w}, errors.Wrap(err, "failed to get terminal size")
}

func isRunningUnderTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
