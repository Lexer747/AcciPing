// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package ping

import (
	"context"
	"flag"
	"fmt"

	"github.com/Lexer747/AcciPing/ping"
)

type Config struct {
	url *string

	*flag.FlagSet
}

func GetFlags() *Config {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	ret := &Config{
		url:     f.String("url", "www.google.com", "the url to target for ping testing"),
		FlagSet: f,
	}
	return ret
}

// A very basic demo and use of the library, pings google.com 4 times.
func RunPing(c *Config) {
	p := ping.NewPing()
	ctx, cancelFunc := context.WithCancel(context.Background())
	channel, err := p.CreateChannel(ctx, *c.url, 45, 0)
	if err != nil {
		panic(err)
	}
	const count = 4
	fmt.Printf("Pinging to %q (%d times) at %q\n", *c.url, count, p.LastIP())
	for range count {
		result := <-channel
		fmt.Println(result.String())
	}
	cancelFunc()
}
