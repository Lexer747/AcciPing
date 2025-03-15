// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package data

import (
	"fmt"
	"math"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/Lexer747/acci-ping/ping"
	"github.com/Lexer747/acci-ping/utils/numeric"
	"github.com/Lexer747/acci-ping/utils/sliceutils"
)

type Data struct {
	URL         string
	Header      *Header
	Network     *Network
	InsertOrder []DataIndexes
	Blocks      []*Block
	TotalCount  int64
	Runs        *Runs
	PingsMeta   version
}

type DataIndexes struct {
	BlockIndex, RawIndex int
}

func NewData(URL string) *Data {
	return newVersionedData(URL, currentDataVersion)
}

func newVersionedData(URL string, v version) *Data {
	d := &Data{
		URL:         URL,
		Header:      &Header{Stats: &Stats{}, TimeSpan: &TimeSpan{Begin: time.UnixMilli(0), End: time.UnixMilli(0), Duration: 0}},
		Network:     &Network{IPs: []net.IP{}, BlockIndexes: []int{}, curBlockIndex: 0},
		InsertOrder: []DataIndexes{},
		Blocks:      []*Block{},
		TotalCount:  0,
		Runs:        &Runs{GoodPackets: &Run{}, DroppedPackets: &Run{}},
		PingsMeta:   v,
	}
	return d
}

func (d *Data) AddPoint(p ping.PingResults) {
	blockIndex := d.Network.AddPoint(p.IP)
	if blockIndex >= len(d.Blocks) {
		d.addBlock()
	}
	curBlock := d.getBlock(blockIndex)
	rawIndex := curBlock.AddPoint(p.Data)
	d.Header.AddPoint(p.Data)
	d.Runs.AddPoint(d.TotalCount, p.Data)
	d.TotalCount++
	d.InsertOrder = append(d.InsertOrder, DataIndexes{
		BlockIndex: blockIndex,
		RawIndex:   rawIndex,
	})
}

func (d *Data) Get(index int64) ping.PingDataPoint {
	this := d.InsertOrder[index]
	return d.Blocks[this.BlockIndex].Raw[this.RawIndex]
}
func (d *Data) GetFull(index int64) ping.PingResults {
	this := d.InsertOrder[index]
	dataPoint := d.Blocks[this.BlockIndex].Raw[this.RawIndex]
	i := slices.Index(d.Network.BlockIndexes, this.BlockIndex)
	ip := d.Network.IPs[i]
	return ping.PingResults{
		Data: dataPoint,
		IP:   ip,
	}
}
func (d *Data) End(index int64) bool {
	return int(index) == len(d.InsertOrder)
}
func (d *Data) IsLast(index int64) bool {
	return d.End(index - 1)
}

func (d *Data) addBlock() {
	d.Blocks = append(d.Blocks, &Block{
		Header: &Header{Stats: &Stats{}, TimeSpan: &TimeSpan{}},
		Raw:    make([]ping.PingDataPoint, 0, 1024),
	})
}

func (d *Data) getBlock(blockIndex int) *Block {
	return d.Blocks[blockIndex]
}

func (d *Data) String() string {
	return fmt.Sprintf("%s: PingsMeta#%d [%s] | %s | %s", d.URL, d.PingsMeta, d.Network.String(), d.Header.String(), d.Runs.String())
}

func (d *Data) In(tz *time.Location) *Data {
	ret := newVersionedData(d.URL, d.PingsMeta)
	for i := range d.TotalCount {
		p := d.GetFull(i)
		p.Data.Timestamp = p.Data.Timestamp.In(tz)
		ret.AddPoint(p)
	}
	return ret
}

// TimeSpan is the time properties of a given thing
type TimeSpan struct {
	Begin    time.Time
	End      time.Time
	Duration time.Duration
}

func (ts *TimeSpan) Equal(other *TimeSpan) bool {
	return ts.Duration == other.Duration && ts.Begin.Equal(other.Begin) && ts.End.Equal(other.End)
}

// Merge takes two [TimeSpan] pointers and returns the new span containing both inputs, this may be
// the same as one of the inputs if a span completely overlaps another.
func (ts *TimeSpan) Merge(other *TimeSpan) *TimeSpan {
	ret := &TimeSpan{Begin: ts.Begin, End: ts.End}
	if other.Begin.Before(ret.Begin) {
		ret.Begin = other.Begin
	}
	if other.End.After(ret.End) {
		ret.End = other.End
	}
	ret.Duration = ret.End.Sub(ret.Begin)
	return ret
}

