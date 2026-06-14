//go:build !gui_ffmpeg_cgo

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
