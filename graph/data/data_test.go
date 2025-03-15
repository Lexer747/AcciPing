// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package data_test

import (
	"fmt"
	"math/rand/v2"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Lexer747/acci-ping/graph/data"
	"github.com/Lexer747/acci-ping/ping"
	"github.com/Lexer747/acci-ping/utils/sliceutils"
	"github.com/Lexer747/acci-ping/utils/th"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
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
			asSlice := data.Stats{}
			asSlice.AddPoints(test.Values)
			asSingles := data.Stats{}
			for _, p := range test.Values {
				asSingles.AddPoint(p)
			}
			th.AssertFloatEqual(t, test.ExpectedMean, asSlice.Mean, 7, "asSlice Mean")
			th.AssertFloatEqual(t, test.ExpectedMean, asSingles.Mean, 7, "asSingles Mean")
			th.AssertFloatEqual(t, test.ExpectedVariance, asSlice.Variance, 7, "asSlice Variance")
			th.AssertFloatEqual(t, test.ExpectedVariance, asSingles.Variance, 7, "asSingles Variance")
			th.AssertFloatEqual(t, test.ExpectedStandardDeviation, asSlice.StandardDeviation, 5, "asSlice StandardDeviation")
			th.AssertFloatEqual(t, test.ExpectedStandardDeviation, asSingles.StandardDeviation, 5, "asSingles StandardDeviation")
			assert.Check(t, is.Equal(test.ExpectedMax, asSlice.Max), "asSlice Max")
			assert.Check(t, is.Equal(test.ExpectedMax, asSingles.Max), "asSingles Max")
			assert.Check(t, is.Equal(test.ExpectedMin, asSlice.Min), "asSlice Min")
			assert.Check(t, is.Equal(test.ExpectedMin, asSingles.Min), "asSingles Min")
			assert.Check(t, is.Equal(uint64(len(test.Values)), asSlice.GoodCount), "asSlice Count")
			assert.Check(t, is.Equal(uint64(len(test.Values)), asSingles.GoodCount), "asSingles Count")
		})
	}
	type MergeCase struct {
		Inputs                    []int
		ExpectedMean              float64
		ExpectedVariance          float64
		ExpectedStandardDeviation float64
		ExpectedMin, ExpectedMax  time.Duration
	}
	mergeCases := []MergeCase{
		{
			Inputs:                    []int{0, 0, 0},
			ExpectedMean:              5.2 * 1000 * 1000,
			ExpectedVariance:          1.885714e+12, // Thanks to floating noise these diverge from correct values.
			ExpectedStandardDeviation: 1.3732e+06,
			ExpectedMin:               3 * time.Millisecond,
			ExpectedMax:               7 * time.Millisecond,
		},
		{
			Inputs:                    []int{2, 3},
			ExpectedMean:              4.428571,
			ExpectedVariance:          17.64835,
			ExpectedStandardDeviation: 4.201,
			ExpectedMin:               1,
			ExpectedMax:               11,
		},
	}
	for i, test := range mergeCases {
		t.Run(fmt.Sprintf("%d:%+v", i, test.Inputs), func(t *testing.T) {
			t.Parallel()
			allInOne := data.Stats{}
			var merged *data.Stats
			for _, tc := range test.Inputs {
				x := testCases[tc]
				allInOne.AddPoints(x.Values)
				toMerge := data.Stats{}
				toMerge.AddPoints(x.Values)
				merged = merged.Merge(&toMerge)
			}
			th.AssertFloatEqual(t, test.ExpectedMean, allInOne.Mean, 7, "allInOne Mean")
			th.AssertFloatEqual(t, test.ExpectedMean, merged.Mean, 7, "merged Mean")
			th.AssertFloatEqual(t, test.ExpectedVariance, allInOne.Variance, 7, "allInOne Variance")
			th.AssertFloatEqual(t, test.ExpectedVariance, merged.Variance, 7, "merged Variance")
			th.AssertFloatEqual(t, test.ExpectedStandardDeviation, allInOne.StandardDeviation, 5, "allInOne StandardDeviation")
			th.AssertFloatEqual(t, test.ExpectedStandardDeviation, merged.StandardDeviation, 5, "merged StandardDeviation")
			assert.Check(t, is.Equal(test.ExpectedMax, allInOne.Max), "allInOne Max")
			assert.Check(t, is.Equal(test.ExpectedMax, merged.Max), "merged Max")
			assert.Check(t, is.Equal(test.ExpectedMin, allInOne.Min), "allInOne Min")
			assert.Check(t, is.Equal(test.ExpectedMin, merged.Min), "merged Min")
		})
	}
}

