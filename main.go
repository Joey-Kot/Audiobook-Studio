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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"audiobook-studio/internal/app"
	"audiobook-studio/internal/config"
)

func usage() {
	name := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, `Usage: %s [options]

Audiobook-Studio converts text files into MP3 audiobooks with an OpenAI-compatible TTS API.

Options:
  -config <path>          Config JSON path. Defaults to ./config.json.
  -input <path>           Input .txt file or directory.
  -output-dir <path>      Output directory.
  -api-base-url <url>     OpenAI-compatible speech endpoint.
  -token <token>          Bearer API token.
  -model <name>           TTS model.
  -voice-json <json>      JSON merged into the TTS request body.
  -split-threshold <n>    Text split threshold in runes.
  -concurrency <n>        Parallel TTS request count.
  -request-timeout <n>    Request timeout seconds.
  -ffmpeg <path>          ffmpeg executable path.
  -bitrate <n>            MP3 output bitrate in kbps.

If config.json does not exist and no -input is provided, a default config is created.
`, name)
}

func main() {
	flag.Usage = usage
	configPath := flag.String("config", "config.json", "config JSON path")
	help := flag.Bool("h", false, "show help")
	help2 := flag.Bool("help", false, "show help")
	fv := config.BindFlags(flag.CommandLine)
	flag.Parse()
	config.MarkSet(flag.CommandLine, fv)

	if *help || *help2 {
		usage()
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		if os.IsNotExist(err) && fv.InputPath == "" {
			if err := config.SaveDefault(*configPath); err != nil {
				fmt.Fprintf(os.Stderr, "failed to create default config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("default config created at %s. Edit it and rerun with -input.\n", *configPath)
			return
		}
		if os.IsNotExist(err) {
			cfg = config.DefaultConfig()
		} else {
			fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
			os.Exit(1)
		}
	}
	config.ApplyFlags(&cfg, fv)
	if err := config.Validate(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		os.Exit(1)
	}
	if fv.InputPath == "" {
		usage()
		return
	}
	if err := app.RunFileMode(context.Background(), cfg, fv.InputPath, fv.OutputPath); err != nil {
		fmt.Fprintf(os.Stderr, "conversion failed: %v\n", err)
		os.Exit(1)
	}
}
