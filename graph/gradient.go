// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package graph

import (
	"fmt"

	t "github.com/Lexer747/acci-ping/graph/terminal/typography"
	"github.com/Lexer747/acci-ping/utils/check"
)

func solve(x []int, y []int) []string {
	check.Check(len(x) == len(y), "x and y should be equal len")
	if len(x) <= 1 {
		return []string{}
	}
	if len(x) == 2 {
		return []string{gradientSolve(x[0], x[1], y[0], y[1])}
	}
	result := make([]string, len(x)-1)
	xDirs := make([]direction, len(x)-1)
	yDirs := make([]direction, len(x)-1)
	xEqualsCount := 0
	yEqualsCount := 0
	for i := range len(x) - 1 {
		xDirs[i] = getDir(x[i], x[i+1])
		// // y values are inverted
		yDirs[i] = getDir(y[i+1], y[i])
		if xDirs[i] == equal {
			xEqualsCount++
		}
		if yDirs[i] == equal {
			yEqualsCount++
		}
	}
	if yEqualsCount > len(yDirs)/2 {
		solve := []string{}
		var specific bool
		for i := range len(xDirs) - 1 {
			if specific {
				specific = false
				continue
			}
			specific, solve = solveShallowTwoDirections(xDirs[i], yDirs[i], xDirs[i+1], yDirs[i+1])
			result[i] = solve[0]
			if specific {
				result[i+1] = solve[1]
			}
		}
		result[len(result)-1] = solve[1]
	} else {
		solve := []string{}
		for i := range len(xDirs) - 1 {
			solve = solveTwoDirections(xDirs[i], yDirs[i], xDirs[i+1], yDirs[i+1])
			result[i] = solve[0]
		}
		result[len(result)-1] = solve[1]
	}

	return result
}

func gradientSolve(beginX, beginY, endX, endY int) string {
	xDir := getDir(beginX, endX)
	// y values are inverted
	yDir := getDir(endY, beginY)
	return solveDirections(xDir, yDir)
}

func solveShallowTwoDirections(firstX direction, firstY direction, secondX direction, secondY direction) (bool, []string) {
	first := solveDirections(firstX, firstY)
	second := solveDirections(secondX, secondY)
	switch {
	case (first == "/" && second == "-") || (first == "/" && second == ""):
		return true, []string{t.TopLine, t.BottomLine}
	case (first == "\\" && second == "-") || (first == "\\" && second == ""):
		return true, []string{t.BottomLine, t.TopLine}
	case (first == "-" && second == "/") || (first == "" && second == "/"):
		return false, []string{first, t.TopLine}
	case (first == "-" && second == "\\") || (first == "" && second == "\\"):
		return false, []string{first, t.BottomLine}
	default:
		return false, []string{first, second}
	}
}

func solveTwoDirections(firstX direction, firstY direction, secondX direction, secondY direction) []string {
	first := solveDirections(firstX, firstY)
	second := solveDirections(secondX, secondY)
	switch {
	case first == "-" && second == t.Vertical:
		return []string{" ", t.Vertical}
	case first == t.Vertical && second == "-":
		return []string{t.Vertical, " "}
	default:
		return []string{first, second}
	}
}

func solveDirections(xDir direction, yDir direction) string {
	if xDir == positive && yDir == positive {
		return "/"
	} else if xDir == equal && yDir == positive {
		return t.Vertical
	} else if xDir == negative && yDir == positive {
		return "\\"
	}
	if (xDir == positive || xDir == negative) && yDir == equal {
		return "-"
	} else if xDir == equal && yDir == equal {
		return ""
	}
	if xDir == positive && yDir == negative {
		return "\\"
	} else if xDir == equal && yDir == negative {
		return t.Vertical
	} else if xDir == negative && yDir == negative {
		return "/"
	}
	panic(fmt.Sprintf("Case not covered %d %d", xDir, yDir))
}

func getDir(begin int, end int) direction {
	switch {
	case begin < end:
		return negative
	case begin == end:
		return equal
	default:
		return positive
	}
}

type direction int

const (
	positive direction = 1
	equal    direction = 0
	negative direction = -1
)
