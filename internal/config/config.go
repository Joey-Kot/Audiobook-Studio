package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds all user-editable settings for CLI and GUI workflows.
type Config struct {
	APIBaseURL      string `json:"API_BASE_URL"`
	APIToken        string `json:"API_TOKEN"`
	Model           string `json:"MODEL"`
	VoiceJSON       string `json:"VOICE_JSON"`
	SplitThreshold  int    `json:"SPLIT_THRESHOLD"`
	OutputDir       string `json:"OUTPUT_DIR"`
	Concurrency     int    `json:"CONCURRENCY"`
	RequestTimeout  int    `json:"REQUEST_TIMEOUT"`
	FFmpegPath      string `json:"FFMPEG_PATH"`
	OutputBitrateKB int    `json:"OUTPUT_BITRATE_KB"`
}

// DefaultConfig returns conservative defaults that work with OpenAI-compatible TTS APIs.
func DefaultConfig() Config {
	return Config{
		APIBaseURL:      "https://api.openai.com/v1/audio/speech",
		APIToken:        "",
		Model:           "gpt-4o-mini-tts",
		VoiceJSON:       `{"voice":"alloy","response_format":"mp3"}`,
		SplitThreshold:  1200,
		OutputDir:       "output",
		Concurrency:     2,
		RequestTimeout:  120,
		FFmpegPath:      "ffmpeg",
		OutputBitrateKB: 128,
	}
}

// Load reads a JSON config file and overlays it on defaults.
func Load(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Save writes cfg as indented JSON, creating parent directories when needed.
func Save(path string, cfg Config) error {
	if path == "" {
		return fmt.Errorf("config path is empty")
	}
	if err := Validate(&cfg); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0600)
}

// SaveDefault writes the default config to path.
func SaveDefault(path string) error {
	return Save(path, DefaultConfig())
}

// Validate checks the user-editable settings.
func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if strings.TrimSpace(cfg.APIBaseURL) == "" {
		return fmt.Errorf("API_BASE_URL is required")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return fmt.Errorf("MODEL is required")
	}
	if strings.TrimSpace(cfg.VoiceJSON) == "" {
		return fmt.Errorf("VOICE_JSON is required")
	}
	var voice map[string]any
	if err := json.Unmarshal([]byte(cfg.VoiceJSON), &voice); err != nil {
		return fmt.Errorf("VOICE_JSON must be valid JSON: %w", err)
	}
	if cfg.SplitThreshold < 80 {
		return fmt.Errorf("SPLIT_THRESHOLD must be at least 80")
	}
	if cfg.Concurrency < 1 {
		return fmt.Errorf("CONCURRENCY must be at least 1")
	}
	if cfg.RequestTimeout < 1 {
		return fmt.Errorf("REQUEST_TIMEOUT must be at least 1")
	}
	if cfg.OutputBitrateKB < 16 {
		return fmt.Errorf("OUTPUT_BITRATE_KB must be at least 16")
	}
	if strings.TrimSpace(cfg.OutputDir) == "" {
		return fmt.Errorf("OUTPUT_DIR is required")
	}
	if strings.TrimSpace(cfg.FFmpegPath) == "" {
		return fmt.Errorf("FFMPEG_PATH is required")
	}
	return nil
}

// EnsureOutputDir creates and returns the absolute output directory.
func EnsureOutputDir(cfg Config) (string, error) {
	if strings.TrimSpace(cfg.OutputDir) == "" {
		return "", fmt.Errorf("OUTPUT_DIR is required")
	}
	abs, err := filepath.Abs(cfg.OutputDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return "", err
	}
	return abs, nil
}
