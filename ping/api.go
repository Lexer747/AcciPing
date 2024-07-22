// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package ping

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"sync"
	"time"

	"github.com/Lexer747/AcciPing/utils/bytes"
	"github.com/Lexer747/AcciPing/utils/errors"
	"github.com/Lexer747/AcciPing/utils/sliceutils"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Ping struct {
	connect    *icmp.PacketConn
	id         uint16
	currentURL string
	timeout    time.Duration

	addresses *queryCache
}

func (p *Ping) LastIP() string {
	if p.addresses == nil {
		return "<IP NOT YET FOUND>"
	}
	return p.addresses.GetLastIP()
}

func NewPing() *Ping {
	return &Ping{
		id: uint16(os.Getpid() + 1234),
	}
}

func (p *Ping) OneShot(url string) (time.Duration, error) {
	// first get the ip for a given url
	cache, err := IPv4DNSQuery(url)
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
	timeoutCtx, _ := context.WithTimeoutCause(context.Background(), time.Second, pingTimeout{Duration: time.Second})
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
	Duration  time.Duration
	Timestamp time.Time
	err       error
}

func (p PingResults) String() string {
	const f = "15:04:05.99"
	if !p.Dropped() {
		return fmt.Sprintf("%s: %s", p.Timestamp.Format(f), p.Duration.String())
	}
	return fmt.Sprintf("%s: DROPPED, reason %q", p.Timestamp.Format(f), p.Error())
}

func (p PingResults) Error() string {
	return p.err.Error()
}
func (p PingResults) Dropped() bool {
	return p.err != nil
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
	p.addresses, _ = IPv4DNSQuery(url)

	p.timeout = time.Second
	var rateLimit *time.Ticker
	if pingsPerMinute != 0 { // Zero is the sentinel, go as fast as possible
		maxPingDuration := PingsPerMinuteToDuration(pingsPerMinute)
		rateLimit = time.NewTicker(maxPingDuration)
		p.timeout = max(min(p.timeout, maxPingDuration), 500*time.Millisecond)
	}

	client := make(chan PingResults, channelSize)
	run := func() {
		defer close(client)
		defer closer()
		var seq uint16
		buffer := make([]byte, 255)
		var errorDuringLoop bool
		for {
			timestamp := time.Now()
			if p.addresses == nil {
				p.addresses, err = IPv4DNSQuery(url)
				if err != nil {
					client <- packetLoss(timestamp, err)
					<-rateLimit.C
					continue // Try again
				}
				// Reset our listening, it's a chance our NIC died in which case we need to restart this.
				closer()
				for {
					closer, err := p.startListening(url)
					if err != nil {
						continue
					}
					defer closer()
					break
				}
			}
			ip, ok := p.addresses.Get()
			if !ok {
				p.addresses = nil // start again, do a new DNS query
				continue
			}

			if seq, errorDuringLoop = p.pingOnChannel(ctx, timestamp, ip, seq, client, buffer); errorDuringLoop {
				// Keep track of this address as maybe being unreliable
				p.addresses.Dropped()
			}
			select {
			case <-ctx.Done():
				return
			default:
				if rateLimit != nil {
					// This throttles us if required, it will also drop ticks if we are pinging something very slow
					<-rateLimit.C
				} else {
					// Don't block on the receiving channel we just get new pings as fast as possible!
				}
			}
		}
	}
	go run()
	return client, nil
}

func PingsPerMinuteToDuration(pingsPerMinute float64) time.Duration {
	gapBetweenPings := math.Round((60 * 1000) / (pingsPerMinute))
	return time.Millisecond * time.Duration(gapBetweenPings)
}

// queryCache provides an interface for Ping to consume in which we respect the wishes of the servers we are
// causing load on, if they provide more than one address we should pick one at "random". Given we will re-use
// addresses from an original query we do the easier job of just round-robin.
type queryCache struct {
	m        *sync.Mutex
	store    []queryCacheItem
	index    int
	maxDrops int
}

func (q *queryCache) GetLastIP() string {
	q.m.Lock()
	defer q.m.Unlock()
	return q.store[q.index].ip.String()
}

