// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/errors"
	"github.com/Lexer747/AcciPing/utils/th"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStats(t *testing.T) {
	t.Parallel()
	type Case struct {
		Values                    []time.Duration
		ExpectedMean              float64
		ExpectedVariance          float64
		ExpectedStandardDeviation float64
		ExpectedMin, ExpectedMax  time.Duration
	}
	testCases := []Case{
		{
			Values: []time.Duration{ // Times in milliseconds
				5 * time.Millisecond,
				6 * time.Millisecond,
				5 * time.Millisecond,
				7 * time.Millisecond,
				3 * time.Millisecond,
			},
			ExpectedMean:              5.2 * 1000 * 1000,
			ExpectedVariance:          2.2 * 1000 * 1000 * 1000 * 1000,
			ExpectedStandardDeviation: 1.4832397 * 1000 * 1000,
			ExpectedMin:               3 * time.Millisecond,
			ExpectedMax:               7 * time.Millisecond,
		},
		{
			Values:                    []time.Duration{},
			ExpectedMean:              0,
			ExpectedVariance:          0,
			ExpectedStandardDeviation: 0,
			ExpectedMin:               0,
			ExpectedMax:               0,
		},
		{
			Values:                    []time.Duration{8, 9, 10, 11, 7, 9},
			ExpectedMean:              9,
			ExpectedVariance:          2,
			ExpectedStandardDeviation: 1.4142136,
			ExpectedMin:               7,
			ExpectedMax:               11,
		},
		{
			Values:                    []time.Duration{1, 1, 1, 1, 1, 1, 1, 1},
			ExpectedMean:              1,
			ExpectedVariance:          0,
			ExpectedStandardDeviation: 0,
			ExpectedMin:               1,
			ExpectedMax:               1,
		},
		{
			Values:                    []time.Duration{1001, 1002, 1003},
			ExpectedMean:              1002,
			ExpectedVariance:          1,
			ExpectedStandardDeviation: 1,
			ExpectedMin:               1001,
			ExpectedMax:               1003,
		},
		{
			// https://oeis.org/A000055
			Values: []time.Duration{
				1, 1, 1, 1, 2, 3, 6, 11, 23, 47, 106, 235, 551, 1301, 3159,
				7741, 19320, 48629, 123867, 317955, 823065, 2144505, 5623756, 14828074,
				39299897, 104636890, 279793450, 751065460, 2023443032, 5469566585,
				14830871802, 40330829030, 109972410221, 300628862480, 823779631721,
				2262366343746, 6226306037178,
			},
			ExpectedMean:              264510990000,
			ExpectedVariance:          11688720e+17,
			ExpectedStandardDeviation: 1081144100000,
			ExpectedMin:               1,
			ExpectedMax:               6226306037178,
		},
		{
			Values: []time.Duration{
				1, -4, -4, -4, 2, 3, 6, -41, 23, 47, -406, 235, 551, -4301, 3159,
				7741, -49320, 48629, -423867, 317955, 823065, 2144505, 5623756, -44828074,
				39299897, -404636890, 279793450, 751065460, 2023443032, 5469566585,
				14830871802, 40330829030, -409972410221, 300628862480, 823779631721,
				-2262366343746, -6226306037178,
			},
			ExpectedMean:              -208404120000,
			ExpectedVariance:          12004762e+17,
			ExpectedStandardDeviation: 1095662500000,
			ExpectedMin:               -6226306037178,
			ExpectedMax:               823779631721,
		},
	}
	for i, test := range testCases {
		t.Run(fmt.Sprintf("%d:%+v", i, test.Values), func(t *testing.T) {
			t.Parallel()
			asSlice := graph.Stats{}
			asSlice.AddPoints(test.Values)
			asSingles := graph.Stats{}
			for _, p := range test.Values {
				asSingles.AddPoint(p)
			}
			th.AssertFloatEqual(t, test.ExpectedMean, asSlice.Mean, 7, "asSlice Mean")
			th.AssertFloatEqual(t, test.ExpectedMean, asSingles.Mean, 7, "asSingles Mean")
			th.AssertFloatEqual(t, test.ExpectedVariance, asSlice.Variance, 7, "asSlice Variance")
			th.AssertFloatEqual(t, test.ExpectedVariance, asSingles.Variance, 7, "asSingles Variance")
			th.AssertFloatEqual(t, test.ExpectedStandardDeviation, asSlice.StandardDeviation, 5, "asSlice StandardDeviation")
			th.AssertFloatEqual(t, test.ExpectedStandardDeviation, asSingles.StandardDeviation, 5, "asSingles StandardDeviation")
			assert.Equal(t, test.ExpectedMax, asSlice.Max, "asSlice Max")
			assert.Equal(t, test.ExpectedMax, asSingles.Max, "asSingles Max")
			assert.Equal(t, test.ExpectedMin, asSlice.Min, "asSlice Min")
			assert.Equal(t, test.ExpectedMin, asSingles.Min, "asSingles Min")
			assert.Equal(t, uint(len(test.Values)), asSlice.GoodCount, "asSlice Count")
			assert.Equal(t, uint(len(test.Values)), asSingles.GoodCount, "asSingles Count")
		})
	}
}

