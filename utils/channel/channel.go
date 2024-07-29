// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package channel

import (
	"context"
	"sync"
)

// TeeChannel, duplicates the channel such that both returned channels receive values from [c], this
// duplication is unsynchronised. Both channels are closed when the [ctx] is done.
func TeeChannel[T any](ctx context.Context, c chan T) (
	chan T,
	chan T,
) {
	left := make(chan T)
	right := make(chan T)
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

// TeeSyncChannel, duplicates the channel such that both returned channels receive values from [c], this
// duplication is syrnchronised such that either channel is at most 1 value ahead. Both channels are closed
// when the [ctx] is done.
func TeeSyncChannel[T any](ctx context.Context, c chan T) (
	chan T,
	chan T,
) {
	left := make(chan T)
	right := make(chan T)
	go func() {
		for {
			wg := sync.WaitGroup{}
			select {
			case <-ctx.Done():
			case v := <-c:
				wg.Add(2)
				go func() {
					left <- v
					wg.Done()
				}()
				go func() {
					right <- v
					wg.Done()
				}()
				wg.Wait()
			}
		}
	}()
	return left, right
}