func (ts *TimeSpan) Contains(t time.Time) bool {
	largeEnough := ts.Begin.Before(t) || ts.Begin.Equal(t)
	smallEnough := ts.End.After(t) || ts.End.Equal(t)
	return (smallEnough) && (largeEnough)
}

// AddTimestamp adds the timestamp to the span, only works when initialized with a non-zero time
func (ts *TimeSpan) AddTimestamp(t time.Time) {
	if ts.Begin.After(t) {
		ts.Begin = t
	}
	if ts.End.Before(t) {
		ts.End = t
	}
	ts.Duration = ts.End.Sub(ts.Begin)
}

// Header describes the statistical properties of a group of objects.
type Header struct {
	Stats    *Stats
	TimeSpan *TimeSpan
}

func (h *Header) AddPoint(p ping.PingDataPoint) {
	if h.Stats.GoodCount == 0 {
		h.TimeSpan = &TimeSpan{Begin: p.Timestamp, End: p.Timestamp}
	} else {
		h.TimeSpan.AddTimestamp(p.Timestamp)
	}
	if p.Dropped() {
		h.Stats.AddDroppedPacket()
	} else {
		h.Stats.AddPoint(p.Duration)
	}
}

func (h *Header) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s | %s", h.TimeSpan.String(), h.Stats.String())
	return b.String()
}

type Network struct {
	IPs           []net.IP
	BlockIndexes  []int
	curBlockIndex int
}

// AddPoint will insert the IP into the network and return the block index for this IP, noting that it will
// return an out of range index if this IP has not been seen before.
func (n *Network) AddPoint(ip net.IP) int {
	ip = ip.To16() // Ensure all saved IPs are in IPv6 format
	if ip == nil {
		ip = net.IPv6zero // DNS failure, etc
	}
	i, found := slices.BinarySearchFunc(n.IPs, ip, ipOrdering)
	if found {
		return n.BlockIndexes[i]
	}
	cur := n.curBlockIndex
	n.IPs = slices.Insert(n.IPs, i, ip)
	n.BlockIndexes = slices.Insert(n.BlockIndexes, i, cur)
	n.curBlockIndex++
	return cur
}

func (n *Network) String() string {
	return sliceutils.Join(n.IPs, ",")
}

// Run will store the largest and current run of a given item.
type Run struct {
	LongestIndexEnd int64
	Longest         uint64
	Current         uint64
}

func (r *Run) Inc(index int64) {
	r.Current++
	if r.Current > r.Longest {
		r.Longest = r.Current
		r.LongestIndexEnd = index
	}
}

func (r *Run) Reset() {
	r.Current = 0
}

// Runs stores the longest consecutive sequence of good and dropped packets
type Runs struct {
	GoodPackets    *Run
	DroppedPackets *Run
}

func (r *Runs) AddPoint(index int64, p ping.PingDataPoint) {
	if p.Dropped() {
		r.GoodPackets.Reset()
		r.DroppedPackets.Inc(index)
	} else {
		r.GoodPackets.Inc(index)
		r.DroppedPackets.Reset()
	}
}
func (r *Runs) String() string {
	switch {
	case r.GoodPackets.Longest == 0 && r.DroppedPackets.Longest == 0:
		return ""
	case r.GoodPackets.Longest == 0:
		return fmt.Sprintf("Longest Drop Streak %d", r.DroppedPackets.Longest)
	case r.DroppedPackets.Longest == 0:
		return fmt.Sprintf("Longest Streak %d", r.GoodPackets.Longest)
	default:
		return fmt.Sprintf("Longest Streak %d | Longest Drop Streak %d", r.GoodPackets.Longest, r.DroppedPackets.Longest)
	}
}

type Block struct {
	Header *Header
	Raw    []ping.PingDataPoint
}

// AddPoint will insert a dataPoint into this block, returning the index into the block in which this was inserted.
func (b *Block) AddPoint(p ping.PingDataPoint) int {
	b.Raw = append(b.Raw, p)
	b.Header.AddPoint(p)
	return len(b.Raw) - 1
}

type Stats struct {
	Min, Max          time.Duration
	Mean              float64
	GoodCount         uint64
	Variance          float64
	StandardDeviation float64
	PacketsDropped    uint64
	sumOfSquares      float64
}

