# Lyrike Studio TUI

## TL;DR
> Summary: Build a new terminal-first lyrics studio in `/home/duckviet/lyrike-studio-tui` using Go 1.23+ and Bubble Tea/Bubbles/Lip Gloss v2, informed by the existing `/home/duckviet/lrclib-upload` web app and specs. Keep the TUI focused on local editing, playback control through mpv IPC, LRC timing workflows, and explicit integration with the existing FastAPI backend.
> Deliverables:
> - A git-tracked Go module `github.com/duckviet/lyrike-studio-tui` with CLI entrypoint, domain core, TUI shell, playback adapter, backend client, draft persistence, and docs.
> - A bounded regression fix in `/home/duckviet/lrclib-upload/backend/routes/local_api.py` for the peaks-cache behavior, with a backend test.
> - Agent-executed unit, integration, TUI, and manual QA evidence under `.omo/evidence/`.
> Effort: Large
> Risk: High - new TUI application plus IPC playback, local persistence, and cross-repo backend compatibility.

## Scope
### Must Have
- Initialize this workspace as a Go module and git repository if it is still empty.
- Use Go 1.23+ and Bubble Tea/Bubbles/Lip Gloss v2 APIs; allow Go toolchain auto-download if the local bootstrap is older.
- Create an `AGENTS.md` hierarchy for root, `cmd/`, `internal/domain/`, `internal/tui/`, `internal/playback/`, `internal/integrations/`, and `docs/`.
- Port the lyrics domain into Go: standard LRC parsing/rendering, enhanced LRC metadata preservation, line timing, edit actions, undo/redo history, and tap-sync.
- Implement an mpv Unix IPC playback adapter as the authoritative clock source, including missing-mpv and disconnected-socket guidance.
- Implement an ASCII waveform/timeline surface with seek, zoom, loop region, and active-line follow behavior.
- Integrate with the existing FastAPI backend in `/home/duckviet/lrclib-upload/backend` for fetch, audio, peaks, transcription SSE, and publish.
- Persist drafts atomically in XDG state/config paths and recover them on restart.
- Provide a three-panel terminal UI: media/fetch panel, waveform/playback panel, lyrics/editor/publish panel.
- Fix only the peaks-cache bug in `/home/duckviet/lrclib-upload/backend/routes/local_api.py`, with one regression test in that backend.

### Must Not Have
- Do not add browser UI, WaveSurfer, iframe embedding, desktop GUI, or Windows named-pipe support.
- Do not implement word-level tap-sync.
- Do not replace the existing FastAPI service or move its ownership into this repo.
- Do not weaken, skip, or delete existing backend or frontend tests.
- Do not commit generated caches, downloaded media, transcripts, peaks, or local draft data.
- Do not use Go TUI v1 libraries if v2 is available for the chosen package.

## Verification Strategy
> Zero human intervention - all verification is agent-executed.
- Test decision: TDD for every behavior-changing task. Capture RED before production changes and GREEN after.
- QA policy: every task has an agent-executed scenario through CLI stdout, tmux, HTTP, or filesystem evidence.
- Evidence root: `.omo/evidence/`.
- Final checks:
  - `go test ./...`
  - `go test -race ./...`
  - `gofmt -w`/`gofumpt` if available, then no formatting diff.
  - `go vet ./...`
  - Backend targeted test from `/home/duckviet/lrclib-upload/backend`.
  - tmux-driven TUI smoke path with captured transcript.

## Execution Strategy
### Parallel Execution Waves
Wave 1 (foundation, mostly parallel):
- Task 1: Workspace, module, repo hygiene, and docs scaffolding.
- Task 2: Lyrics domain parser/renderer and tests.
- Task 3: Playback contracts and fake clock/player test seam.
- Task 4: Backend client contracts and fixtures.
- Task 5: Draft persistence contracts.

Wave 2 (core behavior):
- Task 6: Domain edit actions, history, and tap-sync.
- Task 7: mpv IPC adapter and missing-mpv guidance.
- Task 8: Backend fetch/audio/peaks/SSE/publish client.
- Task 9: Atomic XDG draft persistence.
- Task 10: Peaks-cache backend regression fix.

Wave 3 (TUI integration):
- Task 11: Three-panel app model, routing, and responsive terminal layout.
- Task 12: ASCII waveform and transport controls.
- Task 13: Lyrics editor, keyboard actions, undo/redo, and tap-sync.
- Task 14: Publish flow panel and status machine.

Wave 4 (hardening):
- Task 15: End-to-end tmux TUI scenario with fake backend and fake mpv.
- Task 16: Documentation, examples, and operator guidance.
- Task 17: Final quality, manual QA, and review gates.

Critical path: Task 1 -> Task 2 -> Task 6 -> Task 11 -> Task 13 -> Task 15 -> Task 17.

