// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

// th stands for "test helper"
package th

import (
	"testing"

	"github.com/Lexer747/AcciPing/utils/numeric"
	"github.com/stretchr/testify/assert"
)

func AssertFloatEqual(t *testing.T, expected float64, actual float64, sigFigs int, msgAndArgs ...interface{}) {
	t.Helper()
	a := numeric.RoundToNearestSigFig(actual, sigFigs)
	e := numeric.RoundToNearestSigFig(expected, sigFigs)
	assert.Equal(t, e, a, msgAndArgs...) //nolint:testifylint
}
