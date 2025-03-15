// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"os"

	acciping "github.com/Lexer747/AcciPing/cmd/AcciPing"
	"github.com/Lexer747/AcciPing/cmd/drawframe"
	"github.com/Lexer747/AcciPing/cmd/ping"
	"github.com/Lexer747/AcciPing/cmd/rawdata"
	"github.com/Lexer747/AcciPing/utils/exit"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "drawframe":
			df := drawframe.GetFlags()
			exit.OnError(df.Parse(os.Args[2:]))
			drawframe.RunDrawFrame(df)
			exit.Success()
		case "rawdata":
			rd := rawdata.GetFlags()
			exit.OnError(rd.Parse(os.Args[2:]))
			rawdata.RunPrintData(rd)
			exit.Success()
		case "ping":
			p := ping.GetFlags()
			exit.OnError(p.Parse(os.Args[2:]))
			ping.RunPing(p)
			exit.Success()
		default:
			// fallthrough
		}
	}
	a := acciping.GetFlags()
	exit.OnError(a.Parse(os.Args[1:]))
	acciping.RunAcciPing(a)
	exit.Success()
}
