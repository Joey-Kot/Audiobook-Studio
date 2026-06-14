package config

import "flag"

// FlagValues tracks CLI overrides.
type FlagValues struct {
	APIBaseURL         string
	APIBaseURLSet      bool
	APIToken           string
	APITokenSet        bool
	Model              string
	ModelSet           bool
	VoiceJSON          string
	VoiceJSONSet       bool
	SplitThreshold     int
	SplitThresholdSet  bool
	OutputDir          string
	OutputDirSet       bool
	Concurrency        int
	ConcurrencySet     bool
	RequestTimeout     int
	RequestTimeoutSet  bool
	FFmpegPath         string
	FFmpegPathSet      bool
	OutputBitrateKB    int
	OutputBitrateKBSet bool
	InputPath          string
	OutputPath         string
}

// BindFlags registers CLI flags on fs.
func BindFlags(fs *flag.FlagSet) *FlagValues {
	fv := &FlagValues{}
	fs.Func("api-base-url", "OpenAI-compatible TTS endpoint URL", func(v string) error {
		fv.APIBaseURL, fv.APIBaseURLSet = v, true
		return nil
	})
	fs.Func("token", "Bearer API token", func(v string) error {
		fv.APIToken, fv.APITokenSet = v, true
		return nil
	})
	fs.Func("model", "TTS model name", func(v string) error {
		fv.Model, fv.ModelSet = v, true
		return nil
	})
	fs.Func("voice-json", "JSON merged into the TTS request body", func(v string) error {
		fv.VoiceJSON, fv.VoiceJSONSet = v, true
		return nil
	})
	fs.Func("output-dir", "output directory", func(v string) error {
		fv.OutputDir, fv.OutputDirSet = v, true
		return nil
	})
	fs.Func("ffmpeg", "ffmpeg executable path", func(v string) error {
		fv.FFmpegPath, fv.FFmpegPathSet = v, true
		return nil
	})
	fs.IntVar(&fv.SplitThreshold, "split-threshold", 0, "text split threshold in runes")
	fs.IntVar(&fv.Concurrency, "concurrency", 0, "parallel TTS request count")
	fs.IntVar(&fv.RequestTimeout, "request-timeout", 0, "request timeout seconds")
	fs.IntVar(&fv.OutputBitrateKB, "bitrate", 0, "MP3 output bitrate in kbps")
	fs.StringVar(&fv.InputPath, "input", "", "input .txt file or directory")
	fs.StringVar(&fv.OutputPath, "output", "", "single output MP3 path")
	return fv
}

// MarkSet records integer flags that were explicitly provided.
func MarkSet(fs *flag.FlagSet, fv *FlagValues) {
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "split-threshold":
			fv.SplitThresholdSet = true
		case "concurrency":
			fv.ConcurrencySet = true
		case "request-timeout":
			fv.RequestTimeoutSet = true
		case "bitrate":
			fv.OutputBitrateKBSet = true
		}
	})
}

// ApplyFlags overlays explicitly set flags onto cfg.
func ApplyFlags(cfg *Config, fv *FlagValues) {
	if fv == nil {
		return
	}
	if fv.APIBaseURLSet {
		cfg.APIBaseURL = fv.APIBaseURL
	}
	if fv.APITokenSet {
		cfg.APIToken = fv.APIToken
	}
	if fv.ModelSet {
		cfg.Model = fv.Model
	}
	if fv.VoiceJSONSet {
		cfg.VoiceJSON = fv.VoiceJSON
	}
	if fv.SplitThresholdSet {
		cfg.SplitThreshold = fv.SplitThreshold
	}
	if fv.OutputDirSet {
		cfg.OutputDir = fv.OutputDir
	}
	if fv.ConcurrencySet {
		cfg.Concurrency = fv.Concurrency
	}
	if fv.RequestTimeoutSet {
		cfg.RequestTimeout = fv.RequestTimeout
	}
	if fv.FFmpegPathSet {
		cfg.FFmpegPath = fv.FFmpegPath
	}
	if fv.OutputBitrateKBSet {
		cfg.OutputBitrateKB = fv.OutputBitrateKB
	}
}
