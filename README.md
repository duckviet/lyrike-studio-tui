# lyrike-studio-tui

Terminal UI and Go HTTP backend for the Lyrics Studio workflow. One binary
does both jobs:

- `lyrike-studio-tui serve` — runs the backend (fetch, peaks, transcription,
  lrclib proxy, draft storage).
- `lyrike-studio-tui` — runs the TUI and talks to that backend over HTTP.

Drafts live on the server (`LYRIKE_DRAFT_DIR`); the TUI reads and writes them
through `/local-api/projects/*` via `RemoteStore`.

## Table of contents

- [Prerequisites](#prerequisites)
- [Quick start](#quick-start)
- [Demo mode](#demo-mode)
- [Docker](#docker)
- [Deploy on Fly.io](#deploy-on-flyio)
- [Environment variables](#environment-variables)
- [CLI flags](#cli-flags)
- [Development checks](#development-checks)
- [Architecture](#architecture)
- [Docs](#docs)

## Prerequisites

- **Go 1.25**
- **ffmpeg** — peak computation and audio range streaming.
- **yt-dlp** — media info and audio download (standalone binary, no Python
  runtime required).

```bash
# Debian/Ubuntu
sudo apt-get install ffmpeg

# yt-dlp standalone binary
sudo curl -L -o /usr/local/bin/yt-dlp https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp
sudo chmod +x /usr/local/bin/yt-dlp
```

Transcription also needs an `OPENAI_API_KEY` (whisper-1 only).

## Quick start

### 1. Run the backend

```bash
go run ./cmd/lyrike-studio-tui serve --port 8080
```

Override the cache/draft location if you do not want `./.cache`:

```bash
go run ./cmd/lyrike-studio-tui serve --port 8080 --cache-dir ./data
# drafts live under ./data/drafts
```

### 2. Run the TUI against it

```bash
go run ./cmd/lyrike-studio-tui --backend http://127.0.0.1:8080 --video-id <youtube-id>
```

You can also start the TUI without `--video-id` and press `Ctrl-O` to open the
fetch modal, then paste a YouTube URL or video ID (for example
`https://www.youtube.com/watch?v=P0N0h_EOS-c`) and press Enter.

The TUI plays audio natively through beep using the backend's
`/local-api/audio/{id}` URL. If no audio is available it falls back to a fake
player so the editor still works.

## Demo mode

`--demo` launches the TUI with local fixtures and **ignores `--backend`**. Use
it for quick UI checks without running the server:

```bash
go run ./cmd/lyrike-studio-tui --demo
go run ./cmd/lyrike-studio-tui --demo --backend-fixture
```

`--demo --backend-fixture` is the deterministic end-to-end QA harness: it
renders fetch, playback, tap-sync, draft save, publish success, and quit
readiness.

## Docker

```bash
docker build -t lyrike-studio-tui .
docker run --rm -p 8080:8080 -v lyrike-cache:/data lyrike-studio-tui
```

The image is multi-stage: `golang:1.25` builds the binary, then
`debian:bookworm-slim` provides `ffmpeg`, `yt-dlp`, and the binary. Cache and
drafts persist under `/data` (set `LYRIKE_CACHE_DIR=/data/.cache` by default).

## Deploy on Fly.io

```bash
fly deploy
fly secrets set OPENAI_API_KEY=sk-... YOUTUBE_COOKIES=...
```

`fly.toml` is preconfigured for app `lyrike-studio-tui`, region `nrt`,
internal port `8080`, and a 1 CPU / 1 GB VM.

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `OPENAI_API_KEY` | _empty_ | Required for transcription and lyric refinement. |
| `OPENAI_TRANSCRIPTION_MODEL` | `whisper-1` | OpenAI transcription model. |
| `YOUTUBE_COOKIES` | _empty_ | Base64 or plain `cookies.txt` content; written to `/tmp/yt_cookies.txt` at startup. |
| `LYRIKE_CACHE_DIR` | `./.cache` | Media, audio, peaks, transcripts root. |
| `LYRIKE_DRAFT_DIR` | `./.cache/drafts` | Server-side draft storage. |
| `PORT` | `8080` | Backend listen port (overridden by `--port`). |
| `FRONTEND_URL` | _empty_ | Extra CORS origin (comma-separated for multiple). |
| `RATE_LIMIT_PER_MINUTE` | `60` | General per-IP rate limit. |
| `RATE_LIMIT_TRANSCRIBE_PER_MINUTE` | `5` | Per-IP transcription rate limit. |
| `ENABLE_LYRICS_REFINEMENT` | `false` | Enable GPT-4o-mini lyric refinement. |

## CLI flags

### TUI flags

| Flag | Default | Purpose |
|---|---|---|
| `--backend` | `http://127.0.0.1:8000` | Backend URL. |
| `--video-id` | _empty_ | YouTube video ID to sync. |
| `--project` | _empty_ | Project ID for draft save/load. |
| `--url` | _empty_ | Source media URL to sync. |
| `--audio` | _empty_ | Local audio file to play natively via beep. |
| `--import` | _empty_ | Path to a `.lrc` or `.txt` lyric file to import at startup. |
| `--demo` | `false` | Launch with local fixtures. |
| `--backend-fixture` | `false` | Use deterministic fixture data in demo mode. |
| `--version` | `false` | Print version and exit. |

### `serve` flags

| Flag | Default | Purpose |
|---|---|---|
| `--port` | `8080` | HTTP listen port. |
| `--cache-dir` | _empty_ | Override cache directory; drafts move under `<cache-dir>/drafts`. |

## Development checks

```bash
go test ./...
go test -race ./...
go vet ./...
gofmt -l .
go run ./cmd/lyrike-studio-tui --demo --backend-fixture
```

## Architecture

- **Backend** (`internal/server/`): chi v5 router, per-IP rate limiting, CORS,
  disk-only cache, yt-dlp + ffmpeg shell-out, OpenAI whisper-1 transcription,
  lrclib proxy, and server-side draft store. No WhisperX, Demucs, or CDN/R2.
- **TUI** (`internal/tui/`): Bubble Tea v2 UI; real mode uses `RemoteStore`
  over HTTP for drafts and the backend for fetch/peaks/publish.
- **Single binary**: `serve` starts the backend; no subcommand starts the TUI.

See [`docs/server.md`](docs/server.md) for the full route map.

## Docs

- [`docs/server.md`](docs/server.md) — backend architecture and route map.
- [`docs/keybindings.md`](docs/keybindings.md) — TUI keybindings.
- [`docs/troubleshooting.md`](docs/troubleshooting.md) — common fixes.
- [`docs/implementation.md`](docs/implementation.md) — implementation notes.