func assertStatsEqual(t *testing.T, expected data.Stats, actual data.Stats, sigFigs int, msgAndArgs ...interface{}) {
	t.Helper()
	th.AssertFloatEqual(t, expected.Mean, actual.Mean, sigFigs, msgAndArgs...)
	th.AssertFloatEqual(t, expected.Variance, actual.Variance, sigFigs, msgAndArgs...)
	th.AssertFloatEqual(t, expected.StandardDeviation, actual.StandardDeviation, sigFigs, msgAndArgs...)
	if expected.GoodCount != 0 {
		assert.Check(t, is.Equal(expected.GoodCount, actual.GoodCount), msgAndArgs...)
	}
}

func assertTimeSpanEqual(t *testing.T, expected data.TimeSpan, actual data.TimeSpan, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Check(t, is.DeepEqual(expected.Begin, actual.Begin), msgAndArgs...)
	assert.Check(t, is.DeepEqual(expected.End, actual.End), msgAndArgs...)
	assert.Check(t, is.Equal(expected.Duration, actual.Duration), msgAndArgs...)
}

type BlockTest struct {
	ExpectedBlocks []data.Block
	CheckRaw       bool
}

type DataTestCase struct {
	Values             []ping.PingResults
	BlockSize          int
	ExpectedGraphSpan  data.TimeSpan
	ExpectedGraphStats data.Stats
	ExpectedPacketLoss float64
	ExpectedTotalCount int
	ExpectedRuns       data.Runs
	BlockTest          *BlockTest
}

// A fixed time stamp to make all testing relative too
var origin = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

