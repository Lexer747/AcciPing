// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
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

	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/numeric"
	"github.com/Lexer747/AcciPing/utils/sliceutils"
)

type Data struct {
	URL         string
	Header      *Header
	Network     *Network
	InsertOrder []DataIndexes
	Blocks      []*Block
	TotalCount  int64
	Version     byte
}

type DataIndexes struct {
	BlockIndex, RawIndex int
}

func NewData(URL string) *Data {
	d := &Data{
		URL:         URL,
		Header:      &Header{Stats: &Stats{}, TimeSpan: &TimeSpan{Begin: time.UnixMilli(0), End: time.UnixMilli(0), Duration: 0}},
		Network:     &Network{IPs: []net.IP{}, BlockIndexes: []int{}, curBlockIndex: 0},
		InsertOrder: []DataIndexes{},
		Blocks:      []*Block{},
		TotalCount:  0,
		Version:     currentDataVersion,
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
	return fmt.Sprintf("%s: [%s] | %s", d.URL, d.Network.String(), d.Header.String())
}

// TimeSpan is the time properties of a given thing
type TimeSpan struct {
	Begin    time.Time
	End      time.Time
	Duration time.Duration
}

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
	newCount := s.GoodCount + 1
	delta := value - s.Mean
	newMean := s.Mean + (delta / float64(newCount))
	newDelta := value - newMean
	s.sumOfSquares += delta * newDelta

	variance := 0.0
	std := 0.0
	if newCount >= 2 {
		variance = s.sumOfSquares / float64(newCount-1)
		std = math.Sqrt(float64(variance))
	}
	s.GoodCount = newCount
	s.Mean = newMean
	s.Variance = float64(variance)
	s.StandardDeviation = std
}

func (s *Stats) AddPoints(values []time.Duration) {
	// TODO use one pass variance
	// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Weighted_incremental_algorithm
	for _, v := range values {
		s.AddPoint(v)
	}
}

func Merge(stats ...*Stats) *Stats {
	// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Weighted_incremental_algorithm
	panic("todo")
}

func (ts TimeSpan) String() string {
	format := "15:04:05.9999"
	const firstFormat = "02 Jan 2006 15:04:05.99"
	const day = 24 * time.Hour
	const month = 30 * day
	const year = 12 * month
	switch {
	case ts.Duration > time.Minute:
		format = "15:04:05.99"
	case ts.Duration > time.Hour:
		format = "15:04:05.99"
	case ts.Duration > day:
		format = "06 15:04:05"
	case ts.Duration > month:
		format = "Jan 06 15:04"
	case ts.Duration > year:
		format = firstFormat
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

func (s Stats) superShortString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\u03BC %s | \u03C3 %s",
		stringFloatTime(numeric.RoundToNearestSigFig(s.Mean, 4)),
		stringFloatTime(numeric.RoundToNearestSigFig(s.StandardDeviation, 4)))
	if s.PacketsDropped > 0 {
		fmt.Fprintf(&b, " | %.1f%%", numeric.RoundToNearestSigFig(s.PacketLoss(), 4)*100)
	}
	fmt.Fprintf(&b, " | Count %d", s.PacketsDropped+s.GoodCount)
	return b.String()
}

func (s Stats) shortString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\u03BC %s | \u03C3 %s",
		stringFloatTime(s.Mean), stringFloatTime(s.StandardDeviation))
	if s.PacketsDropped > 0 {
		fmt.Fprintf(&b, " | Loss %.1f%%", numeric.RoundToNearestSigFig(s.PacketLoss(), 4)*100)
	}
	fmt.Fprintf(&b, " | Packet Count %d", s.PacketsDropped+s.GoodCount)
	return b.String()
}

func (s Stats) mediumString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Average \u03BC %s | SD \u03C3 %s",
		stringFloatTime(s.Mean), stringFloatTime(s.StandardDeviation))
	if s.PacketsDropped > 0 {
		fmt.Fprintf(&b, " | PacketLoss %.1f%%", numeric.RoundToNearestSigFig(s.PacketLoss(), 4)*100)
	}
	fmt.Fprintf(&b, " | Packet Count %d", s.PacketsDropped+s.GoodCount)
	return b.String()
}

func (s Stats) longString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Average \u03BC %s | SD \u03C3 %s",
		stringFloatTime(s.Mean), stringFloatTime(s.StandardDeviation))
	fmt.Fprintf(&b, " | PacketLoss %.1f%% | Dropped %d", numeric.RoundToNearestSigFig(s.PacketLoss(), 4)*100, s.PacketsDropped)
	fmt.Fprintf(&b, " | Good Packets %d | Packet Count %d", s.GoodCount, s.PacketsDropped+s.GoodCount)
	return b.String()
}

const currentDataVersion = 1
