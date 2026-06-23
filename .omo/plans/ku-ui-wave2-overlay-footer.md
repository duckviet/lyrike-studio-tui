# KU UI Wave 2 Overlay/Footer Continuation

## TL;DR
> Summary:      Continue the already-started Wave 2 overlay primitives + footer work from `.omo/plans/ku-ui-integration.md` without broad refactors. The current workspace is a Go Bubble Tea/Lipgloss TUI, not Rust; use Go/TUI quality gates.
> Deliverables:
> - centered overlay primitives proven by failing-first tests and a real tmux help-overlay surface
> - context-aware footer with status + hints, preserved below overlays
> - no commits; preserve existing dirty/untracked user changes
> Effort:       Short
> Risk:         Medium - worktree already contains partial Wave 2 changes and untracked files that must not be overwritten

## Scope
### Must have
- Continue from current WIP, not from clean `HEAD`.
- Finish only existing-plan Wave 2 overlay primitives and footer/hints.
- Add RED-first tests before production changes; capture RED and GREEN evidence.
- Prove the real TUI surface through `tmux` against `go run ./cmd/lyrike-studio-tui --demo --backend-fixture`.
- Use existing Bubble Tea v2 and Lipgloss v2 patterns.

### Must NOT have (guardrails, anti-slop, scope boundaries)
- Do not create a broad replacement plan for `.omo/plans/ku-ui-integration.md`.
- Do not migrate fuzzy selector/search in this slice; existing Wave 2 also lists task 8, but this wave plan scopes only overlay primitives + footer.
- Do not migrate fetch/project-picker full-screen flows to selector overlays unless needed to keep the overlay/footer tests passing. Current `internal/tui/view.go:41-45` still returns `renderFetchInput`/`renderProjectPicker` early; treat that as a later-wave migration unless the owner expands scope.
- Do not replace or discard untracked user files.
- Do not commit unless explicitly asked.
- Do not use `rtk`; it is not installed in this shell.

## Verification strategy
> Zero human intervention - all verification is agent-executed.
- Test decision: TDD + Go `testing`
- QA policy: every task has agent-executed scenarios
- Evidence:
  - `.omo/evidence/task-6-7-wave2-red.txt`
  - `.omo/evidence/task-6-7-wave2-green.txt`
  - `.omo/evidence/task-6-7-wave2-full-go-test.txt`
  - `.omo/evidence/task-6-7-wave2-tmux-open.txt`
  - `.omo/evidence/task-6-7-wave2-tmux-close.txt`

## Execution strategy
### Parallel execution waves
> Target 5-8 tasks per wave. This is a continuation slice, so keep it small.
> Extract shared dependencies as Wave-1 tasks to maximize parallelism.

Wave 1 (no dependencies):
- Task 1: Baseline guard + RED overlay/footer tests
- Task 2: Finish overlay primitives
- Task 3: Finish footer/hints and key lifecycle

Wave 2 (after Wave 1):
- Task 4: GREEN verification + tmux real-surface QA

Critical path: Task 1 -> Task 2/3 -> Task 4

### Dependency matrix
| Task | Depends on | Blocks | Can parallelize with |
|------|------------|--------|----------------------|
| 1    | none       | 2, 3, 4 | none |
| 2    | 1          | 4      | 3 |
| 3    | 1          | 4      | 2 |
| 4    | 1, 2, 3    | none   | none |

## Todos
> Implementation + Test = ONE task. Never separate.
> Every task MUST have: References + Acceptance Criteria + QA Scenarios + Commit.

