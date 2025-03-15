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

	"github.com/Lexer747/acci-ping/ping"
	"github.com/Lexer747/acci-ping/utils/check"
	"github.com/Lexer747/acci-ping/utils/exit"
)

type Config struct {
	url   *string
	count *int

	*flag.FlagSet
}

func GetFlags() *Config {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	ret := &Config{
		url:     f.String("url", "www.google.com", "the url to target for ping testing"),
		count:   f.Int("n", 4, "the number of packets to send. 0 or smaller means continuous running."),
		FlagSet: f,
	}
	return ret
}

// A very basic demo and use of the library, pings google.com 4 times.
func RunPing(c *Config) {
	check.Check(c.Parsed(), "flags not parsed")
	p := ping.NewPing()
	ctx, cancelFunc := context.WithCancel(context.Background())
	channel, err := p.CreateChannel(ctx, *c.url, 45, 0)
	exit.OnErrorMsg(err, "Couldn't start ping channel")
	if *c.count <= 0 {
		defer cancelFunc()
		fmt.Printf("Pinging to %q continuously at %q\n", *c.url, p.LastIP())
		for {
			result := <-channel
			fmt.Println(result.String())
		}
	} else {
		fmt.Printf("Pinging to %q (%d times) at %q\n", *c.url, *c.count, p.LastIP())
		for range *c.count {
			result := <-channel
			fmt.Println(result.String())
		}
		cancelFunc()
	}
}
