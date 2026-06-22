# tui-fetch-input - Work Plan

## TL;DR (For humans)
<!-- Fill this LAST, after the detailed plan below is written, so it summarizes the REAL plan. -->
<!-- Plain English for a non-engineer: NO file paths, NO todo numbers, NO wave/agent/tool names. -->

**What you'll get:** Trong TUI sẽ có một ô nhập (modal) mở bằng `Ctrl-O` (hoặc nhấn `n` trong project picker) để dán URL YouTube hoặc nhập video ID, ví dụ `https://www.youtube.com/watch?v=P0N0h_EOS-c`. TUI sẽ gọi backend fetch media, tự đặt project ID theo video ID, và nếu đã có draft cho video đó thì load draft cũ thay vì ghi đè. Ô nhập "New project ID" thủ công hiện tại bị bỏ.

**Why this approach:** (1) Project ID nên được suy ra từ video ID thay vì bắt người dùng gõ tay — đúng ngữ nghĩa và tránh nhập sai. (2) Nhập URL trực tiếp trong TUI là cách tự nhiên nhất để bắt đầu một project mới mà không cần flag `--video-id` từ CLI. (3) Khi draft đã tồn tại, load lyrics/metadata cũ để không mất công sức người dùng.

**What it will NOT do:** Không đổi backend API hay `storage.Store`; không import `internal/server` vào TUI; không ghi đè draft cũ; không thêm dependency mới; không làm hỏng test TUI hiện có.

**Effort:** Medium
**Risk:** Low — chỉ thêm UI state và luồng command trong TUI, không đụng backend.
**Decisions to sanity-check:** (1) `Ctrl-O` thay vì phím khác; (2) draft cũ được load tự động khi video ID trùng; (3) `n` trong picker mở fetch modal thay vì nhập project ID.

Your next move: approve để tôi chạy high-accuracy review (dual Momus), hoặc nói "start work" để bắt đầu thực thi. Full execution detail follows below.

---

> TL;DR (machine): <1 line - effort, risk, deliverables>

## Scope
### Must have
- `internal/tui/fetch_input.go`: a `fetchInput` modal state struct plus a pure helper `parseVideoIDInput(raw string) (videoID, sourceURL string, ok bool)` that accepts a YouTube watch URL (`https://www.youtube.com/watch?v=P0N0h_EOS-c`), `youtu.be/<id>`, `music.youtube.com/watch?v=<id>`, or a bare 11-char video ID, and returns the normalized video ID plus the original URL when the input was a URL. No `internal/server` import.
- `Ctrl-O` global key binding that opens the fetch modal at any time (model field `fetchInput`, `openFetchInput()`). The modal collects one text line, supports backspace, `Esc` cancel, and `Enter` submit. `Ctrl-O` is chosen because `Ctrl-F` is already used by the editor for "snap selection to active playing line" (`internal/tui/keys.go:23`, `internal/tui/editor/render_help.go:9`).
- On `Enter`: parse the input. If a draft already exists for the returned video ID, `loadProject(videoID)` first (preserves lyrics/metadata), then issue `backend.Fetch` so audio/peaks load. If no draft exists, set `projectID = videoID`, reset the editor to a default document, mark `dirty = true`, then issue `backend.Fetch`. If the current project is dirty and the entered video ID differs from the current project, show a confirm step before replacing.
- `fetchMediaMsg` handling also sets `m.projectID = videoID` when it is empty and records `m.sourceURL` from `resp.SourceURL` so the CLI-less startup path still derives a project.
- Project picker rework: pressing `n` in the picker opens the fetch modal instead of the manual "New project ID" input. `saveDraft()` with no project also opens the fetch modal instead of the old create picker. The "No projects" picker footer reads `n: new from URL | Esc: cancel`.
- `renderFetchInput` modal renderer integrated into `renderLayout` so the modal overlays the shell when active, mirroring `renderProjectPicker`.
- Docs: `docs/keybindings.md`, `README.md`, and the editor help line list `Ctrl-O` and the new "new from URL" flow.
- Go table tests for the parser, modal open, enter/load-existing, enter/new-project, dirty-confirm, invalid input, picker `n`, and save-without-project flows.

