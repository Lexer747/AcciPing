// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package acciping

import (
	"context"
	"fmt"
	"os"

	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/utils/errors"
)

type Config struct {
	Cpuprofile         string
	Memprofile         string
	FilePath           string
	URL                string
	PingsPerMinute     float64
	PingBufferingLimit int

	TestErrorListener bool
}

// exitOnError should be called when there is no way from the program to continue functioning normally
func exitOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func RunAcciPing(c Config) {
	app := Application{}
	ctx, cancelFunc := context.WithCancelCause(context.Background())
	defer cancelFunc(nil)
	ch, d := app.Init(ctx, c)
	err := app.Run(ctx, cancelFunc, ch, d)
	if err != nil && !errors.Is(err, terminal.UserCancelled) {
		exitOnError(err)
	} else {
		app.Finish()
	}
}
