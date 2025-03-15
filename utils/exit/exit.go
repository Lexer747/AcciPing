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

// OnError should be called when there is no way from the program to continue functioning normally, if err is
// not nil the program will exit and print the error which caused the issue.
func OnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// OnErrorMsg is like [OnError] but has a custom message when err is not nil.
func OnErrorMsg(err error, msg string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, msg+": %s", err.Error())
		os.Exit(1)
	}
}

// OnErrorMsgf is like [OnErrorMsg] but will format the string according to printf before writing it.
func OnErrorMsgf(err error, format string, args ...any) {
	if err != nil {
		fmt.Fprintf(os.Stderr, fmt.Sprintf(format, args...)+": %s", err.Error())
		os.Exit(1)
	}
}

// Success is a alias for [os.Exit(0)].
func Success() {
	os.Exit(0)
}

func Silent() {
	os.Exit(1)
}
