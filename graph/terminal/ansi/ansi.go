// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package ansi

import "strconv"

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

	R = CSI + "0m"
)

var s = strconv.Itoa

var Clear = EraseInDisplay(CursorScreen)
var Home = CursorPosition(1, 1)

func CursorUp(n int) string                 { return CSI + s(n) + "A" }
func CursorDown(n int) string               { return CSI + s(n) + "B" }
func CursorForward(n int) string            { return CSI + s(n) + "C" }
func CursorBack(n int) string               { return CSI + s(n) + "D" }
func CursorNextLine(n int) string           { return CSI + s(n) + "E" }
func CursorPreviousLine(n int) string       { return CSI + s(n) + "F" }
func CursorHorizontalAbsolute(n int) string { return CSI + s(n) + "G" }

func CursorPosition(row, column int) string { return CSI + s(row) + ";" + s(column) + "H" }

func EraseInDisplay(n ED) string { return CSI + s(int(n)) + "J" }
func EraseInLine(n EL) string    { return CSI + s(int(n)) + "K" }

func Black(s string) string     { return CSI + "30" + s + R }
func Gray(s string) string      { return CSI + "90" + s + R }
func LightGray(s string) string { return CSI + "37" + s + R }
func White(s string) string     { return CSI + "97" + s + R }

func DarkRed(s string) string     { return CSI + "31" + s + R }
func DarkGreen(s string) string   { return CSI + "32" + s + R }
func DarkYellow(s string) string  { return CSI + "33" + s + R }
func DarkBlue(s string) string    { return CSI + "34" + s + R }
func DarkMagenta(s string) string { return CSI + "35" + s + R }
func DarkCyan(s string) string    { return CSI + "36" + s + R }

func Red(s string) string     { return CSI + "91" + s + R }
func Green(s string) string   { return CSI + "92" + s + R }
func Yellow(s string) string  { return CSI + "93" + s + R }
func Blue(s string) string    { return CSI + "94" + s + R }
func Magenta(s string) string { return CSI + "95" + s + R }
func Cyan(s string) string    { return CSI + "96" + s + R }