- [ ] 1. Baseline guard + RED overlay/footer tests

  What to do: Preserve the dirty worktree first. Confirm current dirty files are expected WIP: modified `internal/tui/model.go`, `internal/tui/view.go`; untracked `.omo/notepads/ku-ui-wave2.md`, `internal/tui/footer.go`, `internal/tui/overlay.go`, `internal/tui/overlay_footer_test.go`. Extend the existing untracked `internal/tui/overlay_footer_test.go` instead of replacing it. Add/adjust tests that fail before production changes:
  - `TestHelpOverlayKeyLifecycle`: `?` opens `overlayHelp`; `Esc` closes it; `q` closes overlay without quitting while overlay is active.
  - `TestFooterViewIncludesModeHintsAndStatus`: footer includes focus-aware hints and current status, and switches to overlay close hints when `m.overlay != overlayNone`.
  - `TestOverlayCenterPreservesFooterAndCentersBlock`: body overlay is centered while footer remains a separate bottom row.
  Must NOT do: Do not edit production code before RED evidence. Do not delete existing tests or change unrelated assertions.

  Parallelization: Can parallel: NO | Wave 1 | Blocks: [2, 3, 4] | Blocked by: []

  References:
  - Plan:     `.omo/plans/ku-ui-integration.md:158-184` - existing Wave 2 defines task 6 overlay primitives, task 7 footer/hints, and task 8 fuzzy scoring; this slice covers 6 and 7 only.
  - WIP:      `internal/tui/overlay_footer_test.go:10` - existing overlay/footer test file already started.
  - API/Type: `internal/tui/model.go:36` - `Model` struct; current WIP adds `status string`, `statusErr bool`, and `overlay overlayKind` at `internal/tui/model.go:46-48`.
  - API/Type: `internal/tui/overlay.go:9-16` - `overlayKind` and constants.
  - API/Type: `internal/tui/theme/theme.go:55-62` - `FooterKey`, `FooterDesc`, `StatusOK`, `StatusErr`, `ModalBorder`, `ModalTitle`, `Prompt`.
  - Test:     `internal/tui/model_test.go:97` - existing layout fit test pattern.
  - External: `/charmbracelet/bubbletea` - Bubble Tea key press messages.
  - External: `/charmbracelet/lipgloss` - Lipgloss width/rendering helpers.

  Acceptance criteria (agent-executable only):
  - [ ] `git status --short --untracked-files=all` output is saved mentally/notes before edits; no unrelated dirty file is changed.
  - [ ] RED captured: `bash -lc 'set -o pipefail; mkdir -p .omo/evidence; go test ./internal/tui -run "TestHelpOverlayKeyLifecycle|TestFooterViewIncludesModeHintsAndStatus|TestOverlayCenterPreservesFooterAndCentersBlock" -count=1 2>&1 | tee .omo/evidence/task-6-7-wave2-red.txt; test ${PIPESTATUS[0]} -ne 0'`
  - [ ] RED failure is caused by missing/incorrect overlay/footer behavior, not compile errors from unrelated packages.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: RED proof captures missing overlay/footer behavior
    Tool:     bash
    Steps:    bash -lc 'set -o pipefail; mkdir -p .omo/evidence; go test ./internal/tui -run "TestHelpOverlayKeyLifecycle|TestFooterViewIncludesModeHintsAndStatus|TestOverlayCenterPreservesFooterAndCentersBlock" -count=1 2>&1 | tee .omo/evidence/task-6-7-wave2-red.txt; test ${PIPESTATUS[0]} -ne 0'
    Expected: command exits 0 because go test failed; evidence file contains FAIL for at least one of the new overlay/footer tests.
    Evidence: .omo/evidence/task-6-7-wave2-red.txt

  Scenario: Dirty worktree guard
    Tool:     bash
    Steps:    git status --short --untracked-files=all
    Expected: only planned/user-owned Wave 2 files are dirty before production edits; any unexpected file stops execution for user direction.
    Evidence: .omo/evidence/task-6-7-wave2-status-before.txt
  ```

  Commit: NO | Message: `test(tui): prove overlay footer wave red` | Files: [internal/tui/overlay_footer_test.go]

- [ ] 2. Finish overlay primitives

  What to do: Complete the current untracked `internal/tui/overlay.go` primitives and `internal/tui/view.go` integration. Keep `overlayBlock(content, width, th)` theme-derived through `th.ModalBorder`. Keep `overlayCenter(base, box, width, height)` ANSI-safe enough for Lipgloss-rendered strings: use `lipgloss.Width`, clamp negative padding, avoid panics when terminal/body is smaller than the box, and overlay only the body area. In `renderLayout`, preserve `body + "\n" + footerView(...)` so the footer is not covered by overlays.
  Must NOT do: Do not migrate fetch/project picker full-screen views in this task. Do not introduce new global rendering state.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [4] | Blocked by: [1]

  References:
  - WIP:      `internal/tui/overlay.go:9` - `overlayKind`.
  - WIP:      `internal/tui/overlay.go:19` - `overlayBlock(content string, width int, th Theme) string`.
  - WIP:      `internal/tui/overlay.go:26` - `overlayCenter(base, box string, width, height int) string`.
  - WIP:      `internal/tui/overlay.go:72` - `overlayLine`.
  - WIP:      `internal/tui/view.go:66-71` - current overlay placement and footer append.
  - WIP:      `internal/tui/view.go:75-93` - current `renderOverlay` switch.
  - API/Type: `internal/tui/theme/theme.go:106-108` - `ModalBorder`, `ModalTitle`, `Prompt` initialization.
  - External: `/charmbracelet/lipgloss` - `Width`, `StripANSI`, `Place`, styled rendering behavior.

  Acceptance criteria (agent-executable only):
  - [ ] `go test ./internal/tui -run "TestOverlayCenterPreservesFooterAndCentersBlock|TestOverlayCenterOverwritesMiddle" -count=1` passes.
  - [ ] `rg -n "#[0-9A-Fa-f]{6}|lipgloss\\.NewStyle\\(\\)" internal/tui/overlay.go internal/tui/view.go` returns no new hard-coded overlay styles outside the theme package.
  - [ ] `gofmt -w internal/tui/overlay.go internal/tui/view.go internal/tui/overlay_footer_test.go` produces no diff except formatting.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: Overlay primitive unit behavior
    Tool:     bash
    Steps:    bash -lc 'set -o pipefail; go test ./internal/tui -run "TestOverlayCenterPreservesFooterAndCentersBlock|TestOverlayCenterOverwritesMiddle" -count=1 2>&1 | tee .omo/evidence/task-6-overlay-green.txt'
    Expected: command exits 0 and evidence contains PASS for overlay tests.
    Evidence: .omo/evidence/task-6-overlay-green.txt

  Scenario: Overlay does not reintroduce hard-coded styles
    Tool:     bash
    Steps:    bash -lc '! rg -n "#[0-9A-Fa-f]{6}|lipgloss\\.NewStyle\\(\\)" internal/tui/overlay.go internal/tui/view.go'
    Expected: command exits 0; overlay/view use `Theme`, not local hex colors/new anonymous styles.
    Evidence: .omo/evidence/task-6-overlay-style-scan.txt
  ```

  Commit: NO | Message: `feat(tui): add centered overlay primitives` | Files: [internal/tui/overlay.go, internal/tui/view.go, internal/tui/overlay_footer_test.go]

