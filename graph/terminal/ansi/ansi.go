// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package ansi

import (
	"strconv"
)

// Helper section

var Clear = EraseInDisplay(CursorScreen)
var Home = CursorPosition(1, 1)

// Spec definitions

func CursorUp(n int) string   { return CSI + s(n) + "A" }
func CursorDown(n int) string { return CSI + s(n) + "B" }
func CursorForward(n int) string {
	if n <= 0 {
		return ""
	}
	return CSI + s(n) + "C"
}
func CursorBack(n int) string {
	if n <= 0 {
		return ""
	}
	return CSI + s(n) + "D"
}
func CursorNextLine(n int) string           { return CSI + s(n) + "E" }
func CursorPreviousLine(n int) string       { return CSI + s(n) + "F" }
func CursorHorizontalAbsolute(n int) string { return CSI + s(n) + "G" }

type ED int // Erase in Display
type EL int // Erase in Line

const (
	// Control Sequence Introducer | Starts most of the useful sequences, terminated by a byte in the range
	// 0x40 through 0x7E.
	CSI = "\033["

	CursorToScreenEnd         ED = 0
	CursorToScreenBegin       ED = 1
	CursorScreen              ED = 2
	CursorScreenAndScrollBack ED = 3

	CursorToEndOfLine   EL = 0
	CursorToBeginOfLine EL = 1
	EntireLine          EL = 2

	R          = CSI + "0m"
	HideCursor = CSI + "?25l"
	ShowCursor = CSI + "?25h"
)

// Compacted when defaults are passed, some chars may elided:
//
// > The values are 1-based, and default to '1' (top left corner) if omitted. A sequence such as 'CSI ;5H' is a
// > synonym for 'CSI 1;5H' as well as 'CSI 17;H' is the same as 'CSI 17H' and 'CSI 17;1H'. [wikipedia]
//
// [wikipedia]: https://en.wikipedia.org/wiki/ANSI_escape_code
func CursorPosition(row, column int) string {
	if row == 1 && column == 1 {
		return CSI + "H"
	} else if row == 1 {
		return CSI + ";" + s(column) + "H"
	} else if column == 1 {
		return CSI + s(row) + "H"
	}
	return CSI + s(row) + ";" + s(column) + "H"
}

func EraseInDisplay(n ED) string { return CSI + s(int(n)) + "J" }
func EraseInLine(n EL) string    { return CSI + s(int(n)) + "K" }

// Colours Section:

func Black(s string) string     { return CSI + "30m" + s + R }
func Gray(s string) string      { return CSI + "90m" + s + R }
func LightGray(s string) string { return CSI + "37m" + s + R }
func White(s string) string     { return CSI + "97m" + s + R }

func DarkRed(s string) string     { return CSI + "31m" + s + R }
func DarkGreen(s string) string   { return CSI + "32m" + s + R }
func DarkYellow(s string) string  { return CSI + "33m" + s + R }
func DarkBlue(s string) string    { return CSI + "34m" + s + R }
func DarkMagenta(s string) string { return CSI + "35m" + s + R }
func DarkCyan(s string) string    { return CSI + "36m" + s + R }

func Red(s string) string     { return CSI + "91m" + s + R }
func Green(s string) string   { return CSI + "92m" + s + R }
func Yellow(s string) string  { return CSI + "93m" + s + R }
func Blue(s string) string    { return CSI + "94m" + s + R }
func Magenta(s string) string { return CSI + "95m" + s + R }
func Cyan(s string) string    { return CSI + "96m" + s + R }

// Internal

var s = strconv.Itoa