func assertStatsEqual(t *testing.T, expected graph.Stats, actual graph.Stats, sigFigs int, msgAndArgs ...interface{}) {
	t.Helper()
	th.AssertFloatEqual(t, expected.Mean, actual.Mean, sigFigs, msgAndArgs...)
	th.AssertFloatEqual(t, expected.Variance, actual.Variance, sigFigs, msgAndArgs...)
	th.AssertFloatEqual(t, expected.StandardDeviation, actual.StandardDeviation, sigFigs, msgAndArgs...)
	if expected.GoodCount != 0 {
		assert.Equal(t, expected.GoodCount, actual.GoodCount, msgAndArgs...)
	}
}

func assertTimeSpanEqual(t *testing.T, expected graph.TimeSpan, actual graph.TimeSpan, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Equal(t, expected.Begin, actual.Begin, msgAndArgs...)
	assert.Equal(t, expected.End, actual.End, msgAndArgs...)
	assert.Equal(t, expected.Duration, actual.Duration, msgAndArgs...)
}

type BlockTest struct {
	ExpectedBlocks    []graph.Block
	ExpectedGradients []float64
	CheckGradient     bool
	CheckRaw          bool
}

type DataTestCase struct {
	Values             []ping.PingResults
	BlockSize          int
	ExpectedGraphSpan  graph.TimeSpan
	ExpectedGraphStats graph.Stats
	ExpectedPacketLoss float64
	ExpectedTotalCount int
	BlockTest          *BlockTest
}

func TestData(t *testing.T) {
	t.Parallel()
	// A fixed time stamp to make all testing relative too
	origin := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []DataTestCase{
		{
			Values: []ping.PingResults{
				{Duration: 5 * time.Millisecond, Timestamp: origin},
				{Duration: 6 * time.Millisecond, Timestamp: origin.Add(time.Minute)},
				{Duration: 5 * time.Millisecond, Timestamp: origin.Add(2 * time.Minute)},
				{Duration: 7 * time.Millisecond, Timestamp: origin.Add(3 * time.Minute)},
				{Duration: 3 * time.Millisecond, Timestamp: origin.Add(4 * time.Minute)},
			},
			ExpectedGraphSpan: graph.TimeSpan{
				Begin:    origin,
				End:      origin.Add(4 * time.Minute),
				Duration: 4 * time.Minute,
			},
			ExpectedGraphStats: graph.Stats{
				Mean:              asFloat64(5.2, time.Millisecond),
				GoodCount:         5,
				Variance:          2_200_000_000_000, // Variance isn't squared so it gets real big
				StandardDeviation: asFloat64(1.4832397, time.Millisecond),
			},
			ExpectedTotalCount: 5,
		},
		{
			Values: []ping.PingResults{
				{Duration: 5 * time.Nanosecond, Timestamp: origin},
				{Duration: 6 * time.Nanosecond, Timestamp: origin.Add(time.Nanosecond)},
				{Duration: 5 * time.Nanosecond, Timestamp: origin.Add(2 * time.Nanosecond)},
				{Duration: 7 * time.Nanosecond, Timestamp: origin.Add(3 * time.Nanosecond)},
				{Duration: 3 * time.Nanosecond, Timestamp: origin.Add(4 * time.Nanosecond)},
				{Duration: 5 * time.Nanosecond, Timestamp: origin.Add(5 * time.Nanosecond)},
				{Duration: 6 * time.Nanosecond, Timestamp: origin.Add(6 * time.Nanosecond)},
				{Duration: 5 * time.Nanosecond, Timestamp: origin.Add(7 * time.Nanosecond)},
				{Duration: 7 * time.Nanosecond, Timestamp: origin.Add(8 * time.Nanosecond)},
				{Duration: 3 * time.Nanosecond, Timestamp: origin.Add(9 * time.Nanosecond)},
			},
			BlockSize: 5,
			ExpectedGraphSpan: graph.TimeSpan{
				Begin:    origin,
				End:      origin.Add(9 * time.Nanosecond),
				Duration: 9 * time.Nanosecond,
			},
			ExpectedGraphStats: graph.Stats{
				Mean:              asFloat64(5.2, time.Nanosecond),
				GoodCount:         10,
				Variance:          1.955,
				StandardDeviation: asFloat64(1.3984118, time.Nanosecond),
			},
			BlockTest: &BlockTest{
				// This test has enough data to split the storage over multiple blocks, the blocks are
				// near identical except timestamps.
				ExpectedBlocks: []graph.Block{{
					Header: &graph.Header{
						Stats: &graph.Stats{
							Mean:              asFloat64(5.2, time.Nanosecond),
							GoodCount:         5,
							Variance:          2.2,
							StandardDeviation: asFloat64(1.4832397, time.Nanosecond),
						},
						Span: &graph.TimeSpan{Begin: origin, End: origin.Add(4 * time.Nanosecond), Duration: 4 * time.Nanosecond},
					},
					Gradients: []float64{
						(6.0 - 5.0) / 1.0,
						(5.0 - 6.0) / 1.0,
						(7.0 - 5.0) / 1.0,
						(3.0 - 7.0) / 1.0,
					},
				}, {
					Header: &graph.Header{
						Stats: &graph.Stats{
							Mean:              asFloat64(5.2, time.Nanosecond),
							GoodCount:         5,
							Variance:          2.2,
							StandardDeviation: asFloat64(1.4832397, time.Nanosecond),
						},
						Span: &graph.TimeSpan{
							Begin:    origin.Add(5 * time.Nanosecond),
							End:      origin.Add(9 * time.Nanosecond),
							Duration: 4 * time.Nanosecond,
						},
					},
					Gradients: []float64{
						(6.0 - 5.0) / 1.0,
						(5.0 - 6.0) / 1.0,
						(7.0 - 5.0) / 1.0,
						(3.0 - 7.0) / 1.0,
					},
				}},
				ExpectedGradients: []float64{(5.0 - 3.0) / 1.0},
				CheckRaw:          false,
				CheckGradient:     true,
			},
			ExpectedTotalCount: 10,
		},
		{
			Values: []ping.PingResults{
				{Duration: 15 * time.Millisecond, Timestamp: origin},
				{Duration: 16 * time.Millisecond, Timestamp: origin.Add(10 * time.Minute)},
				ping.NewTestPingResult(errors.Errorf("oh noes"), origin.Add(20*time.Minute)),
				{Duration: 17 * time.Millisecond, Timestamp: origin.Add(30 * time.Minute)},
				{Duration: 13 * time.Millisecond, Timestamp: origin.Add(40 * time.Minute)},
			},
			ExpectedGraphSpan: graph.TimeSpan{
				Begin:    origin,
				End:      origin.Add(40 * time.Minute),
				Duration: 40 * time.Minute,
			},
			ExpectedGraphStats: graph.Stats{
				Mean:              asFloat64(15.25, time.Millisecond),
				Variance:          2_916_666_700_000,
				StandardDeviation: asFloat64(1.7078251, time.Millisecond),
			},
			ExpectedPacketLoss: 1.0 / 5.0,
			ExpectedTotalCount: 5,
		},
	}

	for i, test := range testCases {
		t.Run(fmt.Sprintf("%d:%+v", i, test.Values), func(t *testing.T) {
			t.Parallel()
			graphData := graph.NewData()
			if test.BlockSize != 0 {
				graphData = graph.NewData(graph.Options{BlockSize: test.BlockSize})
			}
			for _, v := range test.Values {
				graphData.AddPoint(v)
			}
			assertStatsEqual(t, test.ExpectedGraphStats, *graphData.Header.Stats, 3, "global graph header")
			assertTimeSpanEqual(t, test.ExpectedGraphSpan, *graphData.Header.Span, "global graph header")
			th.AssertFloatEqual(t, test.ExpectedPacketLoss, graphData.Header.Stats.PacketLoss(), 5, "global packet loss percent")
			if test.BlockTest != nil {
				blockVerify(t, graphData, test)
			}
		})
	}
}

