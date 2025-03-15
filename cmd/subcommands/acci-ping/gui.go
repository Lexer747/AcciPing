// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package acciping

import (
	"sync"

	"github.com/Lexer747/acci-ping/graph/terminal"
	"github.com/Lexer747/acci-ping/gui"
	"github.com/Lexer747/acci-ping/utils/check"
)

type GUI struct {
	listeningChars map[rune]terminal.ConditionalListener
	fallbacks      []terminal.Listener

	m        sync.Mutex
	p, lastP uint64
	i, lastI uint64
}

var _ gui.GUI = (&GUI{})

func newGUIState() *GUI {
	return &GUI{
		listeningChars: map[rune]terminal.ConditionalListener{},
		fallbacks:      []terminal.Listener{},
	}
}

type paintUpdate int

const (
	None       paintUpdate = 0b000000000000000
	Paint      paintUpdate = 0b000000000000001
	Invalidate paintUpdate = 0b000000000000010 // TODO invalidate invalidates all components but should only per remove GUI element
)

func (p paintUpdate) String() string {
	switch p {
	case None:
		return "None"
	case Paint:
		return "Paint"
	case Invalidate:
		return "Invalidate"
	}
	panic("exhaustive:enforce")
}

func (g *GUI) paint(update paintUpdate) {
	if update == None {
		return
	}
	g.m.Lock()
	defer g.m.Unlock()
	if (update & Paint) != None {
		g.p++
	}
	if (update & Invalidate) != None {
		g.i++
	}
}

func (g *GUI) Drawn(t gui.Token) {
	token, ok := t.(*paintToken)
	check.Check(ok, "should only be called with original gui token")
	g.lastI = token.i
	g.lastP = token.p
}

// GetState implements gui.GUI.
func (g *GUI) GetState() gui.Token {
	g.m.Lock()
	defer g.m.Unlock()
	return &paintToken{p: g.p, lastP: g.lastP, i: g.i, lastI: g.lastI}
}

type paintToken struct {
	p, lastP uint64
	i, lastI uint64
}

func (p *paintToken) ShouldDraw() bool {
	return p.p > p.lastP
}

func (p *paintToken) ShouldInvalidate() bool {
	return p.i > p.lastI
}

var _ gui.Token = (&paintToken{})
