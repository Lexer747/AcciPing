// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package numeric_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Lexer747/AcciPing/utils/numeric"
	"github.com/Lexer747/AcciPing/utils/test_helpers"
)

func TestNormalize(t *testing.T) {
	type Case struct {
		Min, Max       float64
		NewMin, NewMax float64
		Inputs         []float64
		Expected       []float64
	}
	cases := []Case{
		{
			Min:    float64(7_657_469 * time.Microsecond),
			Max:    float64(12_301_543 * time.Microsecond),
			NewMin: 2,
			NewMax: 24,
			Inputs: []float64{
				float64(7_706_944 * time.Microsecond),
				float64(7_750_314 * time.Microsecond),
				float64(7_789_195 * time.Microsecond),
				float64(12_301_543 * time.Microsecond),
				float64(7_657_469 * time.Microsecond),
			},
			Expected: []float64{
				2.23,
				2.44,
				2.62,
				24,
				2,
			},
		},
	}
	for i, test := range cases {
		t.Run(fmt.Sprintf("%d:%f->%f|%+v", i, test.Min, test.Max, test.Inputs), func(t *testing.T) {
			t.Parallel()
			for i, input := range test.Inputs {
				actual := numeric.NormalizeToRange(input, test.Min, test.Max, test.NewMin, test.NewMax)
				test_helpers.AssertFloatEqual(t, test.Expected[i], actual, 3)
			}
		})
	}
}
