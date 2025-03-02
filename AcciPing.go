// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"flag"

	acciping "github.com/Lexer747/AcciPing/cmd/AcciPing"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var logfile = flag.String("l", "", "write logs to `file`")

func main() {
	flag.Parse()
	acciping.RunAcciPing(acciping.Config{
		Cpuprofile:         *cpuprofile,
		FilePath:           demoFilePath,
		LogFile:            *logfile,
		Memprofile:         *memprofile,
		PingBufferingLimit: channelSize,
		PingsPerMinute:     pingsPerMinute,
		TestErrorListener:  true,
		URL:                demoURL,
	})
}

const demoFilePath = "dev.pings"
const demoURL = "www.google.com"
const pingsPerMinute = 60.0
const channelSize = 10
