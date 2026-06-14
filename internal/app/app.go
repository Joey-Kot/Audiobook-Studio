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

package app

import (
	"context"
	"fmt"
	"path/filepath"

	"audiobook-studio/internal/batch"
	"audiobook-studio/internal/config"
)

// RunFileMode converts one file or every .txt file inside a directory.
func RunFileMode(ctx context.Context, cfg config.Config, inputPath string, outputPath string) error {
	if inputPath == "" {
		return fmt.Errorf("input path is required")
	}
	files, err := batch.DiscoverTextFiles([]string{inputPath})
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .txt files found")
	}
	if outputPath != "" && len(files) == 1 {
		cfg.OutputDir = filepath.Dir(outputPath)
	}
	manager := batch.NewManager(cfg, nil, nil, func(progress batch.BatchProgress) {
		fmt.Printf("[%s] %s %d%% %s\n", progress.Status, progress.FileName, progress.Percent, progress.Message)
	})
	if err := manager.Start(ctx, files); err != nil {
		return err
	}
	if outputPath != "" && len(files) == 1 {
		// OutputPath is accepted by the CLI for compatibility, but the batch engine
		// currently names output after the source file inside OutputDir.
		fmt.Printf("[done] output directory: %s\n", cfg.OutputDir)
	}
	return nil
}
