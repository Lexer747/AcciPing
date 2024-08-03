// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"io"
	"os"

	"github.com/Lexer747/AcciPing/files"
	"github.com/Lexer747/AcciPing/graph"
	"github.com/Lexer747/AcciPing/graph/data"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/ping"
	"github.com/Lexer747/AcciPing/utils/check"
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
	// Now that we have a ping channel which is already running we want to duplicate it, providing one to the
	// Graph and second to a file writer. This de-couples the processes, we don't want the GUI to affect
	// storing data and vice versa.
	graphChannel, fileChannel := siphon.TeeBufferedChannel(ctx, channel, channelSize)
	fileData, err := duplicateData(toUpdate)
	if err != nil {
		panic(err.Error())
	}
	go writeToFile(ctx, fileData, fileChannel, toUpdate)

	// The graph will take ownership of the data channel.
	g, err := graph.NewGraphWithData(ctx, graphChannel, term, pingsPerMinute, existingData)
	if err != nil {
		panic(err.Error())
	}
	_ = g.Term.ClearScreen(true)
	// Very high FPS is good for responsiveness in the UI (since it's locked) and re-drawing on a re-size.
	err = g.Run(ctx, cancelFunc, 120)
	if err != nil && !errors.Is(err, terminal.UserCancelled) {
		panic(err.Error())
	} else {
		_ = g.Term.ClearScreen(true)
		g.Term.Print(g.LastFrame())
		g.Term.Print("\n# Summary\n" + g.Summarise() + "\n")
	}
}

const demoFilePath = "dev.pings"
const demoURL = "www.google.com"

// TODO incremental read/writes, get the URL ASAP then start the channel, then incremental continuation.
func loadFile() (*data.Data, *os.File) {
	d, f, err := files.LoadOrCreateFile(demoFilePath, demoURL)
	if err != nil {
		panic(err.Error())
	}
	return d, f
}

func writeToFile(ctx context.Context, ourData *data.Data, input chan ping.PingResults, fileToUpdate *os.File) {
	defer fileToUpdate.Close()
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
			_, err := fileToUpdate.Seek(0, 0)
			check.NoErr(err, "seeking file")
			err = ourData.AsCompact(fileToUpdate)
			check.NoErr(err, "writing file")
		}
	}
}

func duplicateData(f *os.File) (*data.Data, error) {
	d := &data.Data{}
	file, fileErr := io.ReadAll(f)
	_, readingErr := d.FromCompact(file)
	return d, errors.Join(fileErr, readingErr)
}
