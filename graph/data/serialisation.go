// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package data

import (
	"io"
	"net"

	"github.com/Lexer747/acci-ping/ping"
	"github.com/Lexer747/acci-ping/utils/errors"
)

type Identifier byte

const (
	_ Identifier = 0

	TimeSpanID Identifier = 1
	StatsID    Identifier = 2
	BlockID    Identifier = 3
	HeaderID   Identifier = 4
	DataID     Identifier = 5
	NetworkID  Identifier = 6
	RunsID     Identifier = 7

	_ Identifier = 0xff
)

func ReadData(r io.Reader) (*Data, error) {
	toReadFrom, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "While reading into Data{}")
	}
	d := &Data{}
	_, err = d.FromCompact(toReadFrom)
	if err != nil {
		return nil, errors.Wrap(err, "While reading into Data{}")
	}
	return d, nil
}

type Compact interface {
	// AsCompact convert a [Compact]ing thing into bytes
	AsCompact(w io.Writer) error

	write(toWriteInto []byte) int
	byteLen() int

	// FromCompact converts raw bytes back into said thing
	FromCompact(input []byte) (int, error)
}

var _ Compact = (&Block{})
var _ Compact = (&DataIndexes{})
var _ Compact = (&Data{})
var _ Compact = (&Header{})
var _ Compact = (&Network{})
var _ Compact = (&Stats{})
var _ Compact = (&Runs{})
var _ Compact = (&Run{})
var _ Compact = (&TimeSpan{})

// PhasedWrite is generally used by a compacting implementor to indicate that the data must be written in two
// phases, each phase is of this type. This is useful for types which have dynamic sizes e.g. [Network] which
// will write all the sizes of it's slices in it's first phase, then it's second phase is to write the
// variable length data. This allows the reader to be more simple and efficient as it can read all the sizes
// before consuming all the bytes.
type PhasedWrite = func(ret []byte) int

func (d *Data) AsCompact(w io.Writer) error {
	ret := make([]byte, d.byteLen())
	_ = d.write(ret)
	_, err := w.Write(ret)
	return err
}

func (d *Data) write(ret []byte) int {
	networkHeader, networkData := d.Network.twoPhaseWrite()
	i := writeByte(ret, DataID)
	// We explicitly do not preserve the version in this data, we have migrated and the write code only ever
	// supports the latest version.
	i += writeByte(ret[i:], currentDataVersion)
	i += writeLen(ret[i:], d.InsertOrder)
	i += writeInt64(ret[i:], d.TotalCount)
	i += networkHeader(ret[i:])
	i += writeInt(ret[i:], blockHeaderLen())
	i += writeLen(ret[i:], d.Blocks)
	deferredData := make([]PhasedWrite, len(d.Blocks))
	for blockIndex, block := range d.Blocks {
		header, data := block.twoPhaseWrite()
		deferredData[blockIndex] = data
		i += header(ret[i:])
	}
	i += writeStringLen(ret[i:], d.URL)
	i += d.Runs.write(ret[i:])
	i += d.Header.write(ret[i:])

	// Phase 2 the variable length data
	for _, insert := range d.InsertOrder {
		i += insert.write(ret[i:])
	}
	i += networkData(ret[i:])
	for _, blockData := range deferredData {
		i += blockData(ret[i:])
	}
	i += writeString(ret[i:], d.URL)
	return i
}

func (d *Data) FromCompact(input []byte) (int, error) {
	if d.Network == nil {
		d.Network = &Network{}
	}
	if d.Header == nil {
		d.Header = &Header{}
	}
	i, err := readID(input, DataID)
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Data")
	}
	i += readByte(input[i:], &d.PingsMeta)
	switch d.PingsMeta {
	case noRuns:
		n, err := d.readVersion1(i, input)
		if err != nil {
			return i, errors.Wrap(err, "while reading compact Data")
		}
		i += n
		d.Migrate()
		return i, nil
	case runsWithNoIndex, currentDataVersion:
		n, err := d.readVersion2(i, input)
		if err != nil {
			return i, errors.Wrap(err, "while reading compact Data")
		}
		i += n
		d.Migrate()
		return i, nil
	default:
		panic("exhaustive:enforce")
	}
}

