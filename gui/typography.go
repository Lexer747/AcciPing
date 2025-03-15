// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package gui

import (
	"bytes"
	"strings"

	"github.com/Lexer747/acci-ping/graph/terminal"
)

type Typography struct {
	ToPrint string
	// TextLen isn't always equal to len(ToPrint) because of unicode characters and ansi control characters
	// hence why it's a separate field.
	TextLen int
	// LenFromToPrint if true will cause the draw call to always overwrite TextLen with len(ToPrint)
	LenFromToPrint bool
	Alignment      Alignment
}

func (t Typography) init(maxTextLength int) iTypography {
	return iTypography{
		Typography:    t,
		maxTextLength: maxTextLength,
	}
}

type iTypography struct {
	Typography
	maxTextLength int
}

func (t iTypography) Draw(size terminal.Size, b *bytes.Buffer) {
	if t.TextLen > t.maxTextLength {
		b.WriteString(t.ToPrint)
		return
	}
	switch t.Alignment {
	case Centre:
		padding := (t.maxTextLength - t.TextLen) / 2
		leftPadding, rightPadding := getLeftRightPadding(padding, padding, t.TextLen, t.maxTextLength)
		b.WriteString(strings.Repeat(" ", leftPadding) + t.ToPrint + strings.Repeat(" ", rightPadding))
	case Left:
		padding := t.maxTextLength - t.TextLen
		b.WriteString(t.ToPrint + strings.Repeat(" ", padding))
	case Right:
		padding := t.maxTextLength - t.TextLen
		b.WriteString(strings.Repeat(" ", padding) + t.ToPrint)
	default:
		panic("unknown Alignment: " + t.Alignment.String())
	}
}

func getLeftRightPadding(leftPadding, rightPadding, cur, maxLen int) (int, int) {
	for leftPadding+rightPadding+cur > maxLen {
		if leftPadding+rightPadding+cur%2 == 0 {
			leftPadding--
		} else {
			rightPadding--
		}
	}
	for leftPadding+rightPadding+cur < maxLen {
		if leftPadding+rightPadding+cur%2 == 0 {
			leftPadding++
		} else {
			rightPadding++
		}
	}
	return leftPadding, rightPadding
}
