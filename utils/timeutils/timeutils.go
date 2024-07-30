// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package timeutils

import (
	"time"

	"github.com/Lexer747/AcciPing/utils/numeric"
)

func HumanString(t time.Duration, digits int) string {
	rounded := numeric.RoundToNearestSigFig(float64(t), digits)
	return time.Duration(rounded).String()
}
