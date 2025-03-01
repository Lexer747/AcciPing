// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package ping

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"time"

	"github.com/Lexer747/AcciPing/utils/bytes"
	"github.com/Lexer747/AcciPing/utils/errors"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Ping struct {
	connect    *icmp.PacketConn
	id         uint16
	currentURL string
	timeout    time.Duration

	dnsCacheTrust uint
	addresses     *queryCache
}

type DNSCacheTrust string

const (
	LowTrust     = "Low Trust"
	NominalTrust = "Nominal Trust"
	HighTrust    = "High Trust"
)

func (p *Ping) LastIP() string {
	if p.addresses == nil {
		return "<IP NOT YET FOUND>"
	}
	return p.addresses.GetLastIP()
}

func NewPing() *Ping {
	return &Ping{
		//nolint:gosec
		// G115 overflow is expected and required
		id: uint16(os.Getpid() + 1234),
	}
}

func NewPingWithTrust(trust DNSCacheTrust) *Ping {
	return &Ping{
		//nolint:gosec
		// G115 overflow is expected and required
		id:            uint16(os.Getpid() + 1234),
		dnsCacheTrust: trust.asMaxDropped(),
	}
}

func (p *Ping) OneShot(url string) (time.Duration, error) {
	// first get the ip for a given url
	cache, err := IPv4DNSQuery(url, p.dnsCacheTrust)
	if err != nil {
		return 0, err
	}
	// Don't handle this [!ok] case in OneShot
	selectedIP, _ := cache.Get()

	// Create a listener for the IP we will use
	closer, err := p.startListening(url)
	defer closer()
	if err != nil {
		return 0, err
	}

	raw, err := p.makeOutgoingPacket(1)
	if err != nil {
		return 0, errors.Wrapf(err, "couldn't create outgoing %q packet", url)
	}

	// Actually write the echo request onto the connection:
	if err = p.writeEcho(selectedIP, raw); err != nil {
		return 0, err
	}
	begin := time.Now()

	// Now wait for the result
	buffer := make([]byte, 255)
	timeoutCtx, _ := context.WithTimeoutCause(context.Background(), time.Second, pingTimeout{Duration: 100 * time.Millisecond})
	n, err := p.pingRead(timeoutCtx, buffer)
	duration := time.Since(begin)
	if err != nil {
		return duration, errors.Wrapf(err, "couldn't read packet from %q", url)
	}
	received, err := icmp.ParseMessage(protocolICMP, buffer[:n])
	if err != nil {
		return duration, errors.Wrapf(err, "couldn't parse raw packet from %q, %+v", url, received)
	}
	switch received.Type {
	case ipv4.ICMPTypeEchoReply:
		return duration, nil
	default:
		return duration, errors.Errorf("Didn't receive a good message back from %q, got Code: %d", url, received.Code)
	}
}

type PingResults struct {
	Data        PingDataPoint
	IP          net.IP
	InternalErr error
}

type PingDataPoint struct {
	Duration   time.Duration
	Timestamp  time.Time
	DropReason Dropped
}

type Dropped byte

const (
	NotDropped Dropped = iota
	Timeout
	DNSFailure
	BadResponse

	TestDrop = 0xfe
)

func (p PingResults) String() string {
	if p.IP == nil && p.InternalErr != nil {
		return "Internal Error " + timestampString(p.Data) + " reason " + p.InternalErr.Error()
	} else {
		return p.IP.String() + " | " + p.Data.String()
	}
}

func (d Dropped) String() string {
	switch d {
	case BadResponse:
		return "Bad Response"
	case Timeout:
		return "Timeout"
	case DNSFailure:
		return "DNS Query Failed"
	case TestDrop:
		return "Testing A Dropped Packet :)"

	case NotDropped:
		fallthrough
	default:
		return ""
	}
}

func (p PingDataPoint) String() string {
	if p.Good() {
		return fmt.Sprintf("%s | %s", timestampString(p), p.Duration.String())
	}
	return fmt.Sprintf("%s | DROPPED, reason %q", timestampString(p), p.DropReason.String())
}

func timestampString(p PingDataPoint) string {
	return p.Timestamp.Format(time.RFC3339Nano)
}

func (p PingDataPoint) Dropped() bool {
	return p.DropReason != NotDropped
}
func (p PingDataPoint) Good() bool {
	return p.DropReason == NotDropped
}
func (p PingDataPoint) Equal(other PingDataPoint) bool {
	return p.Duration == other.Duration && p.Timestamp.Equal(other.Timestamp) && p.DropReason == other.DropReason
}

