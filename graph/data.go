// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/numeric"
)

type Data struct {
	*Header
	Blocks                []*Block
	BetweenBlockGradients []float64
	TotalCount            int
	curBlock              int
	configuredBlockLimit  int
}

type Options struct {
	BlockSize int
}

func NewData(o ...Options) *Data {
	defaultBlockLimit := 2048
	if len(o) > 0 { // TODO explain options
		defaultBlockLimit = o[0].BlockSize
	}
	d := &Data{
		Header:               &Header{Stats: &Stats{}},
		curBlock:             0,
		configuredBlockLimit: defaultBlockLimit,
	}
	d.Blocks = []*Block{{
		Header:    &Header{Stats: &Stats{}},
		Raw:       make([]ping.PingResults, 0, defaultBlockLimit),
		Gradients: make([]float64, 0, defaultBlockLimit-1),
	}}
	return d
}

func (d *Data) AddPoint(p ping.PingResults) {
	curBlock := d.getCurrentBlock()
	if len(curBlock.Raw) >= d.configuredBlockLimit {
		// Make a new block and swap to it
		d.addBlock()
		last := curBlock.Raw[len(curBlock.Raw)-1]
		d.BetweenBlockGradients = append(d.BetweenBlockGradients, gradient(last, p))
		curBlock = d.getCurrentBlock()
	}
	curBlock.AddPoint(p)
	d.Header.AddPoint(p)
	d.TotalCount++
}

func (d *Data) addBlock() {
	d.Blocks = append(d.Blocks, &Block{
		Header: &Header{Stats: &Stats{}},
		Raw:    make([]ping.PingResults, 0, d.configuredBlockLimit),
	})
	d.curBlock++
}

func (d *Data) getCurrentBlock() *Block {
	return d.Blocks[d.curBlock]
}

// TimeSpan is the time properties of a given thing
type TimeSpan struct {
	Begin    time.Time
	End      time.Time
	Duration time.Duration
}

func (s *TimeSpan) AddTimestamp(t time.Time) {
	if s.Begin.After(t) {
		s.Begin = t
	}
	if s.End.Before(t) {
		s.End = t
	}
	s.Duration = s.End.Sub(s.Begin)
}

// Header describes the statistical properties of a group of objects.
type Header struct {
	Stats                    *Stats
	Span                     *TimeSpan
	MinGradient, MaxGradient float64
}

func (h *Header) AddPoint(p ping.PingResults) {
	if h.Stats.GoodCount == 0 {
		h.Span = &TimeSpan{Begin: p.Timestamp, End: p.Timestamp}
	} else {
		h.Span.AddTimestamp(p.Timestamp)
	}
	if p.Error == nil {
		h.Stats.AddPoint(p.Duration)
	} else {
		h.Stats.AddDroppedPacket()
	}
}

type Block struct {
	*Header
	Raw       []ping.PingResults
	Gradients []float64
}

func (b *Block) AddPoint(p ping.PingResults) {
	b.Header.AddPoint(p)
	b.Raw = append(b.Raw, ping.PingResults{
		Duration:  p.Duration,
		Timestamp: p.Timestamp,
		Error:     p.Error,
	})
	if b.Header.Stats.GoodCount > 1 {
		last := b.Raw[len(b.Raw)-2]
		grad := gradient(last, p)
		b.Gradients = append(b.Gradients, grad)
		if b.Header.Stats.GoodCount == 2 {
			b.Header.MaxGradient = grad
			b.Header.MinGradient = grad
		} else {
			b.Header.MaxGradient = max(b.Header.MaxGradient, grad)
			b.Header.MinGradient = min(b.Header.MinGradient, grad)
		}
	}
}

func gradient(last ping.PingResults, current ping.PingResults) float64 {
	rise := float64(current.Duration) - float64(last.Duration)
	run := float64(current.Timestamp.Sub(last.Timestamp))
	return rise / run
}

type Stats struct {
	Min, Max          time.Duration
	Mean              float64
	GoodCount         uint
	Variance          float64
	StandardDeviation float64
	PacketsDropped    uint
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

func (s TimeSpan) String() string {
	format := "15:04:05.9999"
	const day = 24 * time.Hour
	const month = 30 * day
	const year = 12 * month
	switch {
	case s.Duration > time.Minute:
		format = "15:04:05.99"
	case s.Duration > time.Hour:
		format = "15:04:05.99"
	case s.Duration > day:
		format = "06 15:04:05"
	case s.Duration > month:
		format = "Jan 06 15:04"
	case s.Duration > year:
		format = "02 Jan 06 15:04"
	}
	return fmt.Sprintf("%s -> %s (%s)", s.Begin.Format(format), s.End.Format(format), s.Duration.String())
}

func stringFloatTime(f float64) string {
	d := time.Duration(f)
	return d.String()
}

func (s Stats) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Average \u03BC %s | SD \u03C3 %s",
		stringFloatTime(s.Mean), stringFloatTime(s.StandardDeviation))
	if s.PacketsDropped > 0 {
		fmt.Fprintf(&b, " | PacketLoss %f%% | Dropped %d", numeric.RoundToNearestSigFig(s.PacketLoss(), 4), s.PacketsDropped)
	}
	fmt.Fprintf(&b, " | Good Packets %d | Total Packets %d", s.GoodCount, s.PacketsDropped+s.GoodCount)
	return b.String()
}

func (h Header) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s | %s", h.Span.String(), h.Stats.String())
	return b.String()
}
