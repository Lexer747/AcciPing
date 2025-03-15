// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package timeutils

import (
	"fmt"
	"time"

	"github.com/Lexer747/acci-ping/utils/numeric"
)

// HumanString truncates a duration to a given number of sig figs then prints it as a human readable duration.
//
// e.g.
//
//	HumanString(123456 * time.Nanosecond, 3) // prints "123µs"
func HumanString(t time.Duration, digits int) string {
	rounded := numeric.TruncateToNearestSigFigInt(int(t), digits)
	return time.Duration(rounded).String()
}

// TimeDateFormat prints the date as required as input to re-build the timestamp in the go time library.
func TimeDateFormat(t time.Time) string {
	// year int, month Month, day, hour, min, sec, nsec int, loc *Location
	return fmt.Sprintf("time.Date(%d, %s, %d, %d, %d, %d, %d, %s)",
		t.Year(), t.Month().String(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location().String(),
	)
}
