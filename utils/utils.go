// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package utils

// Err helpfully ignores the first argument and returns the error, this is usually used for interfaces like:
//
//	Write(p []byte) (n int, err error)
//
// In which n is not important only the error matters.
func Err[T any](_ T, e error) error {
	return e
}