### Dependency Matrix
| Task | Depends on | Blocks | Can parallelize with |
|------|------------|--------|----------------------|
| 1 | none | 2, 3, 4, 5, 11, 17 | none |
| 2 | 1 | 6, 13, 15 | 3, 4, 5 |
| 3 | 1 | 7, 12, 15 | 2, 4, 5 |
| 4 | 1 | 8, 14, 15 | 2, 3, 5 |
| 5 | 1 | 9, 11, 15 | 2, 3, 4 |
| 6 | 2 | 13, 15 | 7, 8, 9, 10 |
| 7 | 3 | 12, 15 | 6, 8, 9, 10 |
| 8 | 4 | 14, 15 | 6, 7, 9, 10 |
| 9 | 5 | 11, 15 | 6, 7, 8, 10 |
| 10 | 4 | 8, 17 | 6, 7, 9 |
| 11 | 1, 5 | 12, 13, 14, 15 | none |
| 12 | 3, 7, 11 | 15 | 13, 14 |
| 13 | 2, 6, 11 | 15 | 12, 14 |
| 14 | 4, 8, 11 | 15 | 12, 13 |
| 15 | 7, 8, 9, 12, 13, 14 | 17 | none |
| 16 | 11, 12, 13, 14 | 17 | 15 |
| 17 | all prior tasks | none | none |

## Todos
> Implementation + Test = one task. Every task must capture RED, GREEN, and surface evidence before completion.

- [x] 1. Initialize workspace, module, repo hygiene, and local rules

  What to do: Create `go.mod`, `cmd/lyrike-studio-tui/main.go`, top-level `README.md`, `.gitignore`, `docs/`, and scoped `AGENTS.md` files. Use module path `github.com/duckviet/lyrike-studio-tui`. Add a minimal CLI that starts the TUI and supports `--version`.
  Must NOT do: Do not import TUI packages before the module and rules are in place. Do not commit caches or generated local state.

  Parallelization: Can parallel: NO | Wave 1 | Blocks: [2, 3, 4, 5, 11, 17] | Blocked by: []

  References:
  - Project rules: `/home/duckviet/lyrike-studio-tui/AGENTS.md` - root RTK and AGENTS rules.
  - Existing context: `/home/duckviet/lrclib-upload/docs/AI_CONTEXT.md` - source-of-truth conventions.
  - Architecture context: `/home/duckviet/lrclib-upload/docs/ARCHITECTURE.md` - current service boundaries.
  - Glossary: `/home/duckviet/lrclib-upload/docs/GLOSSARY.md` - domain language.
  - External: `https://pkg.go.dev/github.com/charmbracelet/bubbletea/v2` - Bubble Tea v2 API.

  Acceptance criteria:
  - [x] `go test ./...` exits 0.
  - [x] `go run ./cmd/lyrike-studio-tui --version` prints a non-empty version string and exits 0.
  - [x] `find . -name AGENTS.md -print` lists root plus scoped instruction files.

  QA scenarios:
  ```
  Scenario: CLI version smoke
    Tool:     bash
    Steps:    go run ./cmd/lyrike-studio-tui --version | tee .omo/evidence/task-1-version.txt
    Expected: output contains "lyrike-studio-tui" and command exits 0.
    Evidence: .omo/evidence/task-1-version.txt

  Scenario: repository ignores local state
    Tool:     bash
    Steps:    git status --short --ignored | tee .omo/evidence/task-1-git-ignore.txt
    Expected: no media, cache, transcript, peaks, draft, or build-cache files appear as tracked additions.
    Evidence: .omo/evidence/task-1-git-ignore.txt
  ```

  Commit: YES | Message: `chore(workspace): initialize tui module` | Files: [`go.mod`, `cmd/`, `docs/`, `README.md`, `.gitignore`, `AGENTS.md`]