func (d *Data) readVersion2(i int, input []byte) (int, error) {
	insertOrderLen := 0
	i += readLen(input[i:], &insertOrderLen)
	i += readInt64(input[i:], &d.TotalCount)
	networkHeaderReader, networkDataReader := d.Network.twoPhaseRead()
	var IPsLen, blockIndexesLen int
	n, err := networkHeaderReader(input[i:], &IPsLen, &blockIndexesLen)
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Data")
	}
	i += n
	// drop block header len, we know it's fixed until new versions are introduced
	i += readInt(input[i:], &n)
	blockLen := 0
	i += readLen(input[i:], &blockLen)
	d.Blocks = make([]*Block, blockLen)
	blockSizes := make([]*int, blockLen)
	blockReads := make([]BlockRead, blockLen)
	for index := range blockLen {
		d.Blocks[index] = &Block{}
		blockSizes[index] = new(int)
		header, data := d.Blocks[index].twoPhaseRead()
		n, err := header(input[i:], blockSizes[index])
		if err != nil {
			return i, errors.Wrap(err, "while reading compact Data")
		}
		i += n
		blockReads[index] = data
	}
	URLLen := 0
	i += readLen(input[i:], &URLLen)
	if d.Runs == nil {
		d.Runs = &Runs{}
	}
	n, err = d.Runs.fromCompact(input[i:], d.PingsMeta)
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Data")
	}
	i += n
	n, err = d.Header.FromCompact(input[i:])
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Data")
	}
	i += n

	// Phase 2 read the variable sized data
	d.InsertOrder = make([]DataIndexes, insertOrderLen)
	for index := range d.InsertOrder {
		insert := &d.InsertOrder[index]
		n, err := insert.FromCompact(input[i:])
		if err != nil {
			return i, errors.Wrap(err, "while reading compact Data")
		}
		i += n
	}
	i += networkDataReader(input[i:], IPsLen, blockIndexesLen)
	for index, blockData := range blockReads {
		i += blockData(input[i:], *blockSizes[index])
	}
	i += readString(input[i:], &d.URL, URLLen)
	return i, nil
}

func (d *Data) readVersion1(i int, input []byte) (int, error) {
	insertOrderLen := 0
	i += readLen(input[i:], &insertOrderLen)
	i += readInt64(input[i:], &d.TotalCount)
	networkHeaderReader, networkDataReader := d.Network.twoPhaseRead()
	var IPsLen, blockIndexesLen int
	n, err := networkHeaderReader(input[i:], &IPsLen, &blockIndexesLen)
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Data")
	}
	i += n
	// drop block header len, we know it's fixed until new versions are introduced
	i += readInt(input[i:], &n)
	blockLen := 0
	i += readLen(input[i:], &blockLen)
	d.Blocks = make([]*Block, blockLen)
	blockSizes := make([]*int, blockLen)
	blockReads := make([]BlockRead, blockLen)
	for index := range blockLen {
		d.Blocks[index] = &Block{}
		blockSizes[index] = new(int)
		header, data := d.Blocks[index].twoPhaseRead()
		n, err := header(input[i:], blockSizes[index])
		if err != nil {
			return i, errors.Wrap(err, "while reading compact Data")
		}
		i += n
		blockReads[index] = data
	}
	URLLen := 0
	i += readLen(input[i:], &URLLen)
	n, err = d.Header.FromCompact(input[i:])
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Data")
	}
	i += n

	// Phase 2 read the variable sized data
	d.InsertOrder = make([]DataIndexes, insertOrderLen)
	for index := range d.InsertOrder {
		insert := &d.InsertOrder[index]
		n, err := insert.FromCompact(input[i:])
		if err != nil {
			return i, errors.Wrap(err, "while reading compact Data")
		}
		i += n
	}
	i += networkDataReader(input[i:], IPsLen, blockIndexesLen)
	for index, blockData := range blockReads {
		i += blockData(input[i:], *blockSizes[index])
	}
	i += readString(input[i:], &d.URL, URLLen)
	return i, nil
}