func TestData(t *testing.T) {
	t.Parallel()

	testCases := []DataTestCase{
		{
			Values: sameIP([]ping.PingDataPoint{
				{Duration: 5 * time.Millisecond, Timestamp: origin},
				{Duration: 6 * time.Millisecond, Timestamp: origin.Add(time.Minute)},
				{Duration: 5 * time.Millisecond, Timestamp: origin.Add(2 * time.Minute)},
				{Duration: 7 * time.Millisecond, Timestamp: origin.Add(3 * time.Minute)},
				{Duration: 3 * time.Millisecond, Timestamp: origin.Add(4 * time.Minute)},
			}),
			ExpectedGraphSpan: data.TimeSpan{
				Begin:    origin,
				End:      origin.Add(4 * time.Minute),
				Duration: 4 * time.Minute,
			},
			ExpectedGraphStats: data.Stats{
				Mean:              asFloat64(5.2, time.Millisecond),
				GoodCount:         5,
				Variance:          2_200_000_000_000, // Variance isn't squared so it gets real big
				StandardDeviation: asFloat64(1.4832397, time.Millisecond),
			},
			ExpectedRuns: data.Runs{GoodPackets: &data.Run{
				LongestIndexEnd: 4,
				Longest:         5,
				Current:         5,
			}},
			ExpectedTotalCount: 5,
		},
		{
			Values: slices.Concat(
				specificIP([]ping.PingDataPoint{
					{Duration: 5 * time.Nanosecond, Timestamp: origin},
					{Duration: 6 * time.Nanosecond, Timestamp: origin.Add(time.Nanosecond)},
					{Duration: 5 * time.Nanosecond, Timestamp: origin.Add(2 * time.Nanosecond)},
					{Duration: 7 * time.Nanosecond, Timestamp: origin.Add(3 * time.Nanosecond)},
					{Duration: 3 * time.Nanosecond, Timestamp: origin.Add(4 * time.Nanosecond)},
				}, net.IPv4allrouter),
				specificIP([]ping.PingDataPoint{
					{Duration: 5 * time.Nanosecond, Timestamp: origin.Add(5 * time.Nanosecond)},
					{Duration: 6 * time.Nanosecond, Timestamp: origin.Add(6 * time.Nanosecond)},
					{Duration: 5 * time.Nanosecond, Timestamp: origin.Add(7 * time.Nanosecond)},
					{Duration: 7 * time.Nanosecond, Timestamp: origin.Add(8 * time.Nanosecond)},
					{Duration: 3 * time.Nanosecond, Timestamp: origin.Add(9 * time.Nanosecond)},
				}, net.IPv4bcast),
			),
			BlockSize: 5,
			ExpectedGraphSpan: data.TimeSpan{
				Begin:    origin,
				End:      origin.Add(9 * time.Nanosecond),
				Duration: 9 * time.Nanosecond,
			},
			ExpectedGraphStats: data.Stats{
				Mean:              asFloat64(5.2, time.Nanosecond),
				GoodCount:         10,
				Variance:          1.955,
				StandardDeviation: asFloat64(1.3984118, time.Nanosecond),
			},
			BlockTest: &BlockTest{
				// This test has two IPs so will need two blocks. The actual blocks are designed to be
				// identical except IP.
				ExpectedBlocks: []data.Block{{
					Header: &data.Header{
						Stats: &data.Stats{
							Mean:              asFloat64(5.2, time.Nanosecond),
							GoodCount:         5,
							Variance:          2.2,
							StandardDeviation: asFloat64(1.4832397, time.Nanosecond),
						},
						TimeSpan: &data.TimeSpan{Begin: origin, End: origin.Add(4 * time.Nanosecond), Duration: 4 * time.Nanosecond},
					},
				}, {
					Header: &data.Header{
						Stats: &data.Stats{
							Mean:              asFloat64(5.2, time.Nanosecond),
							GoodCount:         5,
							Variance:          2.2,
							StandardDeviation: asFloat64(1.4832397, time.Nanosecond),
						},
						TimeSpan: &data.TimeSpan{
							Begin:    origin.Add(5 * time.Nanosecond),
							End:      origin.Add(9 * time.Nanosecond),
							Duration: 4 * time.Nanosecond,
						},
					},
				}},
				CheckRaw: false,
			},
			ExpectedRuns: data.Runs{GoodPackets: &data.Run{
				LongestIndexEnd: 9,
				Longest:         10,
				Current:         10,
			}},
			ExpectedTotalCount: 10,
		},
		{
			Values: sameIP([]ping.PingDataPoint{
				{Duration: 15 * time.Millisecond, Timestamp: origin},
				{Duration: 16 * time.Millisecond, Timestamp: origin.Add(10 * time.Minute)},
				{DropReason: ping.TestDrop, Timestamp: origin.Add(20 * time.Minute)},
				{Duration: 17 * time.Millisecond, Timestamp: origin.Add(30 * time.Minute)},
				{Duration: 13 * time.Millisecond, Timestamp: origin.Add(40 * time.Minute)},
			}),
			ExpectedGraphSpan: data.TimeSpan{
				Begin:    origin,
				End:      origin.Add(40 * time.Minute),
				Duration: 40 * time.Minute,
			},
			ExpectedGraphStats: data.Stats{
				Mean:              asFloat64(15.25, time.Millisecond),
				Variance:          2_916_666_700_000,
				StandardDeviation: asFloat64(1.7078251, time.Millisecond),
			},
			ExpectedPacketLoss: 1.0 / 5.0,
			ExpectedTotalCount: 5,
			BlockTest: &BlockTest{
				ExpectedBlocks: []data.Block{{
					Header: &data.Header{
						Stats: &data.Stats{
							Mean:              asFloat64(15.25, time.Millisecond),
							Variance:          2_916_666_700_000,
							StandardDeviation: asFloat64(1.7078251, time.Millisecond),
						},
						TimeSpan: &data.TimeSpan{Begin: origin, End: origin.Add(40 * time.Minute), Duration: 40 * time.Minute},
					},
				}},
			},
			ExpectedRuns: data.Runs{GoodPackets: &data.Run{
				LongestIndexEnd: 1,
				Longest:         2,
				Current:         2,
			}, DroppedPackets: &data.Run{
				LongestIndexEnd: 2,
				Longest:         1,
				Current:         0,
			}},
		},
	}

	for i, test := range testCases {
		sliceAsStr := strings.Join(sliceutils.Map(test.Values, func(p ping.PingResults) string {
			return p.String()
		}), ",")
		t.Run(fmt.Sprintf("%d:[%s]", i, sliceAsStr), func(t *testing.T) {
			t.Parallel()
			graphData := data.NewData("www.google.com")
			for _, v := range test.Values {
				graphData.AddPoint(v)
			}
			assertStatsEqual(t, test.ExpectedGraphStats, *graphData.Header.Stats, 3, "global graph header")
			assertTimeSpanEqual(t, test.ExpectedGraphSpan, *graphData.Header.TimeSpan, "global graph header")
			th.AssertFloatEqual(t, test.ExpectedPacketLoss, graphData.Header.Stats.PacketLoss(), 5, "global packet loss percent")
			if test.BlockTest != nil {
				blockVerify(t, graphData, test)
			}
			assertRunsEqual(t, test.ExpectedRuns, *graphData.Runs)
		})
	}
}

func assertRunsEqual(t *testing.T, expect, actual data.Runs) {
	t.Helper()
	if expect.GoodPackets != nil {
		assert.DeepEqual(t, expect.GoodPackets, actual.GoodPackets)
	}
	if expect.DroppedPackets != nil {
		assert.DeepEqual(t, expect.DroppedPackets, actual.DroppedPackets)
	}
}

func specificIP(input []ping.PingDataPoint, IP net.IP) []ping.PingResults {
	return sliceutils.Map(input, func(in ping.PingDataPoint) ping.PingResults {
		return ping.PingResults{
			Data: in,
			IP:   IP,
		}
	})
}

func sameIP(input []ping.PingDataPoint) []ping.PingResults {
	return specificIP(input, net.IPv4allrouter)
}

