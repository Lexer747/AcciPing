// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph_test

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Lexer747/acci-ping/graph/terminal"
	"github.com/Lexer747/acci-ping/graph/terminal/ansi"
	"github.com/Lexer747/acci-ping/utils/check"
)

func makeBuffer(size terminal.Size) []string {
	output := make([]string, size.Height)
	for i := range output {
		output[i] = strings.Repeat(" ", size.Width)
	}
	return output
}

type ansiState struct {
	cursorRow, cursorColumn int
	buffer                  []string
	size                    terminal.Size
	ansiText                string
	asRunes                 []rune
	head                    int
	outOfBounds             bool
}

func (a *ansiState) peekN(n int) rune     { return a.asRunes[a.head+n] }
func (a *ansiState) peek() rune           { return a.peekN(1) }
func (a *ansiState) isNext(r rune) bool   { return a.peek() == r }
func (a *ansiState) consume()             { a.head++ }
func (a *ansiState) isDigit() bool        { return a.peek() >= '0' && a.peek() <= '9' }
func (a *ansiState) isNegativeSign() bool { return a.peek() == '-' }
func (a *ansiState) consumeIfNext(r rune) bool {
	if ok := a.isNext(r); ok {
		a.consume()
		return true
	}
	return false
}
func (a *ansiState) consumeExact(s string) {
	start := a.head - 1
	for _, r := range s {
		check.Checkf(a.consumeIfNext(r), "consumeExact Expected %q got %q", s, string(a.asRunes[start:a.head]))
	}
}
func (a *ansiState) consumeOneOf(s string) bool {
	for _, r := range s {
		if a.consumeIfNext(r) {
			return true
		}
	}
	return false
}
func (a *ansiState) consumeDigits() int {
	digits := []rune{}
	if a.isNegativeSign() {
		digits = append(digits, '-')
		a.consume()
	}
	for a.isDigit() {
		digits = append(digits, a.peek())
		a.consume()
	}
	parsed, _ := strconv.ParseInt(string(digits), 10, 0)
	return int(parsed)
}

func playAnsiOntoStringBuffer(ansiText string, buffer []string, size terminal.Size) []string {
	a := &ansiState{
		cursorRow:    1,
		cursorColumn: 1,
		buffer:       buffer,
		size:         size,
		ansiText:     ansiText,
		asRunes:      []rune(ansiText),
		head:         0,
	}

	for {
		c := a.peekN(0)
		switch c {
		case '\033':
			start := a.head
			if a.consumeIfNext('[') {
				a.handleControl(start)
			}
		default:
			a.write(c)
			a.changeCursor(a.cursorColumn+1, a.cursorRow)
		}
		a.consume()
		if a.EoF() {
			break
		}
	}
	return a.buffer
}

func (a *ansiState) EoF() bool {
	return a.head >= len(a.asRunes)
}

func (a *ansiState) write(c rune) {
	if a.outOfBounds {
		panic(fmt.Sprintf("row out of bounds writing (%c): row is %d, terminal is %d", c, a.cursorRow, a.size.Height))
	}
	y := []rune(a.buffer[a.cursorRow-1])
	y[a.cursorColumn-1] = c
	a.buffer[a.cursorRow-1] = string(y)
}

func (a *ansiState) handleControl(start int) {
	switch {
	case a.isNext('?'):
		// show hide cursor control bytes
		a.consumeExact("25")
		if !a.consumeOneOf("lh") {
			panic(fmt.Sprintf("unexpected control byte sequence %q", string(a.asRunes[start:a.head])))
		}
	case a.isNext('H'): // CursorPosition
		// Shortest possible hand for 'CSI1;1H'
		a.changeCursor(1, 1)
		a.consume()
	case a.isNext(';'): // CursorPosition
		// The first row param has been omitted (meaning it's one)
		a.consume()
		d := a.consumeDigits()
		a.consumeExact("H")
		a.changeCursor(d, 1)
	case a.isDigit() || a.isNegativeSign():
		d := a.consumeDigits()
		switch a.peek() {
		case 'm':
			a.consume()
		case ';': // CursorPosition
			// Both params present
			a.consume()
			col := a.consumeDigits()
			a.consumeExact("H")
			a.changeCursor(col, d)
		case 'H': // CursorPosition
			// The second column param has been omitted (meaning it's one)
			a.changeCursor(1, d)
			a.consume()
		case 'A': // CursorUp
			a.changeCursor(a.cursorColumn, a.cursorRow-d)
			a.consume()
		case 'B': // CursorDown
			a.changeCursor(a.cursorColumn, a.cursorRow+d)
			a.consume()
		case 'C': // CursorForward
			a.changeCursor(a.cursorColumn+d, a.cursorRow)
			a.consume()
		case 'D': // CursorBack
			a.changeCursor(a.cursorColumn-d, a.cursorRow)
			a.consume()
		case 'E': // CursorNextLine
			panic("todo CursorNextLine")
		case 'F': // CursorPreviousLine
			panic("todo CursorPreviousLine")
		case 'G': // CursorHorizontalAbsolute
			panic("todo CursorHorizontalAbsolute")
		case 'J': // EraseInDisplay
			switch ansi.ED(d) {
			case ansi.CursorToScreenEnd:
			case ansi.CursorToScreenBegin:
			case ansi.CursorScreen:
				a.buffer = makeBuffer(a.size)
			case ansi.CursorScreenAndScrollBack:
			default:
				panic("unknown EraseInDisplay enum")
			}
			a.consume()
		case 'K': // EraseInLine
			panic("todo EraseInLine")
		}
	default:
	}
}

func (a *ansiState) changeCursor(newC, newR int) {
	a.cursorColumn = newC
	a.cursorRow = newR
	// Positive wrapping, go to the next line
	if a.cursorColumn > a.size.Width {
		a.cursorColumn = 1
		a.cursorRow++
	}
	// Negative wrapping, go back to the last line, last col
	if a.cursorColumn <= 0 {
		a.cursorColumn = a.size.Width - 1
		a.cursorRow--
	}
	if a.cursorRow > a.size.Height {
		a.outOfBounds = true
	} else {
		a.outOfBounds = false
	}
	check.Check(a.cursorColumn != 0 && a.cursorRow != 0, "cursor should not be 0")
}
