// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// # Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only
package ping_test

import (
	"testing"
	"time"

	"github.com/Lexer747/AcciPing/ping"

	"github.com/stretchr/testify/require"
)

func TestOneShot_google_com(t *testing.T) {
	p := ping.NewPing()
	duration, err := p.OneShot("www.google.com")
	require.NoError(t, err)
	require.GreaterOrEqual(t, duration, time.Millisecond)
}
