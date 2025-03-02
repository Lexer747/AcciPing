// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package acciping

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/utils/check"
	"github.com/Lexer747/AcciPing/utils/errors"
)

type Config struct {
	Cpuprofile         string
	FilePath           string
	LogFile            string
	Memprofile         string
	PingBufferingLimit int
	PingsPerMinute     float64
	URL                string

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
	closeCPUProfile := startCPUProfiling(c.Cpuprofile)
	defer closeCPUProfile()
	defer concludeMemProfile(c.Memprofile)
	closeLogFile := initLogging(c.LogFile)
	defer closeLogFile()

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

func initLogging(file string) func() {
	if file != "" {
		f, err := os.Create(file)
		check.NoErr(err, "could not create CPU profile")
		h := slog.NewTextHandler(f, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		slog.SetDefault(slog.New(h))
		slog.Debug("Logging started", "file", file)
		return func() {
			slog.Debug("Logging finished, closing", "file", file)
			check.NoErr(f.Close(), "failed to close log file")
		}
	}
	// If no file is specified we want to stop all logging
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError,
	})
	slog.SetDefault(slog.New(h))
	return func() {}
}
