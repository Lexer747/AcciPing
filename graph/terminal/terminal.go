// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package terminal

import (
	"context"
	"io"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Lexer747/acci-ping/graph/terminal/ansi"
	"github.com/Lexer747/acci-ping/utils"
	"github.com/Lexer747/acci-ping/utils/bytes"
	"github.com/Lexer747/acci-ping/utils/errors"

	"golang.org/x/term"
)

type Size struct {
	Height int
	Width  int
}

func (s Size) String() string {
	return "W: " + strconv.Itoa(s.Width) + " H: " + strconv.Itoa(s.Height)
}

func Parse(s string) (Size, bool) {
	split := strings.Split(s, "x")
	if len(split) != 2 {
		return Size{}, false
	}
	height, hErr := strconv.ParseInt(split[0], 10, 32)
	width, wErr := strconv.ParseInt(split[1], 10, 32)
	if hErr != nil || wErr != nil {
		return Size{}, false
	}
	return Size{Height: int(height), Width: int(width)}, true
}

type Terminal struct {
	size      Size
	listeners []ConditionalListener
	fallbacks []Listener

	stdin                *stdin
	stdout               *stdout
	terminalSizeCallBack func() Size

	isTestTerminal bool
	isDynamicSize  bool

	// should be called if a panic occurs otherwise stacktraces are unreadable
	cleanup func()

	listenMutex *sync.Mutex
}

func NewTerminal() (*Terminal, error) {
	// TODO check both stdout and stderr for a terminal size
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return nil, errors.Errorf("Not an expected terminal environment cannot get terminal size")
	}
	size, err := getCurrentTerminalSize(os.Stdout)
	if err != nil {
		return nil, err
	}
	t := &Terminal{
		size:          size,
		listeners:     []ConditionalListener{},
		fallbacks:     []Listener{},
		stdin:         &stdin{realFile: os.Stdin},
		stdout:        &stdout{realFile: os.Stdout},
		listenMutex:   &sync.Mutex{},
		isDynamicSize: true,
	}
	return t, t.supportsRaw()
}

func NewFixedSizeTerminal(s Size) (*Terminal, error) {
	t := &Terminal{
		size:          s,
		listeners:     []ConditionalListener{},
		fallbacks:     []Listener{},
		stdin:         &stdin{realFile: os.Stdin},
		stdout:        &stdout{realFile: os.Stdout},
		listenMutex:   &sync.Mutex{},
		isDynamicSize: false,
	}
	return t, t.supportsRaw()
}

// NewParsedFixedSizeTerminal will construct a new fixed size terminal which cannot change size, parsing the
// size from the input parameter string, which is in format <H>x<W>, where H and W are integers.
func NewParsedFixedSizeTerminal(size string) (*Terminal, error) {
	s, ok := Parse(size)
	if !ok {
		return nil, errors.Errorf("Cannot parse %q as terminal a size, should be in the form \"<H>x<W>\", where H and W are integers.", size)
	}
	return NewFixedSizeTerminal(s)
}

func (t *Terminal) Size() Size {
	return t.size
}

type Listener struct {
	// Name is used for if a listener errors for easier identification, it may be omitted.
	Name string
	// Action the callback which will be invoked when a user inputs the applicable rune, the rune passed is
	// the same rune passed to applicable. Note the terminal size will have been updated before this called,
	// but this is actually racey if the user is typing while changing size. If an error occurs in this action
	// the terminal will panic and exit.
	Action func(rune) error
}

type ConditionalListener struct {
	Listener
	// Applicable is the applicability of this listen, i.e. for which input runes do you want this action to
	// be fired
	Applicable func(rune) bool
}

type userControlCErr struct{}

var UserCancelled = userControlCErr{}

func (userControlCErr) Error() string {
	return "user cancelled"
}

