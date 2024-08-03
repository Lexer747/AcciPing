// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"fmt"

	"github.com/Lexer747/AcciPing/ping"
)

// A very basic demo and use of the library, pings google.com 4 times.
func main() {
	p := ping.NewPing()
	ctx, cancelFunc := context.WithCancel(context.Background())
	const google = "www.google.com"
	channel, err := p.CreateChannel(ctx, google, 45, 0)
	if err != nil {
		panic(err)
	}
	const count = 4
	fmt.Printf("Pinging to %q (%d times) at %q\n", google, count, p.LastIP())
	for range count {
		result := <-channel
		fmt.Println(result.String())
	}
	cancelFunc()
}
