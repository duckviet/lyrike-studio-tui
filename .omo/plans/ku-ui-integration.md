# ku-ui-integration - Work Plan

## TL;DR (For humans)

**What you'll get:** A reusable TUI UI layer in `lyrike-studio-tui` with a centralized theme, centered overlay/modal system, context-aware footer with key hints, and a fuzzy selector. The existing project picker becomes the first fuzzy selector; fetch/confirm/help modals become centered overlays instead of full-screen replacements.

**Why this approach:** Both projects use Bubble Tea v2 + Lipgloss v2, so we can port `ku`'s proven, self-contained UI primitives (`Theme`, `overlayCenter`, `selector`, `confirmView`, `helpView`) and adapt them to `lyrike-studio-tui`'s panel layout. We build bottom-up (theme → overlay helpers → footer → selector → modals) so each layer is testable before the next.

**What it will NOT do:** It will not port `ku`'s Kubernetes-specific logic, sidebar, table views, terminal overlay, or non-UI packages. It will not change the backend, playback engine, or domain model. It will not redesign the four main panels (media, waveform, editor, publish).

**Effort:** Medium
**Risk:** Medium - Lipgloss v2 overlay composition with ANSI content is subtle; regressions in focus/key routing are possible.
**Decisions to sanity-check:**
1. Add `charm.land/bubbles/v2` dependency (needed for fuzzy selector text input) — OK?
2. First fuzzy selector target is the project picker (replaces hand-rolled list) — OK?
3. Footer replaces the plain `status []string` with a persistent hints + status bar — OK?
4. Existing purple/pink hard-coded colors become the default theme — OK?

Your next move: Approve this plan, then run `/start-work` or `$start-work ku-ui-integration`. Full execution detail follows below.

---

> TL;DR (machine): Medium effort, medium risk — port ku's Bubble Tea v2 theme, overlay, footer, fuzzy selector, and confirm/help modals into lyrike-studio-tui; deliver reusable UI layer + centered overlays replacing full-screen states.

## Scope
### Must have
1. **Centralized theme system** (`internal/tui/theme.go`):
   - Port `ku/internal/ui/theme.go` `Palette` and `Theme` types.
   - Map existing `lyrike-studio-tui` colors (`#7D56F4`, `#FF3366`, etc.) into a default palette.
   - Replace hard-coded styles in `internal/tui/view.go`, `media/panel.go`, `editor/styles.go`, `waveform/styles.go` with theme-derived styles.

2. **Overlay / modal primitives** (`internal/tui/overlay.go`):
   - Port `overlayCenter` and `overlayBlock` from `ku/internal/ui/notification_view.go`.
   - Add `overlayKind` enum and `Model.overlay` field.
   - Route overlay state in `Model.Update` and `Model.View`.

3. **Context-aware footer** (`internal/tui/footer.go`):
   - Port `hint` struct and `footerView` / `hints` / `renderHints` patterns from `ku/internal/ui/app.go`.
   - Replace `Model.status []string` with `status string` + `statusErr bool` + `[]hint`.
   - Render footer as the last line, reducing body height by 1.
   - Provide per-focus and per-overlay hint sets.

4. **Fuzzy selector** (`internal/tui/selector.go`, `internal/tui/fuzzy.go`):
   - Add `charm.land/bubbles/v2` to `go.mod`.
   - Port `selector`, `selItem`, `selResult` from `ku/internal/ui/selector.go`.
   - Port `fuzzyRank` / `fuzzyScore` from `ku/internal/ui/fuzzy.go`.
   - Replace `project_picker.go` with a selector overlay; preserve `n` (new project) and `Esc` behavior.

5. **Confirm dialog** (`internal/tui/confirm.go`):
   - Port `confirmView` from `ku/internal/ui/confirm_view.go`.
   - Replace inline dirty-replace prompts in `fetch_input.go` and `project_picker.go`.

6. **Integration verification**:
   - `go test ./...`, `go vet ./...`, `gofmt -l .` pass.
   - `--demo --backend-fixture` run reaches publish success and quit readiness.
   - Fuzzy selector can be opened, filtered, and a project selected in demo mode.
   - Footer displays context-aware hints.

