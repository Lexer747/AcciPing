// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Lexer747/AcciPing/files"
	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/graph/terminal"
)

func main() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage of %s: reads '.pings' files and outputs the final frame of the capture\n"+
			"\t drawframe FILE\n\n"+
			"e.g. %s my_ping_capture.ping\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
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

	term, err := terminal.NewTerminal()
	if err != nil {
		panic(err.Error())
	}

	g, err := graph.NewGraphWithData(context.Background(), nil, term, 0, d)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println()
	err = g.OneFrame()
	if err != nil {
		panic(err.Error())
	}
	fmt.Println()
}
