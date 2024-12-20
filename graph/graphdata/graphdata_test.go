// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graphdata_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/graphdata"
	"github.com/Lexer747/AcciPing/graph/th"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/sliceutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTimeSpan(start, end time.Time) *data.TimeSpan {
	return &data.TimeSpan{
		Begin:    start,
		End:      end,
		Duration: end.Sub(start),
	}
}

func TestGraphData_TimeSpan_files(t *testing.T) {
	t.Parallel()
	t.Run("Many spans over gaps",
		TimeSpanFileTest{
			File:              "../data/testdata/input/TimeSpanTestCase1.pings",
			ExpectedSpanCount: 6,
			ExpectedSpans: []*data.TimeSpan{
				newTimeSpan(
					time.Date(2024, time.October, 31, 10, 42, 8, 531000000, time.Local),
					time.Date(2024, time.October, 31, 10, 42, 58, 531000000, time.Local),
				),
				newTimeSpan(
					time.Date(2024, time.November, 1, 13, 52, 55, 304000000, time.Local),
					time.Date(2024, time.November, 1, 13, 53, 1, 305000000, time.Local),
				),
				newTimeSpan(
					time.Date(2024, time.November, 1, 13, 53, 58, 355000000, time.Local),
					time.Date(2024, time.November, 1, 13, 54, 34, 356000000, time.Local),
				),
				newTimeSpan(
					time.Date(2024, time.November, 8, 11, 29, 5, 732000000, time.Local),
					time.Date(2024, time.November, 8, 11, 29, 14, 733000000, time.Local),
				),
				newTimeSpan(
					time.Date(2024, time.November, 8, 11, 29, 39, 877000000, time.Local),
					time.Date(2024, time.November, 8, 11, 29, 41, 878000000, time.Local),
				),
				newTimeSpan(
					time.Date(2024, time.November, 8, 11, 45, 37, 177000000, time.Local),
					time.Date(2024, time.November, 8, 11, 45, 41, 178000000, time.Local),
				),
			},
		}.Run,
	)
	t.Run("medium-395-02-08-2024.pings",
		TimeSpanFileTest{
			File:              "../data/testdata/input/medium-395-02-08-2024.pings",
			ExpectedSpanCount: 1,
			ExpectedSpans: []*data.TimeSpan{
				newTimeSpan(
					time.Date(2024, time.August, 2, 20, 40, 41, 175000000, time.Local),
					time.Date(2024, time.August, 2, 20, 47, 15, 175000000, time.Local),
				),
			},
		}.Run,
	)
	t.Run("long-gap.pings",
		TimeSpanFileTest{
			File:              "../data/testdata/input/long-gap.pings",
			ExpectedSpanCount: 5,
			ExpectedSpans: []*data.TimeSpan{
				newTimeSpan(
					time.Date(2024, time.August, 3, 0, 41, 6, 657000000, time.Local),
					time.Date(2024, time.August, 3, 0, 41, 37, 657000000, time.Local),
				),
				newTimeSpan(
					time.Date(2024, time.August, 3, 0, 55, 35, 613000000, time.Local),
					time.Date(2024, time.August, 3, 0, 55, 50, 614000000, time.Local),
				),
				newTimeSpan(
					time.Date(2024, time.August, 3, 1, 2, 10, 106000000, time.Local),
					time.Date(2024, time.August, 3, 1, 2, 28, 106000000, time.Local),
				),
				newTimeSpan(
					time.Date(2024, time.August, 3, 10, 52, 20, 596000000, time.Local),
					time.Date(2024, time.August, 3, 10, 55, 6, 597000000, time.Local),
				),
				// Several day gap
				newTimeSpan(
					time.Date(2024, time.August, 19, 18, 51, 55, 743000000, time.Local),
					time.Date(2024, time.August, 19, 18, 52, 25, 744000000, time.Local),
				),
			},
		}.Run,
	)
}

var origin = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

func Test_Basic(t *testing.T) {
	t.Parallel()
	require.NotPanics(t, func() {
		_ = graphdata.NewGraphData(data.NewData("foo.bar"))
	})
	t.Run("Basic",
		BasicTimeSpanTest{
			Points: []ping.PingDataPoint{
				// Span 1
				{Timestamp: origin.Add(time.Second * 1)},
				{Timestamp: origin.Add(time.Second * 2)},
				{Timestamp: origin.Add(time.Second * 3)},
				// Span 2
				{Timestamp: origin.Add(time.Second * 200)},
			},
			ExpectedSpanCount: 2,
		}.Run,
	)
}