func (d *Data) byteLen() int {
	return idLen + // Identifier
		1 + // Version
		int64Len + // TotalCount
		d.Runs.byteLen() +
		d.Header.byteLen() +
		d.Network.byteLen() +
		intLen + // blockHeaderLen
		// Begin Variable sized items:
		sliceLenCompact(d.Blocks) +
		sliceLenFixed(d.InsertOrder, dataIndexesLen) +
		stringLen(d.URL)
}

func (b *Block) AsCompact(w io.Writer) error {
	ret := make([]byte, b.byteLen())
	_ = b.write(ret)
	_, err := w.Write(ret)
	return err
}

func (b *Block) FromCompact(input []byte) (int, error) {
	header, data := b.twoPhaseRead()
	rawLen := 0
	i, err := header(input, &rawLen)
	if err != nil {
		return i, err
	}
	return data(input[i:], rawLen), nil
}

func (b *Block) write(ret []byte) int {
	header, data := b.twoPhaseWrite()
	i := header(ret)
	i += data(ret[i:])
	return i
}

func (b *Block) twoPhaseWrite() (PhasedWrite, PhasedWrite) {
	return func(ret []byte) int {
			i := writeByte(ret, BlockID)
			i += writeLen(ret[i:], b.Raw)
			i += b.Header.write(ret[i:])
			return i
		}, func(ret []byte) int {
			i := 0
			for _, raw := range b.Raw {
				i += writePingDataPoint(ret[i:], raw)
			}
			return i
		}
}

type BlockRead = func(input []byte, rawLen int) int

func (b *Block) twoPhaseRead() (
	func(input []byte, rawLen *int) (int, error),
	BlockRead,
) {
	if b.Header == nil {
		b.Header = &Header{}
	}
	return func(input []byte, blockLen *int) (int, error) {
			i, err := readID(input, BlockID)
			if err != nil {
				return i, errors.Wrap(err, "while reading compact Block")
			}
			i += readLen(input[i:], blockLen)
			n, err := b.Header.FromCompact(input[i:])
			if err != nil {
				return i, errors.Wrap(err, "while reading compact Block")
			}
			return i + n, err
		},
		func(input []byte, rawLen int) int {
			b.Raw = make([]ping.PingDataPoint, rawLen)
			i := 0
			for rawIndex := range b.Raw {
				i += readPingDataPoint(input[i:], &b.Raw[rawIndex])
			}
			return i
		}
}

func (b *Block) byteLen() int {
	return idLen + headerLen + sliceLenFixed(b.Raw, pingDataPointLen)
}

func blockHeaderLen() int {
	return idLen + headerLen + sliceLenFixed([]byte{}, 0)
}

func (h *Header) AsCompact(w io.Writer) error {
	ret := make([]byte, h.byteLen())
	_ = h.write(ret)
	_, err := w.Write(ret)
	return err
}

func (h *Header) write(ret []byte) int {
	i := writeByte(ret, HeaderID)
	i += h.Stats.write(ret[i:])
	i += h.TimeSpan.write(ret[i:])
	return i
}

func (h *Header) FromCompact(input []byte) (int, error) {
	i, err := readID(input, HeaderID)
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Header")
	}
	if h.Stats == nil {
		h.Stats = &Stats{}
	}
	n, err := h.Stats.FromCompact(input[i:])
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Header")
	}
	i += n
	if h.TimeSpan == nil {
		h.TimeSpan = &TimeSpan{}
	}
	n, err = h.TimeSpan.FromCompact(input[i:])
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Header")
	}
	i += n
	return i, nil
}

func (h *Header) byteLen() int {
	return headerLen
}

func (n *Network) AsCompact(w io.Writer) error {
	thisLen := n.byteLen()
	ret := make([]byte, thisLen)
	n.write(ret)
	_, err := w.Write(ret)
	return err
}

func (n *Network) write(ret []byte) int {
	header, data := n.twoPhaseWrite()
	i := header(ret)
	i += data(ret[i:])
	return i
}

