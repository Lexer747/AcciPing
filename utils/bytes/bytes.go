// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package bytes

import (
	"fmt"
	"strings"
)

func Clear(buffer []byte, n int) {
	for i := range n {
		buffer[i] = 0
	}
}

func HexPrint(buffer []byte) string {
	var b strings.Builder
	b.WriteString("[")
	for i, bite := range buffer {
		fmt.Fprintf(&b, "0x%x", bite)
		if i < len(buffer)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString("]")
	return b.String()
}
