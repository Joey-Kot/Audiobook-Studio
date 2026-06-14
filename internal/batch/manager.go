package batch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"audiobook-studio/internal/audio/ffmpeg"
	"audiobook-studio/internal/config"
	"audiobook-studio/internal/splitter"
	"audiobook-studio/internal/tts"
)

// Status describes file-level processing state.
type Status string

const (
	StatusQueued    Status = "queued"
	StatusSplitting Status = "splitting"
	StatusTTS       Status = "tts"
	StatusMerging   Status = "merging"
	StatusDone      Status = "done"
	StatusCanceled  Status = "canceled"
	StatusError     Status = "error"
)

// BatchProgress is emitted as work advances.
type BatchProgress struct {
	FileIndex int    `json:"fileIndex"`
	FileName  string `json:"fileName"`
	CharCount int    `json:"charCount"`
	Percent   int    `json:"percent"`
	Status    Status `json:"status"`
	Message   string `json:"message,omitempty"`
	Output    string `json:"output,omitempty"`
}

// Synthesizer is the TTS dependency used by Manager.
type Synthesizer interface {
	SynthesizeContext(ctx context.Context, text string, voiceJSON string) ([]byte, error)
}

// AudioEncoder is the ffmpeg dependency used by Manager.
type AudioEncoder interface {
	DecodeToPCM(input []byte) ([]byte, int, error)
	MergeToMP3(segments [][]byte, outputPath string) error
}

// Manager coordinates batch text-to-audiobook jobs.
type Manager struct {
	cfg         config.Config
	synth       Synthesizer
	encoder     AudioEncoder
	progress    func(BatchProgress)
	cancelMu    sync.Mutex
	cancel      context.CancelFunc
	runID       int64
	paused      atomic.Bool
	pauseMu     sync.Mutex
	pauseNotify chan struct{}
}

// NewManager creates a Manager from config and optional dependencies.
func NewManager(cfg config.Config, synth Synthesizer, encoder AudioEncoder, progress func(BatchProgress)) *Manager {
	if synth == nil {
		synth = tts.New(cfg.APIBaseURL, cfg.APIToken, cfg.Model, cfg.RequestTimeout)
	}
	if encoder == nil {
		enc := ffmpeg.New(cfg.FFmpegPath, cfg.OutputBitrateKB)
		encoder = enc
	}
	return &Manager{
		cfg:         cfg,
		synth:       synth,
		encoder:     encoder,
		progress:    progress,
		pauseNotify: make(chan struct{}),
	}
}

// Start processes the provided files in order.
func (m *Manager) Start(ctx context.Context, files []string) error {
	return m.StartWithNames(ctx, files, nil)
}

// StartWithNames processes the provided files in order with optional output base names.
func (m *Manager) StartWithNames(ctx context.Context, files []string, outputNames map[string]string) error {
	if len(files) == 0 {
		return fmt.Errorf("no input files")
	}
	if err := config.Validate(&m.cfg); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	m.cancelMu.Lock()
	m.runID++
	runID := m.runID
	m.cancel = cancel
	m.cancelMu.Unlock()
	defer func() {
		m.cancelMu.Lock()
		if m.runID == runID {
			m.cancel = nil
		}
		m.cancelMu.Unlock()
		cancel()
	}()

	outputDir, err := config.EnsureOutputDir(m.cfg)
	if err != nil {
		return err
	}
	for i, file := range files {
		if err := ctx.Err(); err != nil {
			m.emit(BatchProgress{FileIndex: i, FileName: filepath.Base(file), Status: StatusCanceled, Message: err.Error()})
			return err
		}
		if err := m.processFile(ctx, i, file, outputDir, outputNames[file]); err != nil {
			if ctx.Err() != nil {
				m.emit(BatchProgress{FileIndex: i, FileName: filepath.Base(file), Status: StatusCanceled, Message: ctx.Err().Error()})
				return ctx.Err()
			}
			m.emit(BatchProgress{FileIndex: i, FileName: filepath.Base(file), Status: StatusError, Message: err.Error()})
			return err
		}
	}
	return nil
}