- [x] 2. Implement LRC domain parser and renderer

  What to do: Add `internal/domain/lyrics` types for metadata, timed lines, enhanced inline timestamps, parse errors, and render options. Support standard LRC and preserve enhanced LRC data without implementing word tap-sync.
  Must NOT do: Do not pass raw maps or untyped JSON through domain APIs. Do not discard unknown metadata.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [6, 13, 15] | Blocked by: [1]

  References:
  - Spec: `/home/duckviet/lrclib-upload/docs/specs/004-lyrics-core.md` - lyrics core operations.
  - Existing frontend model: `/home/duckviet/lrclib-upload/lyrike-studio/features/publish/model/publishFlow.ts` - state-machine style to mirror for deterministic flows.
  - Existing docs: `/home/duckviet/lrclib-upload/docs/GLOSSARY.md` - LRC terms.
  - External: `https://en.wikipedia.org/wiki/LRC_(file_format)` - baseline LRC format.

  Acceptance criteria:
  - [x] RED test first for parse/render round-trip, invalid timestamp, duplicate/out-of-order timestamp handling.
  - [x] `go test ./internal/domain/lyrics -run Test` exits 0.

  QA scenarios:
  ```
  Scenario: parse and render sample LRC
    Tool:     bash
    Steps:    go test ./internal/domain/lyrics -run TestParseRenderSample -v | tee .omo/evidence/task-2-lyrics.txt
    Expected: test output shows PASS and rendered output preserves title, artist, line text, and timestamps.
    Evidence: .omo/evidence/task-2-lyrics.txt

  Scenario: invalid timestamp is rejected
    Tool:     bash
    Steps:    go test ./internal/domain/lyrics -run TestParseRejectsInvalidTimestamp -v | tee .omo/evidence/task-2-invalid.txt
    Expected: test output shows PASS and error contains "invalid timestamp".
    Evidence: .omo/evidence/task-2-invalid.txt
  ```

  Commit: YES | Message: `feat(lyrics): add lrc parser and renderer` | Files: [`internal/domain/lyrics/`]

- [x] 3. Define playback contracts and deterministic fake player

  What to do: Add `internal/playback` interfaces for player commands, observed time, pause/play state, errors, and event subscription. Add a fake player for tests and TUI smoke scenarios.
  Must NOT do: Do not connect to mpv in this task. Do not use wall-clock sleeps in tests.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [7, 12, 15] | Blocked by: [1]

  References:
  - Spec: `/home/duckviet/lrclib-upload/docs/specs/003-waveform-editor.md` - waveform playback behavior.
  - mpv docs: `https://mpv.io/manual/stable/#json-ipc` - authoritative IPC contract for Task 7.
  - mpv property: `https://mpv.io/manual/stable/#command-interface-time-pos` - `time-pos` clock source.

  Acceptance criteria:
  - [x] RED test first for seek/play/pause event flow.
  - [x] `go test ./internal/playback -run TestFakePlayer -v` exits 0.

  QA scenarios:
  ```
  Scenario: fake player emits clock events
    Tool:     bash
    Steps:    go test ./internal/playback -run TestFakePlayerEmitsClockEvents -v | tee .omo/evidence/task-3-fake-player.txt
    Expected: PASS and test asserts ordered time events without sleep.
    Evidence: .omo/evidence/task-3-fake-player.txt
  ```

  Commit: YES | Message: `feat(playback): define player clock contracts` | Files: [`internal/playback/`]

- [x] 4. Define backend client contracts and fixtures

  What to do: Add typed contracts under `internal/integrations/backend` for fetch, audio URL/range expectations, peaks, transcription SSE, challenge, and publish. Add fixtures matching `/local-api` responses from the existing FastAPI app.
  Must NOT do: Do not perform live network calls in unit tests. Do not encode browser-only assumptions.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [8, 14, 15] | Blocked by: [1]

  References:
  - Backend route: `/home/duckviet/lrclib-upload/backend/routes/local_api.py` - local API contracts.
  - Cache route: `/home/duckviet/lrclib-upload/backend/routes/cache_proxy.py` - audio/peaks cache contracts.
  - Spec: `/home/duckviet/lrclib-upload/docs/specs/002-media-pipeline.md` - media pipeline split.
  - Spec: `/home/duckviet/lrclib-upload/docs/specs/005-publish-migration.md` - publish flow.

  Acceptance criteria:
  - [x] RED test first for JSON decoding and error handling.
  - [x] `go test ./internal/integrations/backend -run TestDecode -v` exits 0.

  QA scenarios:
  ```
  Scenario: decode backend fetch fixture
    Tool:     bash
    Steps:    go test ./internal/integrations/backend -run TestDecodeFetchResponse -v | tee .omo/evidence/task-4-fetch-fixture.txt
    Expected: PASS and decoded video ID/title/duration match fixture.
    Evidence: .omo/evidence/task-4-fetch-fixture.txt
  ```

  Commit: YES | Message: `feat(backend): define local api contracts` | Files: [`internal/integrations/backend/`]

