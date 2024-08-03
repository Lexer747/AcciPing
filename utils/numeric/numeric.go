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

// TruncateToNearestSigFigInt is RoundToNearestSigFig but staying in purely integer land
func TruncateToNearestSigFigInt(input int, sigFig int) int {
	if input == 0 {
		return 0
	}
	exp := ExponentInt(input)
	if sigFig > exp {
		return input
	}
	power := sigFig - exp
	rounded := PowInt(input, 10, min(power, -1*power))
	result := PowInt(rounded, 10, max(power, -1*power))
	return result
}

func ExponentInt(input int) int {
	count := 0
	for i := Abs(input); i > 0; i /= 10 {
		count++
	}
	return count
}
func PowInt(start, base, input int) int {
	if input < 0 {
		ret := start
		for range Abs(input) {
			ret = ret / base
		}
		return ret
	} else {
		ret := start
		for range input {
			ret = ret * base
		}
		return ret
	}
}

func Exponent(input float64) float64 {
	return math.Ceil(math.Log10(math.Abs(input)))
}

// Normalize will scale [v] to between [0,1], based on it's ratio between the input [min,max].
func Normalize(v, minimum, maximum float64) float64 {
	return NormalizeToRange(v, minimum, maximum, 0, 1)
}

// NormalizeToRange scales [v] which is located within the range [min,max] and then rescales [v] such that it
// is the same ratio inside the new range [newMin,newMax].
//
// Inspired by my original https://github.com/Lexer747/PingPlotter/blob/master/src/Graph/Internal.hs#L15
func NormalizeToRange(v, minimum, maximum, newMin, newMax float64) float64 {
	return (((newMax - newMin) * (v - minimum)) / (maximum - minimum)) + newMin
}

type Number interface {
	constraints.Float | constraints.Signed
}

// Abs returns a concrete, but constraints confuse this linter.
//
//nolint:ireturn
func Abs[N Number](n N) N {
	if n < 0 {
		return N(-1) * n
	}
	return n
}
