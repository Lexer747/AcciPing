// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graphdata

import "github.com/Lexer747/acci-ping/ping"

func Add(si *SpanInfo, p ping.PingDataPoint, index int64) {
	if si.Count == 0 {
		si.addFirstPoint(p, index)
	} else {
		si.add(p, index)
	}
}
