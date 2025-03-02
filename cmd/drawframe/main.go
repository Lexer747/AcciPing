// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/Lexer747/AcciPing/draw"
	"github.com/Lexer747/AcciPing/files"
	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/gui"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var termSize = flag.String("term-size", "", "controls the terminal size and fixes it to the input,"+
	" input is in the form \"<H>x<W>\" e.g. 20x80. H and W must be integers - where H == height, and W == width of the terminal.")
var profiling = false

func main() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage of %s: reads '.pings' files and outputs the final frame of the capture\n"+
			"\t drawframe [options] FILE\n\n"+
			"e.g. %s my_ping_capture.ping\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	closeProfile := startCPUProfiling()
	defer closeProfile()
	defer concludeMemProfile()
	toPrint := flag.Args()
	if len(toPrint) == 0 {
		fmt.Fprint(os.Stderr, "No files found, exiting. Use -h/--help to print usage instructions.\n")
		os.Exit(0)
	}

	term, err := makeTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open terminal to draw: %s", err.Error())
		os.Exit(1)
	}

	for _, path := range toPrint {
		run(term, path)
	}
	fmt.Println()
	fmt.Println()
	fmt.Println()
}

func run(term *terminal.Terminal, path string) {
	fs, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't stat path %q, failed with: %s", path, err.Error())
		os.Exit(1)
	}
	if fs.IsDir() {
		err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if filepath.Ext(p) != ".pings" {
				return nil
			}
			do(p, term)
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't walk path %q, failed with: %s", path, err.Error())
			os.Exit(1)
		}
	} else {
		do(path, term)
	}
}

func do(path string, term *terminal.Terminal) {
	d, f, err := files.LoadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't open and read file, failed with: %s", err.Error())
		os.Exit(1)
	}
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

func makeTerminal() (*terminal.Terminal, error) {
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

func concludeMemProfile() {
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
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

func startCPUProfiling() func() {
	profiling = *cpuprofile != "" || *memprofile != ""
	if *cpuprofile != "" {
		runtime.SetCPUProfileRate(1000000)
		f, err := os.Create(*cpuprofile)
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