### Must NOT have (guardrails, anti-slop, scope boundaries)
- Do NOT port `ku`'s Kubernetes/resource logic, sidebar, table views, terminal overlay, or command preview modal.
- Do NOT change backend routes, storage, playback, transcription, or publish logic.
- Do NOT redesign the media/waveform/editor/publish panels beyond style migration.
- Do NOT introduce global mutable state; keep Bubble Tea model purity.
- Do NOT use `as any`, `@ts-ignore`, empty `catch` blocks, or other type-safety / error-handling shortcuts (Go has none, but the rule stands: no panic suppression).
- Do NOT rewrite git history; commit only after explicit user request.
- Do NOT leave the app in a broken state between waves; each wave must build and the demo harness must still run.

## Verification strategy
> Zero human intervention - all verification is agent-executed.
- Test decision: tests-after. The TUI currently has no unit tests for UI rendering; we will rely on `go test ./...`, `go vet`, `gofmt`, and the deterministic `--demo --backend-fixture` end-to-end harness plus targeted manual checks via `interactive_bash`.
- Evidence: `.omo/evidence/task-<N>-ku-ui-integration.<ext>` (screenshots, build output, demo run logs).

### Per-todo QA pattern
Every implementation todo includes:
1. **Build check:** `go build ./cmd/lyrike-studio-tui` exits 0.
2. **Static check:** `go vet ./...` and `gofmt -l .` clean for changed files.
3. **Regression check:** `go run ./cmd/lyrike-studio-tui --demo --backend-fixture` completes the full demo flow without hanging or crashing.
4. **Feature check:** a focused manual verification (e.g., open fuzzy selector with `Ctrl-P`, filter with `foo`, press `Enter`).

### End-to-end harness
The existing `--demo --backend-fixture` flag exercises fetch → playback → tap-sync → draft save → publish success → quit readiness. It MUST continue to pass after every wave. Any failure blocks the wave.

## Execution strategy
### Parallel execution waves
We build bottom-up so each wave is independently verifiable.

- **Wave 1 — Foundation:** Add dependency, theme system, and shared helpers (clamp, truncate, spread). No behavioral change yet.
- **Wave 2 — Overlay primitives:** Add `overlayKind`, `overlayCenter`, `overlayBlock`, and the footer. Existing full-screen states still work; footer appears.
- **Wave 3 — Fuzzy selector:** Port selector + fuzzy ranking; replace `project_picker.go` with a centered selector overlay.
- **Wave 4 — Confirm dialog & modal conversions:** Port `confirmView`; convert dirty-replace prompts and, if safe, the fetch input into centered input overlays.
- **Wave 5 — Help overlay & polish:** Optional help overlay wired to `?`, final theming sweep, and documentation update.

### Dependency matrix
| Todo | Depends on | Blocks | Can parallelize with |
| --- | --- | --- | --- |
| 1 Add `bubbles/v2` | — | 9 | 2, 3, 4 |
| 2 Port `Palette`/`Theme` | — | 4, 6, 7, 11 | 1, 3, 4 |
| 3 Port layout helpers (`util.go`) | — | 6, 7, 13 | 1, 2, 4 |
| 4 Migrate existing styles to theme | 2 | 6 | 1, 2, 3 |
| 5 Initialize `Model.theme` | 2 | — | — |
| 6 Overlay enum + `overlayCenter`/`overlayBlock` | 2, 3 | 9, 11, 13, 14 | — |
| 7 Context-aware footer | 2, 3 | 15 | 6 |
| 8 Port fuzzy ranking (`fuzzy.go`) | — | 9 | 1 |
| 9 Port selector component (`selector.go`) | 1, 2, 6, 8 | 10 | — |
| 10 Replace `project_picker.go` with selector overlay | 9 | — | 11 |
| 11 Port `confirmView` (`confirm.go`) | 2, 6 | 12 | 10 |
| 12 Convert dirty-replace prompts to confirm overlay | 11 | — | 13 |
| 13 Convert `fetch_input.go` to input overlay | 6 | — | 12 |
| 14 Help overlay | 2, 6 | — | 15 |
| 15 Final polish + docs | 4, 7 | — | 14 |

## Todos
> Implementation + Test = ONE todo. Never separate.
<!-- APPEND TASK BATCHES BELOW THIS LINE WITH edit/apply_patch - never rewrite the headers above. -->

#### Wave 1 — Foundation

