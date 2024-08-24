// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package th

import (
	"os"
	"testing"

	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/stretchr/testify/require"
)

func GetFromFile(t *testing.T, FileName string) *data.Data {
	t.Helper()
	f, err := os.OpenFile(FileName, os.O_RDONLY, 0)
	require.NoError(t, err)
	defer f.Close()
	d, err := data.ReadData(f)
	require.NoError(t, err)
	return d
}
