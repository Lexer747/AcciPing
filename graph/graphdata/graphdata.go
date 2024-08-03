// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graphdata

import (
	"io"
	"sync"

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
		new := g.spans[g.spanIndex].AddPoint(d.Get(i))
		if new {
			panic("todo")
		}
	}
	return g
}

func (gd *GraphData) AddPoint(p ping.PingResults) {
	gd.Lock()
	defer gd.Unlock()
	gd.data.AddPoint(p)
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

	gap := p.Timestamp.Sub(si.LastPoint.Timestamp)
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
