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
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/Lexer747/AcciPing/files"
	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/terminal"
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
	if len(toPrint) > 1 {
		fmt.Fprint(os.Stderr, "More than one file found, exiting. Only one file at a time supported.\n"+
			"Use -h/--help to print usage instructions.\n")
		os.Exit(1)
	}

	filepath := toPrint[0]

	d, f, err := files.LoadFile(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't open and read file, failed with: %s", err.Error())
		os.Exit(1)
	}
	f.Close()
	var term *terminal.Terminal
	if termSize != nil && *termSize != "" {
		term, err = terminal.NewParsedFixedSizeTerminal(*termSize)
	} else {
		term, err = terminal.NewTerminal()
	}
	if err != nil {
		panic(err.Error())
	}

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
	fmt.Println()
	fmt.Println()
	fmt.Println()
}

func printGraph(term *terminal.Terminal, d *data.Data) {
	g, err := graph.NewGraphWithData(context.Background(), nil, term, 0, d)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println()
	err = g.OneFrame()
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
