// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package ping

import (
	"context"
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
	IPString   string
}

func NewPing() *Ping {
	return &Ping{
		id: uint16(os.Getpid() + 1234),
	}
}

func (p *Ping) OneShot(url string) (time.Duration, error) {
	// first get the ip for a given url
	selectedIP, err := DNSQuery(url)
	if err != nil {
		return 0, err
	}

	// Create a listener for the IP we will use
	closer, err := p.startListening(selectedIP, url)
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
	n, _, err := p.connect.ReadFrom(buffer) // blocking
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
	Error     error
}

func (p PingResults) String() string {
	const f = "15:04:05.99"
	if p.Error == nil {
		return p.Timestamp.Format(f) + ": " + p.Duration.String()
	}
	return p.Timestamp.Format(f) + ": DROPPED"
}

func (p *Ping) CreateChannel(ctx context.Context, url string, pingsPerMinute float64, channelSize int) (chan PingResults, error) {
	if pingsPerMinute < 0 {
		return nil, errors.Errorf("Invalid pings per minute %f, should be larger than 0", pingsPerMinute)
	}

	// first get the ip for a given url
	selectedIP, err := DNSQuery(url)
	if err != nil {
		return nil, err
	}

	// Create a listener for the IP we will use
	closer, err := p.startListening(selectedIP, url)
	if err != nil {
		return nil, err
	}

	var rateLimit *time.Ticker
	if pingsPerMinute != 0 { // Zero is the sentinel, go as fast as possible
		rateLimit = time.NewTicker(PingsPerMinuteToDuration(pingsPerMinute))
	}

	client := make(chan PingResults, channelSize)
	run := func() {
		defer close(client)
		defer closer()
		var seq uint16
		buffer := make([]byte, 255)
		var errorDuringLoop bool
		for {
			if seq, errorDuringLoop = p.pingOnChannel(rateLimit, seq, client, selectedIP, buffer); errorDuringLoop {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
				// Don't block on the receiving channel we just get new pings as fast as possible!
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

func DNSQuery(url string) (net.IP, error) {
	ips, err := net.LookupIP(url)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't DNS query %q", url)
	}

	var selectedIP net.IP
	for _, ip := range ips {
		if isIpv4(ip) {
			selectedIP = ip
			break
		}
	}
	if selectedIP == nil {
		return nil, errors.Errorf("Couldn't resolve %q to valid IPv4 address, ipv6 addresses are not supported", url)
	}
	return selectedIP, nil
}

func packetLoss(Timestamp time.Time, Error error) PingResults {
	return PingResults{Duration: -314, Timestamp: Timestamp, Error: Error}
}

func goodPacket(Duration time.Duration, Timestamp time.Time) PingResults {
	return PingResults{Duration: Duration, Timestamp: Timestamp, Error: nil}
}

// pingOnChannel performs a single ping to the already discovered IP, using the buffer as a scratch buffer,
// and writes ALL results to the channel (including errors). It self limits it's execution if it was called
// too recently compared to the desired rate.
func (p *Ping) pingOnChannel(
	rateLimit *time.Ticker,
	seq uint16,
	client chan PingResults,
	selectedIP net.IP,
	buffer []byte,
) (uint16, bool) {
	if rateLimit != nil && seq != 0 {
		// This throttles us if required, it will also drop ticks if we are pinging something very slow
		<-rateLimit.C
	}
	// Can gain some speed here by not remaking this each time, only to change the sequence number.
	raw, err := p.makeOutgoingPacket(seq)
	if err != nil {
		client <- packetLoss(time.Now(), err)
		return seq, true
	}

	// Actually write the echo request onto the connection:
	if err = p.writeEcho(selectedIP, raw); err != nil {
		client <- PingResults{Timestamp: time.Now(), Error: err}
		return seq, true
	}
	begin := time.Now()
	n, _, err := p.connect.ReadFrom(buffer) // blocking
	duration := time.Since(begin)
	if err != nil {
		client <- packetLoss(time.Now(), errors.Wrapf(err, "couldn't read packet from %q", p.currentURL))
		return seq, true
	}
	received, err := icmp.ParseMessage(protocolICMP, buffer[:n])
	if err != nil {
		client <- packetLoss(begin, errors.Wrapf(err, "couldn't parse raw packet from %q, %+v", p.currentURL, received))
		return seq, true
	}
	switch received.Type {
	case ipv4.ICMPTypeEchoReply:
		// Clear the buffer for next packet
		bytes.Clear(buffer, n)
		seq++ // Deliberate wrap-around
		client <- goodPacket(duration, begin)
		return seq, false
	default:
		client <- packetLoss(begin, errors.Errorf("Didn't receive a good message back from %q, got Code: %d", p.currentURL, received.Code))
		return seq, true
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

func (p *Ping) startListening(selectedIP net.IP, url string) (closer func(), err error) {
	// TODO supporting windows (privileges etc)
	p.connect, err = icmp.ListenPacket("udp4", listenAddr.String())
	p.IPString = selectedIP.String()
	p.currentURL = url
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't listen to %q found from %q", p.IPString, p.currentURL)
	}
	return func() {
		p.connect.Close()
		p.currentURL = ""
		p.IPString = ""
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
