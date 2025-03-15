// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package data

import (
	"encoding/binary"
	"math"
	"net"
	"time"

	"github.com/Lexer747/acci-ping/ping"
	"github.com/Lexer747/acci-ping/utils/errors"
)

// sliceLenCompact works out the dynamic size for all items in a slice.s
func sliceLenCompact[S ~[]T, T Compact](slice S) int {
	i := int64Len // 1 int64 to encode the length
	for _, item := range slice {
		i += item.byteLen()
	}
	return i
}

// sliceLenFixed works out the size of a slice where all the items are of fixed size.
func sliceLenFixed[S ~[]T, T any](slice S, itemLen int) int {
	return int64Len + len(slice)*itemLen
}

func stringLen[S ~string](str S) int {
	return int64Len + // 1 int64 to encode the length
		len([]byte(str)) // The number of bytes in the string
}

func writePingDataPoint(b []byte, p ping.PingDataPoint) int {
	i := writeDuration(b, p.Duration)
	i += writeTime(b[i:], p.Timestamp)
	i += writeByte(b[i:], p.DropReason)
	return i
}

func readPingDataPoint(b []byte, p *ping.PingDataPoint) int {
	i := readDuration(b, &p.Duration)
	i += readTime(b[i:], &p.Timestamp)
	i += readByte(b[i:], &p.DropReason)
	return i
}

func writeTime(b []byte, t time.Time) int {
	return writeInt64(b, t.UnixMilli())
}

func readTime(b []byte, t *time.Time) int {
	var i int64
	ret := readInt64(b, &i)
	*t = time.UnixMilli(i)
	return ret
}

func writeDuration(b []byte, d time.Duration) int {
	return writeInt64(b, int64(d))
}

func readDuration(b []byte, d *time.Duration) int {
	var i int64
	ret := readInt64(b, &i)
	*d = time.Duration(i)
	return ret
}

func writeIP(b []byte, ip net.IP) int {
	ensure16 := ip.To16()
	copy(b, ensure16)
	return netIPLen
}

func readIP(b []byte, ip net.IP) int {
	copy(ip, b[:netIPLen])
	return netIPLen
}

func readID(b []byte, id Identifier) (int, error) {
	if len(b) <= 0 {
		return 0, errors.Errorf("Cannot read id, not enough bytes")
	}
	if id != Identifier(b[0]) {
		return 0, errors.Errorf("Unexpected id %d != %d", b[0], id)
	}
	return 1, nil
}

func writeByte[b ~byte](buf []byte, toWrite b) int {
	buf[0] = byte(toWrite)
	return 1
}

func readByte[b ~byte](buf []byte, toRead *b) int {
	*toRead = b(buf[0])
	return 1
}

func writeStringLen[S ~string](b []byte, str S) int {
	return writeLen(b, []byte(str))
}

func writeString[S ~string](b []byte, str S) int {
	asBytes := []byte(str)
	copy(b, asBytes)
	return len(asBytes)
}

func readString[S ~string](b []byte, s *S, strLen int) int {
	*s = S(string(b[:strLen]))
	return strLen
}

func writeLen[S ~[]T, T any](b []byte, slice S) int {
	binary.LittleEndian.PutUint64(b, uint64(len(slice)))
	return int64Len
}

func readLen(b []byte, i *int) int {
	//nolint:gosec
	// G115 if this overflows it means the underlying file was written with a system supporting 64 bits (as it
	// reached that length of slice), but this current code reading the file is only 32 bits, in which case it
	// won't be able to store the result anyway.
	*i = int(binary.LittleEndian.Uint64(b))
	return int64Len
}

func writeInt64(b []byte, i int64) int {
	//nolint:gosec
	// G115 converting to a uint64 is an overflow but we are simply writing the raw bits to the buffer for later.
	binary.LittleEndian.PutUint64(b, uint64(i))
	return int64Len
}

func readInt64(b []byte, i *int64) int {
	//nolint:gosec
	// G115 converting to a int64 is an overflow but we are simply reading the raw bits to the buffer
	// which started life as a int64.
	*i = int64(binary.LittleEndian.Uint64(b))
	return int64Len
}

func writeInt(b []byte, i int) int {
	//nolint:gosec
	// G115 converting to a uint64 is an overflow but we are simply writing the raw bits to the buffer for later.
	binary.LittleEndian.PutUint64(b, uint64(i))
	return int64Len
}

func readInt(b []byte, i *int) int {
	//nolint:gosec
	// G115 converting to a int64 is an overflow but we are simply reading the raw bits to the buffer
	// which started life as a int.
	*i = int(binary.LittleEndian.Uint64(b))
	return int64Len
}

func writeUint64(b []byte, i uint64) int {
	binary.LittleEndian.PutUint64(b, i)
	return uint64Len
}

func readUint64(b []byte, i *uint64) int {
	*i = binary.LittleEndian.Uint64(b)
	return uint64Len
}

func writeFloat64(b []byte, i float64) int {
	binary.LittleEndian.PutUint64(b, math.Float64bits(i))
	return float64Len
}

func readFloat64(b []byte, i *float64) int {
	*i = math.Float64frombits(binary.LittleEndian.Uint64(b))
	return float64Len
}
