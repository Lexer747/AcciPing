// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package timeutils_test

import (
	"testing"

	"github.com/Lexer747/AcciPing/utils/timeutils"
	"github.com/stretchr/testify/assert"
)

func TestHumanString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "20ns", timeutils.HumanString(19, 1))
	assert.Equal(t, "123ns", timeutils.HumanString(123, 3))
	assert.Equal(t, "123µs", timeutils.HumanString(123456, 3))
	assert.Equal(t, "123ms", timeutils.HumanString(123456789, 3))
	assert.Equal(t, "3h25m0s", timeutils.HumanString(12345678900000, 3))
	// Doesn't do any 60 based rounding
	assert.Equal(t, "3416h40m0s", timeutils.HumanString(12345678900000000, 3))
}
