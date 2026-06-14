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

package config

import (
	"path/filepath"
	"testing"
)

func TestValidateRejectsBadVoiceJSON(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VoiceJSON = "{bad"
	if err := Validate(&cfg); err == nil {
		t.Fatal("expected invalid voice JSON error")
	}
}

func TestValidateSplitThresholdBoundary(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SplitThreshold = 10
	if err := Validate(&cfg); err == nil {
		t.Fatal("expected split threshold boundary error")
	}
	cfg.SplitThreshold = 11
	if err := Validate(&cfg); err != nil {
		t.Fatalf("expected split threshold 11 to be valid: %v", err)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := DefaultConfig()
	cfg.APIToken = "secret"
	cfg.SplitThreshold = 300
	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.APIToken != cfg.APIToken || got.SplitThreshold != cfg.SplitThreshold {
		t.Fatalf("roundtrip mismatch: %#v", got)
	}
}