### Must NOT have (guardrails, anti-slop, scope boundaries)
- Do NOT remove or change the `storage.Store` interface, `FileStore`, or any backend HTTP contract.
- Do NOT import `internal/server` from the TUI; the TUI must not duplicate the backend fetch handler. The parser only extracts the video ID locally; media fetch stays a backend call.
- Do NOT block the TUI on fetch — it must remain an async `tea.Cmd` returning `fetchMediaMsg`.
- Do NOT overwrite an existing draft's lyrics when a draft exists for the video ID; load it.
- Do NOT keep the manual "New project ID" text input as the primary creation path. Project creation is driven by URL/video ID entry.
- Do NOT add new third-party dependencies.
- Do NOT regress existing TUI tests (`go test ./...` must stay green).

## Verification strategy
> Zero human intervention - all verification is agent-executed.
- Test decision: TDD for every behavior-changing todo. Capture RED before production changes and GREEN after.
- Framework: Go `testing` + table-driven tests. Use the existing `memoryDraftStore` fake in `internal/tui/project_picker_test.go` for draft load/new cases; use `httptest` only if a real fetch command assertion is needed.
- Evidence: `.omo/evidence/task-<N>-tui-fetch-input.<ext>`
- Final checks: `go test ./...`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` clean, tmux Ctrl-O smoke against a seeded local `serve`.

## Execution strategy
### Parallel execution waves
> Wave 1 (state + parser + keybinding + render) -> Wave 2 (flow + picker integration) -> Wave 3 (msg fallback + docs + final gate).

### Dependency matrix
| Todo | Depends on | Blocks | Can parallelize with |
| --- | --- | --- | --- |
| 1 | none | 2,3,4,5 | none |
| 2 | 1 | 7 | 3 |
| 3 | 1 | 7 | 2 |
| 4 | 1 | 5,6,7 | none (after 1) |
| 5 | 1,2,4 | 7 | 6 |
| 6 | 2,4,5 | 7 | none |
| 7 | all prior | none | none |

## Todos
> Implementation + Test = ONE todo. Never separate.
<!-- APPEND TASK BATCHES BELOW THIS LINE WITH edit/apply_patch - never rewrite the headers above -->

- [x] 1. Add fetchInput modal state + parseVideoIDInput helper
  What to do: Create `internal/tui/fetch_input.go`. Define `type fetchInput struct { active bool; input string; mode fetchInputMode; targetVideoID string; targetSourceURL string }` with modes `fetchInputClosed`, `fetchInputEnter`, `fetchInputConfirmReplace`. Add `func (f fetchInput) active() bool`. Add `func parseVideoIDInput(raw string) (videoID, sourceURL string, ok bool)` that: trims input; if input contains `://`, parse with `net/url`; for `youtu.be` return first path segment; for `youtube.com`/`music.youtube.com` return `v` query param or `/embed/`/`/v/` prefix; otherwise treat the trimmed string as a bare video ID and validate with `draft.NewProjectID` (video IDs are a subset of project ID chars). Return `(videoID, sourceURL, true)` on success, `("", "", false)` on failure. Do NOT import `internal/server`.
  Must NOT do: Do NOT mutate global state. Do NOT call the backend. Do NOT use regex where `net/url`/`strings` suffice.
  Parallelization: Wave 1 | Blocked by: none | Blocks: 2,3,4,5
  References:
  - Modal pattern: `internal/tui/project_picker.go:14-29`.
  - Model fields to extend: `internal/tui/model.go:31-55`.
  - Project ID validation: `internal/domain/draft/types.go:23-57`.
  - Backend extraction logic (reference only, do NOT import): `internal/server/http.go:587-607` (`extractYouTubeVideoID`) and `internal/server/utils.go:42-62` (`NormalizeVideoID`).
  - Example input: `https://www.youtube.com/watch?v=P0N0h_EOS-c`.
  Acceptance criteria:
  - [ ] RED test first: table-driven `TestParseVideoIDInput` covering watch URL, `youtu.be/<id>`, `music.youtube.com/watch?v=<id>`, bare ID `P0N0h_EOS-c`, invalid `""`, invalid `"abc def"`, non-YouTube URL (`https://example.com` -> ok=false). Add a parity assertion that `parseVideoIDInput` matches `extractYouTubeVideoID` output for the same inputs (reference `internal/server/http.go:587-607`).
  - [ ] `go test ./internal/tui -run TestParseVideoIDInput -count=1 -v` exits 0.
  - [ ] `go vet ./internal/tui` exits 0.
  QA scenarios:
  ```
  Scenario: parser parity
    Tool: bash
    Steps: go test ./internal/tui -run TestParseVideoIDInput -count=1 -v | tee .omo/evidence/task-1-tui-fetch-input.txt
    Expected: PASS; watch URL -> P0N0h_EOS-c; youtu.be -> id; bare id -> id; invalid -> ok=false.
    Evidence: .omo/evidence/task-1-tui-fetch-input.txt
  ```
  Commit: YES | `feat(tui): add fetch input modal state and url parser` | Files: [`internal/tui/fetch_input.go`, `internal/tui/fetch_input_test.go`]

