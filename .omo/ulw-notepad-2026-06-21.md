# Ultrawork Notepad — Lyrike Studio TUI Gate Resolution
Started: 2026-06-21T09:30:00+07:00

## Plan (exhaustively detailed)
1. [x] Fix test race condition in `internal/playback/mpv/mpv_test.go` by adding a ready channel to synchronize connection setup.
2. [x] Refactor `internal/playback/mpv/player.go` to move private observation/updates helpers to `events.go`, keeping `player.go` well under 250 lines.
3. [x] Refactor `internal/storage/store.go` to move JSON serialisation structs (`storedSnapshot`, `storedMetadata`) and conversion functions (`toStored`, `toStoredMetadata`, `fromStored`) to a new file `internal/storage/conversions.go`, keeping `store.go` well under 250 lines.
4. [x] Integrate the publish panel into `internal/tui/model.go` and `internal/tui/view.go`. Allow pressing `p` in the lyrics editor panel to start validation and transition the right panel to show the publish panel state.
5. [x] Create a real end-to-end `httptest.Server` in `cmd/lyrike-studio-tui/main.go` and `tui.DemoFixtureModel` when `--backend-fixture` is requested. Implement actual client calls to fetch metadata, peaks, request challenge, solve PoW, and publish through the TUI.
6. [x] Verify all test files build and pass. Generate dynamic QA screenshots/tmux transcripts.
7. [x] Perform cleanup of leftover `.omo/run-continuation/` files.
8. [x] Self-review and final review.
9. [ ] Implement the actual real version of the application: support command line flags for real backend URL (`--backend`), mpv socket path (`--mpv-socket`), and fetch target (`--video-id` / `--url`). Wire them into uvicorn and real mpv player connection setup.

## Success criteria + QA scenarios
1. All Go package unit tests (`go test ./...`) pass synchronously. [PASS]
2. The publish panel renders dynamically in the right pane when `p` is pressed in editor mode. [PASS]
3. Running with `--demo --backend-fixture` runs real HTTP calls to local mock server and fetches media/peaks and publishes successfully. [PASS]
4. All production Go files are under 250 pure LOC. [PASS]
5. Cleanup: no background ports or tmux sessions left. [PASS]
6. Running `go run ./cmd/lyrike-studio-tui --backend http://127.0.0.1:9999 --video-id dQw4w9WgXcQ` works and connects to uvicorn, fetching real peaks/metadata.

## Now
9. Implement the actual real version of the application in `main.go`, `model.go`, and `demo.go`.

## Todo
- Update `NewModel` signature and logic in `model.go` to support custom video IDs and URLs.
- Update `demo.go` calling signatures.
- Update `main.go` to support real `--backend`, `--mpv-socket`, `--video-id`, and `--url` flags.
- Verify unit tests and e2e run.