func (q *queryCache) Get() (net.IP, bool) {
	q.m.Lock()
	defer q.m.Unlock()
	if len(q.store) == 1 {
		if !q.store[0].stale {
			return q.store[0].ip, true
		}
		return nil, false
	}
	for start := q.index; start != q.index; q.advance() {
		r := q.store[q.index]
		if !r.stale {
			return r.ip, true
		}
	}
	return nil, false
}

func (q *queryCache) Dropped() {
	q.m.Lock()
	defer q.m.Unlock()
	cur := q.store[q.index]
	stale := cur.dropCount >= q.maxDrops
	q.store[q.index] = queryCacheItem{
		ip:        cur.ip,
		stale:     stale,
		dropCount: cur.dropCount + 1,
	}
	q.advance()
}

func (q *queryCache) advance() {
	q.index = (q.index + 1) % len(q.store)
}

type queryCacheItem struct {
	ip        net.IP
	stale     bool
	dropCount int
}

func IPv4DNSQuery(url string) (*queryCache, error) {
	ips, err := net.LookupIP(url)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't DNS query %q", url)
	}
	if len(ips) == 0 {
		return nil, errors.Errorf("Couldn't resolve %q to any address. Network down?", url)
	}

	results := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		if isIpv4(ip) {
			results = append(results, ip)
			break
		}
	}
	if len(results) == 0 {
		return nil, errors.Errorf("Couldn't resolve %q to valid IPv4 address, ipv6 addresses are not supported", url)
	}

	cache := sliceutils.Map(results, func(ip net.IP) queryCacheItem { return queryCacheItem{ip: ip} })
	return &queryCache{
		m:     &sync.Mutex{},
		store: cache,
	}, nil
}

func packetLoss(Timestamp time.Time, Error error) PingResults {
	return PingResults{Duration: -314, Timestamp: Timestamp, err: Error}
}

func goodPacket(Duration time.Duration, Timestamp time.Time) PingResults {
	return PingResults{Duration: Duration, Timestamp: Timestamp, err: nil}
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
		client <- packetLoss(timestamp, err)
		return seq, true
	}

	// Actually write the echo request onto the connection:
	if err = p.writeEcho(selectedIP, raw); err != nil {
		client <- packetLoss(timestamp, err)
		return seq, true
	}
	begin := time.Now()
	timeoutCtx, _ := context.WithTimeoutCause(ctx, p.timeout, pingTimeout{Duration: p.timeout})
	n, err := p.pingRead(timeoutCtx, buffer)
	duration := time.Since(begin)
	if err != nil {
		client <- packetLoss(timestamp, errors.Wrapf(err, "couldn't read packet from %q", p.currentURL))
		return seq, true
	}
	received, err := icmp.ParseMessage(protocolICMP, buffer[:n])
	if err != nil {
		client <- packetLoss(timestamp, errors.Wrapf(err, "couldn't parse raw packet from %q, %+v", p.currentURL, received))
		return seq, true
	}
	switch received.Type {
	case ipv4.ICMPTypeEchoReply:
		// Clear the buffer for next packet
		bytes.Clear(buffer, n)
		seq++ // Deliberate wrap-around
		client <- goodPacket(duration, timestamp)
		return seq, false
	default:
		client <- packetLoss(timestamp, errors.Errorf("Didn't receive a good message back from %q, got Code: %d", p.currentURL, received.Code))
		return seq, true
	}
}

type pingTimeout struct {
	time.Duration
}

func (pt pingTimeout) Error() string { return "PingTimeout {" + pt.String() + "}" }

func (p *Ping) pingRead(ctx context.Context, buffer []byte) (n int, err error) {
	c := make(chan struct{})
	go func() {
		n, _, err = p.connect.ReadFrom(buffer)
		c <- struct{}{}
	}()
	select {
	case <-ctx.Done():
		err = context.Cause(ctx)
	case <-c:
	}
	return n, err
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

func NewTestPingResult(err error, timestamp time.Time) PingResults {
	return PingResults{Timestamp: timestamp, err: err}
}