- [x] 5. Define atomic draft persistence contracts

  What to do: Add `internal/domain/draft` and `internal/storage` contracts for draft IDs, video IDs, metadata, lyrics doc snapshot, and atomic write/read/delete using XDG paths.
  Must NOT do: Do not write to the user home directory in tests; use `t.TempDir()` and injected paths.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [9, 11, 15] | Blocked by: [1]

  References:
  - Spec: `/home/duckviet/lrclib-upload/docs/specs/004-lyrics-core.md` - autosave recovery prompt.
  - Existing config style: `/home/duckviet/lrclib-upload/backend/core/config.py` - explicit cache root contract.
  - External: `https://specifications.freedesktop.org/basedir-spec/latest/` - XDG Base Directory spec.

  Acceptance criteria:
  - [x] RED test first for atomic write and recovery after partial temp file.
  - [x] `go test ./internal/storage -run TestDraft -v` exits 0.

  QA scenarios:
  ```
  Scenario: draft round-trip in temp XDG state
    Tool:     bash
    Steps:    XDG_STATE_HOME="$(mktemp -d)" go test ./internal/storage -run TestDraftRoundTrip -v | tee .omo/evidence/task-5-draft.txt
    Expected: PASS and no files are written outside the temp XDG path.
    Evidence: .omo/evidence/task-5-draft.txt
  ```

  Commit: YES | Message: `feat(storage): add atomic draft contracts` | Files: [`internal/domain/draft/`, `internal/storage/`]

- [x] 6. Add lyric edit actions, undo/redo history, and tap-sync

  What to do: Implement typed commands for set timestamp, edit text, insert line, delete line, reorder line, undo, redo, and tap-sync against the authoritative playback clock.
  Must NOT do: Do not implement word-level tap-sync. Do not mutate command input parameters.

  Parallelization: Can parallel: YES | Wave 2 | Blocks: [13, 15] | Blocked by: [2]

  References:
  - Spec: `/home/duckviet/lrclib-upload/docs/specs/004-lyrics-core.md` - command-pattern history manager.
  - Domain from Task 2: `internal/domain/lyrics`.
  - Playback contract from Task 3: `internal/playback`.

  Acceptance criteria:
  - [x] RED tests first for each command and undo/redo edge.
  - [x] `go test ./internal/domain/... -run 'Test.*History|Test.*TapSync' -v` exits 0.

  QA scenarios:
  ```
  Scenario: keyboard-style tap sync sequence
    Tool:     bash
    Steps:    go test ./internal/domain/... -run TestTapSyncUsesPlaybackClock -v | tee .omo/evidence/task-6-tap-sync.txt
    Expected: PASS and timestamps match fake playback times.
    Evidence: .omo/evidence/task-6-tap-sync.txt
  ```

  Commit: YES | Message: `feat(lyrics): add edit history and tap sync` | Files: [`internal/domain/lyrics/`, `internal/domain/history/`]

- [x] 7. Implement mpv Unix IPC playback adapter

  What to do: Add Unix socket JSON IPC client for mpv, observe `time-pos`, send play/pause/seek commands, handle socket disconnects, and expose actionable missing-mpv guidance.
  Must NOT do: Do not add Windows named-pipe support. Do not treat the TUI timer as authoritative clock.

  Parallelization: Can parallel: YES | Wave 2 | Blocks: [12, 15] | Blocked by: [3]

  References:
  - mpv IPC: `https://mpv.io/manual/stable/#json-ipc`.
  - mpv option: `https://mpv.io/manual/stable/#options-input-ipc-server`.
  - mpv property: `https://mpv.io/manual/stable/#command-interface-time-pos`.
  - Contract from Task 3: `internal/playback`.

  Acceptance criteria:
  - [x] RED integration test first using a local Unix socket fake mpv server.
  - [x] `go test ./internal/playback/mpv -run Test -v` exits 0.

  QA scenarios:
  ```
  Scenario: fake mpv socket reports time-pos
    Tool:     bash
    Steps:    go test ./internal/playback/mpv -run TestObserveTimePosFromSocket -v | tee .omo/evidence/task-7-mpv-observe.txt
    Expected: PASS and adapter emits observed time from fake JSON IPC event.
    Evidence: .omo/evidence/task-7-mpv-observe.txt

  Scenario: missing mpv guidance
    Tool:     bash
    Steps:    go test ./internal/playback/mpv -run TestMissingMpvGuidance -v | tee .omo/evidence/task-7-missing-mpv.txt
    Expected: PASS and error message contains the concrete mpv install/start command.
    Evidence: .omo/evidence/task-7-missing-mpv.txt
  ```

  Commit: YES | Message: `feat(playback): add mpv ipc adapter` | Files: [`internal/playback/mpv/`]

