// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package ping_test

import (
	"context"
	"testing"
	"time"

	"github.com/Lexer747/AcciPing/ping"

	"github.com/stretchr/testify/require"
)

func TestOneShot_google_com(t *testing.T) {
	t.Parallel()
	p := ping.NewPing()
	duration, err := p.OneShot("www.google.com")
	require.NoError(t, err)
	require.GreaterOrEqual(t, duration, time.Millisecond)
}

func TestChannel_google_com(t *testing.T) {
	t.Parallel()
	p := ping.NewPing()
	ctx, cancelFunc := context.WithCancel(context.Background())
	_, err := p.CreateChannel(ctx, "www.google.com", -1, 0)
	require.Error(t, err, "invalid pings per minute")
	channel, err := p.CreateChannel(ctx, "www.google.com", 0, 0)
	require.NoError(t, err)
	for range 2 {
		result := <-channel
		require.GreaterOrEqual(t, result.Data.Duration, time.Millisecond)
	}
	cancelFunc()
}

func TestUint16Wrapping(t *testing.T) {
	t.Parallel()
	var i uint16 = 1
	for i != 0 {
		i++
	}
	require.Equal(t, uint16(1), i+1)
}
