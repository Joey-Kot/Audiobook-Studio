package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"audiobook-studio/internal/batch"
	"audiobook-studio/internal/config"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type ConfigPayload struct {
	Path string        `json:"path"`
	JSON string        `json:"json"`
	Data config.Config `json:"data"`
}

type AppState struct {
	Running bool                  `json:"running"`
	Paused  bool                  `json:"paused"`
	Files   []batch.BatchProgress `json:"files"`
	Config  config.Config         `json:"config"`
}

type App struct {
	ctx        context.Context
	mu         sync.Mutex
	configPath string
	cfg        config.Config
	manager    *batch.Manager
	cancel     context.CancelFunc
	runID      int64
	running    bool
	paused     bool
	files      map[int]batch.BatchProgress
}

func NewApp() *App {
	return &App{files: map[int]batch.BatchProgress{}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	path, err := defaultConfigPath()
	if err != nil {
		a.emitError(err)
		return
	}
	a.configPath = path
	cfg, err := ensureConfig(path)
	if err != nil {
		a.emitError(err)
		return
	}
	a.cfg = cfg
	wailsruntime.EventsEmit(ctx, "app:state", a.GetState())
}

func (a *App) shutdown(ctx context.Context) {
	a.CancelBatch()
}

func (a *App) LoadConfig() (ConfigPayload, error) {
	cfg, err := ensureConfig(a.configPath)
	if err != nil {
		return ConfigPayload{}, err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return ConfigPayload{}, err
	}
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	return ConfigPayload{Path: a.configPath, JSON: string(raw), Data: cfg}, nil
}

func (a *App) SaveConfig(raw string) (ConfigPayload, error) {
	var cfg config.Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ConfigPayload{}, err
	}
	if err := config.Save(a.configPath, cfg); err != nil {
		return ConfigPayload{}, err
	}
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	return a.LoadConfig()
}

func (a *App) StartBatch(paths []string) error {
	files, err := batch.DiscoverTextFiles(paths)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .txt files found")
	}
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("batch is already running")
	}
	cfg := a.cfg
	ctx, cancel := context.WithCancel(context.Background())
	a.runID++
	runID := a.runID
	a.cancel = cancel
	a.running = true
	a.paused = false
	a.files = map[int]batch.BatchProgress{}
	manager := batch.NewManager(cfg, nil, nil, a.onProgress)
	a.manager = manager
	a.mu.Unlock()

	go func() {
		err := manager.Start(ctx, files)
		a.mu.Lock()
		a.running = false
		a.paused = false
		if a.runID == runID {
			a.cancel = nil
		}
		a.mu.Unlock()
		if err != nil {
			a.emitError(err)
		}
		a.emitState()
	}()
	a.emitState()
	return nil
}

func (a *App) ConvertText(text string, outputName string) error {
	tmp, err := os.CreateTemp("", "audiobook-studio-*.txt")
	if err != nil {
		return err
	}
	if _, err := tmp.WriteString(text); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	files := []string{tmp.Name()}
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("batch is already running")
	}
	cfg := a.cfg
	ctx, cancel := context.WithCancel(context.Background())
	a.runID++
	runID := a.runID
	a.cancel = cancel
	a.running = true
	a.paused = false
	a.files = map[int]batch.BatchProgress{}
	manager := batch.NewManager(cfg, nil, nil, a.onProgress)
	a.manager = manager
	a.mu.Unlock()

	go func() {
		outputNames := map[string]string{}
		if outputName != "" {
			outputNames[tmp.Name()] = outputName
		}
		err := manager.StartWithNames(ctx, files, outputNames)
		_ = os.Remove(tmp.Name())
		a.mu.Lock()
		a.running = false
		a.paused = false
		if a.runID == runID {
			a.cancel = nil
		}
		a.mu.Unlock()
		if err != nil {
			a.emitError(err)
		}
		a.emitState()
	}()
	a.emitState()
	return nil
}

func (a *App) CancelBatch() {
	a.mu.Lock()
	manager := a.manager
	cancel := a.cancel
	a.mu.Unlock()
	if manager != nil {
		manager.Cancel()
	}
	if cancel != nil {
		cancel()
	}
}

func (a *App) PauseBatch() {
	a.mu.Lock()
	manager := a.manager
	a.paused = true
	a.mu.Unlock()
	if manager != nil {
		manager.Pause()
	}
	a.emitState()
}

func (a *App) ResumeBatch() {
	a.mu.Lock()
	manager := a.manager
	a.paused = false
	a.mu.Unlock()
	if manager != nil {
		manager.Resume()
	}
	a.emitState()
}

func (a *App) GetState() AppState {
	a.mu.Lock()
	defer a.mu.Unlock()
	files := make([]batch.BatchProgress, 0, len(a.files))
	for i := 0; i < len(a.files); i++ {
		if p, ok := a.files[i]; ok {
			files = append(files, p)
		}
	}
	return AppState{Running: a.running, Paused: a.paused, Files: files, Config: a.cfg}
}

func (a *App) SelectOutputDir() (string, error) {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select output directory",
	})
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", nil
	}
	a.mu.Lock()
	a.cfg.OutputDir = dir
	cfg := a.cfg
	a.mu.Unlock()
	if err := config.Save(a.configPath, cfg); err != nil {
		return "", err
	}
	return dir, nil
}

func (a *App) SelectTextFiles() ([]string, error) {
	paths, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select text files",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Text files (*.txt)", Pattern: "*.txt"},
		},
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func (a *App) SelectInputDirectory() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select input directory",
	})
}

func (a *App) onProgress(progress batch.BatchProgress) {
	a.mu.Lock()
	a.files[progress.FileIndex] = progress
	a.mu.Unlock()
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "batch:progress", progress)
	}
	a.emitState()
}

func (a *App) emitState() {
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "app:state", a.GetState())
	}
}

func (a *App) emitError(err error) {
	if a.ctx != nil && err != nil {
		wailsruntime.EventsEmit(a.ctx, "app:error", err.Error())
	}
}

func defaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Audiobook-Studio", "config.json"), nil
}

func ensureConfig(path string) (config.Config, error) {
	if _, err := os.Stat(path); err == nil {
		return config.Load(path)
	} else if !os.IsNotExist(err) {
		return config.Config{}, err
	}
	cfg := config.DefaultConfig()
	if err := config.Save(path, cfg); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}