- [x] 8. Implement backend HTTP client

  What to do: Implement typed HTTP client for fetch, audio/peaks URLs or payloads, transcription SSE, publish challenge, PoW token submission, and structured errors.
  Must NOT do: Do not solve PoW in the backend client. Do not use bare default `http.Client` without timeout.

  Parallelization: Can parallel: YES | Wave 2 | Blocks: [14, 15] | Blocked by: [4]

  References:
  - Backend route: `/home/duckviet/lrclib-upload/backend/routes/local_api.py`.
  - Cache route: `/home/duckviet/lrclib-upload/backend/routes/cache_proxy.py`.
  - Frontend API: `/home/duckviet/lrclib-upload/lyrike-studio/lib/api.ts`.
  - Publish flow: `/home/duckviet/lrclib-upload/lyrike-studio/features/publish/model/publishFlow.ts`.

  Acceptance criteria:
  - [x] RED tests first using `httptest.Server` for success, HTTP error, invalid JSON, and SSE status stream.
  - [x] `go test ./internal/integrations/backend -run TestClient -v` exits 0.

  QA scenarios:
  ```
  Scenario: fetch through httptest backend
    Tool:     bash
    Steps:    go test ./internal/integrations/backend -run TestClientFetchSuccess -v | tee .omo/evidence/task-8-fetch-client.txt
    Expected: PASS and request path is `/local-api/fetch`.
    Evidence: .omo/evidence/task-8-fetch-client.txt

  Scenario: transcription SSE stream
    Tool:     bash
    Steps:    go test ./internal/integrations/backend -run TestClientTranscriptionSSE -v | tee .omo/evidence/task-8-sse.txt
    Expected: PASS and client receives queued/running/done events in order.
    Evidence: .omo/evidence/task-8-sse.txt
  ```

  Commit: YES | Message: `feat(backend): add local api client` | Files: [`internal/integrations/backend/`]

- [x] 9. Implement XDG atomic draft storage

  What to do: Implement draft storage with temp-file write, fsync where available, atomic rename, read/list/delete, and corrupt-file error reporting.
  Must NOT do: Do not immediately re-read after write as redundant verification in production code; tests own verification.

  Parallelization: Can parallel: YES | Wave 2 | Blocks: [11, 15] | Blocked by: [5]

  References:
  - Storage contract from Task 5: `internal/storage`.
  - XDG spec: `https://specifications.freedesktop.org/basedir-spec/latest/`.

  Acceptance criteria:
  - [x] RED tests first for round-trip, corrupt JSON, and interrupted temp file.
  - [x] `go test ./internal/storage -run TestDraftStore -v` exits 0.

  QA scenarios:
  ```
  Scenario: CLI-visible draft storage temp path
    Tool:     bash
    Steps:    XDG_STATE_HOME="$(mktemp -d)" go test ./internal/storage -run TestDraftStoreRoundTrip -v | tee .omo/evidence/task-9-draft-store.txt
    Expected: PASS and test logs the temp draft path.
    Evidence: .omo/evidence/task-9-draft-store.txt
  ```

  Commit: YES | Message: `feat(storage): persist drafts atomically` | Files: [`internal/storage/`]

- [x] 10. Fix existing backend peaks-cache bug

  What to do: In `/home/duckviet/lrclib-upload/backend/routes/local_api.py`, fix only the peaks-cache behavior discovered during planning. Add or update one regression test under `/home/duckviet/lrclib-upload/backend/tests`.
  Must NOT do: Do not refactor unrelated backend routes. Do not alter frontend behavior.

  Parallelization: Can parallel: YES | Wave 2 | Blocks: [8, 17] | Blocked by: [4]

  References:
  - Backend route: `/home/duckviet/lrclib-upload/backend/routes/local_api.py` - peaks endpoint/cache behavior.
  - Backend config: `/home/duckviet/lrclib-upload/backend/core/config.py` - `PEAKS_CACHE_DIR` contract.
  - Cache proxy: `/home/duckviet/lrclib-upload/backend/routes/cache_proxy.py` - existing peaks cache conventions.
  - Backend tests: `/home/duckviet/lrclib-upload/backend/tests`.

  Acceptance criteria:
  - [x] RED pytest first that fails on the current peaks-cache bug for the right reason.
  - [x] Targeted pytest for the new regression exits 0.
  - [x] Existing related backend tests still exit 0.

  QA scenarios:
  ```
  Scenario: peaks cache hit uses normalized video ID and source
    Tool:     bash
    Steps:    cd /home/duckviet/lrclib-upload/backend && pytest tests -k 'peaks or local_api' -q | tee /home/duckviet/lyrike-studio-tui/.omo/evidence/task-10-peaks-pytest.txt
    Expected: pytest exits 0 and includes the new regression test.
    Evidence: .omo/evidence/task-10-peaks-pytest.txt

  Scenario: local API peaks HTTP smoke
    Tool:     curl
    Steps:    Start backend test app on localhost, then curl -i 'http://127.0.0.1:<port>/local-api/peaks/<video_id>?source=original' | tee /home/duckviet/lyrike-studio-tui/.omo/evidence/task-10-peaks-curl.txt
    Expected: HTTP status is 200 for seeded cache or 404 with structured JSON for absent cache, never 500.
    Evidence: .omo/evidence/task-10-peaks-curl.txt
  ```

  Commit: YES | Message: `fix(backend): correct peaks cache lookup` | Files: [`/home/duckviet/lrclib-upload/backend/routes/local_api.py`, `/home/duckviet/lrclib-upload/backend/tests/`]