func (n *Network) twoPhaseWrite() (PhasedWrite, PhasedWrite) {
	return func(ret []byte) int {
			i := writeByte(ret, NetworkID)
			i += writeInt(ret[i:], n.curBlockIndex)
			i += writeLen(ret[i:], n.IPs)
			i += writeLen(ret[i:], n.BlockIndexes)
			return i
		}, func(ret []byte) int {
			i := 0
			for _, ip := range n.IPs {
				i += writeIP(ret[i:], ip)
			}
			for _, index := range n.BlockIndexes {
				i += writeInt(ret[i:], index)
			}
			return i
		}
}

func (n *Network) twoPhaseRead() (
	func(input []byte, IPsLen, blockIndexesLen *int) (int, error),
	func(input []byte, IPsLen, blockIndexesLen int) int) {
	return func(input []byte, IPsLen, blockIndexesLen *int) (int, error) {
			i, err := readID(input, NetworkID)
			if err != nil {
				return i, errors.Wrap(err, "while reading compact Network")
			}
			i += readInt(input[i:], &n.curBlockIndex)
			i += readLen(input[i:], IPsLen)
			i += readLen(input[i:], blockIndexesLen)
			return i, nil
		},
		func(input []byte, IPsLen, blockIndexesLen int) int {
			n.IPs = make([]net.IP, IPsLen)
			n.BlockIndexes = make([]int, blockIndexesLen)
			i := 0
			for ip := range n.IPs {
				n.IPs[ip] = make(net.IP, netIPLen)
				i += readIP(input[i:], n.IPs[ip])
			}
			for blockIndex := range n.BlockIndexes {
				i += readInt(input[i:], &n.BlockIndexes[blockIndex])
			}
			return i
		}
}

func (n *Network) byteLen() int {
	return sliceLenFixed(n.IPs, netIPLen) + sliceLenFixed(n.BlockIndexes, intLen) + intLen + idLen
}

func (n *Network) FromCompact(input []byte) (int, error) {
	header, data := n.twoPhaseRead()
	IPsLen := 0
	BlockIndexesLen := 0
	i, err := header(input, &IPsLen, &BlockIndexesLen)
	if err != nil {
		return i, err
	}
	return data(input[i:], IPsLen, BlockIndexesLen), nil
}

func (s *Stats) AsCompact(w io.Writer) error {
	ret := make([]byte, statsLen)
	_ = s.write(ret)
	_, err := w.Write(ret)
	return err
}

func (s *Stats) write(ret []byte) int {
	i := writeByte(ret, StatsID)
	i += writeDuration(ret[i:], s.Min)
	i += writeDuration(ret[i:], s.Max)
	i += writeFloat64(ret[i:], s.Mean)
	i += writeUint64(ret[i:], s.GoodCount)
	i += writeFloat64(ret[i:], s.Variance)
	i += writeFloat64(ret[i:], s.StandardDeviation)
	i += writeUint64(ret[i:], s.PacketsDropped)
	i += writeFloat64(ret[i:], s.sumOfSquares)
	return i
}

func (s *Stats) FromCompact(input []byte) (int, error) {
	i, err := readID(input, StatsID)
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Stats")
	}
	i += readDuration(input[i:], &s.Min)
	i += readDuration(input[i:], &s.Max)
	i += readFloat64(input[i:], &s.Mean)
	i += readUint64(input[i:], &s.GoodCount)
	i += readFloat64(input[i:], &s.Variance)
	i += readFloat64(input[i:], &s.StandardDeviation)
	i += readUint64(input[i:], &s.PacketsDropped)
	i += readFloat64(input[i:], &s.sumOfSquares)
	return i, nil
}

func (s *Stats) byteLen() int {
	return statsLen
}

func (ts *TimeSpan) AsCompact(w io.Writer) error {
	ret := make([]byte, timeSpanLen)
	_ = ts.write(ret)
	_, err := w.Write(ret)
	return err
}

func (ts *TimeSpan) write(ret []byte) int {
	i := writeByte(ret, TimeSpanID)
	i += writeTime(ret[i:], ts.Begin)
	i += writeTime(ret[i:], ts.End)
	i += writeDuration(ret[i:], ts.Duration)
	return i
}