// Merge combines two [Stats] pointers into a new [Stats] pointer containing all the data from both
// inputs. This means the total count will be the sum of the two input counts. If either is nil then
// a new [Stats] is not created and the non-nil pointer is returned. If both are nil then this
// panics.
func (s *Stats) Merge(other *Stats) *Stats {
	if s == nil {
		return other
	}
	if other == nil {
		return s
	}
	ret := &Stats{}
	ret.Min = min(s.Min, other.Min)
	ret.Max = max(s.Max, other.Max)
	ret.GoodCount = s.GoodCount + other.GoodCount
	ret.Mean = (s.Mean*float64(s.GoodCount) + other.Mean*float64(other.GoodCount)) / float64(ret.GoodCount)
	ret.PacketsDropped = s.PacketsDropped + other.PacketsDropped
	// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Weighted_incremental_algorithm
	// Illuminating question:
	// https://math.stackexchange.com/questions/2867951/formula-of-combined-variance-of-two-data-sets-yields-wrong-output
	ret.sumOfSquares = s.sumOfSquares + other.sumOfSquares + // First add the sums of squares to keep the original variance
		float64(s.GoodCount)*math.Pow(s.Mean-ret.Mean, 2) + // The sum of squares of set [s] is compared to the [ret] mean
		float64(other.GoodCount)*math.Pow(other.Mean-ret.Mean, 2) // The sum of squares of set [other] is compared to the [ret] mean
	ret.computeVariance()
	return ret
}

// Math proof for why this works:
// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Welford's_online_algorithm
//
// This will compute the variance and update this [Stats] pointer with the variance and standard
// deviation based on the current sumOfSquares.
func (s *Stats) computeVariance() {
	variance := 0.0
	std := 0.0
	if s.GoodCount >= 2 {
		variance = s.sumOfSquares / float64(s.GoodCount-1)
		std = math.Sqrt(variance)
	}
	s.Variance = variance
	s.StandardDeviation = std
}

func (s Stats) PacketLoss() float64 {
	return float64(s.PacketsDropped) / float64(s.GoodCount+s.PacketsDropped)
}

func (s *Stats) AddDroppedPacket() {
	s.PacketsDropped++
}

// TODO float imprecision
// TODO https://en.wikipedia.org/wiki/Kahan_summation_algorithm
// Math proof for why this works:
// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Welford's_online_algorithm
func (s *Stats) AddPoint(input time.Duration) {
	s.Max = max(s.Max, input)
	s.Min = min(s.Min, input)
	if s.GoodCount == 0 {
		s.Max = input
		s.Min = input
	}
	value := float64(input)
	s.GoodCount++
	delta := value - s.Mean
	newMean := s.Mean + (delta / float64(s.GoodCount))
	newDelta := value - newMean
	s.sumOfSquares += delta * newDelta

	s.Mean = newMean
	s.computeVariance()
}

func (s *Stats) AddPoints(values []time.Duration) {
	// TODO use one pass variance
	// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Weighted_incremental_algorithm
	for _, v := range values {
		s.AddPoint(v)
	}
}

func (ts TimeSpan) FormatDraw(width, padding int) (string, []string) {
	var format string
	const firstFormat = "02 Jan 2006 15:04:05.00"
	const halfDay = 12 * time.Hour
	const halfMonth = 30 * halfDay
	const halfYear = 12 * halfMonth
	switch {
	case ts.Duration > halfYear:
		format = firstFormat
	case ts.Duration > halfMonth:
		format = "Jan 02 15:04"
	case ts.Duration > halfDay:
		format = "02 15:04:05"
	case ts.Duration > 15*time.Minute:
		format = "15:04:05"
	case ts.Duration > time.Minute:
		format = "15:04:05.00"
	case ts.Duration > 30*time.Second:
		format = "04:05.0000"
	default:
		format = "05.0000"
	}
	startString := ts.Begin.Format(firstFormat)
	if width < len(firstFormat) {
		return startString, []string{}
	}
	remaining := width - (len(startString) + padding + padding)
	count := remaining / (len(format) + padding)
	if count <= 0 {
		return startString, []string{}
	}
	step := ts.Duration / time.Duration(count)
	steps := make([]string, count)
	for c := range count {
		steps[c] = ts.Begin.Add(step * time.Duration(c+1)).Format(format)
	}
	return startString, steps
}

