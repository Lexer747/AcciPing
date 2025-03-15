// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package timeutils_test

import (
	"testing"

	"github.com/Lexer747/acci-ping/utils/timeutils"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestHumanString(t *testing.T) {
	t.Parallel()
	assert.Check(t, is.Equal("10ns", timeutils.HumanString(19, 1)))
	assert.Check(t, is.Equal("19ns", timeutils.HumanString(19, 3)))
	assert.Check(t, is.Equal("123ns", timeutils.HumanString(123, 3)))
	assert.Check(t, is.Equal("123µs", timeutils.HumanString(123456, 3)))
	assert.Check(t, is.Equal("123ms", timeutils.HumanString(123456789, 3)))
	assert.Check(t, is.Equal("3h25m0s", timeutils.HumanString(12345678900000, 3)))
	// Doesn't do any minute/hour based rounding
	assert.Check(t, is.Equal("34166h40m0s", timeutils.HumanString(123456789123456789, 3)))
}

func TestBug(t *testing.T) {
	t.Parallel()
	// Previous versions of this code would result in "129.399999ms"
	assert.Check(t, is.Equal("129.3ms", timeutils.HumanString(129379939, 4)))
}
