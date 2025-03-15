// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package th

import (
	"os"
	"testing"

	"github.com/Lexer747/acci-ping/graph/data"
	"gotest.tools/v3/assert"
)

func GetFromFile(t *testing.T, FileName string) *data.Data {
	t.Helper()
	f, err := os.OpenFile(FileName, os.O_RDONLY, 0)
	assert.NilError(t, err)
	defer f.Close()
	d, err := data.ReadData(f)
	assert.NilError(t, err)
	return d
}
