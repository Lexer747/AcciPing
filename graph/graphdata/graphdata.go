// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graphdata

import (
	"io"
	"math"
	"sync"
	"time"

	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/ping"
)

type GraphData struct {
	data      *data.Data
	spans     []*SpanInfo
	spanIndex int
	m         *sync.Mutex
}

func NewGraphData(d *data.Data) *GraphData {
	g := &GraphData{
		data: d,
		spans: []*SpanInfo{{
			TimeStats: &data.Stats{},
			TimeSpan:  &data.TimeSpan{},
			LastPoint: ping.PingDataPoint{},
		}},
		m: &sync.Mutex{},
	}
	for i := range d.TotalCount {
		g.addPointToSpans(d.Get(i))
	}
	return g
}

func (gd *GraphData) addPointToSpans(p ping.PingDataPoint) {
	differentSpan := gd.spans[gd.spanIndex].AddPoint(p)
	if differentSpan {
		gd.spans = append(gd.spans, &SpanInfo{
			TimeStats: &data.Stats{},
			TimeSpan:  &data.TimeSpan{},
			LastPoint: ping.PingDataPoint{},
		})
		gd.spanIndex++
		gd.spans[gd.spanIndex].AddPoint(p)
	}
}

func (gd *GraphData) AddPoint(p ping.PingResults) {
	gd.Lock()
	defer gd.Unlock()
	gd.data.AddPoint(p)
	gd.addPointToSpans(p.Data)
}

func (gd *GraphData) TotalCount() int64 {
	gd.Lock()
	defer gd.Unlock()
	return gd.data.TotalCount
}

func (gd *GraphData) String() string {
	gd.Lock()
	defer gd.Unlock()
	return gd.data.String()
}

func (gd *GraphData) AsCompact(w io.Writer) error {
	gd.m.Lock()
	defer gd.m.Unlock()
	return gd.data.AsCompact(w)
}

// NOTE: GraphData does not have a [data.FromCompact] implementation because it is meant to be less strict layer on-top
// the [data] package, which contains types and values which are only useful in drawing a graph but are not meaningful
// to actual data captured by a series of pings.

// We expose public locks because we provide a handful of APIs which are meant to be used while the lock is already
// held. In particular the drawing code is expected to do many large reads and unlock early while it paints this result.

func (gd *GraphData) Lock() {
	gd.m.Lock()
}
func (gd *GraphData) Unlock() {
	gd.m.Unlock()
}

func (gd *GraphData) LockFreeTotalCount() int64      { return gd.data.TotalCount }
func (gd *GraphData) LockFreeHeader() *data.Header   { return gd.data.Header }
func (gd *GraphData) LockFreeURL() string            { return gd.data.URL }
func (gd *GraphData) LockFreeRuns() *data.Runs       { return gd.data.Runs }
func (gd *GraphData) LockFreeTimeSpans() []*SpanInfo { return gd.spans }

type SpanInfo struct {
	TimeStats *data.Stats
	TimeSpan  *data.TimeSpan
	LastPoint ping.PingDataPoint
	Count     int
}

func (si *SpanInfo) AddPoint(p ping.PingDataPoint) bool {
	add := func() {
		si.TimeSpan.AddTimestamp(p.Timestamp)
		si.Count++
		si.LastPoint = p
	}
	if si.Count == 0 {
		add()
		return false
	} else if si.Count == 1 {
		gap := p.Timestamp.Sub(si.LastPoint.Timestamp)
		si.TimeStats.AddPoint(gap)
		add()
		return false
	}
	// Problem statement:
	//
	// We want to determine if a given new point is essentially from a new sampling domain. The main use case
	// is that we capture some packets on day 1. Then capture some more packets on day 2. When something like
	// this happens we want to essentially plot two related graphs on a single axis.
	//
	// In general anytime we encounter some large gap between samples we want to split the graph by that gap
	// to maximizes the information we are displaying (especially given the small amount of space we are
	// typically working with). However one obvious consequence which should be accounted for is that dropped
	// packets already live in their own sampling domain. As the timeout will not be the same as the ping
	// ticker frequency. Furthermore it's a requirement that two different frequency samples can be appended
	// together should be treated equally, e.g. a capture running at 5 pings/minute on day 1 would still look
	// good on the same graph when day 2 is 100 pings/minute.
	//
	// Solution:
	//
	// We record the difference in timestamps into a [data.Stats] struct which will work out the statistical
	// nature of the current sampling, if we detect the next point is some outlier then we consider a new
	// span. Where outlier is a flexible definition to just mean whatever is the best heuristic for pretty
	// graphs.
	gap := p.Timestamp.Sub(si.LastPoint.Timestamp)
	allowedStandardDeviations := 3.0
	if p.Dropped() {
		// At low ping rate this might be too high, given a reasonable 1 ping/minute, a 1s timeout is
		// completely reasonable in which case this should just stay as 3 stds away. Scale this somehow?
		allowedStandardDeviations = 6.0
	}
	if time.Duration(math.Round(si.TimeStats.StandardDeviation*allowedStandardDeviations)) < gap {
		// This gap is officially too big, don't add this point.
		// TODO account for very early small stats with low confidence
		return true
	}
	// This gap is small enough add it to this span
	si.TimeStats.AddPoint(gap)
	add()
	return false
}

type Iter struct {
	Total int64
	d     *data.Data
}

func (gd *GraphData) LockFreeIter() *Iter {
	return &Iter{
		Total: gd.LockFreeTotalCount(),
		d:     gd.data,
	}
}

func (i *Iter) Get(index int64) ping.PingDataPoint {
	return i.d.Get(index)
}

func (i *Iter) IsLast(index int64) bool {
	return i.d.IsLast(index)
}