- [x] 1. Add `charm.land/bubbles/v2` dependency
  What to do / Must NOT do: Run `go get charm.land/bubbles/v2` in `lyrike-studio-tui`. Ensure `go.mod` and `go.sum` are updated. Do NOT use the dependency anywhere yet.
  Parallelization: Wave 1 | Blocked by: — | Blocks: 7
  References: `/home/duckviet/ku/go.mod` (bubbletea v2, lipgloss v2, bubbles v2 versions); `/home/duckviet/lyrike-studio-tui/go.mod`
  Acceptance criteria: `go mod tidy` exits 0; `grep -q 'charm.land/bubbles/v2' go.mod`.
  QA scenarios: `go build ./cmd/lyrike-studio-tui` still passes; `go test ./...` still passes. Evidence: `.omo/evidence/task-1-ku-ui-integration.log`
  Commit: Y | chore(deps): add charm.land/bubbles/v2 for fuzzy selector input

- [x] 2. Port `Palette` and `Theme` from `ku`
  What to do / Must NOT do: Create `internal/tui/theme.go` with `Palette`, `Theme`, and `NewTheme(name, p Palette) Theme` adapted from `ku/internal/ui/theme.go`. Keep only semantic fields needed by the four target patterns (FooterKey/Desc, StatusOK/Err, ModalBorder/Title, Prompt, SelItem/SelItemSel/SelDesc, PaneActive/Inactive, Rule, Good/Warn/Bad/Dim). Do NOT port Kubernetes-specific status styles.
  Parallelization: Wave 1 | Blocked by: — | Blocks: 4, 6, 7, 11
  References: `ku/internal/ui/theme.go` (`Palette`, `Theme`, `NewTheme`); `lyrike-studio-tui/internal/tui/view.go:14-22` (existing border styles)
  Acceptance criteria: `internal/tui/theme.go` compiles; `Theme` struct has at least the 15 semantic styles above; a default theme function returns the current hard-coded palette.
  QA scenarios: `go build ./cmd/lyrike-studio-tui`; `go vet ./internal/tui/...`. Evidence: `.omo/evidence/task-2-ku-ui-integration.log`
  Commit: Y | feat(tui): add semantic Theme and Palette types

- [x] 3. Port shared layout helpers from `ku/util.go`
  What to do / Must NOT do: Create `internal/tui/util.go` with `clamp`, `truncate` (ANSI-aware), `spread` (left/right line layout), `paneContentWidth`, `paneContentHeight`, and any other helpers needed by overlay/footer. Do NOT port helpers tied to tables/sidebars.
  Parallelization: Wave 1 | Blocked by: — | Blocks: 6, 7, 13
  References: `ku/internal/ui/util.go`; `lyrike-studio-tui/internal/tui/view.go` (layout math)
  Acceptance criteria: Helpers compile; unit-test `clamp` and `truncate` with ANSI strings.
  QA scenarios: `go test ./internal/tui/...` passes. Evidence: `.omo/evidence/task-3-ku-ui-integration.log`
  Commit: Y | feat(tui): add layout helpers for overlay and footer

- [x] 4. Migrate existing hard-coded styles to theme-derived styles
  What to do / Must NOT do: Replace direct `lipgloss.NewStyle().BorderForeground(lipgloss.Color("#7D56F4"))` etc. in `view.go`, `media/panel.go`, `editor/styles.go`, `waveform/styles.go` with `m.theme.FocusedBorder` / `m.theme.NormalBorder` / panel styles. Do NOT change layout geometry or behavior.
  Parallelization: Wave 1 | Blocked by: 2 | Blocks: 6
  References: `lyrike-studio-tui/internal/tui/view.go:14-22`; `internal/tui/media/panel.go`; `internal/tui/editor/styles.go`; `internal/tui/waveform/styles.go`
  Acceptance criteria: No hard-coded `#7D56F4`, `#FF3366`, `#555555`, `#888888` remain in changed files; build passes; demo run still renders panels.
  QA scenarios: `go run ./cmd/lyrike-studio-tui --demo --backend-fixture` completes. Evidence: `.omo/evidence/task-4-ku-ui-integration.log`
  Commit: Y | refactor(tui): migrate hard-coded styles to theme

- [x] 5. Initialize `Model.theme` at startup
  What to do / Must NOT do: Initialize `Model.theme` with the default palette at model creation. Do NOT add CLI flags for themes in this wave.
  Parallelization: Wave 1 | Blocked by: 2 | Blocks: —
  References: `lyrike-studio-tui/internal/tui/model.go` (`Model` struct); `cmd/lyrike-studio-tui/main.go` (model creation)
  Acceptance criteria: `Model.theme` is non-zero; all panels use it; no visual regression in demo.
  QA scenarios: `go run ./cmd/lyrike-studio-tui --demo --backend-fixture` still completes. Evidence: `.omo/evidence/task-5-ku-ui-integration.log`
  Commit: N | (rolled into task 2 or 4)

