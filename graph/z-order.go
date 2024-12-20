// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import "sync/atomic"

var (
	barIndex      = newIndex()
	xAxisIndex    = newIndex()
	yAxisIndex    = newIndex()
	keyIndex      = newIndex()
	gradientIndex = newIndex()
	dataIndex     = newIndex()
	spinnerIndex  = newIndex()

	indexCount atomic.Int64
)

func newIndex() int {
	cur := int(indexCount.Add(1))
	return cur - 1
}

// Z-order is top to bottom so the first item added to ret is at the back, the last item is at the front
var paintOrder = []int{
	// gradient is on the bottom since it's the most "fluffy" part of the presentation, it's interpolated data
	gradientIndex,
	barIndex,
	// bars should be overwritten by data and axis
	dataIndex,
	yAxisIndex,
	xAxisIndex,
	// key is inside the frame itself so should come on top of data to be readable
	keyIndex,
	// if we can't see the spinner we may be worried the program is dead
	spinnerIndex,
}
