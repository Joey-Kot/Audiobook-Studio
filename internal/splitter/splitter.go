// Copyright (C) 2026 Joey Kot <joey.kot.x@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed WITHOUT ANY WARRANTY; without even the
// implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
// See <https://www.gnu.org/licenses/> for more details.

package splitter

import "strings"

// DefaultThreshold is the fallback chunk character limit.
const DefaultThreshold = 300

var preferredBreaks = map[rune]bool{
	'。': true, '！': true, '？': true, '，': true, '、': true, '；': true, '：': true,
	'.': true, '!': true, '?': true, ',': true, ';': true, ':': true,
	'\n': true,
}

// Split splits text into chunks at punctuation near threshold.
func Split(text string, threshold int) []string {
	if threshold <= 0 {
		threshold = DefaultThreshold
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}

	var chunks []string
	for len(runes) > threshold {
		cut := findCut(runes, threshold)
		part := strings.TrimSpace(string(runes[:cut]))
		if part != "" {
			chunks = append(chunks, part)
		}
		runes = trimLeadingSpace(runes[cut:])
	}
	if tail := strings.TrimSpace(string(runes)); tail != "" {
		chunks = append(chunks, tail)
	}
	return chunks
}

func findCut(runes []rune, threshold int) int {
	if len(runes) <= threshold {
		return len(runes)
	}
	min := threshold / 2
	if min < 1 {
		min = 1
	}
	max := threshold + threshold/3
	if max > len(runes) {
		max = len(runes)
	}

	best := -1
	bestDistance := len(runes)
	for i := min; i < max; i++ {
		if preferredBreaks[runes[i]] {
			d := abs(i + 1 - threshold)
			if d < bestDistance {
				best = i + 1
				bestDistance = d
			}
		}
	}
	if best > 0 {
		return best
	}

	for i := threshold; i > 0; i-- {
		if runes[i-1] == ' ' || runes[i-1] == '\t' {
			return i
		}
	}
	return threshold
}

func trimLeadingSpace(runes []rune) []rune {
	for len(runes) > 0 {
		switch runes[0] {
		case ' ', '\t', '\n', '\r':
			runes = runes[1:]
		default:
			return runes
		}
	}
	return runes
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