// NOTE doesn't iterate the data in duration order.
func blockVerify(t *testing.T, graphData *data.Data, test DataTestCase) {
	t.Helper()
	assert.Assert(t, is.Len(graphData.Blocks, len(test.BlockTest.ExpectedBlocks)))
	for i, block := range graphData.Blocks {
		expectedBlock := test.BlockTest.ExpectedBlocks[i]
		assertStatsEqual(t, *expectedBlock.Header.Stats, *block.Header.Stats, 4)
		assertTimeSpanEqual(t, *expectedBlock.Header.TimeSpan, *block.Header.TimeSpan, 4)
		if test.BlockTest.CheckRaw {
			assert.Assert(t, is.Len(block.Raw, len(expectedBlock.Raw)), "block %d was unexpected len", i)
			for rawIndex, datum := range block.Raw {
				assert.Check(t, is.DeepEqual(expectedBlock.Raw[rawIndex], datum), "raw inside block %d at index %d", i, rawIndex)
			}
		}
	}
}

func asFloat64(scalar float64, t time.Duration) float64 {
	return scalar * float64(t)
}

type DataOrderingTestCase struct {
	Name          string
	Values        []ping.PingResults
	ExpectedOrder []ping.PingDataPoint
	ExpectFailure bool
}

func TestDataOrdering(t *testing.T) {
	t.Parallel()

	t1min := ping.PingDataPoint{Duration: 15 * time.Millisecond, Timestamp: origin.Add(1 * time.Minute)}
	t2min := ping.PingDataPoint{Duration: 25 * time.Millisecond, Timestamp: origin.Add(2 * time.Minute)}
	t3min := ping.PingDataPoint{Duration: 35 * time.Millisecond, Timestamp: origin.Add(3 * time.Minute)}
	t4min := ping.PingDataPoint{Duration: 45 * time.Millisecond, Timestamp: origin.Add(4 * time.Minute)}
	t5min := ping.PingDataPoint{Duration: 55 * time.Millisecond, Timestamp: origin.Add(5 * time.Minute)}
	t6min := ping.PingDataPoint{Duration: 65 * time.Millisecond, Timestamp: origin.Add(6 * time.Minute)}
	t7min := ping.PingDataPoint{Duration: 75 * time.Millisecond, Timestamp: origin.Add(7 * time.Minute)}
	expectedOrder := []ping.PingDataPoint{t1min, t2min, t3min, t4min, t5min, t6min, t7min}

	sorted := []ping.PingResults{
		{Data: t1min, IP: []byte{}},
		{Data: t2min, IP: []byte{}},
		{Data: t3min, IP: []byte{}},
		{Data: t4min, IP: []byte{}},
		{Data: t5min, IP: []byte{}},
		{Data: t6min, IP: []byte{}},
		{Data: t7min, IP: []byte{}},
	}

	inc := 0

	testCases := []DataOrderingTestCase{
		{
			Name:          "ID",
			Values:        sorted,
			ExpectedOrder: expectedOrder,
		},
		{
			Name:          "Shuffled",
			Values:        sliceutils.Shuffle(sorted),
			ExpectedOrder: expectedOrder,
			ExpectFailure: true, // Data should not be sorted for us. It merely should preserve insertion order for any range of IPs.
		},
		{
			Name: "Incrementing IPs",
			Values: sliceutils.Map(sorted, func(p ping.PingResults) ping.PingResults {
				inc++
				return ping.PingResults{
					Data:        p.Data,
					IP:          net.IPv4(byte(inc), 0, 0, 0),
					InternalErr: nil,
				}
			}),
			ExpectedOrder: expectedOrder,
		},
		{
			Name: "Random IPs",
			Values: sliceutils.Map(sorted, func(p ping.PingResults) ping.PingResults {
				// Not a security use of randomness
				x := rand.Int32() //nolint:gosec
				return ping.PingResults{
					Data:        p.Data,
					IP:          net.IPv4(byte(x), byte(x>>8), byte(x>>16), byte(x>>24)),
					InternalErr: nil,
				}
			}),
			ExpectedOrder: expectedOrder,
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			graphData := data.NewData("www.google.com")
			for _, v := range test.Values {
				graphData.AddPoint(v)
			}
			if test.ExpectFailure {
				collected := make([]ping.PingDataPoint, graphData.TotalCount)
				for i := range graphData.TotalCount {
					collected[i] = graphData.Get(i)
				}
				assert.Check(t, !is.DeepEqual(test.ExpectedOrder, collected)().Success())
			} else {
				for i := range graphData.TotalCount {
					cur := graphData.Get(i)
					assert.Check(t, is.DeepEqual(test.ExpectedOrder[i], cur))
				}
			}
		})
	}
}