// Cancel stops the current batch.
func (m *Manager) Cancel() {
	m.cancelMu.Lock()
	cancel := m.cancel
	m.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Pause pauses between TTS work units.
func (m *Manager) Pause() {
	m.paused.Store(true)
}

// Resume resumes paused work.
func (m *Manager) Resume() {
	if m.paused.Swap(false) {
		m.pauseMu.Lock()
		ch := m.pauseNotify
		m.pauseNotify = make(chan struct{})
		m.pauseMu.Unlock()
		close(ch)
	}
}

func (m *Manager) processFile(ctx context.Context, fileIndex int, path string, outputDir string, outputName string) error {
	name := filepath.Base(path)
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(string(raw))
	charCount := utf8.RuneCountInString(text)
	m.emit(BatchProgress{FileIndex: fileIndex, FileName: name, CharCount: charCount, Percent: 1, Status: StatusSplitting})
	parts := splitter.Split(text, m.cfg.SplitThreshold)
	if len(parts) == 0 {
		return fmt.Errorf("%s has no text", name)
	}

	encoded := make([][]byte, len(parts))
	m.emit(BatchProgress{FileIndex: fileIndex, FileName: name, CharCount: charCount, Percent: 5, Status: StatusTTS, Message: fmt.Sprintf("%d chunks", len(parts))})

	concurrency := m.cfg.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	type job struct {
		index int
		text  string
	}
	jobs := make(chan job)
	errCh := make(chan error, 1)
	var completed atomic.Int64
	var wg sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if err := m.waitIfPaused(ctx); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
				audio, err := m.synth.SynthesizeContext(ctx, j.text, m.cfg.VoiceJSON)
				if err != nil {
					select {
					case errCh <- fmt.Errorf("chunk %d: %w", j.index+1, err):
					default:
					}
					return
				}
				pcm, _, err := m.encoder.DecodeToPCM(audio)
				if err != nil {
					select {
					case errCh <- fmt.Errorf("chunk %d decode: %w", j.index+1, err):
					default:
					}
					return
				}
				encoded[j.index] = pcm
				done := int(completed.Add(1))
				percent := 5 + int(float64(done)/float64(len(parts))*80)
				m.emit(BatchProgress{FileIndex: fileIndex, FileName: name, CharCount: charCount, Percent: percent, Status: StatusTTS, Message: fmt.Sprintf("%d/%d chunks", done, len(parts))})
			}
		}()
	}

sendLoop:
	for i, part := range parts {
		select {
		case <-ctx.Done():
			break sendLoop
		case err := <-errCh:
			close(jobs)
			wg.Wait()
			return err
		case jobs <- job{index: i, text: part}:
		}
	}
	close(jobs)
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	baseName := strings.TrimSuffix(name, filepath.Ext(name))
	if outputName != "" {
		baseName = strings.TrimSuffix(filepath.Base(outputName), filepath.Ext(outputName))
	}
	outputPath := filepath.Join(outputDir, baseName+".mp3")
	m.emit(BatchProgress{FileIndex: fileIndex, FileName: name, CharCount: charCount, Percent: 90, Status: StatusMerging})
	if err := m.encoder.MergeToMP3(encoded, outputPath); err != nil {
		return err
	}
	m.emit(BatchProgress{FileIndex: fileIndex, FileName: name, CharCount: charCount, Percent: 100, Status: StatusDone, Output: outputPath})
	return nil
}

func (m *Manager) waitIfPaused(ctx context.Context) error {
	for m.paused.Load() {
		m.pauseMu.Lock()
		ch := m.pauseNotify
		m.pauseMu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ch:
		}
	}
	return nil
}

func (m *Manager) emit(progress BatchProgress) {
	if m.progress != nil {
		m.progress(progress)
	}
}

// DiscoverTextFiles expands files/directories into a sorted list of .txt files.
func DiscoverTextFiles(paths []string) ([]string, error) {
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			files = append(files, path)
			continue
		}
		err = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".txt") {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}