func (p *Ping) CreateChannel(ctx context.Context, url string, pingsPerMinute float64, channelSize int) (chan PingResults, error) {
	if pingsPerMinute < 0 {
		return nil, errors.Errorf("Invalid pings per minute %f, should be larger than 0", pingsPerMinute)
	}

	// Create a listener for the IP we will use
	closer, err := p.startListening(url)
	if err != nil {
		return nil, err
	}

	// Block the main thread to init this for the first time (most consumers will want to have a [GetLastIP]
	// value as soon as this method returns), if we get an error let the main loop do the retying.
	p.addresses, _ = IPv4DNSQuery(url, p.dnsCacheTrust)

	rateLimit := p.buildRateLimiting(pingsPerMinute)

	client := make(chan PingResults, channelSize)
	p.startChannel(ctx, client, closer, url, rateLimit)
	return client, nil
}

func (p *Ping) startChannel(ctx context.Context, client chan PingResults, closer func(), url string, rateLimit *time.Ticker) {
	run := func() {
		defer close(client)
		defer closer()
		var seq uint16
		buffer := make([]byte, 255)
		var errorDuringLoop bool
		for {
			timestamp := time.Now()

			ip, newCloser := p.dnsRetry(url, client, timestamp, rateLimit, closer)
			if newCloser != nil {
				defer newCloser()
				closer = newCloser
				// Reset the timestamp, we were stuck in DNS for too long
				timestamp = time.Now()
			}

			if seq, errorDuringLoop = p.pingOnChannel(ctx, timestamp, ip, seq, client, buffer); errorDuringLoop {
				// Keep track of this address as maybe being unreliable
				p.addresses.Dropped(ip)
			}
			select {
			case <-ctx.Done():
				return
			default:
				if rateLimit != nil {
					// This throttles us if required, it will also drop ticks if we are pinging something very slow
					<-rateLimit.C
				}
			}
		}
	}
	go run()
}

func (p *Ping) dnsRetry(url string, client chan PingResults, timestamp time.Time, rateLimit *time.Ticker, closer func()) (net.IP, func()) {
	var err error
	var newCloser func()
HARD_RETRY:
	if p.addresses == nil {
		// Keeping doing a DNS query until we get a valid result, count each failure as a dropped packet
		for p.addresses == nil {
			// start again, do a new DNS query
			p.addresses, err = IPv4DNSQuery(url, p.dnsCacheTrust)
			if err != nil {
				client <- packetLoss(nil, timestamp, DNSFailure)
				<-rateLimit.C
				timestamp = time.Now()
			}
		}
		// Reset our listening, it's a chance our NIC died in which case we need to restart this.
		// I don't think we can tell that the inner listener died.
		closer()
		for {
			newCloser, err = p.startListening(url)
			if err == nil {
				break
			}
		}
	}
	ip, ok := p.addresses.Get()
	if !ok {
		p.addresses = nil
		goto HARD_RETRY // Avoid recursion, if we made it here either we have a fresh restart the entire address pool is exhausted
	}
	return ip, newCloser
}

func (p *Ping) buildRateLimiting(pingsPerMinute float64) *time.Ticker {
	p.timeout = time.Second
	var rateLimit *time.Ticker
	// Zero is the sentinel, go as fast as possible
	if pingsPerMinute != 0 {
		maxPingDuration := PingsPerMinuteToDuration(pingsPerMinute)
		rateLimit = time.NewTicker(maxPingDuration)
		p.timeout = max(min(p.timeout, maxPingDuration), 500*time.Millisecond)
	}
	return rateLimit
}

func PingsPerMinuteToDuration(pingsPerMinute float64) time.Duration {
	if pingsPerMinute == 0 {
		return 0
	}
	gapBetweenPings := math.Round((60 * 1000) / (pingsPerMinute))
	return time.Millisecond * time.Duration(gapBetweenPings)
}

func internalErr(IP net.IP, Timestamp time.Time, err error) PingResults {
	return PingResults{
		Data:        PingDataPoint{Timestamp: Timestamp},
		IP:          IP,
		InternalErr: err,
	}
}

func packetLoss(IP net.IP, Timestamp time.Time, Reason Dropped) PingResults {
	return PingResults{
		Data: PingDataPoint{
			Timestamp:  Timestamp,
			DropReason: Reason,
		},
		IP: IP,
	}
}

func goodPacket(IP net.IP, Duration time.Duration, Timestamp time.Time) PingResults {
	return PingResults{
		Data: PingDataPoint{
			Duration:   Duration,
			Timestamp:  Timestamp,
			DropReason: NotDropped,
		},
		IP: IP,
	}
}

