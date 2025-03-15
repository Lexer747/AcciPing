// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package ping

import (
	"net"
	"slices"
	"sync"

	"github.com/Lexer747/acci-ping/utils/check"
	"github.com/Lexer747/acci-ping/utils/errors"
	"github.com/Lexer747/acci-ping/utils/sliceutils"
)

// queryCache provides an interface for Ping to consume in which we respect the wishes of the servers we are
// causing load on, if they provide more than one address we should pick one at "random". Given we will re-use
// addresses from an original query we do the easier job of just round-robin.
//
// Thread safe.
type queryCache struct {
	m        *sync.Mutex
	store    []queryCacheItem
	index    int
	maxDrops uint
}

// GetLastIP will return the last IP address this cache used, formatted according to [net.IP.String].
func (q *queryCache) GetLastIP() string {
	q.m.Lock()
	defer q.m.Unlock()
	return q.store[q.index].ip.String()
}

// Get will return an IP for use which is not considered stale and true. If the cache is exhausted an all IPs
// are stale then it will return nil and false.
func (q *queryCache) Get() (net.IP, bool) {
	q.m.Lock()
	defer q.m.Unlock()
	// If there's only one IP to pick from then we can do a more simple lookup.
	if len(q.store) == 1 {
		if !q.store[0].stale {
			return q.store[0].ip, true
		}
		return nil, false
	}
	// We must iterate the cache, returning the first IP which isn't stale.
	for start := q.index; start != q.index; q.advance() {
		r := q.store[q.index]
		if !r.stale {
			return r.ip, true
		}
	}
	// No non-stale IPs found
	return nil, false
}

// Dropped tells this cache that the passed IP dropped a packet. Once enough drops have occurred for a given
// IP in the cache then the cache will consider that IP stale. Panic's if the IP isn't in the cache.
func (q *queryCache) Dropped(IP net.IP) {
	q.m.Lock()
	defer q.m.Unlock()
	// We could keep the cache sorted and use binary searches, but for now we consider this a cold path and so
	// do not optimise for it.
	index := slices.IndexFunc(q.store, func(q queryCacheItem) bool {
		return q.ip.Equal(IP)
	})
	check.Check(index != -1, "Unknown IP")

	// Now perform the update
	cur := q.store[index]
	stale := cur.dropCount > q.maxDrops
	q.store[q.index] = queryCacheItem{
		ip:        cur.ip,
		stale:     stale,
		dropCount: cur.dropCount + 1,
	}
}

func (q *queryCache) advance() {
	q.index = (q.index + 1) % len(q.store)
}

type queryCacheItem struct {
	ip        net.IP
	stale     bool
	dropCount uint
}

// IPv4DNSQuery builds a new [ping.queryCache] for a given URL. If no IPv4 addresses are found then an error
// is returned. The max drops specifies to the cache how many dropped packets an address is allowed before we
// consider that address too un-reliable, services may rotate their addresses in which case this cache will
// clear itself of these now defunct addresses. If maxDrops is 0, then only a single dropped packet will mean
// the address is considered stale.
func IPv4DNSQuery(url string, maxDrops uint) (*queryCache, error) {
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
		m:        &sync.Mutex{},
		store:    cache,
		maxDrops: maxDrops,
	}, nil
}
