// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

// th stands for "test helper"
package th

import (
	"reflect"
	"testing"

	"github.com/Lexer747/acci-ping/utils/numeric"
	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func AssertFloatEqual(t *testing.T, expected float64, actual float64, sigFigs int, msgAndArgs ...interface{}) {
	t.Helper()
	a := numeric.RoundToNearestSigFig(actual, sigFigs)
	e := numeric.RoundToNearestSigFig(expected, sigFigs)
	assert.Check(t, is.Equal(e, a), msgAndArgs...)
}

var AllowAllUnexported = cmp.Exporter(func(reflect.Type) bool { return true })