// pingOnChannel performs a single ping to the already discovered IP, using the buffer as a scratch buffer,
// and writes ALL results to the channel (including errors). It self limits it's execution if it was called
// too recently compared to the desired rate.
func (p *Ping) pingOnChannel(
	ctx context.Context,
	timestamp time.Time,
	selectedIP net.IP,
	seq uint16,
	client chan PingResults,
	buffer []byte,
) (uint16, bool) {
	// Can gain some speed here by not remaking this each time, only to change the sequence number.
	raw, err := p.makeOutgoingPacket(seq)
	if err != nil {
		client <- internalErr(selectedIP, timestamp, err)
		return seq, true
	}

	// Actually write the echo request onto the connection:
	if err = p.writeEcho(selectedIP, raw); err != nil {
		client <- internalErr(selectedIP, timestamp, err)
		return seq, true
	}
	begin := time.Now()
	timeout := pingTimeout{Duration: p.timeout}
	timeoutCtx, _ := context.WithTimeoutCause(ctx, p.timeout, timeout)
	n, err := p.pingRead(timeoutCtx, buffer)
	duration := time.Since(begin)
	if err != nil && errors.Is(err, timeout) {
		client <- packetLoss(selectedIP, timestamp, Timeout)
		return seq, true
	} else if err != nil {
		client <- internalErr(selectedIP, timestamp, errors.Wrapf(err, "couldn't read packet from %q", p.currentURL))
		return seq, true
	}
	received, err := icmp.ParseMessage(protocolICMP, buffer[:n])
	if err != nil {
		client <- internalErr(selectedIP, timestamp, errors.Wrapf(err, "couldn't parse raw packet from %q, %+v", p.currentURL, received))
		return seq, true
	}
	switch received.Type {
	case ipv4.ICMPTypeEchoReply:
		// Clear the buffer for next packet
		bytes.Clear(buffer, n)
		seq++ // Deliberate wrap-around
		client <- goodPacket(selectedIP, duration, timestamp)
		return seq, false
	default:
		client <- packetLoss(selectedIP, timestamp, BadResponse)
		return seq, true
	}
}

type pingTimeout struct {
	time.Duration
}

func (pt pingTimeout) Error() string { return "PingTimeout {" + pt.String() + "}" }

func (p *Ping) pingRead(ctx context.Context, buffer []byte) (int, error) {
	type read struct {
		n   int
		err error
	}
	c := make(chan read)
	go func() {
		n, _, err := p.connect.ReadFrom(buffer)
		c <- read{n: n, err: err}
	}()
	select {
	case <-ctx.Done():
		err := context.Cause(ctx)
		return 0, err
	case success := <-c:
		return success.n, success.err
	}
}

func (p *Ping) makeOutgoingPacket(seq uint16) ([]byte, error) {
	outGoingPacket := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Body: &icmp.Echo{
			// This identifier is purely to help distinguish other ongoing echos since we are listening on the
			// broad cast. Its a u16 in the spec, as is Seq.
			ID:   int(p.id),
			Seq:  int(seq),
			Data: []byte("#"),
		},
	}
	raw, err := outGoingPacket.Marshal(nil)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (p *Ping) writeEcho(selectedIP net.IP, raw []byte) error {
	udpDst := &net.UDPAddr{IP: selectedIP}
	if _, err := p.connect.WriteTo(raw, udpDst); err != nil {
		return errors.Wrapf(err, "couldn't write packet to connection %q", p.currentURL)
	}
	return nil
}

func (p *Ping) startListening(url string) (closer func(), err error) {
	// TODO supporting windows (privileges etc)
	p.connect, err = icmp.ListenPacket("udp4", listenAddr.String())
	p.currentURL = url
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't listen")
	}
	return func() {
		p.connect.Close()
		p.currentURL = ""
	}, nil
}

func isIpv4(ip net.IP) bool {
	const IPv4len = 4
	const IPv6len = 16
	isZeros := func(p net.IP) bool {
		for i := range p {
			if p[i] != 0 {
				return false
			}
		}
		return true
	}
	if len(ip) == IPv4len {
		return true
	}
	if len(ip) == IPv6len &&
		isZeros(ip[0:10]) &&
		ip[10] == 0xff &&
		ip[11] == 0xff {
		return true
	}
	return false
}

var listenAddr = net.IPv4zero

func (dct DNSCacheTrust) asMaxDropped() uint {
	switch dct {
	case LowTrust:
		return 0
	case NominalTrust:
		return 1
	case HighTrust:
		return 5
	}
	panic("exhaustive:enforce")
}
