// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package terminal_test

import (
	"testing"

	"github.com/Lexer747/AcciPing/graph/terminal"

	"github.com/stretchr/testify/require"
)

func TestGetCurrentTerminalSize(t *testing.T) {
	t.Parallel()
	_, err := terminal.GetCurrentTerminalSize()
	require.Error(t, err)
}
