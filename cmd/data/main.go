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

	"github.com/Lexer747/AcciPing/graph/data"
)

// Parses any `.ping` and prints them to stdout
func main() {
	printAll := false
	flag.BoolVar(&printAll, "a", false, "prints all raw values")
	flag.Parse()
	toPrint := flag.Args()
	for _, file := range toPrint {
		f, err := os.OpenFile(file, os.O_RDONLY, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open %q, %s\n", file, err.Error())
		}
		d, err := data.ReadData(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse %q, %s\n", file, err.Error())
		}
		defer f.Close()
		if printAll {
			fmt.Fprintf(os.Stdout, "BEGIN %s: %s\n", d.URL, d.Header.String())
			for i := range d.TotalCount {
				p := d.GetFull(i)
				fmt.Fprintf(os.Stdout, "%d: %s\n", i, p.String())
			}
			fmt.Fprintf(os.Stdout, "END %s: %s\n", d.URL, d.Header.String())
		} else {
			fmt.Fprintln(os.Stdout, d.String())
		}
	}
}
