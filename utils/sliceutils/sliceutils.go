// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package sliceutils

func Map[IN, OUT any, S ~[]IN](slice S, f func(IN) OUT) []OUT {
	ret := make([]OUT, len(slice))
	for i, in := range slice {
		ret[i] = f(in)
	}
	return ret
}
