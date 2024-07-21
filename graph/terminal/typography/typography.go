// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package typography

const (
	Bullet       = "\u2022"
	HollowBullet = "\u25E6"
	Diamond      = "\u2BC1"

	Vertical   = "\u2502"
	Horizontal = "\u2500"

	VerySteepUpSlope = "\u002F"
	SteepUpSlope     = "\u2215"
	UpSlope          = "\u2571"
	GentleUpSlope    = "\uFF0F"

	SteepDownSlope  = "\u005C"
	DownSlope       = "\u2572"
	GentleDownSlope = "\uFF3C"

	Block       = "\u2588"
	LightBlock  = "\u2591"
	MediumBlock = "\u2592"
	DarkBlock   = "\u2593"
)

// Gradient operates on the [0,1] range.
func Gradient(g float64) string {
	switch {
	case g > 0.9:
		return VerySteepUpSlope
	case g > 0.8:
		return SteepUpSlope
	case g > 0.65:
		return UpSlope
	case g > 0.55:
		return GentleUpSlope
	case g > 0.45:
		return Horizontal
	case g > 0.3:
		return GentleDownSlope
	case g > 0.2:
		return DownSlope
	default:
		return SteepDownSlope
	}
}
