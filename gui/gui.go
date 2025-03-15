// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package gui

import (
	"bytes"
	"strconv"

	"github.com/Lexer747/acci-ping/graph/terminal"
)

type GUI interface {
	GetState() Token
	Drawn(t Token)
}

type Token interface {
	ShouldDraw() bool
	ShouldInvalidate() bool
}

// Draw is the high level interface that any GUI component should implement which will draw itself to the byte
// buffer.
type Draw interface {
	Draw(size terminal.Size, b *bytes.Buffer)
}

var _ Draw = (&Box{})
var _ Draw = (&iTypography{})

type Position struct {
	Vertical   Alignment
	Horizontal Alignment
	Padding    Padding
}

type Padding struct {
	Top, Bottom, Left, Right int
}

func (p Padding) Equal(other Padding) bool {
	return p.Top == other.Top &&
		p.Bottom == other.Bottom &&
		p.Left == other.Left &&
		p.Right == other.Right
}

var NoPadding Padding = Padding{}

type Alignment int

const (
	Left   Alignment = 1
	Centre Alignment = 2
	Right  Alignment = 3
)

func (a Alignment) String() string {
	switch a {
	case Left:
		return "Left"
	case Centre:
		return "Centre"
	case Right:
		return "Right"
	default:
		return "Unknown Alignment: " + strconv.Itoa(int(a))
	}
}
