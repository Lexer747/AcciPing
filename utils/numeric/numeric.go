// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package numeric

import (
	"math"

	"golang.org/x/exp/constraints"
)

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

// Normalize will scale [v] to between [0,1], based on it's ratio between the input [min,max].
func Normalize(v, min, max float64) float64 {
	return NormalizeToRange(v, min, max, 0, 1)
}

// NormalizeToRange scales [v] which is located within the range [min,max] and then rescales [v] such that it
// is the same ratio inside the new range [newMin,newMax].
//
// Inspired by my original https://github.com/Lexer747/PingPlotter/blob/master/src/Graph/Internal.hs#L15
func NormalizeToRange(v, min, max, newMin, newMax float64) float64 {
	return (((newMax - newMin) * (v - min)) / (max - min)) + newMin
}

type Number interface {
	constraints.Float | constraints.Signed
}

func Abs[N Number](n N) N {
	if n < 0 {
		return N(-1) * n
	}
	return n
}
