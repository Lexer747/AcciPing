// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package acciping

import (
	"bytes"
	"context"

	"github.com/Lexer747/acci-ping/draw"
	"github.com/Lexer747/acci-ping/graph/terminal"
	"github.com/Lexer747/acci-ping/graph/terminal/ansi"
	"github.com/Lexer747/acci-ping/gui"
)

// help which should only be called once the paint buffer is initialised.
func (app *Application) help(
	ctx context.Context,
	startShowHelp bool,
	helpChannel chan rune,
	terminalSizeUpdates chan terminal.Size,
) {
	helpBuffer := app.drawBuffer.Get(draw.HelpIndex)
	h := help{showHelp: startShowHelp}
	app.GUI.paint(h.render(app.term.Size(), helpBuffer))
	for {
		select {
		case <-ctx.Done():
			return
		case newSize := <-terminalSizeUpdates:
			app.GUI.paint(h.render(newSize, helpBuffer))
		case toShow := <-helpChannel:
			switch toShow {
			case 'h':
				h.showHelp = !h.showHelp
			default:
			}
			app.GUI.paint(h.render(app.term.Size(), helpBuffer))
		}
	}
}

type help struct {
	showHelp bool
}

func (h help) render(size terminal.Size, buf *bytes.Buffer) paintUpdate {
	ret := None
	shouldInvalidate := buf.Len() != 0
	if shouldInvalidate {
		ret = ret | Invalidate
	}
	buf.Reset()
	if h.showHelp {
		box := h.makeHelpBox()
		box.Draw(size, buf)
		return ret | Paint
	}
	return ret
}

func helpAction(ch chan rune) func(r rune) error {
	return func(r rune) error {
		ch <- r
		return nil
	}
}

func (h help) makeHelpBox() gui.Box {
	return gui.Box{
		BoxText: helpCopy,
		Position: gui.Position{
			Vertical:   gui.Centre,
			Horizontal: gui.Right,
			Padding:    gui.Padding{Left: 4},
		},
		Style: gui.SharpCorners,
	}
}

var helpCopy = []gui.Typography{
	{ToPrint: ansi.Yellow("Help"), TextLen: 4, Alignment: gui.Centre},
	{ToPrint: "", TextLen: 0, Alignment: gui.Centre},
	{ToPrint: "Press " + ansi.Green("ctrl+c") + " to exit.", TextLen: 6 + 6 + 9, Alignment: gui.Left},
	{ToPrint: "Press " + ansi.Green("h") + " to open/close this window.", TextLen: 6 + 1 + 27, Alignment: gui.Left},
}