func blockVerify(t *testing.T, graphData *graph.Data, test DataTestCase) {
	t.Helper()
	require.Len(t, graphData.Blocks, len(test.BlockTest.ExpectedBlocks))
	for i, block := range graphData.Blocks {
		expectedBlock := test.BlockTest.ExpectedBlocks[i]
		assertStatsEqual(t, *expectedBlock.Stats, *block.Header.Stats, 4)
		assertTimeSpanEqual(t, *expectedBlock.Span, *block.Header.Span, 4)
		if test.BlockTest.CheckRaw {
			require.Lenf(t, block.Raw, len(expectedBlock.Raw), "block %d was unexpected len", i)
			for rawIndex, datum := range block.Raw {
				assert.Equal(t, expectedBlock.Raw[rawIndex], datum, "raw inside block %d at index %d", i, rawIndex)
			}
		}
		if test.BlockTest.CheckGradient {
			require.Lenf(t, block.Gradients, len(expectedBlock.Gradients), "block %d was unexpected len", i)
			expectedMin := block.Gradients[0]
			expectedMax := block.Gradients[0]
			for gradientIndex, datum := range block.Gradients {
				expected := expectedBlock.Gradients[gradientIndex]
				expectedMin = min(expectedMin, expected)
				expectedMax = max(expectedMax, expected)
				th.AssertFloatEqual(t, expected, datum, 6, "gradient inside block %d at index %d", i, gradientIndex)
			}
			th.AssertFloatEqual(t, expectedMin, block.MinGradient, 4, "min gradient for block %d", i)
			th.AssertFloatEqual(t, expectedMax, block.MaxGradient, 4, "max gradient for block %d", i)
			if i > 0 {
				th.AssertFloatEqual(t, test.BlockTest.ExpectedGradients[i-1], graphData.BetweenBlockGradients[i-1], 6,
					"gradient between blocks %d and %d", i-1, i)
			}
		}
	}
}

func asFloat64(scalar float64, t time.Duration) float64 {
	return scalar * float64(t)
}
