// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

//nolint:stylecheck
package env

import "os"

func SHOULD_TEST_NETWORK() bool {
	str := os.Getenv("SHOULD_TEST_NETWORK")
	return str == "1"
}

func LOCAL_FRAME_DIFFS() bool {
	str := os.Getenv("LOCAL_FRAME_DIFFS")
	return str == "1"
}
