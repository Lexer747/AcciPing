// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package data

import (
	"cmp"
	"net"
)

// ipOrdering Doesn't work if passed different length addresses v4/v6, otherwise it's a simple byte wise ordering
func ipOrdering(a, b net.IP) int {
	for i := range a {
		c := cmp.Compare(a[i], b[i])
		if c != 0 {
			return c
		}
	}
	return 0
}
