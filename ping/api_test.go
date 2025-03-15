// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package ping_test

import (
	"cmp"
	"context"
	"testing"
	"time"

	"github.com/Lexer747/acci-ping/ping"
	"github.com/Lexer747/acci-ping/utils/env"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestOneShot_google_com(t *testing.T) {
	shouldTest(t)
	t.Parallel()
	p := ping.NewPing()
	duration, err := p.OneShot("www.google.com")
	assert.NilError(t, err)
	assert.Assert(t, cmp.Compare(duration, time.Millisecond) >= 0)
}

func TestChannel_google_com(t *testing.T) {
	shouldTest(t)
	t.Parallel()
	p := ping.NewPing()
	ctx, cancelFunc := context.WithCancel(context.Background())
	_, err := p.CreateChannel(ctx, "www.google.com", -1, 0)
	assert.Assert(t, is.ErrorContains(err, ""), "invalid pings per minute")
	channel, err := p.CreateChannel(ctx, "www.google.com", 0, 0)
	assert.NilError(t, err)
	for range 2 {
		result := <-channel
		assert.Assert(t, cmp.Compare(result.Data.Duration, time.Millisecond) >= 0)
	}
	cancelFunc()
}

func TestUint16Wrapping(t *testing.T) {
	shouldTest(t)
	t.Parallel()
	var i uint16 = 1
	for i != 0 {
		i++
	}
	assert.Equal(t, uint16(1), i+1)
}

func shouldTest(t *testing.T) {
	t.Helper()
	if !env.SHOULD_TEST_NETWORK() {
		t.Skip()
	}
}