- [x] 2. Wire Ctrl-O global key to open the fetch modal
  What to do: Add `keyActionOpenFetch` to `internal/tui/keymap.go` and map `Ctrl-O` (`key.Code == 'o' && key.Mod == tea.ModCtrl`) in `globalKeyAction`. Add `case keyActionOpenFetch: m = m.openFetchInput()` in `applyRootKeyAction` (`internal/tui/keys.go`). Add `func (m Model) openFetchInput() Model` in `fetch_input.go` that sets `m.fetchInput = fetchInput{active: true, mode: fetchInputEnter}` and a status line. Add a `fetchInput` field to `Model` (`internal/tui/model.go`). Do not render yet (todo 3). Do NOT touch the editor's `Ctrl-F`/`f` snap routing at `keys.go:23`.
  Must NOT do: Do NOT bind plain `f` or `Ctrl-F` (editor snap). Do NOT open the modal while the project picker or metadata editor is active; `Update` already gates on `m.picker.active()` and `m.metadataEditor.active`.
  Parallelization: Wave 1 | Blocked by: 1 | Blocks: 7 | Can parallelize with: 3
  References:
  - Keymap: `internal/tui/keymap.go:5-36`.
  - Root action dispatch: `internal/tui/keys.go:32-57`.
  - Update gating: `internal/tui/update.go:30-37`.
  - Model struct: `internal/tui/model.go:31-55`.
  Acceptance criteria:
  - [ ] RED test first: `TestCtrlOOpensFetchInput` sends `tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl}` and asserts `got.fetchInput.active` is true and `got.fetchInput.mode == fetchInputEnter`.
  - [ ] `go test ./internal/tui -run TestCtrlO -count=1 -v` exits 0.
  - [ ] Existing `TestProjectPicker_ctrlPOpensProjectList` still passes.
  QA scenarios:
  ```
  Scenario: Ctrl-O opens modal
    Tool: bash
    Steps: go test ./internal/tui -run TestCtrlFOpensFetchInput -count=1 -v | tee .omo/evidence/task-2-tui-fetch-input.txt
    Expected: PASS; fetchInput.active true.
    Evidence: .omo/evidence/task-2-tui-fetch-input.txt
  ```
  Commit: YES | `feat(tui): bind Ctrl-O to open fetch modal` | Files: [`internal/tui/keymap.go`, `internal/tui/keys.go`, `internal/tui/model.go`, `internal/tui/fetch_input.go`, `internal/tui/fetch_input_test.go`]

- [x] 3. Render the fetch input modal
  What to do: Add `renderFetchInput(f fetchInput, width, height int) string` in `internal/tui/fetch_input.go` (or `view.go`) showing `Fetch Media\nYouTube URL or video ID: <input>\nEnter: fetch | Esc: cancel` for `fetchInputEnter`, and `Unsaved changes will be replaced.\nEnter: fetch <targetVideoID> | Esc: cancel` for `fetchInputConfirmReplace`. In `renderLayout` (`internal/tui/view.go`), before the project picker branch, add `if m.fetchInput.active { return renderFetchInput(m.fetchInput, m.width, m.height) }`. Reuse `focusedBorder` like `renderProjectPicker`.
  Must NOT do: Do NOT let the modal text overlap at 80x24; cap input display width to `width-4`. Do NOT render the modal when inactive.
  Parallelization: Wave 1 | Blocked by: 1 | Blocks: 7 | Can parallelize with: 2
  References:
  - Layout: `internal/tui/view.go:47-81`.
  - Picker renderer: `internal/tui/project_picker.go:167-204`.
  - Border styles: `internal/tui/view.go:14-22`.
  Acceptance criteria:
  - [ ] RED test first: `TestRenderFetchInput` asserts `renderFetchInput(fetchInput{active: true, mode: fetchInputEnter, input: "abc"}, 80, 24)` contains `YouTube URL or video ID:` and `abc`; confirm mode contains `Unsaved changes`. Add `TestRenderFetchInputFits80x24` with a long input (e.g. 120 chars) that asserts every output line width ≤ 80 and total line count ≤ 24.
  - [ ] `go test ./internal/tui -run TestRenderFetchInput -count=1 -v` exits 0.
  QA scenarios:
  ```
  Scenario: modal render
    Tool: bash
    Steps: go test ./internal/tui -run TestRenderFetchInput -count=1 -v | tee .omo/evidence/task-3-tui-fetch-input.txt
    Expected: PASS; enter mode shows prompt + input; confirm mode shows replace warning.
    Evidence: .omo/evidence/task-3-tui-fetch-input.txt
  ```
  Commit: YES | `feat(tui): render fetch input modal` | Files: [`internal/tui/fetch_input.go`, `internal/tui/view.go`, `internal/tui/fetch_input_test.go`]

