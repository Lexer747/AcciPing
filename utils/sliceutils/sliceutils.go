// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package sliceutils

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
)

func Map[IN, OUT any, S ~[]IN](slice S, f func(IN) OUT) []OUT {
	ret := make([]OUT, len(slice))
	for i, in := range slice {
		ret[i] = f(in)
	}
	return ret
}

func OneOf[S ~[]T, T any](slice S, f func(T) bool) bool {
	for _, item := range slice {
		if f(item) {
			return true
		}
	}
	return false
}

func AllOf[S ~[]T, T any](slice S, f func(T) bool) bool {
	for _, item := range slice {
		if !f(item) {
			return false
		}
	}
	return true
}

func Fold[IN, OUT any, S ~[]IN](slice S, base OUT, f func(IN, OUT) OUT) OUT {
	ret := base
	for _, in := range slice {
		ret = f(in, ret)
	}
	return ret
}

func Shuffle[S ~[]T, T any](slice S) S {
	ret := slices.Clone(slice)
	shuf := func(i, j int) {
		t := ret[i]
		ret[i] = ret[j]
		ret[j] = t
	}
	rand.Shuffle(len(ret), shuf)
	return ret
}

func Join[S ~[]T, T fmt.Stringer](slice S, sep string) string {
	return strings.Join(Map(slice, T.String), sep)
}

func Remove[S ~[]T, T comparable](slice S, elements ...T) S {
	toDelete := map[T]struct{}{}
	for _, t := range elements {
		toDelete[t] = struct{}{}
	}
	ret := S{}
	for _, s := range slice {
		if _, found := toDelete[s]; found {
			continue
		} else {
			ret = append(ret, s)
		}
	}
	return ret
}
