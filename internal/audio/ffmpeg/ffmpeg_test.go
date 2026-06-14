package ffmpeg

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDecodeAndMergeWithFFmpeg(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "tone.wav")
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=0.05", "-y", source)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate tone: %v: %s", err, out)
	}
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	encoder := New("ffmpeg", 64)
	pcm, rate, err := encoder.DecodeToPCM(input)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(pcm) == 0 || rate != sampleRate {
		t.Fatalf("unexpected pcm len/rate: %d/%d", len(pcm), rate)
	}
	output := filepath.Join(dir, "out.mp3")
	if err := encoder.MergeToMP3([][]byte{pcm, pcm}, output); err != nil {
		t.Fatalf("merge: %v", err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output mp3 is empty")
	}
}

func TestDecodeCommonTTSFormatsWithFFmpeg(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	cases := []struct {
		name string
		args []string
	}{
		{name: "tone.wav", args: []string{"-f", "lavfi", "-i", "sine=frequency=440:duration=0.03", "-y"}},
		{name: "tone.mp3", args: []string{"-f", "lavfi", "-i", "sine=frequency=440:duration=0.03", "-codec:a", "libmp3lame", "-y"}},
		{name: "tone.opus", args: []string{"-f", "lavfi", "-i", "sine=frequency=440:duration=0.03", "-codec:a", "libopus", "-y"}},
		{name: "tone.aac", args: []string{"-f", "lavfi", "-i", "sine=frequency=440:duration=0.03", "-codec:a", "aac", "-y"}},
		{name: "tone.s16le", args: []string{"-f", "lavfi", "-i", "sine=frequency=440:duration=0.03", "-f", "s16le", "-acodec", "pcm_s16le", "-ac", "1", "-ar", "24000", "-y"}},
	}
	dir := t.TempDir()
	encoder := New("ffmpeg", 64)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name)
			args := append(append([]string{"-hide_banner", "-loglevel", "error"}, tc.args...), path)
			if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
				t.Fatalf("generate %s: %v: %s", tc.name, err, out)
			}
			input, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", tc.name, err)
			}
			pcm, rate, err := encoder.DecodeToPCM(input)
			if err != nil {
				t.Fatalf("decode %s: %v", tc.name, err)
			}
			if len(pcm) == 0 || rate != sampleRate {
				t.Fatalf("unexpected pcm len/rate for %s: %d/%d", tc.name, len(pcm), rate)
			}
		})
	}
}
