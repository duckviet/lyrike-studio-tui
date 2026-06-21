# lyrike-studio-tui

Terminal UI prototype for the Lyrics Studio workflow.

## Run

From this repository root:

```bash
go run ./cmd/lyrike-studio-tui --version
go run ./cmd/lyrike-studio-tui --demo
go run ./cmd/lyrike-studio-tui --demo --backend-fixture
```

The fixture demo is the current end-to-end QA harness. It renders deterministic backend and playback milestones: fetch, playback, tap-sync, draft save, publish success, and quit readiness.

## Real backend and mpv modes

The implemented lower layers are ready for real wiring:

```bash
# Start the Python backend from the sibling repo, then wire this TUI to it.
cd /home/duckviet/lrclib-upload/backend
PYTHONPATH=. uv run python main.py

# mpv IPC mode expects a Unix socket created by mpv.
mpv --input-ipc-server=/tmp/lyrike-mpv.sock <audio-file>
```

The current CLI exposes the deterministic demo surface. Full command-line flags for selecting backend URL and mpv socket are not exposed yet.

## Development checks

```bash
go test ./...
go run ./cmd/lyrike-studio-tui --demo --backend-fixture
```

See:

- `docs/keybindings.md`
- `docs/troubleshooting.md`
- `docs/implementation.md`
