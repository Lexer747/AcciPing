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

func main() {
	p := ping.NewPing()
	ctx, cancelFunc := context.WithCancel(context.Background())
	channel, err := p.CreateChannel(ctx, "www.google.com", 30)
	if err != nil {
		panic(err)
	}
	for range 4 {
		result := <-channel
		fmt.Printf("Duration: %s | Timestamp %s | Err: '%+v'\n", result.Duration, result.Timestamp.String(), result.Error)
	}
	cancelFunc()
}