- [x] 4. Implement updateFetchInput text handling + applyFetch flow
  What to do: In `Update` (`internal/tui/update.go`), after the project picker/metadata editor gates, add `if m.fetchInput.active { return m.updateFetchInput(msg) }`. Implement `updateFetchInput(msg tea.KeyPressMsg) (Model, tea.Cmd)` in `fetch_input.go`: `Esc` closes; `Backspace` deletes last rune; printable text appends; `Enter` parses via `parseVideoIDInput`. On invalid input, set status `invalid url or video id` and keep modal. On valid input, if `m.dirty && m.projectID != "" && videoID != m.projectID`, switch to `fetchInputConfirmReplace` storing `m.fetchInput.targetVideoID = videoID` and `m.fetchInput.targetSourceURL = sourceURL`. On confirm `Enter`, read `videoID = m.fetchInput.targetVideoID`, `sourceURL = m.fetchInput.targetSourceURL`, then call `m, cmd = m.applyFetch(videoID, sourceURL)`. `applyFetch`: attempt `m.draftStore.Load(draft.ProjectID(videoID))`; if it returns no error and a non-zero snapshot, call `m.loadProject(draft.ProjectID(videoID))` (reuses `project_picker.go`), set `m.sourceURL = sourceURL` (or keep fetched), set `m.fetchInput = fetchInput{}`; if load returns a not-found error, set `m.projectID = draft.ProjectID(videoID)`, `m.videoID = videoID`, `m.sourceURL = sourceURL`, reset `m.trackName, m.artistName, m.albumName = "", "", ""`, reset `m.editor = editor.NewPanel(newDefaultDocument())`, `m.media = media.NewPanel()`, `m.dirty = true`, `m.fetchInput = fetchInput{}`. Return a fetch `tea.Cmd` that calls `m.client.Fetch` with `{VideoID: videoID, URL: sourceURL}` and returns `fetchMediaMsg`; if `m.client == nil`, return nil cmd and status `backend unavailable`. Add `newDefaultDocument()` in `fetch_input.go` using `lyrics.NewDocument` with one 10s placeholder line (reuse `cmd/lyrike-studio-tui/main.go:232-239` pattern; duplication accepted with a `// dup ok: TUI default doc` comment). Also update the test fake `memoryDraftStore.Load` in `internal/tui/project_picker_test.go` to return `draft.Snapshot{}, errors.New("not found")` when `id` is not in `s.loads`, matching `FileStore` not-found semantics.
  Must NOT do: Do NOT block on the HTTP call. Do NOT overwrite lyrics when a draft loads. Do NOT treat a zero-value snapshot as found. Do NOT panic when `m.client == nil`. Do NOT leave stale `albumName`/`trackName`/`artistName` on the new-project path.
  Parallelization: Wave 2 | Blocked by: 1 | Blocks: 5,6,7
  References:
  - Fetch msg handling: `internal/tui/update.go:44-63`.
  - Init fetch command pattern: `internal/tui/model.go:107-124`.
  - loadProject: `internal/tui/project_picker.go:147-165`.
  - saveDraft/no-project: `internal/tui/keys.go:117-122`.
  - backend.Fetch: `internal/integrations/backend/client.go:42-65`.
  - Fake store: `internal/tui/project_picker_test.go:11-35` (update `Load` not-found semantics).
  - Real not-found semantics: `internal/storage/store.go:108-117` (`StorageError{Code: CodeDraftNotFound}`).
  Acceptance criteria:
  - [ ] RED tests first: `TestFetchInputEnterLoadsExistingDraft` (store has draft for `P0N0h_EOS-c`; submit bare id; assert `projectID == "P0N0h_EOS-c"`, editor document came from the loaded snapshot, `dirty == false`, `fetchInput.active == false`); `TestFetchInputEnterNewProject` (empty store with fixed not-found error; submit id; assert `projectID == id`, `dirty == true`, `trackName == ""`, `albumName == ""`, editor reset); `TestFetchInputDirtyConfirm` (dirty current project `oldid`; submit `newid`; assert `fetchInput.mode == fetchInputConfirmReplace` and `targetVideoID == "newid"`; second `Enter` applies and `projectID == "newid"`); `TestFetchInputInvalidInput` (submit `"abc def"`; assert modal still active, status contains `invalid`).
  - [ ] `memoryDraftStore.Load` returns a not-found error for missing IDs (unit-asserted or covered by `TestFetchInputEnterNewProject`).
  - [ ] `go test ./internal/tui -run TestFetchInput -count=1 -v` exits 0.
  QA scenarios:
  ```
  Scenario: fetch input flow
    Tool: bash
    Steps: go test ./internal/tui -run 'TestFetchInput' -count=1 -v | tee .omo/evidence/task-4-tui-fetch-input.txt
    Expected: PASS; load-existing, new-project, dirty-confirm, invalid all green.
    Evidence: .omo/evidence/task-4-tui-fetch-input.txt
  ```
  Commit: YES | `feat(tui): implement fetch input submit and project flow` | Files: [`internal/tui/fetch_input.go`, `internal/tui/update.go`, `internal/tui/fetch_input_test.go`]

