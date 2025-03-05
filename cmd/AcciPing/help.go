// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package acciping

import (
	"context"
	"strconv"

	"github.com/Lexer747/AcciPing/utils/errors"
)

// help which should only be called once the paint buffer is initialised.
func (app *Application) help(ctx context.Context, helpChannel chan rune) {
	// helpBuffer := app.drawBuffer.Get(draw.ToastIndex)
	for {
		select {
		case <-ctx.Done():
			return
		case toShow := <-helpChannel:
			switch toShow {
			default:
				app.errorChannel <- errors.Errorf("Unsupported help type %s", strconv.QuoteRune(toShow))
			}
		}
	}
}

func helpAction(ch chan rune) func(r rune) error {
	return func(r rune) error {
		ch <- r
		return nil
	}
}
