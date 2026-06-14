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

import "fmt"

const sampleRate = 24000

// Encoder wraps audio conversion settings.
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
	pcm, rate, err := decodeToPCM(e, input)
	if err == nil {
		return pcm, rate, nil
	}
	if raw, rawErr := rawPCM(input); rawErr == nil {
		return raw, sampleRate, nil
	}
	return nil, 0, err
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
	for i, segment := range segments {
		if len(segment) == 0 {
			return fmt.Errorf("segment %d is empty", i)
		}
	}
	return mergeToMP3(e, segments, outputPath)
}

func rawPCM(input []byte) ([]byte, error) {
	if len(input) < 2 || len(input)%2 != 0 {
		return nil, fmt.Errorf("input is not valid s16le pcm")
	}
	return append([]byte(nil), input...), nil
}
