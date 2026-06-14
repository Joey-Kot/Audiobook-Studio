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
