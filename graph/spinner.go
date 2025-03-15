// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import (
	"time"

	"github.com/Lexer747/acci-ping/graph/terminal"
	"github.com/Lexer747/acci-ping/graph/terminal/ansi"
	"github.com/Lexer747/acci-ping/graph/terminal/typography"
)

var spinnerArray = [...]string{
	typography.UpperLeftQuadrantCircularArc,
	typography.UpperRightQuadrantCircularArc,
	typography.LowerRightQuadrantCircularArc,
	typography.LowerLeftQuadrantCircularArc,
}

func spinner(s terminal.Size, i int, timeBetweenFrames time.Duration) string {
	// TODO refactor into a generic only paint me every X fps.
	// We want 175ms between spinner updates
	a := i
	x := timeBetweenFrames.Milliseconds()
	if x != 0 && int(175/x) != 0 {
		a = i / int(175/x)
	}
	return ansi.CursorPosition(1, s.Width-3) + ansi.Cyan(spinnerArray[a%len(spinnerArray)])
}
