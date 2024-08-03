// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Lexer747/AcciPing/graph/data"
)

// Parses any `.ping` and prints them to stdout
func main() {
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage of %s: reads '.pings' files and outputs the raw data to the stdout\n"+
			"\t data [-a][-csv] FILES\n\n"+
			"e.g. %s my_ping_capture.ping\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	printAll := false
	flag.BoolVar(&printAll, "a", false, "prints all raw values otherwise only summarises '.pings' files")
	toCSV := false
	flag.BoolVar(&toCSV, "csv", false, "writes '.pings' files as '.csv'")
	flag.Parse()
	toPrint := flag.Args()
	if len(toPrint) == 0 {
		fmt.Fprintf(os.Stderr, "No files found, exiting. Use -h/--help to print usage instructions.\n")
		os.Exit(0)
	}
	for _, file := range toPrint {
		f, err := os.OpenFile(file, os.O_RDONLY, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open %q, %s\n", file, err.Error())
			continue
		}
		d, err := data.ReadData(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse %q, %s\n", file, err.Error())
			continue
		}
		defer f.Close()
		handle(printAll, toCSV, d)
	}
}

func handle(printAll, toCSV bool, d *data.Data) {
	// In precedence order of flags
	switch {
	case printAll:
		fmt.Fprintf(os.Stdout, "BEGIN %s: %s\n", d.URL, d.Header.String())
		for i := range d.TotalCount {
			p := d.GetFull(i)
			fmt.Fprintf(os.Stdout, "%d: %s\n", i, p.String())
		}
		fmt.Fprintf(os.Stdout, "END %s: %s\n", d.URL, d.Header.String())
	case toCSV:
		handleCSV(d)
	default:
		fmt.Fprintln(os.Stdout, d.String())
	}
}

func handleCSV(d *data.Data) {
	fmt.Fprintln(os.Stdout, "timestamp(RFC3339Nano),latency,dropped,ip,header")
	fmt.Fprintf(os.Stdout, ",,,,%q\n", d.String())
	for i := range d.TotalCount {
		p := d.GetFull(i)
		fmt.Fprintf(
			os.Stdout,
			"%q,%q,%q,%q,\n",
			p.Data.Timestamp.Format(time.RFC3339Nano),
			p.Data.Duration.String(),
			p.Data.DropReason.String(),
			p.IP.String(),
		)
	}
}
