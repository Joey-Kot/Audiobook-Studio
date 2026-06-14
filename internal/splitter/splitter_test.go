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

import (
	"strings"
	"testing"
)

func TestSplitChineseNearPunctuation(t *testing.T) {
	got := Split("第一句很短。第二句稍微长一点，第三句继续延伸。第四句结束。", 14)
	if len(got) < 2 {
		t.Fatalf("expected multiple chunks, got %#v", got)
	}
	if !strings.HasSuffix(got[0], "。") && !strings.HasSuffix(got[0], "，") {
		t.Fatalf("expected punctuation cut, got %q", got[0])
	}
}

func TestSplitEnglishNearPunctuation(t *testing.T) {
	got := Split("One short sentence. Another sentence, with a comma. Final sentence.", 24)
	if len(got) < 2 {
		t.Fatalf("expected multiple chunks, got %#v", got)
	}
	if !strings.HasSuffix(got[0], ".") && !strings.HasSuffix(got[0], ",") {
		t.Fatalf("expected punctuation cut, got %q", got[0])
	}
}

func TestSplitMixedText(t *testing.T) {
	got := Split("第一段中文。Then English follows, with punctuation. 最后一段。", 20)
	if len(got) < 2 {
		t.Fatalf("expected mixed text split, got %#v", got)
	}
}

func TestSplitLongWithoutPunctuationFallback(t *testing.T) {
	text := strings.Repeat("你", 55)
	got := Split(text, 20)
	if len(got) != 3 {
		t.Fatalf("expected 3 chunks, got %d: %#v", len(got), got)
	}
	if got[0] != strings.Repeat("你", 20) {
		t.Fatalf("unexpected first chunk length %d", len([]rune(got[0])))
	}
}

func TestSplitEmptyAndBoundary(t *testing.T) {
	if got := Split("   ", 10); len(got) != 0 {
		t.Fatalf("expected no chunks, got %#v", got)
	}
	if got := Split("short", 10); len(got) != 1 || got[0] != "short" {
		t.Fatalf("unexpected boundary result %#v", got)
	}
}
