// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package drawbuffer

import "bytes"

// Collection is a helper type for the graph drawing code, instead of writing everything as literal go strings
// (the output type expected by the terminal) we keep a byte buffer for every z-index in our program. This
// allows the program to re-use the memory we allocate every frame, this means the total memory we need to
// allocate for drawing is bounded for the amount of the single largest frame we ever draw. This has huge
// performance improvements over creating string literals because it's gets the GC out of our way.
type Collection struct {
	storage []*bytes.Buffer
}

// NewCollection creates a new [Collection] of [n] z-buffers.
func NewCollection(zMax int) *Collection {
	ret := &Collection{
		storage: make([]*bytes.Buffer, zMax),
	}
	for i := range zMax {
		ret.storage[i] = &bytes.Buffer{}
	}
	return ret
}

// Get the underlying buffer for this z-index
func (b *Collection) Get(z int) *bytes.Buffer {
	return b.storage[z]
}

// Reset will reset all the buffers so that they no longer contain the last frame but are all empty.
func (b *Collection) Reset() {
	// TODO an optimization here is too not reset at frame start but just reset the writer pointer per frame
	// to the start of the buffer then before drawing clear all the bytes from the writer pointer till the end
	// of the buffer.
	for _, buffer := range b.storage {
		buffer.Reset()
	}
}
