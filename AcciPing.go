// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"

	"github.com/Lexer747/AcciPing/ping"
)

func main() {
	p := ping.NewPing()
	duration, err := p.OneShot("www.google.com")
	fmt.Printf("Duration: %s | Err: '%+v'\n", duration, err)
}
