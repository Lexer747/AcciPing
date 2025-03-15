// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package drawframe

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/Lexer747/acci-ping/draw"
	"github.com/Lexer747/acci-ping/files"
	"github.com/Lexer747/acci-ping/graph"
	"github.com/Lexer747/acci-ping/graph/data"
	"github.com/Lexer747/acci-ping/graph/terminal"
	"github.com/Lexer747/acci-ping/gui"
	"github.com/Lexer747/acci-ping/utils/check"
	"github.com/Lexer747/acci-ping/utils/exit"
)

type Config struct {
	cpuprofile *string
	memprofile *string
	termSize   *string

	*flag.FlagSet
}

func GetFlags() *Config {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	ret := &Config{
		cpuprofile: f.String("cpuprofile", "", "write cpu profile to `file`"),
		memprofile: f.String("memprofile", "", "write memory profile to `file`"),
		termSize: f.String("term-size", "", "controls the terminal size and fixes it to the input,"+
			" input is in the form \"<H>x<W>\" e.g. 20x80. H and W must be integers - where H == height, and W == width of the terminal."),
		FlagSet: f,
	}
	f.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage of %s: reads '.pings' files and outputs the final frame of the capture\n"+
			"\t drawframe [options] FILE\n\n"+
			"e.g. %s my_ping_capture.ping\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	return ret
}

func RunDrawFrame(c *Config) {
	check.Check(c.Parsed(), "flags not parsed")
	closeProfile := startCPUProfiling(*c.cpuprofile)
	defer closeProfile()
	defer concludeMemProfile(*c.memprofile)
	profiling := *c.cpuprofile != "" || *c.memprofile != ""

	toPrint := c.Args()
	if len(toPrint) == 0 {
		fmt.Fprint(os.Stderr, "No files found, exiting. Use -h/--help to print usage instructions.\n")
		exit.Success()
	}

	term, err := makeTerminal(c.termSize)
	exit.OnErrorMsg(err, "failed to open terminal to draw")

	for _, path := range toPrint {
		run(term, path, profiling)
	}
	fmt.Println()
	fmt.Println()
	fmt.Println()
}

func run(term *terminal.Terminal, path string, profiling bool) {
	fs, err := os.Stat(path)
	exit.OnErrorMsgf(err, "Couldn't stat path %q, failed with", path)
	if fs.IsDir() {
		err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if filepath.Ext(p) != ".pings" {
				return nil
			}
			do(p, term, profiling)
			return nil
		})
		exit.OnErrorMsgf(err, "Couldn't walk path %q, failed with", path)
	} else {
		do(path, term, profiling)
	}
}

func do(path string, term *terminal.Terminal, profiling bool) {
	d, f, err := files.LoadFile(path)
	exit.OnErrorMsg(err, "Couldn't open and read file, failed with")
	f.Close()
	if err != nil {
		panic(err.Error())
	}

	// TODO dont profile like this when on a folder.
	if profiling {
		timer := time.NewTimer(time.Second * 60)
		running := true
		for running {
			printGraph(term, d)
			select {
			case <-timer.C:
				running = false
			default:
			}
		}
	} else {
		printGraph(term, d)
	}
}

func makeTerminal(termSize *string) (*terminal.Terminal, error) {
	if termSize != nil && *termSize != "" {
		return terminal.NewParsedFixedSizeTerminal(*termSize)
	} else {
		return terminal.NewTerminal()
	}
}

func printGraph(term *terminal.Terminal, d *data.Data) {
	g := graph.NewGraphWithData(context.Background(), nil, term, gui.NoGUI(), 0, d, draw.NewPaintBuffer())
	fmt.Println()
	err := g.OneFrame()
	if err != nil {
		panic(err.Error())
	}
}

func concludeMemProfile(path string) {
	if path != "" {
		f, err := os.Create(path)
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

func startCPUProfiling(path string) func() {
	if path != "" {
		runtime.SetCPUProfileRate(1000000)
		f, err := os.Create(path)
		if err != nil {
			panic("could not create CPU profile: " + err.Error())
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			panic("could not start CPU profile: " + err.Error())
		}
		return func() {
			pprof.StopCPUProfile()
			f.Close()
		}
	}
	return func() {}
}