- [x] 5. Replace project picker "New project ID" with fetch modal + save fallback
  What to do: In `internal/tui/project_picker.go`, change `updateProjectPickerChoose` `case msg.Code == 'n'` to call `m = m.openFetchInput()` and close the picker (`m.picker = projectPicker{}`) instead of switching to `projectPickerCreate`. Remove the `projectPickerCreate` mode constant, `updateProjectPickerCreate`, the `Enter`-with-zero-projects → `projectPickerCreate` branch in `updateProjectPickerChoose` (line 90-93), and the `case projectPickerCreate:` branch in `renderProjectPicker` (line 170-174). Update `withProjects` so zero projects keeps `projectPickerChoose` (do not force create). Update `renderProjectPicker` zero-projects footer to `n: new from URL | Esc: cancel`. In `internal/tui/keys.go` `saveDraft()`, replace `m.picker = projectPicker{mode: projectPickerCreate}` with `m = m.openFetchInput()` and status `fetch a video before saving`.
  Must NOT do: Do NOT remove the dirty-load confirm path. Do NOT delete `loadProject`. Do NOT change `OpenProjectPickerOnStartup`. Do NOT leave any `projectPickerCreate` references (build must pass).
  Parallelization: Wave 2 | Blocked by: 1,2,4 | Blocks: 7 | Can parallelize with: 6
  References:
  - Picker update: `internal/tui/project_picker.go:75-133`.
  - Picker render: `internal/tui/project_picker.go:167-204`.
  - Save no-project: `internal/tui/keys.go:117-122`.
  - Save test: `internal/tui/project_picker_test.go:56-69`.
  Acceptance criteria:
  - [ ] RED tests first: `TestProjectPickerNOpensFetchInput` (choose mode + `n` -> `fetchInput.active`, picker closed); `TestSaveWithoutProjectOpensFetchInput` (no project + Ctrl-S -> `fetchInput.active == true` and `picker.active() == false`); `TestProjectPickerNoProjectsShowsNewFromURL` (empty store -> render contains `new from URL`).
  - [ ] `go test ./internal/tui -run 'TestProjectPicker|TestSaveWithoutProject' -count=1 -v` exits 0.
  - [ ] `go build ./internal/tui` passes with no `projectPickerCreate` references (`grep -n projectPickerCreate internal/tui/*.go` returns nothing).
  - [ ] Existing `TestProjectSave_withoutProjectOpensCreatePicker` is renamed to `TestProjectSave_withoutProjectOpensFetchInput` and asserts `got.fetchInput.active` is true and `got.picker.active() == false`.
  QA scenarios:
  ```
  Scenario: picker + save fallback
    Tool: bash
    Steps: go test ./internal/tui -run 'TestProjectPicker|TestSaveWithoutProject' -count=1 -v | tee .omo/evidence/task-5-tui-fetch-input.txt
    Expected: PASS; n opens fetch modal; save without project opens fetch modal.
    Evidence: .omo/evidence/task-5-tui-fetch-input.txt
  ```
  Commit: YES | `feat(tui): replace manual project id input with fetch modal` | Files: [`internal/tui/project_picker.go`, `internal/tui/keys.go`, `internal/tui/project_picker_test.go`]

