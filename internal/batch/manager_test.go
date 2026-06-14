package batch

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"audiobook-studio/internal/config"
)

type fakeSynth struct{}

func (fakeSynth) SynthesizeContext(ctx context.Context, text string, voiceJSON string) ([]byte, error) {
	return []byte("audio:" + text), nil
}

type fakeEncoder struct {
	merged [][]byte
	output string
}

func (f *fakeEncoder) DecodeToPCM(input []byte) ([]byte, int, error) {
	return append([]byte("pcm:"), input...), 24000, nil
}

func (f *fakeEncoder) MergeToMP3(segments [][]byte, outputPath string) error {
	f.merged = append(f.merged, segments...)
	f.output = outputPath
	return os.WriteFile(outputPath, []byte("mp3"), 0644)
}

func TestManagerProcessesFileInOrder(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "book.txt")
	if err := os.WriteFile(input, []byte("first sentence. second sentence. third sentence. fourth sentence. fifth sentence. sixth sentence."), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.OutputDir = filepath.Join(dir, "out")
	cfg.SplitThreshold = 80
	cfg.Concurrency = 2
	encoder := &fakeEncoder{}
	var final BatchProgress
	manager := NewManager(cfg, fakeSynth{}, encoder, func(progress BatchProgress) {
		final = progress
	})
	if err := manager.Start(context.Background(), []string{input}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if final.Status != StatusDone || final.Percent != 100 {
		t.Fatalf("unexpected final progress %#v", final)
	}
	if len(encoder.merged) < 2 {
		t.Fatalf("expected multiple merged segments, got %d", len(encoder.merged))
	}
	if _, err := os.Stat(encoder.output); err != nil {
		t.Fatalf("output not written: %v", err)
	}
}
