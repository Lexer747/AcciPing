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
	assert.Equal(t, "10ns", timeutils.HumanString(19, 1))
	assert.Equal(t, "19ns", timeutils.HumanString(19, 3))
	assert.Equal(t, "123ns", timeutils.HumanString(123, 3))
	assert.Equal(t, "123µs", timeutils.HumanString(123456, 3))
	assert.Equal(t, "123ms", timeutils.HumanString(123456789, 3))
	assert.Equal(t, "3h25m0s", timeutils.HumanString(12345678900000, 3))
	// Doesn't do any minute/hour based rounding
	assert.Equal(t, "34166h40m0s", timeutils.HumanString(123456789123456789, 3))
}

func TestBug(t *testing.T) {
	t.Parallel()
	// Previous versions of this code would result in "129.399999ms"
	assert.Equal(t, "129.3ms", timeutils.HumanString(129379939, 4))
}
