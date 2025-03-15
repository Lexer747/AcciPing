// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import (
	"strings"

	"github.com/Lexer747/acci-ping/ping"
	"github.com/Lexer747/acci-ping/utils/check"
)

// This file contains various helper methods for unit tests but which are not safe public API methods.

func (g *Graph) AddPoint(p ping.PingResults) {
	g.data.AddPoint(p)
}

func (g *Graph) ComputeFrame() string {
	var b strings.Builder
	painter := g.computeFrame(0, false)
	err := painter(&b)
	check.NoErr(err, "While painting frame to string buffer")
	return b.String()
}

func (g *Graph) Size() int64 {
	return g.data.TotalCount()
}