func (ts TimeSpan) String() string {
	format := "15:04:05.9999"
	const firstFormat = "02 Jan 2006 15:04:05.99"
	const halfDay = 12 * time.Hour
	const halfMonth = 30 * halfDay
	const halfYear = 12 * halfMonth
	switch {
	case ts.Duration > halfYear:
		format = firstFormat
	case ts.Duration > halfDay:
		format = "06 15:04:05"
	case ts.Duration > halfMonth:
		format = "Jan 06 15:04"
	case ts.Duration > time.Minute, ts.Duration > time.Hour:
		format = "15:04:05.99"
	}
	return fmt.Sprintf("%s -> %s (%s)", ts.Begin.Format(firstFormat), ts.End.Format(format), ts.Duration.String())
}

func stringFloatTime(f float64) string {
	d := time.Duration(f)
	return d.String()
}

func (s Stats) PickString(remainingSpace int) string {
	// heuristic is good enough for now
	switch {
	case remainingSpace > 100:
		return s.longString()
	case remainingSpace > 80 && s.PacketsDropped > 0:
		return s.mediumString()
	case remainingSpace > 55 && s.PacketsDropped == 0:
		return s.mediumString()
	case remainingSpace > 61 && s.PacketsDropped > 0:
		return s.shortString()
	case remainingSpace > 45 && s.PacketsDropped == 0:
		return s.shortString()
	case remainingSpace > 10:
		return s.superShortString()
	default:
		return ""
	}
}

func (s Stats) String() string {
	return s.mediumString()
}

func (s Stats) packetLoss(b *strings.Builder, prefix string) {
	if s.PacketsDropped > 0 {
		percent := numeric.RoundToNearestSigFig(s.PacketLoss(), 4) * 100
		if percent > 0.1 {
			fmt.Fprintf(b, " | %s%.1f%%", prefix, percent)
		}
	}
}

func (s Stats) superShortString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\u03BC %s | \u03C3 %s",
		stringFloatTime(numeric.RoundToNearestSigFig(s.Mean, 4)),
		stringFloatTime(numeric.RoundToNearestSigFig(s.StandardDeviation, 4)))
	s.packetLoss(&b, "")
	fmt.Fprintf(&b, " | Count %d", s.PacketsDropped+s.GoodCount)
	return b.String()
}

func (s Stats) shortString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\u03BC %s | \u03C3 %s",
		stringFloatTime(s.Mean), stringFloatTime(s.StandardDeviation))
	s.packetLoss(&b, "Loss ")
	fmt.Fprintf(&b, " | Packet Count %d", s.PacketsDropped+s.GoodCount)
	return b.String()
}

func (s Stats) mediumString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Average \u03BC %s | SD \u03C3 %s",
		stringFloatTime(s.Mean), stringFloatTime(s.StandardDeviation))
	s.packetLoss(&b, "PacketLoss ")
	fmt.Fprintf(&b, " | Packet Count %d", s.PacketsDropped+s.GoodCount)
	return b.String()
}

func (s Stats) longString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Average \u03BC %s | SD \u03C3 %s",
		stringFloatTime(s.Mean), stringFloatTime(s.StandardDeviation))
	s.packetLoss(&b, "PacketLoss ")
	fmt.Fprintf(&b, " | Dropped %d", s.PacketsDropped)
	fmt.Fprintf(&b, " | Good Packets %d | Packet Count %d", s.GoodCount, s.PacketsDropped+s.GoodCount)
	return b.String()
}

type version byte

const (
	noRuns version = iota + 1
	runsWithNoIndex
	currentDataVersion
)

func (d *Data) Migrate() {
	startingVersion := d.PingsMeta
	// Keep migrating until we are the current version, don't modify the starting version though, we want it preserved.
	for {
		switch startingVersion {
		case noRuns:
			// This migration is literally the same as the next but without indexes, we may as well just defer this to
			// the next migration
		case runsWithNoIndex:
			d.Runs = &Runs{GoodPackets: &Run{}, DroppedPackets: &Run{}}
			for i := range d.TotalCount {
				p := d.Get(i)
				d.Runs.AddPoint(i, p)
			}
		case currentDataVersion:
			return
		}
		// Perform the next migration
		startingVersion++
	}
}
