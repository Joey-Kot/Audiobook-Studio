package ffmpeg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

const sampleRate = 24000

// Encoder wraps the ffmpeg executable and output encoding settings.
type Encoder struct {
	Path            string
	OutputBitrateKB int
}

// New creates an Encoder.
func New(path string, bitrateKB int) Encoder {
	if path == "" {
		path = "ffmpeg"
	}
	if bitrateKB <= 0 {
		bitrateKB = 128
	}
	return Encoder{Path: path, OutputBitrateKB: bitrateKB}
}

// DecodeToPCM decodes arbitrary audio bytes into mono s16le PCM at a stable sample rate.
func DecodeToPCM(input []byte) ([]byte, int, error) {
	return New("", 0).DecodeToPCM(input)
}

// DecodeToPCM decodes arbitrary audio bytes into mono s16le PCM at a stable sample rate.
func (e Encoder) DecodeToPCM(input []byte) ([]byte, int, error) {
	if len(input) == 0 {
		return nil, 0, fmt.Errorf("input audio is empty")
	}
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

// MergeToMP3 concatenates PCM segments and writes a single MP3.
func MergeToMP3(segments [][]byte, outputPath string) error {
	return New("", 0).MergeToMP3(segments, outputPath)
}

// MergeToMP3 concatenates PCM segments and writes a single MP3.
func (e Encoder) MergeToMP3(segments [][]byte, outputPath string) error {
	if len(segments) == 0 {
		return fmt.Errorf("no audio segments to merge")
	}
	if outputPath == "" {
		return fmt.Errorf("output path is empty")
	}
	var pcm bytes.Buffer
	for i, segment := range segments {
		if len(segment) == 0 {
			return fmt.Errorf("segment %d is empty", i)
		}
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
