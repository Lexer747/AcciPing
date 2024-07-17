// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package ping

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/Lexer747/AcciPing/utils/errors"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type ping struct {
	connect *icmp.PacketConn
	id      uint16
}

func NewPing() *ping {
	return &ping{
		id: uint16(os.Getpid() + 1234),
	}
}

func (p *ping) OneShot(url string) (time.Duration, error) {
	// first get the ip for a given url
	selectedIp, err := p.DNSQuery(url)
	if err != nil {
		return 0, err
	}

	// Create a listener for the IP we will use
	closer, err := p.startListening(selectedIp, url)
	defer closer()
	if err != nil {
		return 0, err
	}

	raw, err := p.makeOutgoingPacket(1)
	if err != nil {
		return 0, errors.Wrapf(err, "couldn't create outgoing %q packet", url)
	}

	// Actually write the echo request onto the connection:
	if err = p.writeEcho(selectedIp, raw, url); err != nil {
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

func (ping) DNSQuery(url string) (net.IP, error) {
	ips, err := net.LookupIP(url)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't DNS query %q", url)
	}

	var selectedIp net.IP
	for _, ip := range ips {
		if isIpv4(ip) {
			selectedIp = ip
			break
		}
	}
	if selectedIp == nil {
		return nil, errors.Errorf("Couldn't resolve %q to valid IPv4 address, ipv6 addresses are not supported", url)
	}
	return selectedIp, nil
}

type PingResults struct {
	Duration time.Duration
	Error    error
}

func (p *ping) CreateChannel(ctx context.Context, url string) (chan PingResults, error) {
	client := make(chan PingResults)

	// first get the ip for a given url
	selectedIp, err := p.DNSQuery(url)
	if err != nil {
		return nil, err
	}

	// Create a listener for the IP we will use
	closer, err := p.startListening(selectedIp, url)
	if err != nil {
		return nil, err
	}

	run := func() {
		defer close(client)
		defer closer()
		var seq uint16 = 0
		buffer := make([]byte, 255)
		for {
			raw, err := p.makeOutgoingPacket(seq)
			if err != nil {
				client <- PingResults{Error: err}
				return
			}
			// Actually write the echo request onto the connection:
			if err = p.writeEcho(selectedIp, raw, url); err != nil {
				client <- PingResults{Error: err}
				return
			}
			begin := time.Now()
			n, _, err := p.connect.ReadFrom(buffer) // blocking
			duration := time.Since(begin)
			if err != nil {
				client <- PingResults{
					Duration: duration,
					Error:    errors.Wrapf(err, "couldn't read packet from %q", url),
				}
				return
			}
			received, err := icmp.ParseMessage(protocolICMP, buffer[:n])
			if err != nil {
				client <- PingResults{
					Duration: duration,
					Error:    errors.Wrapf(err, "couldn't parse raw packet from %q, %+v", url, received),
				}
				return
			}
			switch received.Type {
			case ipv4.ICMPTypeEchoReply: // Happy path
				// Clear the buffer for next packet
				for i := range n {
					buffer[i] = 0
				}
				seq++ // Todo wrap around?
			default:
				client <- PingResults{
					Duration: duration,
					Error:    errors.Errorf("Didn't receive a good message back from %q, got Code: %d", url, received.Code),
				}
				return
			}

			select {
			case <-ctx.Done():
				client <- PingResults{Error: ctx.Err()}
				return
			case client <- PingResults{Duration: duration}:
			}
		}
	}
	go run()
	return client, nil
}

func (p *ping) makeOutgoingPacket(seq uint16) ([]byte, error) {
	outGoingPacket := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Body: &icmp.Echo{
			ID:   int(p.id), // This identifier is purely to help distinguish other ongoing echos since we are listening on the broad cast. its a u16
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

func (p *ping) writeEcho(selectedIp net.IP, raw []byte, url string) error {
	udpDst := &net.UDPAddr{IP: selectedIp}
	if _, err := p.connect.WriteTo(raw, udpDst); err != nil {
		return errors.Wrapf(err, "couldn't write packet to connection %q", url)
	}
	return nil
}

func (p *ping) startListening(selectedIp net.IP, url string) (closer func(), err error) {
	// TODO supporting windows (privileges etc)
	p.connect, err = icmp.ListenPacket("udp4", listenAddr.String())
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't listen to %q found from %q", selectedIp.String(), url)
	}
	return func() { p.connect.Close() }, nil
}

func isIpv4(ip net.IP) bool {
	const IPv4len = 4
	const IPv6len = 16
	isZeros := func(p net.IP) bool {
		for i := 0; i < len(p); i++ {
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
