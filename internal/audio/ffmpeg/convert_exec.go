//go:build !gui_ffmpeg_cgo

package ffmpeg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func decodeToPCM(e Encoder, input []byte) ([]byte, int, error) {
	cmd := exec.Command(e.Path, "-hide_banner", "-loglevel", "error", "-i", "pipe:0", "-f", "s16le", "-acodec", "pcm_s16le", "-ac", "1", "-ar", strconv.Itoa(sampleRate), "pipe:1")
	cmd.Stdin = bytes.NewReader(input)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, 0, fmt.Errorf("ffmpeg decode failed: %w: %s", err, stderr.String())
	}
	return out.Bytes(), sampleRate, nil
}

func mergeToMP3(e Encoder, segments [][]byte, outputPath string) error {
	var pcm bytes.Buffer
	for _, segment := range segments {
		pcm.Write(segment)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	cmd := exec.Command(e.Path, "-hide_banner", "-loglevel", "error", "-f", "s16le", "-acodec", "pcm_s16le", "-ac", "1", "-ar", strconv.Itoa(sampleRate), "-i", "pipe:0", "-codec:a", "libmp3lame", "-b:a", fmt.Sprintf("%dk", e.OutputBitrateKB), "-y", outputPath)
	cmd.Stdin = bytes.NewReader(pcm.Bytes())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg encode failed: %w: %s", err, stderr.String())
	}
	return nil
}