#### Wave 2 — Overlay primitives and footer

- [ ] 6. Implement overlay enum and `overlayCenter`/`overlayBlock`
  What to do / Must NOT do: Add `internal/tui/overlay.go` with `overlayKind` enum (`overlayNone`, `overlaySelector`, `overlayHelp`, `overlayConfirm`, `overlayInput`), `overlayCenter(base, box, width, height)`, and `overlayBlock`. Add `Model.overlay overlayKind`. In `view.go` render the active overlay on top of the normal body. In `update.go`, route keys to the overlay first when `m.overlay != overlayNone`. Do NOT convert any existing full-screen state yet.
  Parallelization: Wave 2 | Blocked by: 2, 3 | Blocks: 9, 11, 13, 14
  References: `ku/internal/ui/app.go:32-41` (`overlayKind`), `app.go:2250` (render switch), `notification_view.go:81-91` (`overlayCenter`/`overlayBlock`); `lyrike-studio-tui/internal/tui/model.go`, `view.go`, `update.go`
  Acceptance criteria: Overlay helpers compile; `Model.overlay` exists; overlay render path is exercised by a no-op test overlay or temporary stub.
  QA scenarios: `go run ./cmd/lyrike-studio-tui --demo --backend-fixture` still completes; temporary stub overlay can be opened/closed with a key. Evidence: `.omo/evidence/task-6-ku-ui-integration.log`
  Commit: Y | feat(tui): add overlay enum and compositing helpers

- [ ] 7. Add context-aware footer
  What to do / Must NOT do: Add `internal/tui/footer.go` with `hint` struct, `Model.hints()`, `footerView()`, and `renderHints()`. Replace `Model.status []string` with `status string` and `statusErr bool`. Reserve the last terminal row for the footer; reduce body height by 1. Provide hints for each focus (`focusMedia`, `focusWaveform`, `focusEditor`, `focusPublish`) and for overlays. Do NOT port `ku`'s Kubernetes/resource-specific hint branches.
  Parallelization: Wave 2 | Blocked by: 2, 3 | Blocks: 15
  References: `ku/internal/ui/app.go:2469-2623` (`footerView`, `hints`, `renderHints`, `hint`); `lyrike-studio-tui/internal/tui/model.go:36-44`; `internal/tui/view.go:47-81`; `internal/tui/keymap.go`/`keys.go`
  Acceptance criteria: Footer renders at bottom; hints change when focus changes; status messages still appear on the right; no layout overflow.
  QA scenarios: Demo run shows footer; cycling focus (`Tab`/`Shift-Tab`) changes hints. Evidence: `.omo/evidence/task-7-ku-ui-integration.log` + screenshot
  Commit: Y | feat(tui): add context-aware footer with key hints

#### Wave 3 — Fuzzy selector

- [ ] 8. Port fuzzy ranking algorithm
  What to do / Must NOT do: Create `internal/tui/fuzzy.go` with `fuzzyRank[T any]` and `fuzzyScore(pattern, text)` copied/adapted from `ku/internal/ui/fuzzy.go`. Add unit tests for scoring and ranking.
  Parallelization: Wave 3 | Blocked by: — | Blocks: 9
  References: `ku/internal/ui/fuzzy.go`
  Acceptance criteria: `go test ./internal/tui/...` passes; fuzzy ranking produces expected order for sample inputs.
  QA scenarios: Test `fuzzyScore("ab", "alpha bravo")` matches; test `fuzzyRank` sorts by score. Evidence: `.omo/evidence/task-8-ku-ui-integration.log`
  Commit: Y | feat(tui): port fuzzy ranking algorithm

- [ ] 9. Port reusable fuzzy selector component
  What to do / Must NOT do: Create `internal/tui/selector.go` with `selItem`, `selResult`, `selector`, and `selKind` enum. Use `bubbles/v2/textinput` for the search box. Support up/down/enter/esc, freeform entry, and loading state. Do NOT wire it to any screen yet.
  Parallelization: Wave 3 | Blocked by: 1, 2, 6, 8 | Blocks: 10
  References: `ku/internal/ui/selector.go`; `lyrike-studio-tui/internal/tui/theme.go`
  Acceptance criteria: Component compiles; `selector.Update` returns correct `selResult`; `selector.View` renders within given width/height.
  QA scenarios: Unit-test `selector` with a few items; run a temporary TUI program that opens the selector. Evidence: `.omo/evidence/task-9-ku-ui-integration.log`
  Commit: Y | feat(tui): add reusable fuzzy selector component

