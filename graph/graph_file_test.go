// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/terminal"
	termTh "github.com/Lexer747/AcciPing/graph/terminal/th"
	graphTh "github.com/Lexer747/AcciPing/graph/th"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/stretchr/testify/require"
)

const (
	inputPath  = "data/testdata/input"
	outputPath = "data/testdata/output"
)

type FileTest struct {
	FileName string
	Sizes    []terminal.Size
}

var StandardTestSizes = []terminal.Size{
	{Height: 40, Width: 80}, // Viewing height
	{Height: 25, Width: 80},
	{Height: 16, Width: 284}, // My small vscode window
	{Height: 30, Width: 300}, // My average vscode window
	{Height: 74, Width: 354}, // Fullscreen
}

func TestFiles(t *testing.T) {
	t.Parallel()
	t.Run("Small", FileTest{
		FileName: "small-2-02-08-2024",
		Sizes:    StandardTestSizes,
	}.Run)
	t.Run("Medium", FileTest{
		FileName: "medium-395-02-08-2024",
		Sizes:    StandardTestSizes,
	}.Run)
	t.Run("Medium with Drops", FileTest{
		FileName: "medium-309-with-induced-drops-02-08-2024",
		Sizes:    StandardTestSizes,
	}.Run)
	t.Run("Medium with minute Gaps", FileTest{
		FileName: "medium-minute-gaps",
		Sizes:    StandardTestSizes,
	}.Run)
	t.Run("Medium with hour Gaps", FileTest{
		FileName: "medium-hour-gaps",
		Sizes:    StandardTestSizes,
	}.Run)
	t.Run("Hotel", FileTest{
		FileName: "medium-hotel",
		Sizes:    StandardTestSizes,
	}.Run)
	t.Run("Large Hotel", FileTest{
		FileName: "large-hotel",
		Sizes:    StandardTestSizes,
	}.Run)
	t.Run("Gap", FileTest{
		FileName: "long-gap",
		Sizes:    StandardTestSizes,
	}.Run)
	t.Run("Smoke Test", FileTest{
		FileName: "smoke",
		Sizes:    StandardTestSizes,
	}.Run)
	t.Run("Span bugs", FileTest{
		FileName: "huge-over-days",
		Sizes:    StandardTestSizes,
	}.Run)
}

func (ft FileTest) Run(t *testing.T) {
	t.Parallel()

	d := graphTh.GetFromFile(t, ft.getInputFileName())
	for _, size := range ft.Sizes {
		actualStrings := produceFrame(t, size, d)

		// ft.update(t, size, actualStrings)
		ft.assertEqual(t, size, actualStrings)
	}
}

func (ft FileTest) assertEqual(t *testing.T, size terminal.Size, actualStrings []string) {
	t.Helper()
	outputFile := ft.getOutputFileName(size)
	expectedBytes, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	actualJoined := strings.Join(actualStrings, "\n")
	actualOutput := outputFile + ".actual"
	if string(expectedBytes) != actualJoined {
		err := os.WriteFile(actualOutput, []byte(actualJoined), 0o777)
		require.NoError(t, err)
		t.Logf("Diff in outputs see %s", actualOutput)
		t.Fail()
	} else {
		os.Remove(actualOutput)
	}
}

func (ft FileTest) getInputFileName() string {
	return fmt.Sprintf("%s/%s.pings", inputPath, ft.FileName)
}
func (ft FileTest) getOutputFileName(size terminal.Size) string {
	return fmt.Sprintf("%s/%s/w%d-h%d.frame", outputPath, ft.FileName, size.Width, size.Height)
}

//nolint:unused
func (ft FileTest) update(t *testing.T, size terminal.Size, actualStrings []string) {
	t.Helper()
	outputFile := ft.getOutputFileName(size)
	err := os.MkdirAll(path.Dir(outputFile), 0o777)
	require.NoError(t, err)
	err = os.WriteFile(outputFile, []byte(strings.Join(actualStrings, "\n")), 0o777)
	require.NoError(t, err)
	t.Fail()
	t.Log("Only call update drawing once")
}

func produceFrame(t *testing.T, size terminal.Size, data *data.Data) []string {
	t.Helper()
	stdin, _, term, setTerm, err := termTh.NewTestTerminal()
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