- [x] 6. Set projectID/sourceURL fallback in fetchMediaMsg + update docs
  What to do: In `internal/tui/update.go` `case fetchMediaMsg:`, after setting `m.videoID`, add `if m.projectID == "" { m.projectID, _ = draft.NewProjectID(msg.resp.VideoID) }` and `if msg.resp.SourceURL != nil { m.sourceURL = *msg.resp.SourceURL }`. Update `docs/keybindings.md` with `Ctrl-O | Global | Open the fetch modal to enter a YouTube URL or video ID.` and update the `Ctrl-P` row to note `n` creates from URL. Update `README.md` Quick Start to mention `Ctrl-O` for in-TUI URL entry. Update the editor help line in `internal/tui/editor/render_help.go` to add `Ctrl-O: Fetch media from URL` (do NOT modify the existing `f / Ctrl-F: Snap selection to active playing line` line at `render_help.go:9`).
  Must NOT do: Do NOT overwrite a non-empty `projectID`. Do NOT change backend response shapes. Do NOT touch the editor `Ctrl-F` snap help.
  Parallelization: Wave 3 | Blocked by: 2,4,5 | Blocks: 7
  References:
  - fetchMediaMsg: `internal/tui/update.go:44-63`.
  - Keybindings doc: `docs/keybindings.md:1-21`.
  - README: `README.md`.
  - Help line: `internal/tui/editor/render_help.go:9` (existing snap help, leave intact) and `:34` (Ctrl-P area, add Ctrl-O nearby).
  Acceptance criteria:
  - [ ] RED test first: `TestFetchMediaMsgSetsProjectIDWhenEmpty` sends `fetchMediaMsg{resp: backend.FetchResponse{VideoID: "P0N0h_EOS-c", SourceURL: ptr(url)}}` to a model with empty projectID; asserts `projectID == "P0N0h_EOS-c"` and `sourceURL == url`. Add `TestFetchMediaMsgPreservesNonEmptyProjectID`: model with `projectID = "existing"`, send same `fetchMediaMsg`; assert `projectID == "existing"`.
  - [ ] `go test ./internal/tui -run TestFetchMediaMsg -count=1 -v` exits 0.
  - [ ] `docs/keybindings.md` lists Ctrl-O; `README.md` mentions Ctrl-O; editor help lists Ctrl-O.
  QA scenarios:
  ```
  Scenario: msg fallback + docs
    Tool: bash
    Steps: go test ./internal/tui -run TestFetchMediaMsg -count=1 -v | tee .omo/evidence/task-6-tui-fetch-input.txt
    Expected: PASS; projectID and sourceURL set from fetch response.
    Evidence: .omo/evidence/task-6-tui-fetch-input.txt
  ```
  Commit: YES | `feat(tui): derive project id from fetch and document Ctrl-O` | Files: [`internal/tui/update.go`, `docs/keybindings.md`, `README.md`, `internal/tui/editor/render_help.go`, `internal/tui/fetch_input_test.go`]

