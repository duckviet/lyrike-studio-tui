# Server

The Go backend is a single binary (`lyrike-studio-tui serve`) that replaces the previous Python FastAPI backend. It lives in `internal/server/` and is wired into the same CLI entrypoint as the TUI.

## Architecture

- **Single binary**: `lyrike-studio-tui serve` starts the HTTP server; `lyrike-studio-tui` (without arguments) starts the TUI.
- **Router**: [chi v5](https://github.com/go-chi/chi) with CORS and per-IP token-bucket rate limiting.
- **Config**: Environment variables only (`OPENAI_API_KEY`, `OPENAI_TRANSCRIPTION_MODEL`, `YOUTUBE_COOKIES`, `LYRIKE_CACHE_DIR`, `LYRIKE_DRAFT_DIR`, `PORT`, etc.). Optional `.env` loading via `godotenv`.
- **Media**: Shells out to `yt-dlp` (info + audio download) and `ffmpeg` (peak computation, range streaming).
- **Transcription**: OpenAI whisper-1 via `openai-go`; optional GPT-4o-mini lyric refinement.
- **Storage**: Disk-only cache (`{cache}/media`, `{cache}/audio`, `{cache}/peaks`, `{cache}/transcripts`, `{cache}/drafts`). No CDN or object storage.
- **Drafts**: Server-side file store under `LYRIKE_DRAFT_DIR`; TUI reads/writes drafts over HTTP (`/local-api/projects/*`).

## Route map

### TUI contract

| Method | Path | Description |
|--------|------|-------------|
| POST | `/local-api/fetch` | Fetch or cache video metadata and audio. |
| POST | `/local-api/transcribe` | Queue a transcription job (returns `queued`/`running`/`completed`). |
| GET | `/local-api/transcribe/stream/{id}` | SSE stream of transcription progress. |
| GET | `/local-api/audio/{id}` | Stream cached audio (supports `Range`). |
| GET | `/local-api/peaks/{id}` | Generate or return cached peak data. |
| GET | `/local-api/projects` | List all saved draft projects. |
| GET | `/local-api/projects/{id}` | Load a single draft project. |
| PUT | `/local-api/projects/{id}` | Save a draft project. |
| DELETE | `/local-api/projects/{id}` | Delete a draft project. |

### Cache proxy (disk-only)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/cache/audio/{id}` | Cached audio with long `Cache-Control`. |
| GET | `/cache/peaks/{id}` | Cached peaks JSON. |
| GET | `/cache/transcript/{id}` | Cached transcript JSON. |

### LRCLIB proxy

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/request-challenge` | Proxies to `lrclib.net/api/request-challenge`. |
| POST | `/api/publish` | Proxies to `lrclib.net/api/publish` with `X-Publish-Token`. |

### Health

| Method | Path | Description |
|--------|------|-------------|
| GET / HEAD | `/` | Health check. |
| GET / HEAD | `/health` | Health check. |
| GET | `/healthz` | Health check. |

## Dropped features

The following Python-backend features were intentionally removed:

- **WhisperX** and word-level alignment — transcription is OpenAI whisper-1 only.
- **Demucs** source separation — `source=demucs` and `source=vocal` always return 404.
- **CDN / R2** — cache is disk-only; no Cloudflare R2 or AWS S3 integration.
- **Browser UI / WaveSurfer** — this is a TUI-only project.
- **Server-side proof-of-work** — PoW solving remains client-side (TUI).