- [x] 11. Build three-panel TUI model and layout

  What to do: Add `internal/tui` app model with media/fetch panel, waveform/playback panel, and lyrics/editor/publish panel. Include focus management, terminal-size adaptation, command routing, and no-overlap layout.
  Must NOT do: Do not add business logic directly to view rendering.

  Parallelization: Can parallel: NO | Wave 3 | Blocks: [12, 13, 14, 15] | Blocked by: [1, 5, 9]

  References:
  - Spec: `/home/duckviet/lrclib-upload/docs/specs/001-layout-shell.md` - layout shell intent.
  - Bubble Tea v2: `https://pkg.go.dev/github.com/charmbracelet/bubbletea/v2`.
  - Bubbles v2: `https://pkg.go.dev/github.com/charmbracelet/bubbles/v2`.
  - Lip Gloss v2: `https://pkg.go.dev/github.com/charmbracelet/lipgloss/v2`.

  Acceptance criteria:
  - [x] RED test first for focus routing and resize behavior.
  - [x] `go test ./internal/tui -run 'Test.*Layout|Test.*Focus' -v` exits 0.

  QA scenarios:
  ```
  Scenario: TUI renders three panels in tmux
    Tool:     tmux
    Steps:    tmux new-session -d -s ulw-qa-task-11 'go run ./cmd/lyrike-studio-tui --demo'; sleep 1; tmux capture-pane -t ulw-qa-task-11 -pS -200 > .omo/evidence/task-11-tui.txt; tmux kill-session -t ulw-qa-task-11
    Expected: capture contains media, waveform, and lyrics panel labels with no panic.
    Evidence: .omo/evidence/task-11-tui.txt
  ```

  Commit: YES | Message: `feat(tui): add three panel shell` | Files: [`internal/tui/`, `cmd/lyrike-studio-tui/`]

- [x] 12. Add ASCII waveform and transport controls

  What to do: Render normalized peaks as an ASCII waveform/timeline with cursor, zoom, loop region, seek actions, play/pause, and active line following playback time.
  Must NOT do: Do not use browser waveform libraries.

  Parallelization: Can parallel: YES | Wave 3 | Blocks: [15] | Blocked by: [3, 7, 11]

  References:
  - Spec: `/home/duckviet/lrclib-upload/docs/specs/003-waveform-editor.md`.
  - Playback adapter from Task 7: `internal/playback/mpv`.
  - TUI shell from Task 11: `internal/tui`.

  Acceptance criteria:
  - [x] RED tests first for peak-to-cell mapping, seek mapping, and loop bounds.
  - [x] `go test ./internal/tui/... -run 'Test.*Waveform|Test.*Transport' -v` exits 0.

  QA scenarios:
  ```
  Scenario: waveform seek through tmux
    Tool:     tmux
    Steps:    tmux new-session -d -s ulw-qa-task-12 'go run ./cmd/lyrike-studio-tui --demo'; tmux send-keys -t ulw-qa-task-12 Space Right Right l; tmux capture-pane -t ulw-qa-task-12 -pS -200 > .omo/evidence/task-12-waveform.txt; tmux kill-session -t ulw-qa-task-12
    Expected: capture shows play state, moved cursor, and loop marker.
    Evidence: .omo/evidence/task-12-waveform.txt
  ```

  Commit: YES | Message: `feat(tui): render ascii waveform controls` | Files: [`internal/tui/`, `internal/playback/`]

- [x] 13. Add lyrics editor interactions

  What to do: Implement synced/plain/meta tabs, inline text editing, line navigation, timestamp tap, insert/delete/reorder, undo/redo, and draft dirty-state indication.
  Must NOT do: Do not implement collaborative editing or AI rewriting.

  Parallelization: Can parallel: YES | Wave 3 | Blocks: [15] | Blocked by: [2, 6, 11]

  References:
  - Spec: `/home/duckviet/lrclib-upload/docs/specs/004-lyrics-core.md`.
  - Domain history from Task 6: `internal/domain/history`.
  - TUI shell from Task 11: `internal/tui`.

  Acceptance criteria:
  - [x] RED tests first for keyboard messages and resulting document state.
  - [x] `go test ./internal/tui/... -run 'Test.*Lyrics|Test.*Undo|Test.*Redo|Test.*Tap' -v` exits 0.

  QA scenarios:
  ```
  Scenario: keyboard-only sync loop
    Tool:     tmux
    Steps:    tmux new-session -d -s ulw-qa-task-13 'go run ./cmd/lyrike-studio-tui --demo'; tmux send-keys -t ulw-qa-task-13 Tab t Down t u C-r; tmux capture-pane -t ulw-qa-task-13 -pS -200 > .omo/evidence/task-13-lyrics.txt; tmux kill-session -t ulw-qa-task-13
    Expected: capture shows timestamps applied, undo, and redo state changes without panic.
    Evidence: .omo/evidence/task-13-lyrics.txt
  ```

  Commit: YES | Message: `feat(tui): add lyrics editing workflow` | Files: [`internal/tui/`, `internal/domain/lyrics/`, `internal/domain/history/`]

