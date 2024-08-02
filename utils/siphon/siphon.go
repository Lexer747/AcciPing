// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package siphon

import (
	"context"
)

// TeeBufferedChannel, duplicates the channel such that both returned channels receive values from [c], this
// duplication is unsynchronised. Both channels are closed when the [ctx] is done.
func TeeBufferedChannel[T any](ctx context.Context, c chan T, channelSize int) (
	chan T,
	chan T,
) {
	left := make(chan T, channelSize)
	right := make(chan T, channelSize)
	go func() {
		defer close(left)
		defer close(right)
		for {
			select {
			case <-ctx.Done():
			case v := <-c:
				go func() {
					left <- v
				}()
				go func() {
					right <- v
				}()
			}
		}
	}()
	return left, right
}