- [ ] 10. Replace project picker with fuzzy selector overlay
  What to do / Must NOT do: Convert `internal/tui/project_picker.go` from a full-screen inline state to a centered selector overlay. Keep keybindings: open with `Ctrl-O`/equivalent, `j`/`k` or `↑`/`↓`, `Enter` to select, `Esc` to cancel, `n` for new project. Delete the old `projectPicker` inline state from `Model` and `view.go`/`update.go`. Preserve dirty-replace confirmation (will become confirm overlay in task 12).
  Parallelization: Wave 3 | Blocked by: 9 | Blocks: —
  References: `lyrike-studio-tui/internal/tui/project_picker.go`; `internal/tui/model.go`; `internal/tui/view.go:54-56`; `internal/tui/update.go`; `ku/internal/ui/app.go:1953-2063` (openers/applySelection)
  Acceptance criteria: Project picker opens as centered fuzzy overlay; filtering works; selecting a project loads it; canceling returns to previous view.
  QA scenarios: `go run ./cmd/lyrike-studio-tui --demo --backend-fixture`: open picker, type to filter, select project, continue flow. Evidence: `.omo/evidence/task-10-ku-ui-integration.log` + screenshot
  Commit: Y | feat(tui): replace project picker with fuzzy selector overlay

#### Wave 4 — Confirm dialog and modal conversions

- [ ] 11. Port generic confirm dialog
  What to do / Must NOT do: Add `internal/tui/confirm.go` with `confirmView{title, message, danger, action, cancel}` and `Model.confirmAction(...)` adapted from `ku/internal/ui/confirm_view.go`. Render with `ModalBorder`; danger mode colors border/title with `theme.P.Bad`. Do NOT wire it to actions yet.
  Parallelization: Wave 4 | Blocked by: 2, 6 | Blocks: 12
  References: `ku/internal/ui/confirm_view.go`; `ku/internal/ui/app.go:1892-1907` (`confirmAction`, `openCommand`); `lyrike-studio-tui/internal/tui/overlay.go`
  Acceptance criteria: Confirm dialog compiles; can be opened/closed with a temporary test action.
  QA scenarios: Temporary keybind opens confirm dialog; `y` accepts, `n`/`Esc` cancels. Evidence: `.omo/evidence/task-11-ku-ui-integration.log`
  Commit: Y | feat(tui): add generic confirm dialog

- [ ] 12. Convert dirty-replace prompts to confirm overlay
  What to do / Must NOT do: Replace inline dirty-replace modes in `fetch_input.go` and `project_picker.go` with `confirmAction("Unsaved changes", "Discard current work?", true, action)`. Ensure accept/cancel handlers return to the correct state. Delete the inline confirm state fields.
  Parallelization: Wave 4 | Blocked by: 11 | Blocks: —
  References: `lyrike-studio-tui/internal/tui/fetch_input.go:25-27`; `internal/tui/project_picker.go:16-21`; `ku/internal/ui/app.go:1892`
  Acceptance criteria: Both dirty-replace flows show a centered confirm dialog; `y` proceeds, `n`/`Esc` cancels without data loss.
  QA scenarios: Demo mode: start a project, trigger fetch (`Ctrl-O`), confirm replace dialog appears and both accept/cancel paths work. Evidence: `.omo/evidence/task-12-ku-ui-integration.log`
  Commit: Y | feat(tui): use confirm overlay for dirty-replace prompts

- [ ] 13. Convert fetch input to centered input overlay
  What to do / Must NOT do: Refactor `internal/tui/fetch_input.go` from a full-screen inline state into a centered input overlay using `bubbles/v2/textinput`. Keep behavior: open with `Ctrl-O`, accept URL/video ID, validate, fetch. Remove inline state from `Model` and `view.go`/`update.go` short-circuit.
  Parallelization: Wave 4 | Blocked by: 6 | Blocks: —
  References: `lyrike-studio-tui/internal/tui/fetch_input.go`; `internal/tui/view.go:51-53`; `internal/tui/update.go:32`; `ku/internal/ui/selector.go` (textinput usage)
  Acceptance criteria: Fetch input opens as centered overlay; typing and submission work; cancel returns to app.
  QA scenarios: Demo mode: open fetch overlay, type a URL or video ID, submit, see fetch status. Evidence: `.omo/evidence/task-13-ku-ui-integration.log`
  Commit: Y | feat(tui): convert fetch input to centered overlay

