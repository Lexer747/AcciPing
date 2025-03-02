// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024-2025 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package acciping

import (
	"bytes"
	"context"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Lexer747/AcciPing/draw"
	"github.com/Lexer747/AcciPing/graph/terminal"
	"github.com/Lexer747/AcciPing/graph/terminal/ansi"
)

// toastNotifications which should only be called once the paint buffer is initialised.
func (app *Application) toastNotifications(ctx context.Context) {
	store := toastStore{
		Mutex:  &sync.Mutex{},
		toasts: map[int]toast{},
	}
	toastBuffer := app.drawBuffer.Get(draw.ToastIndex)
	for {
		select {
		case <-ctx.Done():
			return
		case toShow := <-app.errorChannel:
			if toShow == nil {
				continue
			}
			// A new error has been surfaced:
			store.Lock()
			// First generate a unique id for this error and add it to our map.
			key := app.insertToast(store, toShow)
			// Now refresh the window size and write the toast notification to the window
			store.write(app.term.Size(), toastBuffer)
			store.Unlock()
			// Now after some timeout, remove the notification and re-render
			go func() {
				<-time.After(10 * time.Second)
				store.Lock()
				delete(store.toasts, key)
				store.write(app.term.Size(), toastBuffer)
				store.Unlock()
			}()
		}
	}
}

type toast struct {
	timestamp time.Time
	err       string
}

type toastStore struct {
	*sync.Mutex
	toasts map[int]toast
}

// insertToast should only be called while the lock is held
func (*Application) insertToast(store toastStore, toShow error) int {
	var key int
	for {
		key = rand.Int() //nolint:gosec
		_, ok := store.toasts[key]
		if !ok {
			store.toasts[key] = toast{
				timestamp: time.Now(),
				err:       toShow.Error(),
			}
			break
		}
	}
	return key
}

// write should only be called while the lock is held
func (ts toastStore) write(size terminal.Size, b *bytes.Buffer) {
	b.Reset()
	if len(ts.toasts) == 0 {
		return
	}
	order, length := ts.orderToasts()
	length += 7
	putCentre := centreAndPad(length, b)
	centreY := size.Height / 2
	centreX := size.Width / 2
	startY := centreY - len(order)/2
	startX := centreX - length/2
	bar := strings.Repeat("─", length)
	b.WriteString(ansi.CursorPosition(startY, startX) + "╭" + bar + "╮")
	b.WriteString(ansi.CursorPosition(startY+1, startX) + "│")
	putCentre(ansi.Red(title), len(title))
	b.WriteString("│")
	// TODO trim error box when more than height
	for i, t := range order {
		b.WriteString(ansi.CursorPosition(startY+i+2, startX) + "│⚠️ ")
		putCentre(t.err, len(t.err)+5)
		b.WriteString(" ⚠️ |")
	}
	b.WriteString(ansi.CursorPosition(startY+len(order)+2, startX) + "╰" + bar + "╯")
}

func centreAndPad(length int, b *bytes.Buffer) func(string, int) {
	return func(s string, strLen int) {
		if strLen >= length {
			b.WriteString(s)
			return
		}
		padding := (length - strLen) / 2
		leftPadding, rightPadding := getLeftRightPadding(padding, padding, strLen, length)
		b.WriteString(strings.Repeat(" ", leftPadding) + s + strings.Repeat(" ", rightPadding))
	}
}

func getLeftRightPadding(leftPadding, rightPadding, strLen, length int) (int, int) {
	for leftPadding+rightPadding+strLen > length {
		if leftPadding+rightPadding+strLen%2 == 0 {
			leftPadding--
		} else {
			rightPadding--
		}
	}
	for leftPadding+rightPadding+strLen < length {
		if leftPadding+rightPadding+strLen%2 == 0 {
			leftPadding++
		} else {
			rightPadding++
		}
	}
	return leftPadding, rightPadding
}

// orderToasts will return a slice of ordered toasts where they're sorted by the timestamp in which they were
// added to the storage, it also returns the longest error string. Should only be called while the lock is
// held.
func (ts toastStore) orderToasts() ([]toast, int) {
	order := make([]toast, 0, len(ts.toasts))
	length := 0
	for _, t := range ts.toasts {
		idx, _ := slices.BinarySearchFunc(order, t, func(a, b toast) int { return a.timestamp.Compare(b.timestamp) })
		order = slices.Insert(order, idx, t)
		length = max(length, len(t.err))
	}
	return order, length
}

const title = "An Error Occurred"
