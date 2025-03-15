// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph_test

import (
	"context"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Lexer747/acci-ping/draw"
	"github.com/Lexer747/acci-ping/graph"
	"github.com/Lexer747/acci-ping/graph/terminal"
	"github.com/Lexer747/acci-ping/graph/terminal/th"
	"github.com/Lexer747/acci-ping/gui"
	"github.com/Lexer747/acci-ping/ping"
	"github.com/Lexer747/acci-ping/utils/env"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
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

//nolint:unused
func updateDrawingTest(t *testing.T, test DrawingTest) {
	t.Helper()
	actual := drawGraph(t, test.Size, test.Values)
	err := os.WriteFile(test.ExpectedFile, []byte(strings.Join(actual, "\n")), 0o777)
	assert.NilError(t, err)
	t.Fatal("Only call update drawing once")
}

func drawingTest(t *testing.T, test DrawingTest) {
	// updateDrawingTest(t, test)
	t.Helper()
	actualStrings := drawGraph(t, test.Size, test.Values)
	expectedBytes, err := os.ReadFile(test.ExpectedFile)
	assert.NilError(t, err)
	actualJoined := strings.Join(actualStrings, "\n")
	expected := string(expectedBytes)
	if env.LOCAL_FRAME_DIFFS() {
		actualOutput := test.ExpectedFile + ".actual"
		if expected != actualJoined {
			err := os.WriteFile(actualOutput, []byte(actualJoined), 0o777)
			assert.NilError(t, err)
			t.Logf("Diff in outputs see %s", actualOutput)
			t.Fail()
		} else {
			os.Remove(actualOutput)
		}
	} else {
		assert.Check(t, is.Equal(expected, actualJoined), test.ExpectedFile)
	}
}

func drawGraph(t *testing.T, size terminal.Size, input []ping.PingDataPoint) []string {
	t.Helper()
	if len(input) == 1 {
		panic("drawGraph test doesn't work on inputs size 1")
	}
	g, closer, err := initTestGraph(t, size)
	assert.NilError(t, err)
	defer closer()

	actual := eval(t, g, input)
	output := makeBuffer(size)
	return playAnsiOntoStringBuffer(actual, output, size)
}

func initTestGraph(t *testing.T, size terminal.Size) (*graph.Graph, func(), error) {
	t.Helper()
	stdin, _, term, setTerm, err := th.NewTestTerminal()
	setTerm(size)
	ctx, cancel := context.WithCancel(context.Background())
	// cancel this, we don't want the graph collecting from the channel in the background
	cancel()
	assert.NilError(t, err)
	pingChannel := make(chan ping.PingResults)
	defer close(pingChannel)
	g := graph.NewGraph(ctx, pingChannel, term, gui.NoGUI(), 0, "", draw.NewPaintBuffer())
	return g, func() { stdin.WriteCtrlC(t) }, err
}

func eval(t *testing.T, g *graph.Graph, input []ping.PingDataPoint) string {
	t.Helper()
	for _, p := range input {
		g.AddPoint(ping.PingResults{Data: p, IP: []byte{}})
	}
	assert.Equal(t, int64(len(input)), g.Size())
	actual := g.ComputeFrame()
	assert.Equal(t, int64(len(input)), g.Size())
	return actual
}
