# ku-ui-integration Wave 2 notepad

## Bootstrap

- Skills: `omo:ulw-loop` for evidence-led ultrawork, `omo:programming` for Go/Bubble Tea implementation discipline, `omo:frontend` and `omo:visual-qa` for TUI surface changes.
- Tier: HEAVY. Wave 2 adds overlay primitives and a footer layer across the TUI view/update path.
- `rtk` status: unavailable in this shell (`rtk: command not found`), so commands are run directly with bounded output.
- LazyCodex status: v4.13.0 is installing in the background; user should start a new session after completion.

## Success criteria

1. Overlay primitives compile and can center an overlay box over a base view without changing base-only rendering.
2. Footer renders on the last row, uses theme-derived styles, keeps status on the right, and exposes focus/overlay-specific hints.
3. Existing demo flow still works, and a tmux transcript proves footer visibility and focus hint changes in the running TUI.
4. `go test ./...`, `go vet ./...`, `gofmt -l` for changed files pass or failures are identified as pre-existing.

## Planned evidence

- RED: `go test ./internal/tui -run 'TestOverlayCenter|TestFooter'` before production code.
- GREEN: same focused tests after implementation.
- Static/build: `go test ./...`, `go vet ./...`, `gofmt -l <changed files>`, `go build ./cmd/lyrike-studio-tui`.
- Real surface: `tmux new-session -d -s ulw-qa-ku-wave2 'go run ./cmd/lyrike-studio-tui --demo --backend-fixture'`; send `Tab`, capture pane; PASS if transcript includes footer hints and they change after focus cycling.

## Findings

- Repository is Go/Bubble Tea, not Rust. Applied Go guidance.
- Wave 1 tasks 4 and 5 are checked off in `.omo/plans/ku-ui-integration.md`.

## Evidence log

- RED: `go test ./internal/tui -run 'TestOverlayCenter|TestFooter|TestHints'` failed before production code with missing `overlayCenter`, `statusErr`, `overlay`, and footer/hints API, plus `Model.status` still typed as `[]string`.
- GREEN focused: `.omo/evidence/task-6-7-wave2-green.txt` (`go test ./internal/tui -run 'TestHelpOverlayKeyLifecycle|TestFooterViewRendersHintsAndStatus|TestOverlayCenterPlacesBoxOverBase|TestModelViewIncludesFooter|TestRenderLayoutIncludesFooter|TestHintsChangeWithFocusAndOverlay' -count=1`) passed.
- Full tests: `.omo/evidence/task-6-7-wave2-go-test.txt` (`go test ./...`) passed.
- Vet: `.omo/evidence/task-6-7-wave2-go-vet.txt` (`go vet ./...`) passed.
- Format: `.omo/evidence/task-6-7-wave2-gofmt.txt` empty, so changed files are gofmt-clean.
- Diff check: `.omo/evidence/task-6-7-wave2-diff-check.txt` passed.
- Build: `.omo/evidence/task-6-7-wave2-go-build.txt` (`go build ./cmd/lyrike-studio-tui`) passed.
- LSP diagnostics: attempted `mcp__lsp.diagnostics` on `internal/tui`; tool failed with `Transport closed`. Go compiler/vet/test checks are clean.
- tmux QA: `.omo/evidence/task-6-7-wave2-tmux-open.txt` shows `?` opened Help overlay with `Help` and `Esc close`; `.omo/evidence/task-6-7-wave2-tmux-close.txt` shows `Esc` returned to the normal pane.
- Cleanup: `tmux kill-session -t ulw-qa-ku-wave2` run after QA; no expected QA session remains.
- Diff snapshot: `.omo/evidence/task-6-7-wave2.diff`.

## Review loop

- Reviewer pass 1 rejected in `.omo/evidence/task-6-7-wave2-code-review.md`:
  1. diff evidence omitted untracked implementation/test files;
  2. rune-based styled overlay composition leaked ANSI fragments into tmux output;
  3. `?` opened overlay before active fetch/project/input dispatch, creating possible invisible overlay hijack.
- Fixes:
  1. `.omo/evidence/task-6-7-wave2.diff` now appends `git diff --no-index /dev/null <untracked>` for new implementation/test/plan/notepad files;
  2. `overlayCenter` now uses `lipgloss.Place` when base or overlay strings contain ANSI escapes, preserving plain-text overlay behavior for unstyled content;
  3. `?` overlay opening now happens after active fetch/project/metadata dispatch.
- Added regression tests: `TestStyledOverlayDoesNotLeakANSICodes` and `TestHelpKeyDoesNotHijackFetchInput`.
- Refreshed evidence after fixes: focused tests, full tests, vet, build, gofmt, diff-check, style-scan, tmux open/close, cleanup, and ANSI scan.
