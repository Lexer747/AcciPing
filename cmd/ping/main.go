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
	channel, err := p.CreateChannel(ctx, google, 30, 0)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Beginning pinging to %q at %q\n", google, p.IPString)
	for range 4 {
		result := <-channel
		if result.Error == nil {
			fmt.Printf("Duration: %s | Timestamp %s\n", result.Duration, result.Timestamp.Format("15:04:05.999"))
		} else {
			if result.Duration != 0 {
				fmt.Printf("Failed '%v' | Duration: %s\n", result.Error, result.Duration)
			} else {
				fmt.Printf("Failed '%v'\n", result.Error)
			}
		}
	}
	cancelFunc()
}