- [x] 7. Final verification: full gate + tmux Ctrl-O smoke
  What to do: Run `go test ./...`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty). Run a tmux smoke: seed `.cache_smoke/media/P0N0h_EOS-c.json` (`{"videoId":"P0N0h_EOS-c","trackName":"Smoke Track","artistName":"Smoke Artist","duration":120.5,"cachedAt":"2026-06-22T00:00:00Z"}`), `.cache_smoke/audio/P0N0h_EOS-c/original.mp3` (10 dummy bytes), `.cache_smoke/peaks/P0N0h_EOS-c/original.json` (`{"videoId":"P0N0h_EOS-c","source":"original","duration":120.5,"samples":400,"peaks":[0.1,0.2],"sourceFile":"original.mp3","generatedAt":"2026-06-22T00:00:00Z"}`), start `go run ./cmd/lyrike-studio-tui serve --port 18090 --cache-dir ./.cache_smoke`, run TUI without `--video-id`, press `Ctrl-O`, type `https://www.youtube.com/watch?v=P0N0h_EOS-c`, press Enter, wait, capture pane; assert capture contains the seeded track name and `fetch complete`; quit. Confirm no leftover tmux sessions/ports/temp dirs.
  Must NOT do: Do NOT declare complete from unit tests alone. Do NOT leave QA state running. Do NOT use `--demo` for the serve integration smoke.
  Parallelization: Wave 3 | Blocked by: all prior | Blocks: none
  References:
  - Seed layout: `media/<id>.json`, `audio/<id>/original.mp3`, `peaks/<id>/original.json` under `LYRIKE_CACHE_DIR`.
  - Plan: `.omo/plans/tui-fetch-input.md`.
  Acceptance criteria:
  - [ ] `go test ./...` exits 0.
  - [ ] `go test -race ./...` exits 0.
  - [ ] `go vet ./...` exits 0.
  - [ ] `gofmt -l .` outputs nothing.
  - [ ] tmux capture contains seeded track name and `fetch complete`; no panic.
  - [ ] `tmux ls` shows no `ulw-qa-*` sessions; no ports 18090+ bound.
  QA scenarios:
  ```
  Scenario: full gate
    Tool: bash
    Steps: go test ./... && go test -race ./... && go vet ./... && gofmt -l . | tee .omo/evidence/task-7-tui-fetch-input-gate.txt
    Expected: all exit 0; gofmt empty.
    Evidence: .omo/evidence/task-7-tui-fetch-input-gate.txt

  Scenario: Ctrl-O TUI+serve smoke
    Tool: tmux
    Steps: seed cache; go run ./cmd/lyrike-studio-tui serve --port 18090 --cache-dir ./.cache_smoke &; tmux new-session -d -s ulw-qa-tui-fetch 'go run ./cmd/lyrike-studio-tui --backend http://127.0.0.1:18090'; tmux set-option -t ulw-qa-tui-fetch remain-on-exit on; sleep 4; tmux send-keys -t ulw-qa-tui-fetch C-f; tmux send-keys -t ulw-qa-tui-fetch 'https://www.youtube.com/watch?v=P0N0h_EOS-c' Enter; sleep 4; tmux capture-pane -t ulw-qa-tui-fetch -pS -200 > .omo/evidence/task-7-tui-fetch-input-tmux.txt; tmux send-keys -t ulw-qa-tui-fetch q; tmux kill-session -t ulw-qa-tui-fetch; kill server; rm -rf ./.cache_smoke
    Expected: capture shows seeded track name and fetch complete; no panic.
    Evidence: .omo/evidence/task-7-tui-fetch-input-tmux.txt
  ```
  Commit: YES | `chore(tui): verify fetch input feature` | Files: [all completed task files]

## Final verification wave
> Runs in parallel after ALL todos. ALL must APPROVE. Surface results and wait for the user's explicit okay before declaring complete.
- [x] F1. Plan compliance audit — every todo done, every acceptance criterion met, every evidence file exists.
- [x] F2. Code quality review — `go vet`/`gofmt` clean, no oversized file (>250 LOC pure without `SIZE_OK`), no dead code, idiomatic Go, modal does not overlap at 80x24.
- [x] F3. Real manual QA — tmux Ctrl-O TUI+serve smoke, unit tests, race detector.
- [x] F4. Scope fidelity — no backend contract change, no `internal/server` import from TUI, existing drafts loaded not overwritten, no new deps.

## Commit strategy
- One logical change per commit. Conventional Commits: `<type>(<scope>): <imperative summary>`.
- Every commit must build and pass its task-specific tests before the next task starts.
- Do not auto-commit unless the session explicitly authorizes commits.
- Final commit footer, if authorized: `Plan: .omo/plans/tui-fetch-input.md`.

## Success criteria
- `Ctrl-O` opens a fetch modal in the running TUI; typing a YouTube URL or video ID fetches media from the backend.
- Entering a video ID that already has a draft loads that draft instead of creating a blank project.
- The old "New project ID" manual input is gone; project creation is URL/video driven.
- `go test ./...` and `go test -race ./...` pass; `go vet` clean; `gofmt -l .` empty.
- No existing TUI tests regress; no backend contract or `storage.Store` interface changes.
