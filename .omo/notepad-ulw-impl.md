# ULW Implementation Notepad

## Bootstrap
- Skills: `omo:ulw-loop` for durable ULW execution; `omo:start-work` for plan execution discipline; `omo:programming` for Go code standards.
- Tier: HEAVY. This implementation starts a new Go module and later crosses TUI, domain, playback, storage, and backend integration boundaries.
- Current slice: Task 1 from `.omo/plans/lyrike-studio-tui.md`.

## Task 1 Success Criteria
- RED: `go test ./...` fails before production code because the version label contract is missing.
- GREEN: `go test ./...` passes after implementation.
- Real surface: `go run ./cmd/lyrike-studio-tui --version` prints `lyrike-studio-tui 0.1.0-dev`.
- Cleanup: no `ulw-qa-*` tmux sessions remain.

## Task 1 Evidence
- Invalid RED attempt: `.omo/evidence/task-1-red-go-test.txt` failed before compilation because `go 1.23` tried to download unavailable toolchain name `go1.23`.
- Valid RED: `.omo/evidence/task-1-red-go-test-contract.txt` fails on `undefined: Label`.
- GREEN: `.omo/evidence/task-1-green-go-test.txt` passes.
- CLI surface: `.omo/evidence/task-1-version.txt` prints `lyrike-studio-tui 0.1.0-dev`.
- tmux surface: `.omo/evidence/task-1-tmux-version.txt` prints `lyrike-studio-tui 0.1.0-dev`.
- Diagnostics: `.omo/evidence/task-1-go-vet.txt` and `.omo/evidence/task-1-go-test-race.txt` pass.
- Cleanup: `.omo/evidence/task-1-leftover-tmux-final.txt` is empty.
## Task 2 bootstrap

- Tier: HEAVY — creates a new `internal/domain/lyrics` package with typed domain primitives and validation semantics.
- Skills: `omo:start-work` for continuing the plan execution; `omo:programming` for Go TDD/type-safety constraints.
- Plan/reviewer note: multi-agent plan/reviewer tools are not exposed in this session; proceeding with local evidence-bound plan and will record this limitation in the DoneClaim.
- Success criteria:
  1. Failing-first tests prove LRC/enhanced LRC parsing/validation behavior before production code.
  2. Package exposes typed timestamps/text/line/document primitives and returns typed validation errors at parse/package boundaries.
  3. Automated package and broad Go checks pass from the current worktree.
  4. Manual parse surface proves valid input returns timestamp-ordered typed lines and invalid timestamp returns a typed validation error.
- Manual-QA scenario: run `go test ./internal/domain/lyrics -run TestManualParseSurface -v`; PASS observable is stdout showing ordered typed lines and an `*lyrics.ValidationError` for invalid timestamp.
- Dirty-worktree baseline captured at `.omo/evidence/task-2-dirty-worktree.txt`; pre-existing edits include docs/domain/version/boulder files and were not reverted.

## Task 2 update

- Implemented `internal/domain/lyrics` as a TUI/HTTP/filesystem/process-independent domain package.
- Domain surface: `Timestamp`, `Text`, `WordTiming`, `Line`, `Document`, constructors, `ParseLRC`, `FormatLRC`, `ValidationError`, and stable `ErrorCode` values.
- Validation covered at package boundaries: malformed LRC line, invalid timestamp seconds, empty lyric text, duplicate timestamps, unsorted timestamps, and malformed enhanced markers.
- RED evidence: `.omo/evidence/task-2-red-go-test.txt`; invalid RED: `.omo/evidence/task-2-invalid-red.txt`.
- GREEN evidence: `.omo/evidence/task-2-lyrics.txt`, `.omo/evidence/task-2-invalid.txt`, `.omo/evidence/task-2-go-test-all.txt`.
- Manual-QA evidence: `.omo/evidence/task-2-manual-parse.txt` shows typed ordered lines and typed invalid timestamp error.
- Cleanup receipt: `.omo/evidence/task-2-cleanup.txt`; transient `.serena/` and `.omo/ulw-loop/...` tool state was removed from disk and git index, and no QA helpers/scripts/tmux/processes remain.
- Review note: independent gate-review attempts were inconclusive because both reviewers scoped themselves to final F1 instead of Task 2; orchestrator self-review checked API boundaries, typed errors, line ordering, file LOC, and evidence completeness.
## Task 3 bootstrap — deterministic playback fake

- Tier: HEAVY. Justification: Task 3 creates a new playback package/domain port with typed primitives and deterministic fake behavior.
- Skills used: `omo:programming` for Go TDD/type/error constraints; Serena instructions/project activation for codebase navigation readiness.
- Source context read: `.omo/plans/lyrike-studio-tui.md` Task 3 playback/fake-player section, `internal/playback/AGENTS.md`, `/home/duckviet/lrclib-upload/docs/specs/003-waveform-editor.md`; mpv IPC is explicitly left to Task 7.
- Dirty-worktree baseline captured at `.omo/evidence/task-3-dirty-baseline.txt`; unrelated pre-existing edits are outside Task 3 scope and must be preserved.
- Success criteria:
  1. RED: fake-player behavior tests fail before production implementation; evidence `.omo/evidence/task-3-red-go-test.txt`.
  2. GREEN package: `rtk go test ./internal/playback`; evidence `.omo/evidence/task-3-fake-player.txt`.
  3. GREEN repository: `rtk go test ./...`; evidence `.omo/evidence/task-3-go-test-all.txt`.
  4. Manual/data surface: `rtk go test ./internal/playback -run TestManualFakePlayerSurface -v` prints deterministic play/tick/seek/pause transcript without sleeps; evidence `.omo/evidence/task-3-manual-fake-player.txt`.
  5. Cleanup receipt: `.omo/evidence/task-3-cleanup.txt` states no tmux/process/port/temp helper remains.
- Planned API shape: TUI-independent `internal/playback` package with semantic `Position`, `Duration`, and `State`, a small `Player` port, typed command errors, and a sleep-free `FakePlayer` whose `Tick` advances only while playing and clamps progress at duration.

## Task 3 update

- Implemented `internal/playback` as a TUI/HTTP/filesystem/process-independent package with `Position`, `Duration`, `State`, `Snapshot`, `Player`, typed `CommandError`, and deterministic `FakePlayer`.
- Fake player behavior: starts paused at 0, `Play`/`Pause` transition state, `Tick` advances only while playing, paused ticks are no-ops, progress clamps at duration and pauses, `Seek` is deterministic and preserves playing state unless seeking to the end.
- Adversarial coverage: negative position, zero/negative duration, zero/negative tick, seek past duration, tick while paused, and progress beyond duration.
- RED evidence: `.omo/evidence/task-3-red-go-test.txt`.
- GREEN evidence: `.omo/evidence/task-3-fake-player.txt`, `.omo/evidence/task-3-go-test-all.txt`.
- Manual/data-surface evidence: `.omo/evidence/task-3-manual-fake-player.txt` contains `go test -v` transcript for start/play/tick/seek/tick/pause with typed position/duration/state output.
- Cleanup receipt: `.omo/evidence/task-3-cleanup.txt`.
- Review note: `omo:review-work` was selected for significant implementation, but no `multi_agent_v1`/independent reviewer tool is exposed in this session; local self-review checked API scope, typed errors, deterministic/sleep-free behavior, LOC ceiling, dirty-worktree preservation, and evidence completeness. Review gate result is therefore inconclusive, not an independent approval.