func (ts *TimeSpan) FromCompact(input []byte) (int, error) {
	i, err := readID(input, TimeSpanID)
	if err != nil {
		return i, errors.Wrap(err, "while reading compact TimeSpan")
	}
	i += readTime(input[i:], &ts.Begin)
	i += readTime(input[i:], &ts.End)
	i += readDuration(input[i:], &ts.Duration)
	return i, nil
}

func (ts *TimeSpan) byteLen() int {
	return timeSpanLen
}

func (r *Runs) AsCompact(w io.Writer) error {
	ret := make([]byte, runsLen)
	_ = r.write(ret)
	_, err := w.Write(ret)
	return err
}

func (r *Runs) FromCompact(input []byte) (int, error) {
	return r.fromCompact(input, currentDataVersion)
}
func (r *Runs) fromCompact(input []byte, version version) (int, error) {
	i, err := readID(input, RunsID)
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Runs")
	}
	if r.DroppedPackets == nil {
		r.DroppedPackets = &Run{}
	}
	if r.GoodPackets == nil {
		r.GoodPackets = &Run{}
	}
	n, err := r.GoodPackets.fromCompact(input[i:], version)
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Runs")
	}
	i += n
	n, err = r.DroppedPackets.fromCompact(input[i:], version)
	if err != nil {
		return i, errors.Wrap(err, "while reading compact Runs")
	}
	i += n
	return i, nil
}

func (r *Runs) write(ret []byte) int {
	i := writeByte(ret, RunsID)
	i += r.GoodPackets.write(ret[i:])
	i += r.DroppedPackets.write(ret[i:])
	return i
}

func (r *Runs) byteLen() int {
	return runsLen
}

func (r *Run) AsCompact(w io.Writer) error {
	ret := make([]byte, runLen)
	_ = r.write(ret)
	_, err := w.Write(ret)
	return err
}

func (r *Run) fromCompact(input []byte, version version) (int, error) {
	switch version {
	case noRuns:
		panic("should not be called")
	case runsWithNoIndex:
		i := readUint64(input, &r.Longest)
		i += readUint64(input[i:], &r.Current)
		return i, nil
	case currentDataVersion:
		i := readInt64(input, &r.LongestIndexEnd)
		i += readUint64(input[i:], &r.Longest)
		i += readUint64(input[i:], &r.Current)
		return i, nil
	}
	panic("exhaustive:enforce")
}

func (r *Run) FromCompact(input []byte) (int, error) {
	return r.fromCompact(input, currentDataVersion)
}

func (r *Run) write(ret []byte) int {
	i := writeInt64(ret, r.LongestIndexEnd)
	i += writeUint64(ret[i:], r.Longest)
	i += writeUint64(ret[i:], r.Current)
	return i
}

func (r *Run) byteLen() int {
	return runLen
}

func (di *DataIndexes) AsCompact(w io.Writer) error {
	ret := make([]byte, di.byteLen())
	_ = di.write(ret)
	_, err := w.Write(ret)
	return err
}

func (di *DataIndexes) FromCompact(input []byte) (int, error) {
	i := readInt(input, &di.BlockIndex)
	i += readInt(input[i:], &di.RawIndex)
	return i, nil
}

func (di *DataIndexes) write(toWriteInto []byte) int {
	i := writeInt(toWriteInto, di.BlockIndex)
	i += writeInt(toWriteInto[i:], di.RawIndex)
	return i
}

func (di *DataIndexes) byteLen() int {
	return dataIndexesLen
}

// Lens in Bytes
const (
	intLen          = int64Len
	int64Len        = 8
	uint64Len       = int64Len
	float64Len      = int64Len
	timeLen         = int64Len
	timeDurationLen = int64Len
	idLen           = 1
	netIPLen        = 16 // Always store in ipv6 form

	timeSpanLen      = idLen + 2*timeLen + timeDurationLen
	statsLen         = idLen + 2*timeDurationLen + 4*float64Len + 2*uint64Len
	headerLen        = idLen + timeSpanLen + statsLen
	pingDataPointLen = timeDurationLen + timeLen + 1
	dataIndexesLen   = intLen + intLen
	runLen           = int64Len + uint64Len + uint64Len
	runsLen          = idLen + runLen + runLen
)
