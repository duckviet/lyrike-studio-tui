---
slug: ku-ui-integration
status: awaiting-approval
intent: clear
pending-action: user approves plan at .omo/plans/ku-ui-integration.md, then run /start-work ku-ui-integration
approach: Port proven Bubble Tea v2 UI primitives from /home/duckviet/ku into /home/duckviet/lyrike-studio-tui bottom-up: theme → overlay helpers → footer → fuzzy selector → confirm/help modals. Replace full-screen inline states (project picker, fetch input) with centered overlays.
---

# Draft: ku-ui-integration

## Components (topology ledger)
| id | outcome | status | evidence path |
|---|---|---|---|
| theme | Semantic `Palette` + `Theme` in `internal/tui/theme.go`; existing colors become default theme | active | .omo/evidence/task-2/4-ku-ui-integration.* |
| overlay | `overlayKind` enum + `overlayCenter`/`overlayBlock` in `internal/tui/overlay.go` | active | .omo/evidence/task-6-ku-ui-integration.* |
| footer | Context-aware footer with hints + status in `internal/tui/footer.go` | active | .omo/evidence/task-7-ku-ui-integration.* |
| fuzzy | `fuzzy.go` + `selector.go` using `bubbles/v2/textinput`; replaces project picker | active | .omo/evidence/task-8/9/10-ku-ui-integration.* |
| confirm | Generic `confirmView` in `internal/tui/confirm.go` for dirty-replace prompts | active | .omo/evidence/task-11/12-ku-ui-integration.* |
| help | Scrollable keybindings help overlay in `internal/tui/help.go` | active | .omo/evidence/task-14-ku-ui-integration.* |

## Open assumptions (announced defaults)
| assumption | adopted default | rationale | reversible? |
|---|---|---|---|
| Bubbles v2 dependency | Add `charm.land/bubbles/v2` for fuzzy selector text input | ku already uses it; standard Bubble Tea component | Reversible only by reimplementing textinput manually |
| First fuzzy selector target | Replace `project_picker.go` | It is the existing list selector; immediate user value | Reversible by keeping old picker behind flag |
| Footer design | Replace `status []string` with persistent hints + structured status | Matches ku pattern and provides discoverability | Reversible by adding status-only mode |
| Default color scheme | Map current hard-coded purple/pink/grays into default palette | Preserves existing look while enabling theming | Reversible by changing palette |
| Help overlay | Port ku help overlay and wire to `?` | User explicitly mentioned overlays/modals | Reversible by removing keybinding |

## Findings (cited - path:lines)
- Source framework: `ku/go.mod` uses `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`.
- Target framework: `lyrike-studio-tui/go.mod` uses `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`; no `bubbles`.
- ku overlay composition: `ku/internal/ui/notification_view.go:81-91` (`overlayCenter`, `overlayBlock`).
- ku footer/hints: `ku/internal/ui/app.go:2469-2623`.
- ku selector: `ku/internal/ui/selector.go` + `ku/internal/ui/fuzzy.go`.
- ku confirm: `ku/internal/ui/confirm_view.go`.
- ku theme: `ku/internal/ui/theme.go`.
- lyrike current inline states: `lyrike-studio-tui/internal/tui/view.go:51-56` (fetch/picker short-circuits), `internal/tui/model.go:36-44` (Model struct), `internal/tui/project_picker.go`, `internal/tui/fetch_input.go`.

## Decisions (with rationale)
1. **Bottom-up build order.** Theme and helpers first, then overlay primitives, then components. Each wave is independently buildable and demo-testable, reducing integration risk.
2. **Use `bubbles/v2/textinput`.** Reimplementing a cursor-capable input is error-prone and out of scope; bubbles is the idiomatic Bubble Tea choice.
3. **Project picker is the first fuzzy selector.** It is the only existing list UI in lyrike-studio-tui, so porting it validates the selector immediately.
4. **Keep backend/playback/domain untouched.** The request is strictly TUI UI integration.

## Scope IN
- Theme/Palette system.
- Overlay enum + `overlayCenter`/`overlayBlock`.
- Context-aware footer.
- Fuzzy selector component and fuzzy ranking.
- Confirm dialog.
- Help overlay.
- Conversion of project picker and fetch input to overlays.

## Scope OUT (Must NOT have)
- ku's Kubernetes/resource logic, sidebar, table views, terminal overlay, command preview modal.
- Backend, playback, transcription, publish logic changes.
- Redesign of media/waveform/editor/publish panels beyond style migration.
- New CLI flags or persistent user preferences in this plan.

## Open questions
None remaining; all forks resolved with announced defaults above. User may veto any default at approval.

## Approval gate
status: awaiting-approval
Pending action: user reviews `.omo/plans/ku-ui-integration.md` and replies with approval, scope change, or rejection. Implementation begins only after explicit approval.