- [x] 14. Add publish flow panel

  What to do: Add publish validation, challenge request, local PoW solver or compatible token flow, publish submission, retry, and deterministic step status display.
  Must NOT do: Do not change LRCLIB API contracts. Do not block TUI rendering during PoW.

  Parallelization: Can parallel: YES | Wave 3 | Blocks: [15] | Blocked by: [4, 8, 11]

  References:
  - Existing publish state machine: `/home/duckviet/lrclib-upload/lyrike-studio/features/publish/model/publishFlow.ts`.
  - Existing worker: `/home/duckviet/lrclib-upload/lyrike-studio/features/publish/model/powWorker.ts`.
  - Existing publish mutation: `/home/duckviet/lrclib-upload/lyrike-studio/features/media/queries/publishMutation.ts`.
  - Spec: `/home/duckviet/lrclib-upload/docs/specs/005-publish-migration.md`.

  Acceptance criteria:
  - [x] RED tests first for Validate -> PoW -> Publish -> Done and failure retry.
  - [x] `go test ./internal/tui/... ./internal/integrations/backend/... -run 'Test.*Publish' -v` exits 0.

  QA scenarios:
  ```
  Scenario: publish success with fake backend
    Tool:     tmux
    Steps:    Start fake backend fixture server, then tmux new-session -d -s ulw-qa-task-14 'go run ./cmd/lyrike-studio-tui --backend http://127.0.0.1:<port> --demo'; tmux send-keys -t ulw-qa-task-14 p; tmux capture-pane -t ulw-qa-task-14 -pS -200 > .omo/evidence/task-14-publish.txt; tmux kill-session -t ulw-qa-task-14
    Expected: capture shows Validate, PoW, Publish, and Done as success.
    Evidence: .omo/evidence/task-14-publish.txt
  ```

  Commit: YES | Message: `feat(tui): add publish workflow panel` | Files: [`internal/tui/`, `internal/integrations/backend/`]

- [x] 15. Add end-to-end TUI QA harness

  What to do: Add deterministic demo mode or fixture harness that runs the full TUI against fake backend and fake mpv/player. Drive it with tmux and capture transcript evidence.
  Must NOT do: Do not rely on network, real YouTube, real mpv, or human keystrokes for CI-smoke behavior.

  Parallelization: Can parallel: NO | Wave 4 | Blocks: [17] | Blocked by: [7, 8, 9, 12, 13, 14]

  References:
  - All TUI tasks: `internal/tui`.
  - Backend fixtures from Task 8: `internal/integrations/backend`.
  - Playback fake from Task 3: `internal/playback`.

  Acceptance criteria:
  - [x] `go test ./...` exits 0.
  - [x] `go run ./cmd/lyrike-studio-tui --demo --backend-fixture` can be driven in tmux.
  - [x] Captured tmux transcript contains fetch, playback, tap-sync, draft save, publish success, and quit.

  QA scenarios:
  ```
  Scenario: full TUI demo workflow
    Tool:     tmux
    Steps:    tmux new-session -d -s ulw-qa-task-15 'go run ./cmd/lyrike-studio-tui --demo --backend-fixture'; tmux send-keys -t ulw-qa-task-15 Enter Space t Down t s p q; tmux capture-pane -t ulw-qa-task-15 -pS -500 > .omo/evidence/task-15-full-tui.txt; tmux kill-session -t ulw-qa-task-15
    Expected: capture contains fetched media, waveform movement, timestamped lyric lines, draft saved, publish completed, and graceful quit.
    Evidence: .omo/evidence/task-15-full-tui.txt
  ```

  Commit: YES | Message: `test(tui): add tmux demo qa harness` | Files: [`cmd/lyrike-studio-tui/`, `internal/tui/`, `internal/testfixtures/`]

