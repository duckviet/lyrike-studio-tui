# Troubleshooting

## Missing mpv

The mpv adapter surfaces missing binary/socket errors. Install `mpv`, then start it with a Unix IPC socket:

```bash
mpv --input-ipc-server=/tmp/lyrike-mpv.sock <audio-file>
```

This project intentionally supports Unix sockets only.

## Backend unavailable

Start the sibling backend before real backend wiring:

```bash
cd /home/duckviet/lrclib-upload/backend
PYTHONPATH=. uv run python main.py
```

The deterministic TUI harness does not require the backend:

```bash
go run ./cmd/lyrike-studio-tui --demo --backend-fixture
```

## Corrupt draft

Draft storage reports corrupt JSON as a typed storage error. Move the bad draft out of the XDG draft directory and restart the TUI. The default draft directory is under:

```bash
${XDG_STATE_HOME:-$HOME/.local/state}/lyrike-studio-tui/drafts
```

Draft filenames are project IDs. Use `--project <id>` or `Ctrl-P` in the TUI to select the project to save or reopen.

## Publish failure

Publish failures leave the publish panel in a failed state with retry available. Retry requests a fresh proof-of-work step before publishing again. Backend publish HTTP failures are covered by backend client tests and should be treated as upstream/backend errors rather than local draft corruption.
