# Troubleshooting

## Missing mpv

The mpv adapter surfaces missing binary/socket errors. Install `mpv`, then start it with a Unix IPC socket:

```bash
mpv --input-ipc-server=/tmp/lyrike-mpv.sock <audio-file>
```

This project intentionally supports Unix sockets only.

## Corrupt draft

Draft storage reports corrupt JSON as a typed storage error. Move the bad draft out of the server draft directory (`LYRIKE_DRAFT_DIR`) and restart the backend. Use `--project <id>` or `Ctrl-P` in the TUI to select the project to save or reopen.

See [Draft storage moved server-side](#draft-storage-moved-server-side) below.

## Missing yt-dlp

The backend shells out to `yt-dlp` for media info and audio download. If it is missing:

```bash
# Linux (standalone binary)
curl -L -o /usr/local/bin/yt-dlp https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp
chmod +x /usr/local/bin/yt-dlp
```

## Missing ffmpeg

Peak computation and audio streaming require `ffmpeg`. Install it via your system package manager:

```bash
# Debian/Ubuntu
apt-get install ffmpeg
```

## Missing OpenAI API key

Transcription and lyric refinement require an `OPENAI_API_KEY`. Set it in your environment before starting the server:

```bash
export OPENAI_API_KEY=sk-...
go run ./cmd/lyrike-studio-tui serve --port 8080
```

## Backend unavailable

Start the Go backend before running the TUI in real mode:

```bash
go run ./cmd/lyrike-studio-tui serve --port 8080
```

The deterministic TUI harness does not require the backend:

```bash
go run ./cmd/lyrike-studio-tui --demo --backend-fixture
```

## Draft storage moved server-side

Drafts now live in `LYRIKE_DRAFT_DIR` on the server. Existing XDG-local drafts are **not** auto-migrated:

```bash
${XDG_STATE_HOME:-$HOME/.local/state}/lyrike-studio-tui/drafts
```

Use the TUI project picker (`Ctrl-P`) or `--project <id>` to save and reopen drafts via the backend.

## Publish failure

Publish failures leave the publish panel in a failed state with retry available. Retry requests a fresh proof-of-work step before publishing again. Backend publish HTTP failures are covered by backend client tests and should be treated as upstream/backend errors rather than local draft corruption.
