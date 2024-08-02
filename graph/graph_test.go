// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/graph/terminal/ansi"
	"github.com/Lexer747/AcciPing/graph/terminal/th"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/check"
	"github.com/stretchr/testify/require"
)

func TestSmallDrawing(t *testing.T) {
	t.Parallel()
	test := DrawingTest{
		Size: terminal.Size{Height: 5, Width: 20},
		Values: []ping.PingDataPoint{
			{Duration: 1 * time.Second, Timestamp: time.Time{}.Add(1 * time.Second)},
			{Duration: 2 * time.Second, Timestamp: time.Time{}.Add(2 * time.Second)},
			{Duration: 3 * time.Second, Timestamp: time.Time{}.Add(3 * time.Second)},
		},
		ExpectedFile: "testdata/small.frame",
	}
	drawingTest(t, test)
}

func TestNegativeGradientDrawing(t *testing.T) {
	t.Parallel()
	test := DrawingTest{
		Size: terminal.Size{Height: 15, Width: 80},
		Values: []ping.PingDataPoint{
			{Duration: 6 * time.Second, Timestamp: time.Time{}.Add(1 * time.Second)},
			{Duration: 5 * time.Second, Timestamp: time.Time{}.Add(2 * time.Second)},
			{Duration: 4 * time.Second, Timestamp: time.Time{}.Add(3 * time.Second)},
			{Duration: 3 * time.Second, Timestamp: time.Time{}.Add(4 * time.Second)},
			{Duration: 2 * time.Second, Timestamp: time.Time{}.Add(5 * time.Second)},
			{Duration: 1 * time.Second, Timestamp: time.Time{}.Add(6 * time.Second)},
		},
		ExpectedFile: "testdata/negative-gradient.frame",
	}
	drawingTest(t, test)
}

func TestPacketLossDrawing(t *testing.T) {
	t.Parallel()
	test := DrawingTest{
		Size: terminal.Size{Height: 15, Width: 80},
		Values: []ping.PingDataPoint{
			{Duration: 6 * time.Second, Timestamp: time.Time{}.Add(1 * time.Second)},
			{Duration: 5 * time.Second, Timestamp: time.Time{}.Add(2 * time.Second)},
			{DropReason: ping.TestDrop, Timestamp: time.Time{}.Add(3 * time.Second)},
			{DropReason: ping.TestDrop, Timestamp: time.Time{}.Add(4 * time.Second)},
			{DropReason: ping.TestDrop, Timestamp: time.Time{}.Add(5 * time.Second)},
			{Duration: 4 * time.Second, Timestamp: time.Time{}.Add(6 * time.Second)},
			{Duration: 3 * time.Second, Timestamp: time.Time{}.Add(7 * time.Second)},
			{Duration: 2 * time.Second, Timestamp: time.Time{}.Add(8 * time.Second)},
			{DropReason: ping.TestDrop, Timestamp: time.Time{}.Add(9 * time.Second)},
			{DropReason: ping.TestDrop, Timestamp: time.Time{}.Add(10 * time.Second)},
			{Duration: 7 * time.Second, Timestamp: time.Time{}.Add(11 * time.Second)},
			{DropReason: ping.TestDrop, Timestamp: time.Time{}.Add(12 * time.Second)},
			{Duration: 4 * time.Second, Timestamp: time.Time{}.Add(13 * time.Second)},
			{Duration: 4 * time.Second, Timestamp: time.Time{}.Add(14 * time.Second)},
			{Duration: 13 * time.Second, Timestamp: time.Time{}.Add(15 * time.Second)},
		},
		ExpectedFile: "testdata/packet-loss.frame",
	}
	drawingTest(t, test)
}

func TestLargeDrawing(t *testing.T) {
	t.Parallel()
	test := DrawingTest{
		Size: terminal.Size{Height: 35, Width: 160},
		Values: []ping.PingDataPoint{
			{Duration: 1 * time.Second, Timestamp: time.Time{}.Add(1 * time.Second)},
			{Duration: 2 * time.Second, Timestamp: time.Time{}.Add(2 * time.Second)},
			{Duration: 3 * time.Second, Timestamp: time.Time{}.Add(3 * time.Second)},
		},
		ExpectedFile: "testdata/large.frame",
	}
	drawingTest(t, test)
}

