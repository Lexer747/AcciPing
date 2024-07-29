// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package check

import "fmt"

func Check(shouldBeTrue bool, assertMsg string) {
	if !shouldBeTrue {
		panic(assertMsg)
	}
}

func Checkf(shouldBeTrue bool, format string, a ...any) {
	if !shouldBeTrue {
		panic(fmt.Sprintf(format, a...))
	}
}
