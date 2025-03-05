// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package acciping

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	backoff "github.com/Lexer747/AcciPing/backoff"
	"github.com/Lexer747/AcciPing/draw"
	"github.com/Lexer747/AcciPing/files"
	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/errors"
	"github.com/Lexer747/AcciPing/utils/siphon"
	"golang.org/x/exp/maps"
)

type Application struct {
	GUI
	g    *graph.Graph
	term *terminal.Terminal

	toUpdate *os.File
	config   Config
	// this doesn't need a mutex because we ensure that no two threads have access to the same byte index (I
	// think this is fine when the slice doesn't grow).
	drawBuffer *draw.Buffer

	errorChannel chan error
}

type GUI struct {
	listeningChars map[rune]terminal.ConditionalListener
	fallbacks      []terminal.Listener
}

func (app *Application) Run(
	ctx context.Context,
	cancelFunc context.CancelCauseFunc,
	channel chan ping.PingResults,
	existingData *data.Data,
) error {
	// The ping channel which is already running needs to be duplicated, providing one to the Graph and second
	// to a file writer. This de-couples the processes, we don't want the GUI to affect storing data and vice
	// versa.
	graphChannel, fileChannel := siphon.TeeBufferedChannel(ctx, channel, app.config.PingBufferingLimit)
	fileData, err := duplicateData(app.toUpdate)
	// TODO support no file operation
	exitOnError(err)
	go app.writeToFile(ctx, fileData, fileChannel)

	app.drawBuffer = draw.NewPaintBuffer()

	go app.toastNotifications(ctx)
	helpCh := make(chan rune)
	app.addFallbackListener(helpAction(helpCh))
	go app.help(ctx, helpCh)

	// The graph will take ownership of the data channel and data pointer.
	app.g = graph.NewGraphWithData(ctx, graphChannel, app.term, app.config.PingsPerMinute, existingData, app.drawBuffer)
	_ = app.g.Term.ClearScreen(terminal.UpdateAndMove)

	if app.config.TestErrorListener {
		app.makeErrorGenerator()
	}

	defer close(app.errorChannel)
	defer close(helpCh)
	// Very high FPS is good for responsiveness in the UI (since it's locked) and re-drawing on a re-size.
	// TODO add UI listeners, zooming, changing ping speed - etc
	return app.g.Run(ctx, cancelFunc, 120, app.listeners(), app.fallbacks)
}

func (app *Application) Init(ctx context.Context, c Config) (channel chan ping.PingResults, existingData *data.Data) {
	app.config = c
	app.errorChannel = make(chan error)
	app.GUI = newGUI()
	closeProfile := startCPUProfiling(c.Cpuprofile)
	defer closeProfile()
	defer concludeMemProfile(c.Memprofile)
	p := ping.NewPing()
	var err error
	app.term, err = terminal.NewTerminal()
	exitOnError(err) // If we can't open the terminal for any reason we reasonably can't do anything this program offers.

	existingData, app.toUpdate = loadFile(c.FilePath, c.URL)

	channel, err = p.CreateChannel(ctx, existingData.URL, c.PingsPerMinute, c.PingBufferingLimit)
	// If Creating the channel has an error this means we cannot continue, the network errors are already
	// wrapped and retried by this channel, other errors imply some larger problem
	exitOnError(err)
	return
}

func (app *Application) Finish() {
	_ = app.term.ClearScreen(terminal.UpdateSize)
	app.term.Print(app.g.LastFrame())
	afterGraph := strings.Repeat("\n", app.term.Size().Height)
	app.term.Print(afterGraph + "# Summary\nPing Successfully recorded in file '" + app.config.FilePath + "'\n\t" +
		app.g.Summarise() + "\n")
}

func (app *Application) writeToFile(ctx context.Context, ourData *data.Data, input chan ping.PingResults) {
	defer app.toUpdate.Close()
	backoff := backoff.NewExponentialBackoff(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case p, ok := <-input:
			if !ok {
				return
			}
			ourData.AddPoint(p)
			_, err := app.toUpdate.Seek(0, 0)
			if err != nil {
				app.errorChannel <- err
				backoff.Wait()
				continue
			}
			err = ourData.AsCompact(app.toUpdate)
			if err != nil {
				app.errorChannel <- err
				backoff.Wait()
				continue
			}
			backoff.Success()
		}
	}
}

func (app *Application) makeErrorGenerator() {
	app.addListener('e', func(r rune) error {
		go func() { app.errorChannel <- errors.New("Test Error") }()
		return nil
	})
}

func (app *Application) addListener(r rune, Action func(rune) error) {
	if _, found := app.GUI.listeningChars[r]; found {
		panic(fmt.Sprintf("Adding more than one listener for '%v'", r))
	}
	app.GUI.listeningChars[r] = terminal.ConditionalListener{
		Listener: terminal.Listener{
			Action: Action,
			Name:   "GUI Listener " + strconv.QuoteRune(r),
		},
		Applicable: func(in rune) bool {
			return in == r
		},
	}
}

func (app *Application) addFallbackListener(Action func(rune) error) {
	app.GUI.fallbacks = append(app.GUI.fallbacks, terminal.Listener{
		Action: Action,
		Name:   "GUI Fallback Listener",
	})
}

func (app *Application) listeners() []terminal.ConditionalListener {
	return maps.Values(app.GUI.listeningChars)
}

func duplicateData(f *os.File) (*data.Data, error) {
	d := &data.Data{}
	file, fileErr := io.ReadAll(f)
	_, readingErr := d.FromCompact(file)
	return d, errors.Join(fileErr, readingErr)
}

func concludeMemProfile(memprofile string) {
	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			panic("could not create memory profile: " + err.Error())
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			panic("could not write memory profile: " + err.Error())
		}
	}
}

func startCPUProfiling(cpuprofile string) func() {
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			panic("could not create CPU profile: " + err.Error())
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			panic("could not start CPU profile: " + err.Error())
		}
		return func() {
			f.Close()
			pprof.StopCPUProfile()
		}
	}
	return func() {}
}

// TODO incremental read/writes, get the URL ASAP then start the channel, then incremental continuation.
func loadFile(file, url string) (*data.Data, *os.File) {
	// TODO this currently panics if the url's don't match we should do better
	d, f, err := files.LoadOrCreateFile(file, url)
	exitOnError(err)
	return d, f
}

func newGUI() GUI {
	return GUI{
		listeningChars: map[rune]terminal.ConditionalListener{},
		fallbacks:      []terminal.Listener{},
	}
}