func TestManyDrawing(t *testing.T) {
	t.Parallel()
	// Fixed seed, used in testing only, not sec sensitive
	rng := rand.New(rand.NewPCG(4, 4)) //nolint:gosec
	test := DrawingTest{
		Size: terminal.Size{Height: 25, Width: 100},
		Values: []ping.PingDataPoint{
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(1*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(2*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(3*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(4*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(5*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(6*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(7*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(8*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(9*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(10*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(11*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(12*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(13*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(14*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
			{
				Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
				Timestamp: time.Time{}.Add(15*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
			},
		},
		ExpectedFile: "testdata/many.frame",
	}
	drawingTest(t, test)
}

func TestThousandsDrawing(t *testing.T) {
	t.Parallel()
	// Fixed seed, used in testing only, not sec sensitive
	rng := rand.New(rand.NewPCG(5, 5)) //nolint:gosec
	generated := make([]ping.PingDataPoint, 5000)
	for i := range generated {
		generated[i] = ping.PingDataPoint{
			Duration:  time.Duration(rng.Float64() * float64(10*time.Millisecond)),
			Timestamp: time.Time{}.Add(time.Duration(i)*time.Second + time.Duration(rng.Float64()*float64(time.Second))),
		}
	}
	test := DrawingTest{
		Size:         terminal.Size{Height: 25, Width: 100},
		Values:       generated,
		ExpectedFile: "testdata/thousands.frame",
	}
	drawingTest(t, test)
}

type DrawingTest struct {
	Size         terminal.Size
	Values       []ping.PingDataPoint
	ExpectedFile string
}

// updateDrawingTest used for updating goldens.
//
//nolint:unused
func updateDrawingTest(t *testing.T, test DrawingTest) {
	t.Helper()
	actual := drawGraph(t, test.Size, test.Values)
	err := os.WriteFile(test.ExpectedFile, []byte(strings.Join(actual, "\n")), 0o777)
	require.NoError(t, err)
	t.Fatal("Only call update drawing once")
}

func drawingTest(t *testing.T, test DrawingTest) {
	// updateDrawingTest(t, test)
	t.Helper()
	actualStrings := drawGraph(t, test.Size, test.Values)
	expectedBytes, err := os.ReadFile(test.ExpectedFile)
	require.NoError(t, err)
	actualJoined := strings.Join(actualStrings, "\n")
	actualOutput := test.ExpectedFile + ".actual"
	if string(expectedBytes) != actualJoined {
		err := os.WriteFile(actualOutput, []byte(actualJoined), 0o777)
		require.NoError(t, err)
		t.Fatalf("Diff in outputs see %s", actualOutput)
	} else {
		os.Remove(actualOutput)
	}
}

func drawGraph(t *testing.T, size terminal.Size, input []ping.PingDataPoint) []string {
	t.Helper()
	if len(input) == 1 {
		panic("drawGraph test doesn't work on inputs size 1")
	}
	g, closer, err := initTestGraph(t, size)
	require.NoError(t, err)
	defer closer()

	actual := eval(t, g, input)
	output := makeBuffer(size)
	return playAnsiOntoStringBuffer(actual, output, size)
}

func makeBuffer(size terminal.Size) []string {
	output := make([]string, size.Height)
	for i := range output {
		output[i] = strings.Repeat(" ", size.Width)
	}
	return output
}

func initTestGraph(t *testing.T, size terminal.Size) (*graph.Graph, func(), error) {
	t.Helper()
	stdin, _, term, setTerm, err := th.NewTestTerminal()
	setTerm(size)
	ctx, cancel := context.WithCancel(context.Background())
	// cancel this, we don't want the graph collecting from the channel in the background
	cancel()
	require.NoError(t, err)
	pingChannel := make(chan ping.PingResults)
	defer close(pingChannel)
	g, err := graph.NewGraph(ctx, pingChannel, term, 0, "")
	require.NoError(t, err)
	return g, func() { stdin.WriteCtrlC(t) }, err
}

func eval(t *testing.T, g *graph.Graph, input []ping.PingDataPoint) string {
	t.Helper()
	for _, p := range input {
		g.AddPoint(ping.PingResults{Data: p, IP: []byte{}})
	}
	require.Equal(t, int64(len(input)), g.Size())
	actual := g.ComputeFrame()
	require.Equal(t, int64(len(input)), g.Size())
	return actual
}

type ansiState struct {
	cursorRow, cursorColumn int
	buffer                  []string
	size                    terminal.Size
	ansiText                string
	asRunes                 []rune
	head                    int
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
	if a.cursorColumn > a.size.Width {
		a.cursorColumn = 1
		a.cursorRow++
	}
	if a.cursorRow > a.size.Height {
		panic("row out of bounds")
	}
	check.Check(a.cursorColumn != 0 && a.cursorRow != 0, "cursor should not be 0")
}
