// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import (
	"fmt"

	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/terminal/ansi"
	"github.com/Lexer747/AcciPing/graph/terminal/typography"
	"github.com/Lexer747/AcciPing/ping"
)

// drawWindow is an optimiser and beautifier for the actual points being drawn to a given frame. The
// optimisation is that instead of creating more terminal output by painting over dots we cache all the
// results into this buffer, then draw the cache after iterating all the points.
//
// This also directly enables not painting over labels and the other span based printing choosing which points
// to highlight.
type drawWindow struct {
	cache map[coords]drawnData
	max   int
}

type coords struct {
	x, y int
}

type drawnData struct {
	pingCount int
	isLabel   bool
}

func newDrawWindow(total int64) *drawWindow {
	return &drawWindow{
		cache: make(map[coords]drawnData, int(total)),
	}
}

func (dw *drawWindow) drawPoint(
	p ping.PingDataPoint,
	spanStats, stats *data.Stats,
	x, y, centreX int,
) string {
	leftJustify := x > centreX
	isMin := p.Duration == stats.Min
	isMax := p.Duration == stats.Max
	isMinWithinSpan := p.Duration == spanStats.Min
	isMaxWithinSpan := p.Duration == spanStats.Max
	needsLabel := isMin || isMax || isMinWithinSpan || isMaxWithinSpan
	dw.add(x, y, needsLabel)
	symbol := dw.getOverlap(x, y)
	var colour func(string) string
	if isMinWithinSpan {
		colour = ansi.Green
		symbol = typography.HollowUpTriangle
	}
	if isMaxWithinSpan {
		colour = ansi.Red
		symbol = typography.HollowDownTriangle
	}
	if isMin {
		colour = ansi.Green
		symbol = typography.FilledUpTriangle
	}
	if isMax {
		colour = ansi.Red
		symbol = typography.FilledDownTriangle
	}
	label := ""
	if needsLabel {
		label = p.Duration.String()
		dw.addLabel(x, y, leftJustify, label)
	}
	if leftJustify && needsLabel {
		return ansi.CursorPosition(y, x-len(label)) + colour(label+" "+symbol)
	} else if needsLabel /* rightJustify */ {
		return ansi.CursorPosition(y, x) + colour(symbol+" "+label)
	}
	if dw.shouldSkip(x, y) {
		return ""
	}
	return ansi.CursorPosition(y, x) + symbol
}

func (dw *drawWindow) add(x, y int, label bool) {
	c := coords{x, y}
	if dd, found := dw.cache[c]; found {
		count := dd.pingCount + 1
		dw.cache[c] = drawnData{
			pingCount: count,
			isLabel:   dd.isLabel || label,
		}
		dw.max = max(count, dw.max)
	} else {
		dw.cache[c] = drawnData{
			pingCount: 1,
			isLabel:   label,
		}
	}
}

func (dw *drawWindow) addLabel(x, y int, leftJustify bool, label string) {
	for i := range len(label) + 2 {
		if leftJustify {
			dw.add(x-i+2, y, true)
		} else {
			dw.add(x+i, y, true)
		}
	}
}

func (dw *drawWindow) shouldSkip(x, y int) bool {
	c := coords{x, y}
	if dd, found := dw.cache[c]; found {
		return dd.isLabel
	}
	return false
}

const (
	fewThreshold  = 1
	manyThreshold = 5
)

var (
	single = ansi.White(typography.Multiply)
	few    = ansi.White(typography.SmallSquare)
	many   = ansi.White(typography.Square)

	bar = ansi.Gray("|")
)

func (dw *drawWindow) getOverlap(x, y int) string {
	c := coords{x, y}
	dd := dw.cache[c]
	switch {
	case dd.pingCount <= fewThreshold:
		return single
	case dd.pingCount <= manyThreshold:
		return few
	default:
		return many
	}
}

func (dw *drawWindow) getKey() string {
	if dw.max > manyThreshold {
		return fmt.Sprintf(ansi.Gray("Key")+ansi.White(": ")+single+" = %d "+bar+" "+few+" = %d-%d "+bar+" "+many+" = %d+    ",
			fewThreshold, fewThreshold+1, manyThreshold, manyThreshold+1)
	}
	if dw.max > fewThreshold {
		return fmt.Sprintf(ansi.Gray("Key")+ansi.White(": ")+single+" = %d "+bar+" "+few+" = %d-%d    ",
			fewThreshold, fewThreshold+1, manyThreshold)
	}
	return ""
}
