# Audiobook-Studio

Audiobook-Studio converts `.txt` files into MP3 audiobooks with an OpenAI-compatible TTS endpoint.

## Current Implementation

- Go core packages for config, text splitting, TTS requests, ffmpeg decoding/merging, and batch orchestration.
- CLI entrypoint for file or directory conversion.
- Wails v2 GUI scaffold with config, batch, pause/resume/cancel, single-text conversion bindings, and a dark static frontend.
- Unit tests for config, splitter, TTS, batch, and ffmpeg integration.

## CLI Usage

Create a default config:

```sh
go run .
```

Convert one file:

```sh
go run . -input ./book.txt -token "$OPENAI_API_KEY"
```

Convert a directory of `.txt` files:

```sh
go run . -input ./texts -output-dir ./output -concurrency 2
```

## Config

The default `config.json` includes:

- `API_BASE_URL`: OpenAI-compatible speech endpoint.
- `API_TOKEN`: bearer token.
- `MODEL`: TTS model name.
- `VOICE_JSON`: JSON merged into every TTS request, for example `{"voice":"alloy","response_format":"mp3"}`.
- `SPLIT_THRESHOLD`: approximate chunk size in runes.
- `OUTPUT_DIR`: MP3 output directory.
- `CONCURRENCY`: parallel TTS requests per file.
- `FFMPEG_PATH`: ffmpeg executable.

## GUI

The GUI lives in `GUI/` and follows Wails v2 conventions.

```sh
cd GUI
npm run build
go mod tidy
wails dev
```

`go mod tidy` and `wails dev` require network access the first time because Wails dependencies must be downloaded.

This repository is intended to use GitHub Actions as the build authority. CI downloads Wails dependencies, builds the frontend, generates Wails bindings, and runs platform builds on Windows, macOS, and Linux.

GUI builds use a trimmed static FFmpeg/libmp3lame CGO build produced by `scripts/build-ffmpeg-static.sh` and compile the app with the `gui_ffmpeg_cgo` tag on Windows, macOS, and Linux.

The static FFmpeg configuration decodes common TTS outputs including WAV, MP3, Opus/Ogg, AAC/M4A, and raw PCM. The audiobook output encoder is intentionally fixed to MP3 for now.

The application icons live under `GUI/build/`:

- `GUI/build/appicon.png`
- `GUI/build/icons/icon16.png` through `GUI/build/icons/icon512.png`

## Tests

```sh
GOCACHE="$PWD/.cache/go-build" go test ./...
```

The ffmpeg integration test skips automatically when `ffmpeg` is not installed.
