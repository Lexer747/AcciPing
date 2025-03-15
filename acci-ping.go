// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"flag"
	"os"

	acciping "github.com/Lexer747/acci-ping/cmd/subcommands/acci-ping"
	"github.com/Lexer747/acci-ping/cmd/subcommands/drawframe"
	"github.com/Lexer747/acci-ping/cmd/subcommands/ping"
	"github.com/Lexer747/acci-ping/cmd/subcommands/rawdata"
	"github.com/Lexer747/acci-ping/utils/errors"
	"github.com/Lexer747/acci-ping/utils/exit"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "drawframe":
			df := drawframe.GetFlags()
			FlagParseError(df.Parse(os.Args[2:]))
			drawframe.RunDrawFrame(df)
			exit.Success()
		case "rawdata":
			rd := rawdata.GetFlags()
			FlagParseError(rd.Parse(os.Args[2:]))
			rawdata.RunPrintData(rd)
			exit.Success()
		case "ping":
			p := ping.GetFlags()
			FlagParseError(p.Parse(os.Args[2:]))
			ping.RunPing(p)
			exit.Success()
		default:
			// fallthrough
		}
	}
	a := acciping.GetFlags()
	FlagParseError(a.Parse(os.Args[1:]))
	acciping.RunAcciPing(a)
	exit.Success()
}

func FlagParseError(err error) {
	if errors.Is(err, flag.ErrHelp) {
		exit.Silent()
	} else {
		exit.OnError(err)
	}
}