func Test_Complex(t *testing.T) {
	t.Parallel()
	t.Run("Complex",
		TimeSpanTest{
			Spans: [][]ping.PingDataPoint{
				{
					{Timestamp: origin.Add(time.Millisecond*1 + 1)},
					{Timestamp: origin.Add(time.Millisecond*2 + 2)},
					{Timestamp: origin.Add(time.Millisecond*3 + 5)},
				},
				{
					{Timestamp: origin.Add(time.Second*200 + 1)},
					{Timestamp: origin.Add(time.Second*201 + 2)},
				},
				{
					{Timestamp: origin.Add(time.Minute*70000 + 1)},
					{Timestamp: origin.Add(time.Minute*70002 + 2)},
				},
			},
		}.Run,
	)
}

type BasicTimeSpanTest struct {
	Points            []ping.PingDataPoint
	ExpectedSpanCount int
}

func (test BasicTimeSpanTest) Run(t *testing.T) {
	t.Parallel()
	gd := graphdata.NewGraphData(data.NewData("foo.bar"))
	for _, point := range test.Points {
		gd.AddPoint(ping.PingResults{Data: point})
	}
	require.Len(t, gd.LockFreeSpanInfos(), test.ExpectedSpanCount)

	assertEveryPointHasSpan(t, gd, gd.LockFreeSpanInfos())
}

func assertEveryPointHasSpan(t *testing.T, gd *graphdata.GraphData, actual []*graphdata.SpanInfo) {
	t.Helper()
	iter := gd.LockFreeIter()
	for i := range iter.Total {
		p, _ := iter.Get(i)
		timestamp := p.Timestamp
		sliceutils.OneOf(actual, func(span *graphdata.SpanInfo) bool { return span.TimeSpan.Contains(timestamp) })
		assert.Truef(
			t,
			sliceutils.OneOf(actual, func(span *graphdata.SpanInfo) bool { return span.TimeSpan.Contains(timestamp) }),
			"Missing %q from spans: %+v",
			timestamp.Format("02 Jan 2006 15:04:05.000000"),
			strings.Join(sliceutils.Map(actual,
				func(si *graphdata.SpanInfo) string {
					return si.TimeSpan.String()
				}), ", "),
		)
	}
}

type TimeSpanTest struct {
	Spans [][]ping.PingDataPoint
}

func (test TimeSpanTest) Run(t *testing.T) {
	t.Parallel()
	gd := graphdata.NewGraphData(data.NewData("foo.bar"))
	expectedSpans := make([]*graphdata.SpanInfo, 0, len(test.Spans))
	index := int64(0)
	for i, span := range test.Spans {
		expectedSpans = append(expectedSpans, graphdata.NewSpanInfo())
		for _, point := range span {
			graphdata.Add(expectedSpans[i], point, index)
			gd.AddPoint(ping.PingResults{Data: point})
			index++
		}
		actual := gd.LockFreeSpanInfos()
		assert.Equal(t, graphdata.Spans(expectedSpans), actual, "index %d | %+v", i, span)
	}

	actual := gd.LockFreeSpanInfos()
	require.Len(t, actual, len(expectedSpans))
	for i := range actual {
		assert.Equal(t, expectedSpans[i], actual[i], "index %d", i)
	}
	assertEveryPointHasSpan(t, gd, actual)
}

type TimeSpanFileTest struct {
	File              string
	ExpectedSpanCount int
	ExpectedSpans     []*data.TimeSpan
}

func (test TimeSpanFileTest) Run(t *testing.T) {
	t.Helper()
	t.Parallel()
	d := th.GetFromFile(t, test.File)
	gd := graphdata.NewGraphData(d)
	actualSpans := gd.LockFreeSpanInfos()
	require.Len(t, actualSpans, test.ExpectedSpanCount)
	if len(test.ExpectedSpans) != 0 {
		actual := sliceutils.Map(actualSpans, func(si *graphdata.SpanInfo) *data.TimeSpan { return si.TimeSpan })
		for i, span := range test.ExpectedSpans {
			require.Equal(t, span, actual[i], "index %d", i)
		}
	}
	assertEveryPointHasSpan(t, gd, gd.LockFreeSpanInfos())
}