- [ ] 3. Finish footer/hints and help-overlay key lifecycle

  What to do: Complete `internal/tui/footer.go` and minimal top-level key handling. `footerView(m, width)` should render left-side status with `StatusOK`/`StatusErr`, right-side hints from `renderHints`, and truncate/spread without exceeding terminal width. `Model.hints()` should prioritize overlay-active hints (`Esc`/`q` close) before focus-specific hints. In the existing top-level key dispatch, add the smallest lifecycle: when no blocking input/picker/editor mode owns keys, `?` sets `m.overlay = overlayHelp`; when any overlay is active, `Esc` or `q` sets `m.overlay = overlayNone` and returns without quitting or forwarding to focused panels.
  Must NOT do: Do not rewrite panel keymaps. Do not hijack editor-specific help inside editor submode unless top-level overlay is active.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [4] | Blocked by: [1]

  References:
  - WIP:      `internal/tui/footer.go:15` - current overlay-active hint branch.
  - WIP:      `internal/tui/footer.go:30-36` - current focus-specific hint branches.
  - WIP:      `internal/tui/footer.go:43` - `footerView(m Model, width int) string`.
  - WIP:      `internal/tui/footer.go:52` - `statusErr` style branch.
  - WIP:      `internal/tui/footer.go:65` - `renderHints`.
  - WIP:      `internal/tui/model.go:109-116` - `setStatus`/`setErrorStatus`; use these instead of reverting to `[]string`.
  - Pattern:  `internal/tui/keys.go:42` - existing project-picker top-level key action.
  - Pattern:  `internal/tui/keys.go:62` - existing status-setting key path.
  - Pattern:  `internal/tui/project_picker.go:71-104` - Escape handling for modal-ish state.
  - External: `/charmbracelet/bubbletea` - `tea.KeyPressMsg`, `tea.KeyEscape`, rune key handling.

  Acceptance criteria (agent-executable only):
  - [ ] `go test ./internal/tui -run "TestHelpOverlayKeyLifecycle|TestFooterViewIncludesModeHintsAndStatus" -count=1` passes.
  - [ ] `rg -n "m\\.status = \\[\\]string|len\\(m\\.status\\)|strings\\.Join\\(status" internal/tui` returns no stale slice-status writes.
  - [ ] `go test ./internal/tui -run "TestCtrlOOpensFetchInput|TestProjectPicker|Test_LayoutFits80x24" -count=1` still passes; overlay key handling did not break existing modal-like flows.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: Footer and help overlay lifecycle unit behavior
    Tool:     bash
    Steps:    bash -lc 'set -o pipefail; go test ./internal/tui -run "TestHelpOverlayKeyLifecycle|TestFooterViewIncludesModeHintsAndStatus" -count=1 2>&1 | tee .omo/evidence/task-7-footer-green.txt'
    Expected: command exits 0 and evidence contains PASS for footer/help overlay lifecycle tests.
    Evidence: .omo/evidence/task-7-footer-green.txt

  Scenario: Existing input/picker flows still pass
    Tool:     bash
    Steps:    bash -lc 'set -o pipefail; go test ./internal/tui -run "TestCtrlOOpensFetchInput|TestProjectPicker|Test_LayoutFits80x24" -count=1 2>&1 | tee .omo/evidence/task-7-regression-green.txt'
    Expected: command exits 0; existing fetch input, project picker, and layout tests still pass.
    Evidence: .omo/evidence/task-7-regression-green.txt
  ```

  Commit: NO | Message: `feat(tui): add contextual footer hints` | Files: [internal/tui/footer.go, internal/tui/keys.go, internal/tui/model.go, internal/tui/overlay_footer_test.go]

- [ ] 4. GREEN verification + tmux real-surface QA

  What to do: Run full formatting/tests and real-surface QA. Save evidence. Kill tmux session afterward.
  Must NOT do: Do not declare done from tests alone. Do not leave tmux sessions running.

  Parallelization: Can parallel: NO | Wave 2 | Blocks: [] | Blocked by: [1, 2, 3]

  References:
  - Entry:    `cmd/lyrike-studio-tui/main.go:42-43` - demo and backend fixture flags.
  - Entry:    `cmd/lyrike-studio-tui/main.go:242` - Bubble Tea program runner.
  - Demo:     `internal/tui/demo.go:14` - `DemoModel`.
  - Surface:  `internal/tui/model.go:158` - `View()`.
  - Surface:  `internal/tui/view.go:37` - `renderLayout`.
  - Surface:  `internal/tui/view.go:71` - footer appended below body.

  Acceptance criteria (agent-executable only):
  - [ ] `gofmt -w internal/tui/model.go internal/tui/view.go internal/tui/overlay.go internal/tui/footer.go internal/tui/overlay_footer_test.go internal/tui/keys.go`
  - [ ] `bash -lc 'set -o pipefail; go test ./internal/tui -run "TestHelpOverlayKeyLifecycle|TestFooterViewIncludesModeHintsAndStatus|TestOverlayCenterPreservesFooterAndCentersBlock|TestOverlayCenterOverwritesMiddle" -count=1 2>&1 | tee .omo/evidence/task-6-7-wave2-green.txt'`
  - [ ] `bash -lc 'set -o pipefail; go test ./... -count=1 2>&1 | tee .omo/evidence/task-6-7-wave2-full-go-test.txt'`
  - [ ] `git diff --check` passes.
  - [ ] tmux scenario below passes and `tmux has-session -t ku-wave2-help` fails after cleanup.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: Real TUI help overlay opens and footer hints are visible
    Tool:     tmux
    Steps:    bash -lc 'set -euo pipefail; mkdir -p .omo/evidence; tmux kill-session -t ku-wave2-help 2>/dev/null || true; tmux new-session -d -s ku-wave2-help "TERM=xterm-256color go run ./cmd/lyrike-studio-tui --demo --backend-fixture"; sleep 3; tmux send-keys -t ku-wave2-help "?"; sleep 1; tmux capture-pane -t ku-wave2-help -pS -200 > .omo/evidence/task-6-7-wave2-tmux-open.txt; grep -q "Help" .omo/evidence/task-6-7-wave2-tmux-open.txt; grep -q "Esc" .omo/evidence/task-6-7-wave2-tmux-open.txt; grep -q "q" .omo/evidence/task-6-7-wave2-tmux-open.txt'
    Expected: command exits 0; captured pane contains the help overlay title plus close hints.
    Evidence: .omo/evidence/task-6-7-wave2-tmux-open.txt

  Scenario: Real TUI help overlay closes without quitting
    Tool:     tmux
    Steps:    bash -lc 'set -euo pipefail; tmux send-keys -t ku-wave2-help Escape; sleep 1; tmux capture-pane -t ku-wave2-help -pS -200 > .omo/evidence/task-6-7-wave2-tmux-close.txt; tmux has-session -t ku-wave2-help; if grep -q "Help" .omo/evidence/task-6-7-wave2-tmux-close.txt; then exit 1; fi; tmux kill-session -t ku-wave2-help; if tmux has-session -t ku-wave2-help 2>/dev/null; then exit 1; fi'
    Expected: command exits 0; app session remains alive after Escape, overlay title is absent, then cleanup removes the session.
    Evidence: .omo/evidence/task-6-7-wave2-tmux-close.txt
  ```

  Commit: NO | Message: `feat(tui): complete wave two overlay footer` | Files: [internal/tui/model.go, internal/tui/view.go, internal/tui/overlay.go, internal/tui/footer.go, internal/tui/keys.go, internal/tui/overlay_footer_test.go]

## Final verification wave (MANDATORY - after all implementation tasks)
> Runs in PARALLEL. ALL must APPROVE. Surface results to the caller and wait for an explicit "okay" before declaring complete.
- [ ] F1. Plan compliance audit - every task done, every acceptance criterion met
- [ ] F2. Code quality review - diagnostics clean, idioms match, no dead code
- [ ] F3. Real manual QA - every QA scenario executed with evidence captured
- [ ] F4. Scope fidelity - no fuzzy-selector work, no broad fetch/project-picker migration, no commits

## Commit strategy
- No commits unless the user explicitly asks.
- If later asked to commit, use one logical Conventional Commit and include footer: `Plan: .omo/plans/ku-ui-wave2-overlay-footer.md`.

## Success criteria
- Existing Wave 2 tasks 6 and 7 are completed for overlay primitives + footer/hints; task 8 remains out of this slice.
- RED→GREEN proof is captured.
- `go test ./... -count=1` passes.
- tmux real-surface QA proves `?` opens the help overlay and `Esc` closes it while the app stays alive.
- No user-owned dirty work is overwritten and no commit is made.
