// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package gui

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/Lexer747/acci-ping/graph/terminal"
	"github.com/Lexer747/acci-ping/graph/terminal/ansi"
)

type Box struct {
	// BoxText is the slice of text to show where each element represents a separate line
	BoxText       []Typography
	Position      Position
	Style         Style
	Configuration BoxCfg
}

type Style int

const (
	RoundedCorners Style = 1
	SharpCorners   Style = 2
)

func (s Style) String() string {
	switch s {
	case RoundedCorners:
		return "RoundedCorners"
	case SharpCorners:
		return "SharpCorners"
	default:
		return "Unknown Style: " + strconv.Itoa(int(s))
	}
}

type BoxCfg struct {
	DefaultWidth int
}

func (b Box) Draw(size terminal.Size, buf *bytes.Buffer) {
	p := b.position(size)
	bar := strings.Repeat("─", b.boxTextWidth())
	corners := getCorner(b.Style)
	buf.WriteString(ansi.CursorPosition(p.startY, p.startX) + corners.TopLeft + bar + corners.TopRight)
	// TODO trim error box when more than height
	for i, t := range b.BoxText {
		buf.WriteString(ansi.CursorPosition(p.startY+i+1, p.startX) + "│")
		t.init(b.boxTextWidth()).Draw(size, buf)
		buf.WriteString("|")
	}
	buf.WriteString(ansi.CursorPosition(p.startY+b.height()+1, p.startX) + corners.BottomLeft + bar + corners.BottomRight)
}

type boxPosition struct {
	startY, startX int
}

func (b Box) position(size terminal.Size) boxPosition {
	p := b.Position
	ret := boxPosition{}
	switch {
	case p.Horizontal == Centre && p.Vertical == Centre:
		originX := size.Width / 2
		originY := size.Height / 2
		ret = boxPosition{
			startY: originY - b.height()/2,
			startX: originX - b.width()/2,
		}
	case p.Vertical == Centre && p.Horizontal == Right:
		originY := size.Height / 2
		ret = boxPosition{
			startY: originY - b.height()/2,
			startX: size.Width - b.width(),
		}
	default:
		panic(fmt.Sprintf("unhandled:box:position %+v", p))
	}
	if !p.Padding.Equal(NoPadding) {
		ret.startY = ret.startY - p.Padding.Top + p.Padding.Bottom
		ret.startX = ret.startX - p.Padding.Left + p.Padding.Right
	}
	return ret
}

func (b Box) height() int {
	return len(b.BoxText)
}

func (b Box) width() int {
	return b.boxTextWidth() + b.widthFromStyle()
}

func (b Box) boxTextWidth() int {
	if b.height() == 0 {
		return b.Configuration.DefaultWidth
	}
	ret := 0
	for _, t := range b.BoxText {
		ret = max(ret, t.TextLen)
	}
	return ret
}

func (b Box) widthFromStyle() int {
	switch b.Style {
	case RoundedCorners, SharpCorners:
		return 2
	default:
		panic("unknown box style: " + b.Style.String())
	}
}

type corners struct {
	TopLeft, TopRight, BottomLeft, BottomRight string
}

func getCorner(s Style) corners {
	switch s {
	case RoundedCorners:
		return corners{TopLeft: "╭", TopRight: "╮", BottomLeft: "╰", BottomRight: "╯"}
	case SharpCorners:
		return corners{TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘"}
	default:
		panic("unknown box style: " + s.String())
	}
}
