// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package numeric

import "math"

func RoundToNearestSigFig(input float64, sigFig int) float64 {
	if input == 0 {
		return 0
	}
	power := float64(sigFig) - Exponent(input)
	magnitude := math.Pow(10.0, power)
	shifted := input * magnitude
	rounded := math.Round(shifted)
	return rounded / magnitude
}

func Exponent(input float64) float64 {
	return math.Ceil(math.Log10(math.Abs(input)))
}