// StartRaw takes ownership of the stdin/stdout and control of the incoming context. It will asynchronously
// block on the users input and forward characters to the relevant listener. By default a `ctrl+C` listener is
// added which will call the [stop] function when detected.
//
// The first return value is a clean up function which recover from a panic, putting the terminal back into
// normal mode and unhooking the listeners so that the program terminates gracefully upon a panic in another
// thread. It should be called like so:
//
//	term, _ := terminal.NewTerminal()
//	cleanup, _ := term.StartRaw(ctx, stop)
//	defer cleanup() // Graceful panic recovery
//	<-ctx.Done() // Wait till user cancels with ctrl+C
//
// To block a main thread until the `ctrl+C` listener is hit, simply wait on the input [ctx.Done()] channel.
//
// The `ctrl-c` listener will also provide the [terminal.UserControlCErr] cause when this happens for use with
// [error.Is].
func (t *Terminal) StartRaw(
	ctx context.Context,
	stop context.CancelCauseFunc,
	listeners []ConditionalListener,
	fallbacks []Listener,
) (func(), error) {
	restore := func() {}
	if !t.isTestTerminal {
		inFd := int(t.stdin.realFile.Fd())
		oldState, err := term.MakeRaw(inFd)
		if err != nil {
			return nil, errors.Wrap(err, "failed to set terminal to raw mode")
		}
		restore = func() { _ = term.Restore(inFd, oldState) }
	}
	ctrlCAction := func(rune) error {
		t.Print(ansi.ShowCursor)
		restore()
		stop(UserCancelled)
		return nil
	}
	t.cleanup = func() {
		_ = ctrlCAction('\x00')
	}

	controlCListener := ConditionalListener{
		Applicable: func(r rune) bool { return r == '\x03' },
		Listener: Listener{
			Name:   "ctrl+c",
			Action: ctrlCAction,
		},
	}
	t.listeners = slices.Concat(t.listeners, []ConditionalListener{controlCListener}, listeners)
	if fallbacks != nil {
		t.fallbacks = fallbacks
	}
	t.Print(ansi.HideCursor)
	go t.beingListening(ctx)
	return t.cleanup, nil
}

type ClearBehaviour int

const (
	UpdateSize            ClearBehaviour = 1
	MoveHome              ClearBehaviour = 2
	UpdateSizeAndMoveHome ClearBehaviour = 3
)

func (t *Terminal) ClearScreen(behaviour ClearBehaviour) error {
	if behaviour == UpdateSize || behaviour == UpdateSizeAndMoveHome {
		if err := t.UpdateCurrentTerminalSize(); err != nil {
			return errors.Wrap(err, "while ClearScreen")
		}
	}
	t.Print(strings.Repeat("\n", t.size.Height))
	err := t.Print(ansi.Clear)
	if behaviour == MoveHome || behaviour == UpdateSizeAndMoveHome {
		err = errors.Join(err, t.Print(ansi.Home))
	}
	return errors.Wrap(err, "while ClearScreen")
}

func (t *Terminal) Print(s string) error {
	err := utils.Err(t.Write([]byte(s)))
	return err
}

func (t *Terminal) Write(b []byte) (int, error) {
	return t.stdout.Write(b)
}

type listenResult struct {
	n   int
	err error
}

func (t *Terminal) beingListening(ctx context.Context) {
	buffer := make([]byte, 20)
	listenChannel := make(chan listenResult, 20)
	processingChannel := make(chan struct{})
	// Create a go-routine which continuously reads from stdin
	go func() {
		defer t.cleanup()
		// This is blocking hence why the go-routine wrapper exists, we still only free ourself when
		// the outer context is done which is racey.
		t.listen(ctx, listenChannel, processingChannel, buffer)
	}()

	defer t.cleanup()
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
			if err := t.UpdateCurrentTerminalSize(); err != nil {
				panic(errors.Wrap(err, "unexpected read failure in terminal"))
			}
			if received.n <= 0 {
				return // cancelled
			}
			heard := string(buffer[:received.n])
			slog.Debug("got keyboard input", "received", heard)
			for _, r := range heard {
				// TODO document multiple valid listeners - especially ctrl-C interactions
				t.processListenedRune(r)
			}
			// if we don't have the processing signal this clear would be racey against stdin.
			bytes.Clear(buffer, received.n)
			processingChannel <- struct{}{}
		}
	}
}

// processListenedRune should only be called by the listener thread
func (t *Terminal) processListenedRune(r rune) {
	runFallback := true
	for _, l := range t.listeners {
		if !l.Applicable(r) {
			continue
		}
		err := l.Action(r)
		if err != nil {
			panic(errors.Wrapf(err, "unexpected failure Action %q in terminal", l.Name))
		}
		runFallback = false
	}
	if runFallback {
		for _, l := range t.fallbacks {
			err := l.Action(r)
			if err != nil {
				panic(errors.Wrapf(err, "unexpected failure Action %q in terminal", l.Name))
			}
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
func (t *Terminal) UpdateCurrentTerminalSize() error {
	if !t.isDynamicSize {
		return nil
	}
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
		listeners:            []ConditionalListener{},
		fallbacks:            []Listener{},
		stdin:                &stdin{stubFileReader: stdinReader},
		stdout:               &stdout{stubFileWriter: stdoutWriter},
		terminalSizeCallBack: terminalSizeCallBack,
		isTestTerminal:       true,
		isDynamicSize:        true,
		listenMutex:          &sync.Mutex{},
	}, nil
}

func (t *Terminal) supportsRaw() error {
	inFd := int(t.stdin.realFile.Fd())
	oldState, makeRawErr := term.MakeRaw(inFd)
	restoreErr := term.Restore(inFd, oldState)
	return errors.Wrap(errors.Join(makeRawErr, restoreErr), "failed to set terminal to raw mode")
}
