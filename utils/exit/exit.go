// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package exit

import (
	"fmt"
	"os"
)

// OnError should be called when there is no way from the program to continue functioning normally
func OnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func OnErrorMsg(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, msg+": %s", err.Error())
		os.Exit(1)
	}
}

func Success() {
	os.Exit(0)
}
