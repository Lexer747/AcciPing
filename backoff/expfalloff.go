// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package backoff

import (
	"math"
	"time"
)

type expFallOff struct {
	// Base is the initial smallest duration to wait in milliseconds
	Base     float64
	curCount int
}

// https://en.wikipedia.org/wiki/Exponential_backoff
func NewExponentialBackoff(backoffStart time.Duration) *expFallOff {
	return &expFallOff{
		Base: float64(backoffStart.Milliseconds()),
	}
}

func (e *expFallOff) Wait() {
	e.curCount++
	backoff := time.Duration(math.Pow(e.Base, float64(e.curCount)))
	<-time.After(backoff * time.Millisecond)
}

func (e *expFallOff) Success() {
	e.curCount = 0
}