#### Wave 5 — Help overlay and polish

- [ ] 14. Add help overlay
  What to do / Must NOT do: Port/adapt `ku/internal/ui/help_view.go` into `internal/tui/help.go`. Render grouped keybindings from `keymap.go` in a scrollable centered overlay. Open with `?`, close with `Esc`/`q`. Do NOT port `ku`'s full keyMap if it differs; use `lyrike-studio-tui`'s existing bindings.
  Parallelization: Wave 5 | Blocked by: 2, 6 | Blocks: —
  References: `ku/internal/ui/help_view.go`; `lyrike-studio-tui/internal/tui/keymap.go`/`keys.go`; `docs/keybindings.md`
  Acceptance criteria: `?` opens help overlay; scrolling works; `Esc` closes it.
  QA scenarios: Demo mode: press `?`, verify overlay renders, press `Esc` to close. Evidence: `.omo/evidence/task-14-ku-ui-integration.log` + screenshot
  Commit: Y | feat(tui): add help keybindings overlay

- [ ] 15. Final polish and documentation
  What to do / Must NOT do: Remove any remaining TODOs/stubs. Update `docs/keybindings.md` and `docs/implementation.md` with new footer/overlay/selector behavior. Run final `go vet`, `gofmt`, `go test ./...`, and full demo harness.
  Parallelization: Wave 5 | Blocked by: 4, 7 | Blocks: —
  References: `docs/keybindings.md`; `docs/implementation.md`
  Acceptance criteria: All docs reflect new UI; static checks pass; demo harness passes.
  QA scenarios: `go test ./...`; `go vet ./...`; `gofmt -l .`; `go run ./cmd/lyrike-studio-tui --demo --backend-fixture`. Evidence: `.omo/evidence/task-15-ku-ui-integration.log`
  Commit: Y | docs(tui): update keybindings and implementation docs for new UI layer

## Final verification wave
> Runs in parallel after ALL todos. ALL must APPROVE. Surface results and wait for the user's explicit okay before declaring complete.
- [ ] F1. Plan compliance audit
  Check that every todo produced its declared deliverable and evidence file. Verify no hard-coded colors, no full-screen replacement states remain for fetch/project-picker, and footer/overlay/selector are wired. Tool: manual checklist against `.omo/evidence/`.
- [ ] F2. Code quality review
  Run `go vet ./...`, `gofmt -l .`, `go test ./...`, `go test -race ./...`. All must pass with no output from `gofmt -l`. Tool: bash.
- [ ] F3. Real manual QA
  Run `go run ./cmd/lyrike-studio-tui --demo --backend-fixture` and interactively verify: footer hints change on focus, fuzzy project picker opens/filters/selects, confirm dialog appears on dirty replace, help overlay opens/closes. Tool: `interactive_bash` + screenshot.
- [ ] F4. Scope fidelity
  Confirm no `ku`-specific Kubernetes logic, sidebar, table, or terminal overlay leaked in. Confirm backend, playback, and domain packages unchanged except for any style imports. Tool: `git diff --stat` and targeted `git diff` review.

## Commit strategy
- Each todo with `Commit: Y` becomes one atomic commit when its wave is complete and verified.
- Do NOT commit until the user explicitly requests it.
- Commit message format: `<type>(<scope>): <summary>` in imperative mood, matching existing repo style.
- Keep commits small and focused per todo; do not batch unrelated waves into a single commit.
- Before any commit, run `git status`, `git diff`, and `git log --oneline -5` to ensure only intended files are staged.

## Success criteria
1. `go build ./cmd/lyrike-studio-tui` exits 0.
2. `go test ./...` passes.
3. `go vet ./...` and `gofmt -l .` report no issues.
4. `--demo --backend-fixture` completes the full demo flow.
5. Footer displays context-aware hints that update with focus and overlay state.
6. Project picker is a centered fuzzy selector overlay with working filter and selection.
7. Dirty-replace prompts use the centered confirm dialog.
8. Fetch input (and optionally other modals) render as centered overlays, not full-screen replacements.
9. Help overlay opens with `?` and closes with `Esc`/`q`.
10. No `ku`-specific non-UI logic is ported; scope boundaries are respected.
