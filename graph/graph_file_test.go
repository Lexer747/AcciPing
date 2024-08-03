// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/graph/terminal/th"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/stretchr/testify/require"
)

type FileTest struct {
	FileName           string
	Size               terminal.Size
	ExpectedOutputFile string
}

func TestFiles(t *testing.T) {
	t.Parallel()
	t.Run("Small", FileTest{
		FileName:           "data/testdata/small-2-02-08-2024.pings",
		Size:               terminal.Size{Height: 25, Width: 80},
		ExpectedOutputFile: "data/testdata/small-2-02-08-2024.frame",
	}.Run)
	t.Run("Medium", FileTest{
		FileName:           "data/testdata/medium-395-02-08-2024.pings",
		Size:               terminal.Size{Height: 25, Width: 80},
		ExpectedOutputFile: "data/testdata/medium-395-02-08-2024.frame",
	}.Run)
	t.Run("Medium with Drops", FileTest{
		FileName:           "data/testdata/medium-309-with-induced-drops-02-08-2024.pings",
		Size:               terminal.Size{Height: 25, Width: 80},
		ExpectedOutputFile: "data/testdata/medium-309-with-induced-drops-02-08-2024.frame",
	}.Run)
	t.Run("Medium with minute Gaps", FileTest{
		FileName:           "data/testdata/medium-minute-gaps.pings",
		Size:               terminal.Size{Height: 25, Width: 80},
		ExpectedOutputFile: "data/testdata/medium-minute-gaps.frame",
	}.Run)
	t.Run("Medium with hour Gaps", FileTest{
		FileName:           "data/testdata/medium-hour-gaps.pings",
		Size:               terminal.Size{Height: 25, Width: 80},
		ExpectedOutputFile: "data/testdata/medium-hour-gaps.frame",
	}.Run)
}

func (ft FileTest) Run(t *testing.T) {
	t.Parallel()
	f, err := os.OpenFile(ft.FileName, os.O_RDONLY, 0)
	require.NoError(t, err)
	defer f.Close()
	d, err := data.ReadData(f)
	require.NoError(t, err)

	actualStrings := produceFrame(t, ft.Size, d)

	// ft.update(t, actualStrings)
	ft.requireEqual(t, actualStrings)
}

func (ft FileTest) requireEqual(t *testing.T, actualStrings []string) {
	t.Helper()
	expectedBytes, err := os.ReadFile(ft.ExpectedOutputFile)
	require.NoError(t, err)
	actualJoined := strings.Join(actualStrings, "\n")
	actualOutput := ft.ExpectedOutputFile + ".actual"
	if string(expectedBytes) != actualJoined {
		err := os.WriteFile(actualOutput, []byte(actualJoined), 0o777)
		require.NoError(t, err)
		t.Fatalf("Diff in outputs see %s", actualOutput)
	} else {
		os.Remove(actualOutput)
	}
}

//nolint:unused
func (ft FileTest) update(t *testing.T, actualStrings []string) {
	t.Helper()
	err := os.WriteFile(ft.ExpectedOutputFile, []byte(strings.Join(actualStrings, "\n")), 0o777)
	require.NoError(t, err)
	t.Fatal("Only call update drawing once")
}

func produceFrame(t *testing.T, size terminal.Size, data *data.Data) []string {
	t.Helper()
	stdin, _, term, setTerm, err := th.NewTestTerminal()
	setTerm(size)
	ctx, cancel := context.WithCancel(context.Background())
	// cancel this, we don't want the graph collecting from the channel in the background
	cancel()
	require.NoError(t, err)
	pingChannel := make(chan ping.PingResults)
	close(pingChannel)
	g, err := graph.NewGraphWithData(ctx, pingChannel, term, 0, data)
	require.NoError(t, err)
	defer func() { stdin.WriteCtrlC(t) }()
	output := makeBuffer(size)
	return playAnsiOntoStringBuffer(g.ComputeFrame(), output, size)
}
