// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package data_test

import (
	"bytes"
	"net"
	"os"
	"testing"
	"time"

	"github.com/Lexer747/acci-ping/graph/data"
	"github.com/Lexer747/acci-ping/ping"
	"github.com/Lexer747/acci-ping/utils/th"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestCompactTimeSpan(t *testing.T) {
	t.Parallel()
	testSpan := &data.TimeSpan{
		Begin: time.UnixMilli(1000),
		End:   time.UnixMilli(2000),
	}
	testSpan.Duration = testSpan.End.Sub(testSpan.Begin)
	testCompacter(t, testSpan, &data.TimeSpan{})
}

func TestCompactStats(t *testing.T) {
	t.Parallel()
	testStats := &data.Stats{}
	testStats.AddPoint(2 * time.Millisecond)
	testStats.AddPoint(4 * time.Millisecond)
	testStats.AddPoint(7 * time.Millisecond)
	testStats.AddDroppedPacket()
	testCompacter(t, testStats, &data.Stats{})
}

func TestCompactHeader(t *testing.T) {
	t.Parallel()
	testHeader := &data.Header{Stats: &data.Stats{}, TimeSpan: &data.TimeSpan{}}
	// I don't care about bit-for-bit identical output, hence why we use time.UnixMilli which doesn't preserve
	// wall clocks, locations, etc.
	testHeader.AddPoint(ping.PingDataPoint{Duration: 1, Timestamp: time.UnixMilli(1000)})
	testHeader.AddPoint(ping.PingDataPoint{Duration: 2, Timestamp: time.UnixMilli(3000)})
	testHeader.AddPoint(ping.PingDataPoint{DropReason: ping.TestDrop, Timestamp: time.UnixMilli(5000)})
	testCompacter(t, testHeader, &data.Header{})
}

func TestCompactNetwork(t *testing.T) {
	t.Parallel()
	testNetwork := &data.Network{IPs: []net.IP{}}
	testNetwork.AddPoint([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	testNetwork.AddPoint(net.IPv6loopback)
	testNetwork.AddPoint(net.IPv4zero)
	testCompacter(t, testNetwork, &data.Network{})
}

func TestCompactBlock(t *testing.T) {
	t.Parallel()
	testBlock := &data.Block{
		Header: &data.Header{Stats: &data.Stats{}, TimeSpan: &data.TimeSpan{}},
		Raw:    []ping.PingDataPoint{},
	}
	testBlock.AddPoint(ping.PingDataPoint{Duration: 1, Timestamp: time.UnixMilli(1000)})
	testCompacter(t, testBlock, &data.Block{})
}

func TestCompactLargeBlock(t *testing.T) {
	t.Parallel()
	testBlock := &data.Block{
		Header: &data.Header{Stats: &data.Stats{}, TimeSpan: &data.TimeSpan{}},
		Raw:    []ping.PingDataPoint{},
	}
	for _, p := range makeLargePings() {
		testBlock.AddPoint(p.Data)
	}
	testCompacter(t, testBlock, &data.Block{})
}

func makeLargePings() []ping.PingResults {
	pings := make([]ping.PingResults, 1_000_000)
	randomIPs := []net.IP{
		net.IPv6zero,
		net.IPv6unspecified,
		net.IPv6loopback,
		net.IPv6interfacelocalallnodes,
		net.IPv6linklocalallnodes,
		net.IPv6linklocalallrouters,
		net.IPv4bcast,
		net.IPv4allsys,
		net.IPv4allrouter,
		net.IPv4zero,
	}
	for i := range pings {
		pings[i] = ping.PingResults{
			Data: ping.PingDataPoint{
				Duration:  time.Duration(i) * time.Millisecond,
				Timestamp: time.UnixMilli(int64(i) * 1000),
			},
			IP: randomIPs[i%len(randomIPs)],
		}
	}
	return pings
}

func TestCompactDataIndexes(t *testing.T) {
	t.Parallel()
	testDataIndexes := &data.DataIndexes{BlockIndex: 2, RawIndex: 2}
	testCompacter(t, testDataIndexes, &data.DataIndexes{})
}

func TestCompactRun(t *testing.T) {
	t.Parallel()
	testRun := &data.Run{}
	testRun.Inc(0)
	testRun.Inc(1)
	testRun.Reset()
	testRun.Inc(3)
	testCompacter(t, testRun, &data.Run{})
}

func TestCompactRuns(t *testing.T) {
	t.Parallel()
	testRuns := &data.Runs{GoodPackets: &data.Run{}, DroppedPackets: &data.Run{}}
	testRuns.AddPoint(0, ping.PingDataPoint{DropReason: ping.NotDropped})
	testRuns.AddPoint(1, ping.PingDataPoint{DropReason: ping.NotDropped})
	testRuns.AddPoint(2, ping.PingDataPoint{DropReason: ping.TestDrop})
	testRuns.AddPoint(3, ping.PingDataPoint{DropReason: ping.NotDropped})
	testCompacter(t, testRuns, &data.Runs{})
}

func TestCompactEmptyData(t *testing.T) {
	t.Parallel()
	testData := data.NewData("www.google.com")
	testCompacter(t, testData, &data.Data{})
}

func TestCompactData(t *testing.T) {
	t.Parallel()
	testData := data.NewData("www.google.com")
	testData.AddPoint(ping.PingResults{
		Data: ping.PingDataPoint{Duration: 1, Timestamp: time.UnixMilli(1000)},
		IP:   net.IPv4bcast,
	})
	testData.AddPoint(ping.PingResults{
		Data: ping.PingDataPoint{Duration: 2, Timestamp: time.UnixMilli(2000)},
		IP:   net.IPv4bcast,
	})
	testCompacter(t, testData, &data.Data{})
}

func TestCompactLargeData(t *testing.T) {
	t.Parallel()
	testData := data.NewData("www.google.com")
	for _, p := range makeLargePings() {
		testData.AddPoint(p)
	}
	testCompacter(t, testData, &data.Data{})
}

func testCompacter(t *testing.T, start data.Compact, empty data.Compact) {
	t.Helper()
	var b bytes.Buffer
	err := start.AsCompact(&b)
	assert.NilError(t, err)
	_, err = empty.FromCompact(b.Bytes())
	assert.NilError(t, err)
	assert.Assert(t, is.DeepEqual(start, empty, th.AllowAllUnexported), "AsCompact->FromCompact")
}

type FileTest struct {
	FileName        string
	ExpectedSummary string
	tz              *time.Location
}

func (ft FileTest) Run(t *testing.T) {
	t.Parallel()
	f, err := os.OpenFile(ft.FileName, os.O_RDONLY, 0)
	assert.NilError(t, err)
	defer f.Close()
	d, err := data.ReadData(f)
	d = d.In(ft.tz)
	assert.NilError(t, err)
	assert.Equal(t, ft.ExpectedSummary, d.String())
}

var summer = time.FixedZone("+1", 3_600)

//nolint:lll
func TestFiles(t *testing.T) {
	t.Parallel()
	t.Run("Small",
		FileTest{
			FileName:        "testdata/input/small-2-02-08-2024.pings",
			ExpectedSummary: "www.google.com: PingsMeta#1 [172.217.16.228] | 02 Aug 2024 20:01:58.66 -> 20:01:59.665 (1s) | Average μ 8.052048ms | SD σ 122.04µs | Packet Count 2 | Longest Streak 2",
			tz:              summer,
		}.Run,
	)
	t.Run("Medium",
		FileTest{
			FileName:        "testdata/input/medium-395-02-08-2024.pings",
			ExpectedSummary: "www.google.com: PingsMeta#1 [142.250.200.36] | 02 Aug 2024 20:40:41.17 -> 20:47:15.17 (6m34s) | Average μ 8.404893ms | SD σ 970.911µs | Packet Count 395 | Longest Streak 395",
			tz:              summer,
		}.Run,
	)
	t.Run("Medium with drops",
		FileTest{
			FileName:        "testdata/input/medium-309-with-induced-drops-02-08-2024.pings",
			ExpectedSummary: "www.google.com: PingsMeta#1 [142.250.179.228,142.250.200.4] | 02 Aug 2024 21:04:27.56 -> 21:09:51.56 (5m24s) | Average μ 8.564583ms | SD σ 3.25564ms | PacketLoss 2.6% | Packet Count 309 | Longest Streak 92 | Longest Drop Streak 2",
			tz:              summer,
		}.Run,
	)
	t.Run("Medium with minute Gaps",
		FileTest{
			FileName: "testdata/input/medium-minute-gaps.pings",
			// 60 packets but over 21mins
			ExpectedSummary: "www.google.com: PingsMeta#1 [172.217.16.228,216.58.201.100,216.58.204.68] | 03 Aug 2024 00:41:06.65 -> 01:02:28.1 (21m21.449s) | Average μ 8.167942ms | SD σ 180.4µs | Packet Count 67 | Longest Streak 67",
			tz:              summer,
		}.Run,
	)
	t.Run("Medium with hour Gaps",
		FileTest{
			FileName:        "testdata/input/medium-hour-gaps.pings",
			ExpectedSummary: "www.google.com: PingsMeta#1 [142.250.179.228,172.217.16.228,216.58.201.100,216.58.204.68] | 03 Aug 2024 00:41:06.65 -> 10:55:06.59 (10h13m59.94s) | Average μ 8.207511ms | SD σ 1.398049ms | Packet Count 234 | Longest Streak 234",
			tz:              summer,
		}.Run,
	)
}