- [x] 16. Write docs and operator guidance

  What to do: Document install, mpv startup, backend URL, keybindings, draft location, troubleshooting, and the precise boundaries between the TUI repo and `/home/duckviet/lrclib-upload`.
  Must NOT do: Do not document unimplemented features.

  Parallelization: Can parallel: YES | Wave 4 | Blocks: [17] | Blocked by: [11, 12, 13, 14]

  References:
  - README from Task 1.
  - Existing docs: `/home/duckviet/lrclib-upload/docs/AI_CONTEXT.md`.
  - mpv IPC docs: `https://mpv.io/manual/stable/#options-input-ipc-server`.

  Acceptance criteria:
  - [x] `README.md` includes runnable commands for demo, real backend, and mpv IPC mode.
  - [x] `docs/keybindings.md` lists every implemented keybinding.
  - [x] `docs/troubleshooting.md` covers missing mpv, backend unavailable, corrupt draft, and publish failure.

  QA scenarios:
  ```
  Scenario: docs commands are present
    Tool:     bash
    Steps:    rg -n 'go run ./cmd/lyrike-studio-tui|mpv --input-ipc-server|--backend|keybindings' README.md docs | tee .omo/evidence/task-16-docs.txt
    Expected: command exits 0 and prints all required command references.
    Evidence: .omo/evidence/task-16-docs.txt
  ```

  Commit: YES | Message: `docs: document tui operation` | Files: [`README.md`, `docs/`]

- [x] 17. Final verification, review, cleanup, and handoff

  What to do: Run the full automated gate, targeted backend regression gate, tmux manual QA, LSP/diagnostics if configured, and final review. Confirm no live tmux sessions, bound ports, temp dirs, or background processes remain.
  Must NOT do: Do not declare complete from tests alone. Do not leave QA state running.

  Parallelization: Can parallel: NO | Wave 4 | Blocks: [] | Blocked by: [all tasks]

  References:
  - This plan: `.omo/plans/lyrike-studio-tui.md`.
  - Evidence directory: `.omo/evidence/`.
  - Git history: `git log --oneline --decorate -n 20`.

  Acceptance criteria:
  - [x] `go test ./...` exits 0.
  - [x] `go test -race ./...` exits 0.
  - [x] `go vet ./...` exits 0.
  - [x] Backend targeted pytest exits 0.
  - [x] Full tmux TUI scenario PASS evidence exists.
  - [x] Reviewer approval is unconditional if review gate is triggered.
  - [x] `tmux ls` shows no `ulw-qa-*` sessions.
  - [x] `git status --short` contains only intended changes or is clean after approved commits.

  QA scenarios:
  ```
  Scenario: full Go gate
    Tool:     bash
    Steps:    go test ./... && go test -race ./... && go vet ./... | tee .omo/evidence/task-17-go-gate.txt
    Expected: all commands exit 0.
    Evidence: .omo/evidence/task-17-go-gate.txt

  Scenario: backend regression gate
    Tool:     bash
    Steps:    cd /home/duckviet/lrclib-upload/backend && pytest tests -k 'peaks or local_api' -q | tee /home/duckviet/lyrike-studio-tui/.omo/evidence/task-17-backend-gate.txt
    Expected: pytest exits 0.
    Evidence: .omo/evidence/task-17-backend-gate.txt

  Scenario: no leftover QA state
    Tool:     bash
    Steps:    tmux ls 2>/dev/null | grep 'ulw-qa-' > .omo/evidence/task-17-leftover-qa.txt || true
    Expected: evidence file is empty.
    Evidence: .omo/evidence/task-17-leftover-qa.txt
  ```

  Commit: YES | Message: `chore(release): verify lyrike studio tui` | Files: [all completed task files]

## Final Verification Wave
> Runs after all implementation tasks. All must pass before completion.
- [x] F1. Plan compliance audit - every task done, every acceptance criterion met, every evidence file exists.
- [x] F2. Code quality review - diagnostics clean, idioms match, no dead code, no oversized source file without explicit `SIZE_OK` rationale.
- [x] F3. Real manual QA - every tmux, HTTP, and CLI scenario executed with evidence captured.
- [x] F4. Scope fidelity - no browser UI, WaveSurfer, iframe, Windows named-pipe, or word tap-sync introduced.

## Commit Strategy
- One logical change per commit. Use Conventional Commits: `<type>(<scope>): <imperative summary>`.
- Every commit must build and pass its task-specific tests before the next task starts.
- Do not auto-commit unless the session explicitly authorizes commits. Otherwise stage nothing and provide the intended commit message.
- Final commit footer, if commits are authorized: `Plan: .omo/plans/lyrike-studio-tui.md`.

## Success Criteria
- The plan file exists at `.omo/plans/lyrike-studio-tui.md` and has 17 top-level implementation tasks plus 4 final verification checks.
- The executor can start at Task 1 without further interview.
- Every task has references, acceptance criteria, concrete QA scenarios, evidence paths, and commit instructions.
- Final delivery is not complete until the TUI has been driven through tmux and the backend regression has been driven through pytest/HTTP evidence.
