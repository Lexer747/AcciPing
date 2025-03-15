// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package draw

import (
	"bytes"
	"sync/atomic"

	"github.com/Lexer747/acci-ping/utils/sliceutils"
)

// Buffer is a helper type for the graph drawing code, instead of writing everything as literal go strings
// (the output type expected by the terminal) we keep a byte buffer for every z-index in our program. This
// allows the program to re-use the memory we allocate every frame, this means the total memory we need to
// allocate for drawing is bounded for the amount of the single largest frame we ever draw. This has huge
// performance improvements over creating string literals because it's gets the GC out of our way.
type Buffer struct {
	storage []*bytes.Buffer
}

// TODO paint buffer should be application level and agnostic to the draw buffer itself.
func NewPaintBuffer() *Buffer {
	return newBuffer(int(indexCount.Load()))
}

type Index int

// Get the underlying buffer for this z-index
func (b *Buffer) Get(z Index) *bytes.Buffer {
	return b.storage[z]
}

// Reset will reset all the buffers so that they no longer contain the last frame but are all empty.
func (b *Buffer) Reset(toReset ...Index) {
	// TODO an optimization here is too not reset at frame start but just reset the writer pointer per frame
	// to the start of the buffer then before drawing clear all the bytes from the writer pointer till the end
	// of the buffer.
	for _, idx := range toReset {
		b.Get(idx).Reset()
	}
}

var (
	BarIndex      = newIndex()
	DataIndex     = newIndex()
	GradientIndex = newIndex()
	KeyIndex      = newIndex()
	SpinnerIndex  = newIndex()
	ToastIndex    = newIndex()
	HelpIndex     = newIndex()
	XAxisIndex    = newIndex()
	YAxisIndex    = newIndex()
)

// Z-order is top to bottom so the first item added to ret is at the back, the last item is at the front
var PaintOrder = []Index{
	// gradient is on the bottom since it's the most "fluffy" part of the presentation, it's interpolated data
	GradientIndex,
	BarIndex,
	// bars should be overwritten by data and axis
	DataIndex,
	YAxisIndex,
	XAxisIndex,
	// key is inside the frame itself so should come on top of data to be readable
	KeyIndex,
	// Notifications can appear above the graph as they're ephemeral
	ToastIndex,
	HelpIndex,
	// if we can't see the spinner we may be worried the program is dead
	SpinnerIndex,
}

// GraphIndexes is the [PaintOrder] with the GUI indexes removed
var GraphIndexes = sliceutils.Remove(PaintOrder,
	ToastIndex,
	HelpIndex,
	SpinnerIndex,
)

// GUIIndexes is the above paint order with the GraphIndexes indexes removed
var GUIIndexes = sliceutils.Remove(PaintOrder, GraphIndexes...)

// newBuffer creates a new [Buffer] of [n] z-buffers.
func newBuffer(zMax int) *Buffer {
	ret := &Buffer{
		storage: make([]*bytes.Buffer, zMax),
	}
	for i := range zMax {
		ret.storage[i] = &bytes.Buffer{}
	}
	return ret
}

func newIndex() Index {
	cur := Index(indexCount.Add(1))
	return cur - 1
}

var indexCount atomic.Int32
