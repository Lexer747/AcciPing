// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/errors"
	"github.com/Lexer747/AcciPing/utils/siphon"
)

func main() {
	p := ping.NewPing()
	ctx, cancelFunc := context.WithCancelCause(context.Background())
	defer cancelFunc(nil)
	term, err := terminal.NewTerminal()
	if err != nil {
		panic(err.Error())
	}
	existingData, toUpdate := loadFile()

	const pingsPerMinute = 60.0
	const channelSize = 10
	channel, err := p.CreateChannel(ctx, existingData.URL, pingsPerMinute, channelSize)
	if err != nil {
		panic(err.Error())
	}
	graphChannel, fileChannel := siphon.TeeBufferedChannel(ctx, channel, channelSize)
	go writeToFile(ctx, fileChannel, toUpdate)

	// The graph will take ownership of the data.
	g, err := graph.NewGraphWithData(ctx, graphChannel, term, pingsPerMinute, existingData)
	if err != nil {
		panic(err.Error())
	}
	// Very high FPS is good for responsiveness in the UI (since it's locked) and re-drawing on a re-size.
	err = g.Run(ctx, cancelFunc, 60)
	if err != nil && !errors.Is(err, terminal.UserCancelled) {
		panic(err.Error())
	} else {
		_ = g.Term.ClearScreen(true)
		g.Term.Print(g.LastFrame())
		g.Term.Print("\n# Summary\n" + g.Summarize())
	}
}

func loadFile() (*data.Data, *os.File) {
	const demoFilePath = "dev.pings"
	demoURL := "www.google.com"
	f, err := os.OpenFile(demoFilePath, os.O_RDONLY, 0)
	var existingData *data.Data
	switch {
	case err != nil && !errors.Is(err, os.ErrNotExist):
		// Some error we are not expecting
		panic(err.Error())
	case err != nil && errors.Is(err, os.ErrNotExist):
		defer f.Close()
		// First time, make a new file
		existingData = data.NewData(demoURL)
		newFile, err := os.OpenFile(demoFilePath, os.O_CREATE|os.O_RDWR, 0o777)
		if err != nil {
			panic(err.Error())
		}
		defer newFile.Close()
		if err = existingData.AsCompact(newFile); err != nil {
			panic(err.Error())
		}
	default:
		defer f.Close()
		// File exists, read the data from it
		// TODO incremental read/writes, get the URL ASAP then start the channel, then incremental continuation.
		existingData = &data.Data{}
		fromFile, err := io.ReadAll(f)
		if err != nil {
			panic(err.Error())
		}
		if _, err = existingData.FromCompact(fromFile); err != nil {
			panic(err.Error())
		}
	}

	f, err = os.OpenFile(demoFilePath, os.O_RDWR, 0o777)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(existingData.String())
	return existingData, f
}

func writeToFile(ctx context.Context, input chan ping.PingResults, fileToUpdate *os.File) {
	defer fileToUpdate.Close()
	ourData := &data.Data{}
	// Block: To scope this byte slice, we don't want to expose it to the running loop
	{
		// TODO provide an error channel and surface errors to the graph UI
		file, _ := io.ReadAll(fileToUpdate)
		_, _ = ourData.FromCompact(file)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case p, ok := <-input:
			if !ok {
				return
			}
			ourData.AddPoint(p)
			// TODO provide an error channel and surface errors to the graph UI
			_, _ = fileToUpdate.Seek(0, 0)
			_ = ourData.AsCompact(fileToUpdate)
		}
	}
}
