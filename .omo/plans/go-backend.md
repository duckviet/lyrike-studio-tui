# go-backend - Work Plan

## TL;DR (For humans)
<!-- Fill this LAST, after the detailed plan below is written, so it summarizes the REAL plan. -->
<!-- Plain English for a non-engineer: NO file paths, NO todo numbers, NO wave/agent/tool names. -->

**What you'll get:** Một backend Go hoàn chỉnh sống ngay trong repo `lyrike-studio-tui` (`internal/server/`), thay thế backend Python FastAPI ở repo kế bên. Cùng một HTTP contract (fetch, peaks, audio, transcribe SSE, publish, health) nên TUI hiện tại chạy không đổi. Một binary duy nhất: `lyrike-studio-tui serve` chạy backend, `lyrike-studio-tui` chạy TUI. Draft storage (lưu lyrics dở) dời khỏi máy client vào server — TUI đọc/ghi draft qua HTTP thay vì file local.

**Why this approach:** (1) Port parity — server Go phải thoả mãn đúng contract mà TUI client `internal/integrations/backend` đã pin, nên không phải sửa TUI. (2) Một binary đơn giản hoá deploy và chia sẻ cache dir giữa backend + TUI. (3) Whisper-1 only + bỏ Demucs/WhisperX/torch/numpy cắt toàn bộ heavy Python deps; yt-dlp và ffmpeg vẫn shell-out (không rewrite). (4) Bỏ CDN R2 hoàn toàn theo yêu cầu. (5) Draft store vào server để tập trung hoá persisted state.

**What it will NOT do:** Không port WhisperX/Demucs/CDN/R2; không copy secrets; không đụng repo Python; không đổi TUI client contract hiện có (chỉ thêm draft methods); không thêm browser UI hay Windows support; không regress TUI tests.

**Effort:** Large
**Risk:** High — port toàn bộ backend sang ngôn ngữ mới + chuyển draft storage architecture + giữ contract parity.
**Decisions I made for you:**
- Web framework: **chi v5** (idiomatic Go, minimal).
- yt-dlp: **shell out `exec.Command`** (binary là prereq, không rewrite).
- Transcription: **OpenAI whisper-1 only** (bạn đã xác nhận drop WhisperX).
- Bỏ **Demucs hoàn toàn**; `source=demucs` → 404.
- Bỏ **CDN R2 hoàn toàn** (bạn đã xác nhận); `cache_proxy` routes disk-only.
- Draft storage: **dời vào server** (`internal/server/drafts/`); TUI dùng `RemoteStore` (HTTP) (bạn đã xác nhận).
- Một binary, subcommand `serve`; flag `--backend` vẫn chỉ TUI vào backend nào.
- Cookies: `YOUTUBE_COOKIES` env → `/tmp/yt_cookies.txt` (base64-decode).
- **Không bao giờ copy `.env`/secrets** — document env vars only.
- ffmpeg shell-out cho peaks, numeric value parity (4 decimals; JSON formatting differs `0.0` vs `0` but decodes identically).

Tôi coi request này là open-ended và đã chọn các default trên; nếu bạn có outcome cụ thể khác, nói ra và tôi sẽ chuyển sang hỏi.

Your next move: approve để tôi chạy high-accuracy review (dual Momus), hoặc nói "start work" để bắt đầu thực thi. Full execution detail follows below.

---

> TL;DR (machine): Large/High — port Python FastAPI backend to Go internal/server/ (15 tasks, 4 waves); drop WhisperX/Demucs/CDN; whisper-1 only; move draft store to server; single binary serve+TUI; contract parity with existing TUI client.

## Scope
### Must have
- New `internal/server/` Go package: config, middleware (rate limit + CORS), cache CRUD, media/ytdlp, media/peaks (ffmpeg), transcription (OpenAI whisper-1), lrclib proxy, drafts store, http handlers — reimplementing the Python backend contract.
- `internal/server/drafts/`: FileStore (relocated from `internal/storage/`) + REST handlers `GET/PUT/DELETE /local-api/projects/{id}` and `GET /local-api/projects`.
- `internal/storage/remote.go`: `RemoteStore` (HTTP-backed) implementing the existing `storage.Store` interface; TUI real mode uses it instead of `FileStore`.
- `internal/integrations/backend/drafts.go`: new client methods `SaveDraft/LoadDraft/ListDrafts/DeleteDraft` calling the new endpoints. Existing client methods UNCHANGED.
- `serve` subcommand in `cmd/lyrike-studio-tui/main.go`; TUI wiring updated to use `RemoteStore` when a backend URL is configured.
- Port 4 Python test files to Go tests + httptest integration tests + draft endpoint tests.
- Dockerfile (multi-stage Go + ffmpeg runtime) + fly.toml reuse + README/docs + `internal/server/AGENTS.md`.
- New deps: `chi`, `openai-go`, `golang.org/x/time/rate`. NO CDN SDK.

### Must NOT have (guardrails, anti-slop, scope boundaries)
- Do NOT change existing `internal/integrations/backend/*` contracts for existing endpoints (fetch/peaks/publish/challenge/SSE) — only ADD draft methods.
- Do NOT port WhisperX, Demucs, torch, numpy, boto3, slowapi.
- Do NOT implement Cloudflare R2 / CDN / aws-sdk-go-v2 — dropped entirely.
- Do NOT copy the sibling repo's `.env` or any secrets.
- Do NOT touch `/home/duckviet/lrclib-upload/*`.
- Do NOT add word-level tap-sync, browser UI, WaveSurfer, iframe, or Windows named-pipe support.
- Do NOT implement a native Go YouTube extractor (shell out to yt-dlp) or server-side PoW.
- Do NOT regress existing TUI tests (`go test ./...` must stay green).
- Do NOT keep `FileStore` in use by the TUI in real mode — real mode uses `RemoteStore`; `FileStore` lives only in the server and demo/test fixtures.

## Verification strategy
> Zero human intervention - all verification is agent-executed.
- Test decision: TDD for every behavior-changing task. Capture RED before production changes and GREEN after.
- Framework: Go `testing` + `net/http/httptest`. Table-driven where the Python tests were table-driven.
- Evidence: `.omo/evidence/task-<N>-go-backend.<ext>`
- Final checks: `go test ./...`, `go test -race ./...`, `go vet ./...`, `gofmt -l` clean, httptest route parity smoke, tmux TUI+serve smoke.

## Execution strategy
### Parallel execution waves
> Wave 1 (foundation) → Wave 2 (core services) → Wave 3 (HTTP + drafts + wiring) → Wave 4 (hardening).

### Dependency matrix
| Todo | Depends on | Blocks | Can parallelize with |
| --- | --- | --- | --- |
| 1 | none | 2,3,4,5,6,7 | none |
| 2 | 1 | 8,9,11 | 3,4,5,6,7 |
| 3 | 1 | 8,10,11 | 2,4,5,6,7 |
| 4 | 1 | 8,10,11 | 2,3,5,6,7 |
| 5 | 1 | 11 | 2,3,4,6,7 |
| 6 | 1 | 11 | 2,3,4,5,7 |
| 7 | 1 | 8,11 | 2,3,4,5,6 |
| 8 | 2,3,4,7 | 11 | 9,10 |
| 9 | 2 | 11 | 8,10 |
| 10 | 3,4 | 11 | 8,9 |
| 11 | 3,4,5,6,7,8,9,10 | 12,13 | none |
| 12 | 11 | 14 | 13 |
| 13 | 11 | 14 | 12 |
| 14 | 12,13 | 15 | none |
| 15 | all prior | none | none |

## Todos
> Implementation + Test = ONE todo. Never separate.

- [ ] 1. Scaffold internal/server package, config, go.mod deps, AGENTS.md

  What to do: Create `internal/server/` with `config.go` (env loading: `OPENAI_API_KEY`, `OPENAI_TRANSCRIPTION_MODEL` default `whisper-1`, `TRANSCRIPTION_PROVIDER` fixed to `openai-whisper-1`, `ENABLE_LYRICS_REFINEMENT`, `FRONTEND_URL`, `RATE_LIMIT_PER_MINUTE` default 60, `RATE_LIMIT_TRANSCRIBE_PER_MINUTE` default 5, `YOUTUBE_COOKIES`, `PORT` default 8080, `LYRIKE_CACHE_DIR` default `./.cache`, `LYRIKE_DRAFT_DIR` default `./.cache/drafts`). Create cache dir layout `{cache}/media`, `{cache}/audio`, `{cache}/transcripts`, `{cache}/peaks`, `{cache}/drafts` at startup. Add `godotenv` optional load. Add go.mod deps: `github.com/go-chi/chi/v5`, `github.com/openai/openai-go`, `golang.org/x/time/rate`, `github.com/joho/godotenv`. Add `internal/server/AGENTS.md` mirroring the backend integration rules. Write `WriteCookiesFromEnv(path string) error` — make the cookies path a parameter (default `/tmp/yt_cookies.txt` in production, temp path in tests) so tests don't write to the real `/tmp`. Base64-decode if content does not start with `#` and len>20.
  Must NOT do: Do NOT add R2/CDN config. Do NOT hardcode secrets. Do NOT use viper or heavy config libs. Do NOT hardcode the cookies path in a way tests can't override — make it a parameter.

  Parallelization: Wave 1 | Blocked by: none | Blocks: 2,3,4,5,6,7

  References:
  - Python config: `/home/duckviet/lrclib-upload/backend/core/config.py:1-97` — env vars, cache dir layout, write_cookies_from_env.
  - Python models: `/home/duckviet/lrclib-upload/backend/core/models.py:1-25` — request shapes.
  - TUI go.mod: `/home/duckviet/lyrike-studio-tui/go.mod:1-30` — existing module baseline.
  - TUI AGENTS root: `/home/duckviet/lyrike-studio-tui/AGENTS.md` — RTK + rules.
  - Existing AGENTS pattern: `/home/duckviet/lyrike-studio-tui/internal/integrations/backend/AGENTS.md`.

  Acceptance criteria:
  - [ ] RED test first: `go test ./internal/server -run TestConfig` fails, then passes.
  - [ ] `go test ./internal/server -run TestConfig -v` exits 0 and asserts defaults + env override + cookie writing.
  - [ ] `go build ./internal/server` exits 0.
  - [ ] `go vet ./internal/server` exits 0.

  QA scenarios:
  ```
  Scenario: config defaults and env override
    Tool: bash
    Steps: go test ./internal/server -run TestConfigDefaults -v | tee .omo/evidence/task-1-go-backend-config.txt
    Expected: PASS; LYRIKE_CACHE_DIR defaults to ./.cache, PORT defaults to 8080, OPENAI_API_KEY read from env.
    Evidence: .omo/evidence/task-1-go-backend-config.txt

  Scenario: cookie writing from base64 env
    Tool: bash
    Steps: go test ./internal/server -run TestWriteCookiesFromEnv -v | tee .omo/evidence/task-1-go-backend-cookies.txt
    Expected: PASS; base64-decoded cookies written to /tmp/yt_cookies.txt in a temp test path.
    Evidence: .omo/evidence/task-1-go-backend-cookies.txt
  ```

  Commit: YES | `feat(server): scaffold config and cache layout` | Files: [`internal/server/config.go`, `internal/server/config_test.go`, `internal/server/AGENTS.md`, `go.mod`, `go.sum`]

- [ ] 2. Port core utils: video ID normalization, URL sanitization, JSON helpers

  What to do: Create `internal/server/utils.go` with `NormalizeVideoID(raw string) string` (strip non `[A-Za-z0-9_-]`), `SanitizeYouTubeURL(url string) string` (drop `list`/`index` params on `youtube.com`/`youtu.be`/`music.youtube.com`; pass through non-YouTube), `UTCNowISO() string`, `LoadJSON(path string) (map[string]any, error)`, `SaveJSON(path string, v any) error`. Port `test_utils.py` to `utils_test.go` as table-driven tests: watch URL, short URL, music URL, non-YouTube URL, no-list URL.
  Must NOT do: Do NOT use regex where `strings`/`net/url` suffice for URL parsing. Do NOT mutate input.

  Parallelization: Wave 1 | Blocked by: 1 | Blocks: 8,9,11

  References:
  - Python utils: `/home/duckviet/lrclib-upload/backend/core/utils.py:1-41` — exact logic to port.
  - Python tests: `/home/duckviet/lrclib-upload/backend/tests/test_utils.py:1-60` — 5 sanitize test cases + fetch sanitizes URL integration.
  - Go net/url: `https://pkg.go.dev/net/url` — Parse, Query, Encode.

  Acceptance criteria:
  - [ ] RED test first: `TestNormalizeVideoID` (strips special chars, keeps `-`/`_`, empty input→empty string, all-special→empty string), table-driven `TestSanitizeYouTubeURL` all 5 cases.
  - [ ] `go test ./internal/server -run TestSanitizeYouTubeURL -v` exits 0.
  - [ ] `go test ./internal/server -run TestNormalizeVideoID -v` exits 0.

  QA scenarios:
  ```
  Scenario: URL sanitize parity with Python
    Tool: bash
    Steps: go test ./internal/server -run TestSanitizeYouTubeURL -v | tee .omo/evidence/task-2-go-backend-utils.txt
    Expected: PASS; all 5 cases match Python test_utils.py expectations exactly.
    Evidence: .omo/evidence/task-2-go-backend-utils.txt
  ```

  Commit: YES | `feat(server): port video id and url utils` | Files: [`internal/server/utils.go`, `internal/server/utils_test.go`]

- [ ] 3. Port cache CRUD (metadata, peaks, transcripts) on disk

  What to do: Create `internal/server/cache/store.go` with path helpers and load/save for metadata (`{cache}/media/{id}.json`), peaks (`{cache}/peaks/{id}/{source}.json`), transcripts (`{cache}/transcripts/{id}.json`). Mirror the disk layout from `metadata_service.py` exactly. Atomic JSON writes (temp + rename + fsync dir). Typed errors (not found, corrupt, write failed).
  Must NOT do: Do NOT add CDN/R2 logic. Do NOT use `encoding/json` streaming for small JSON files — read/write whole file.

  Parallelization: Wave 1 | Blocked by: 1 | Blocks: 8,10

  References:
  - Python metadata_service: `/home/duckviet/lrclib-upload/backend/services/metadata_service.py:1-83` — path helpers + CRUD.
  - Python config dirs: `/home/duckviet/lrclib-upload/backend/core/config.py:18-25` — cache layout.
  - TUI atomic write pattern: `/home/duckviet/lyrike-studio-tui/internal/storage/atomic.go:1-48` — temp+rename+syncDir to reuse.

  Acceptance criteria:
  - [ ] RED test first: round-trip metadata/peaks/transcript; corrupt JSON returns typed error; missing file returns not-found.
  - [ ] `go test ./internal/server/cache -run TestStore -v` exits 0.

  QA scenarios:
  ```
  Scenario: cache round-trip in temp dir
    Tool: bash
    Steps: go test ./internal/server/cache -run TestStoreRoundTrip -v | tee .omo/evidence/task-3-go-backend-cache.txt
    Expected: PASS; metadata/peaks/transcript saved and loaded; no files outside temp dir.
    Evidence: .omo/evidence/task-3-go-backend-cache.txt
  ```

  Commit: YES | `feat(server): port disk cache crud` | Files: [`internal/server/cache/store.go`, `internal/server/cache/store_test.go`]

- [ ] 4. Port transcription types, formatter, and word-strip

  What to do: Create `internal/server/transcription/types.go` (`TranscribedWord`, `TranscribedSegment`, `TranscriptionResult`), `formatter.go` (`FormatTime(seconds float64) string` → `[mm:ss.xx]`, `FormatWords`, `BuildSyncedLyrics(result) (synced, plain string)` with enhanced LRC `<mm:ss.xx>word` when words exist, line LRC `[mm:ss.xx] text` when words absent, skip segments with malformed timing), `strip.go` (`StripWords(result) TranscriptionResult` for normal mode). Port `test_transcription_formatter.py` and `test_transcription_service_mode.py` to Go tests.
  Must NOT do: Do NOT add WhisperX alignment logic. Do NOT add Demucs. Do NOT mutate input result.

  Parallelization: Wave 1 | Blocked by: 1 | Blocks: 10,11

  References:
  - Python formatter: `/home/duckviet/lrclib-upload/backend/services/transcription/formatter.py:1-35` — exact format logic.
  - Python types: `/home/duckviet/lrclib-upload/backend/services/transcription/types.py:1-26` — dataclass shapes.
  - Python strip: `/home/duckviet/lrclib-upload/backend/services/transcription_service.py:74-82` — `_strip_words`.
  - Python formatter tests: `/home/duckviet/lrclib-upload/backend/tests/test_transcription_formatter.py:1-80` — 3 cases.
  - Python mode tests: `/home/duckviet/lrclib-upload/backend/tests/test_transcription_service_mode.py:1-43` — strip + format.

  Acceptance criteria:
  - [ ] RED test first: enhanced LRC, line LRC, malformed-skip, strip-words — all 4 cases.
  - [ ] `go test ./internal/server/transcription -run TestFormatter -v` exits 0.
  - [ ] `go test ./internal/server/transcription -run TestStripWords -v` exits 0.

  QA scenarios:
  ```
  Scenario: formatter parity with Python
    Tool: bash
    Steps: go test ./internal/server/transcription -run TestFormatter -v | tee .omo/evidence/task-4-go-backend-formatter.txt
    Expected: PASS; [00:01.00]<00:01.10>Hello <00:01.50>world for words; [00:01.00] Hello world for no words; empty for malformed.
    Evidence: .omo/evidence/task-4-go-backend-formatter.txt
  ```

  Commit: YES | `feat(server): port transcription formatter and types` | Files: [`internal/server/transcription/types.go`, `internal/server/transcription/formatter.go`, `internal/server/transcription/strip.go`, `internal/server/transcription/formatter_test.go`, `internal/server/transcription/strip_test.go`]

- [ ] 5. Port yt-dlp media layer (fetch info + download audio) via exec.Command

  What to do: Create `internal/server/media/ytdlp/ytdlp.go` with `FindCachedAudio(cacheDir, videoID) (string, bool)` (check `{cache}/audio/{id}/original.*`, fallback `{cache}/media/{id}.*` skipping `.json`), `FetchVideoInfo(url string) (map[string]any, error)` (shell out `yt-dlp` with `--quiet --no-warnings --noplaylist --skip-download --nocheckcertificate`, `--extractor-args youtube:player_client=tv,android,mweb,web`, `--extractor-args youtube:player_skip=web`, `--source-address 0.0.0.0`, `--force-ipv4`, smart-TV User-Agent + `Accept: */*` + `Accept-Language: en-US,en;q=0.9` headers, `--cookies-file` if `/tmp/yt_cookies.txt` exists; parse JSON stdout), `DownloadAudio(url, videoID, cacheDir) (string, error)` (`--format bestaudio[ext=m4a]/bestaudio/best`, `--outtmpl {cache}/audio/{id}/original.%(ext)s`, `--extractor-args youtube:player_client=tv,android,mweb` — NO `web`, NO `player_skip` (different from fetch!), smart-TV User-Agent only (no Accept/Accept-Language), `--cookies-file` if exists). Map yt-dlp errors to HTTP status codes by stderr substring: 404 if contains `video unavailable`/`private`/`removed`; 403 if contains `sign in`/`bot`/`captcha`; 502 otherwise; 500 for unknown exceptions. Write cookies to `/tmp/yt_cookies.txt` from env at startup (config task). Return typed HTTP errors.
  Must NOT do: Do NOT implement a native Go YouTube extractor. Do NOT call yt-dlp in unit tests — mock `exec.Command` or test the flag-building separately. Do NOT add a `LOCAL_COOKIES_PATH`/`cookies.txt` fallback — only `/tmp/yt_cookies.txt` from env. Do NOT reuse fetch args for download — fetch and download have DIFFERENT `player_client` lists (fetch includes `web`+`player_skip`, download does not).

  Parallelization: Wave 1 | Blocked by: 1 | Blocks: 11

  References:
  - Python audio_service: `/home/duckviet/lrclib-upload/backend/services/audio_service.py:12-113` — ydl_opts, extractor_args, headers, cookiefile, error mapping.
  - yt-dlp CLI: `https://github.com/yt-dlp/yt-dlp#usage-and-options` — CLI flag equivalents.
  - Go os/exec: `https://pkg.go.dev/os/exec` — Command, Output, CombinedOutput.

  Acceptance criteria:
  - [ ] RED test first: `TestFindCachedAudio` (found in audio dir, found in media fallback, not found), `TestBuildFetchArgs` (asserts flags: `player_client=tv,android,mweb,web`, `player_skip=web`, `source_address 0.0.0.0`, smart-TV UA + Accept + Accept-Language, cookiefile when cookies exist, `check_formats=False`), `TestBuildDownloadArgs` (asserts flags: `player_client=tv,android,mweb` — NO web, NO player_skip, `format=bestaudio[ext=m4a]/bestaudio/best`, `outtmpl={cache}/audio/{id}/original.%(ext)s`, smart-TV UA only), `TestMapYtdlpError` (table: `video unavailable`→404, `private`→404, `removed`→404, `sign in`→403, `bot`→403, `captcha`→403, other→502).
  - [ ] `go test ./internal/server/media/ytdlp -run Test -v` exits 0.
  - [ ] `go vet ./internal/server/media/ytdlp` exits 0.

  QA scenarios:
  ```
  Scenario: yt-dlp flag parity
    Tool: bash
    Steps: go test ./internal/server/media/ytdlp -run TestBuildFetchArgs -v | tee .omo/evidence/task-5-go-backend-ytdlp.txt
    Expected: PASS; flags include player_client=tv,android,mweb,web, source_address 0.0.0.0, smart-TV UA, cookiefile when cookies exist.
    Evidence: .omo/evidence/task-5-go-backend-ytdlp.txt
  ```

  Commit: YES | `feat(server): port yt-dlp media layer` | Files: [`internal/server/media/ytdlp/ytdlp.go`, `internal/server/media/ytdlp/ytdlp_test.go`]

- [ ] 6. Port peaks computation via ffmpeg shell-out

  What to do: Create `internal/server/media/peaks/peaks.go` with `ComputePeaks(audioPath string, samples int) ([]float64, error)` — shell out `ffmpeg -i {path} -f f32le -acodec pcm_f32le -ac 1 -ar {target_ar} -loglevel error -`, where `target_ar = max(1000, (samples*10)/60)`, read stdout as `[]float32` (little-endian), bucket into `samples` buckets taking max-abs per bucket, normalize if max>1.0, round to 4 decimals. Also port `ParseRangeHeader(rangeHeader string, fileSize int64) (start, end int64, error)` and `IterFileRange(path string, start, end int64) io.Reader` (64KB chunks) for audio streaming. Port `test_local_api_peaks_cache.py` peaks logic.
  Must NOT do: Do NOT add a Go audio decoder dependency. Do NOT use numpy — pure Go float32 parsing from `encoding/binary`.

  Parallelization: Wave 1 | Blocked by: 1 | Blocks: 11

  References:
  - Python compute_peaks: `/home/duckviet/lrclib-upload/backend/services/audio_service.py:147-179` — ffmpeg cmd, f32le, bucket logic, normalize.
  - Python parse_range: `/home/duckviet/lrclib-upload/backend/services/audio_service.py:126-145` — regex, suffix, 416 errors.
  - Python iter_file_range: `/home/duckviet/lrclib-upload/backend/services/audio_service.py:115-124` — 64KB chunks, seek.
  - Go encoding/binary: `https://pkg.go.dev/encoding/binary` — Read float32 little-endian.

  Acceptance criteria:
  - [ ] RED test first: `TestParseRangeHeader` (normal, suffix, empty, invalid, out-of-bounds → 416), `TestComputePeaksBuckets` (deterministic f32le fixture → expected peaks, normalize >1.0), `TestComputePeaksEmpty` (empty f32le → `[]`), `TestComputePeaksBucketSizeZero` (samples > sample count → `[max_abs]*samples`), `TestComputePeaksFfmpegFailure` (bad audio → `[]`).
  - [ ] `go test ./internal/server/media/peaks -run Test -v` exits 0.

  QA scenarios:
  ```
  Scenario: range header parsing parity
    Tool: bash
    Steps: go test ./internal/server/media/peaks -run TestParseRangeHeader -v | tee .omo/evidence/task-6-go-backend-peaks.txt
    Expected: PASS; bytes=0-499 → (0,499); bytes=-500 → suffix; bytes=0- → full; invalid → 416 error.
    Evidence: .omo/evidence/task-6-go-backend-peaks.txt
  ```

  Commit: YES | `feat(server): port peaks and range via ffmpeg` | Files: [`internal/server/media/peaks/peaks.go`, `internal/server/media/peaks/range.go`, `internal/server/media/peaks/peaks_test.go`]

- [ ] 7. Port OpenAI whisper-1 transcription provider + refinement

  What to do: Create `internal/server/transcription/openai.go` with `Transcribe(audioPath string) (TranscriptionResult, error)` using `openai-go` SDK: `client.Audio.Transcriptions.New(ctx, ...)` with `Model` from `OPENAI_TRANSCRIPTION_MODEL` env (default `whisper-1`), `ResponseFormat: verbose_json`, `TimestampGranularities: [word, segment]`. Map response segments+global words into `TranscribedSegment`/`TranscribedWord` (distribute global words to segments by timeframe, same as Python). Create `internal/server/transcription/refinement.go` with `RefineLyrics(synced, plain, trackName, artistName string, duration float64) (RefineResult, error)` using `openai-go` chat completions (`gpt-4o-mini`, temp 0.2, JSON response, same system prompt). Skip refinement if `ENABLE_LYRICS_REFINEMENT=false` or no API key. On refinement call failure, return error (the job manager handles fallback to unrefined lyrics).
  Must NOT do: Do NOT add WhisperX. Do NOT add Demucs. Do NOT block the HTTP request — transcription runs in a goroutine; the provider is called from the job manager. Do NOT hardcode `whisper-1` — read from `OPENAI_TRANSCRIPTION_MODEL` env.

  Parallelization: Wave 1 | Blocked by: 1 | Blocks: 11

  References:
  - Python openai_whisper_service: `/home/duckviet/lrclib-upload/backend/services/transcription/openai_whisper_service.py:1-100` — verbose_json, word+segment, word-to-segment mapping.
  - Python refinement: `/home/duckviet/lrclib-upload/backend/services/lyrics_refinement_service.py:1-83` — system prompt, gpt-4o-mini, JSON response.
  - openai-go: `https://pkg.go.dev/github.com/openai/openai-go` — Audio.Transcriptions, Chat.Completions.

  Acceptance criteria:
  - [ ] RED test first: `TestMapWordsToSegments` (deterministic fixture → expected segment/word mapping), `TestRefinePrompt` (asserts system+user prompt shape). Do NOT call real OpenAI API in tests.
  - [ ] `go test ./internal/server/transcription -run TestMapWords -v` exits 0.
  - [ ] `go test ./internal/server/transcription -run TestRefine -v` exits 0.

  QA scenarios:
  ```
  Scenario: word-to-segment mapping parity
    Tool: bash
    Steps: go test ./internal/server/transcription -run TestMapWordsToSegments -v | tee .omo/evidence/task-7-go-backend-openai.txt
    Expected: PASS; global words distributed to segments by timeframe exactly as Python openai_whisper_service.py.
    Evidence: .omo/evidence/task-7-go-backend-openai.txt
  ```

  Commit: YES | `feat(server): port openai whisper-1 provider` | Files: [`internal/server/transcription/openai.go`, `internal/server/transcription/refinement.go`, `internal/server/transcription/openai_test.go`]

- [ ] 8. Port transcription job manager + SSE broadcast

  What to do: Create `internal/server/transcription/job.go` with a job manager: mutex-protected map for `jobs[videoID]` (NOT `sync.Map` — needs atomic check-and-set for dedup), per-video broadcast channels for SSE subscribers, `RunTranscriptionJob(videoID, audioPath, enableRefinement, mode string)` goroutine that: under mutex checks if a job is already running for this videoID → if so, return existing (dedup, do NOT start second goroutine); sets status `running` + broadcasts `{videoId, status, startedAt, updatedAt}` (NO `job` field, matching Python `_broadcast`); calls `Transcribe`, optionally `RefineLyrics` (if refinement fails, log and proceed with original lyrics, set `is_ai_refined=false, model=""`), formats via `BuildSyncedLyrics`/`StripWords` by mode, saves transcript to cache (shape: `{videoId, status, provider, language, plain, synced, is_ai_refined, model, mode, updatedAt}`), sets status `completed` + broadcasts (or `failed` on error with `{videoId, status, error, updatedAt}`). `Subscribe(videoID) chan Event`, `Unsubscribe`, and `CurrentState(videoID) (Event, bool)` — returns the current job state for SSE replay-on-connect. Event JSON shape must match `TranscribeResponse` from `internal/integrations/backend/types.go:214-256` (status discriminator: queued/running/completed/failed). Running broadcast omits `job` field; completed broadcast includes `provider/language/plain/synced/is_ai_refined/model/mode/updatedAt`.
  Must NOT do: Do NOT use wall-clock sleeps in tests. Do NOT leak goroutines — clean unsubscribe on context cancel. Do NOT use `sync.Map` (needs mutex for dedup). Do NOT send `queued` over the broadcast channel (only POST returns queued). Do NOT include `job` in running broadcasts. Do NOT let refinement failure abort the job — fallback to unrefined lyrics.

  Parallelization: Wave 2 | Blocked by: 2,3,4,7 | Blocks: 11

  References:
  - Python transcription_service: `/home/duckviet/lrclib-upload/backend/services/transcription_service.py:34-213` — JOB_LOCK, TRANSCRIBE_JOBS, EVENT_QUEUES, _broadcast, run_transcription_job.
  - TUI contract: `/home/duckviet/lyrike-studio-tui/internal/integrations/backend/types.go:41-256` — TranscriptionStatus, sealed events, DecodeTranscribeResponse.
  - TUI SSE reader: `/home/duckviet/lyrike-studio-tui/internal/integrations/backend/sse.go:15-42` — expects `data: <json>\n\n`.

  Acceptance criteria:
  - [ ] RED test first: `TestJobRunningCompleted` (fake provider → running then completed broadcast, running payload has NO `job` field), `TestJobFailed` (error → failed broadcast), `TestJobDedupRunning` (second submit while running → returns existing, no second goroutine), `TestJobRefinementFailure` (fake refine returns error → completed with `is_ai_refined=false`), `TestSubscribeUnsubscribe` (no goroutine leak), `TestCurrentStateReplay` (after job completes, `CurrentState` returns completed event).
  - [ ] `go test ./internal/server/transcription -run TestJob -v` exits 0.

  QA scenarios:
  ```
  Scenario: job lifecycle broadcasts
    Tool: bash
    Steps: go test ./internal/server/transcription -run TestJobRunningCompleted -v | tee .omo/evidence/task-8-go-backend-job.txt
    Expected: PASS; subscriber receives running then completed events with correct JSON status discriminator.
    Evidence: .omo/evidence/task-8-go-backend-job.txt
  ```

  Commit: YES | `feat(server): port transcription job manager` | Files: [`internal/server/transcription/job.go`, `internal/server/transcription/job_test.go`]

- [ ] 9. Port rate limit + CORS middleware

  What to do: Create `internal/server/middleware/ratelimit.go` with per-IP token-bucket using `golang.org/x/time/rate`: `GetRealIP(r *http.Request) string` (CF-Connecting-IP > X-Forwarded-For[0] > RemoteAddr), per-route limits (60/min fetch, 5/min transcribe, 120/min cache), 429 response with `{"error":"rate_limit_exceeded","detail":"...","retry_after":"60"}` + `Retry-After: 60` header. Create `internal/server/middleware/cors.go` with allowed origins (localhost:3000/5173/4173 + `FRONTEND_URL` env split by comma), `allow_credentials=true`, same methods/headers/expose-headers as `main.py`, preflight `OPTIONS` handling, `max_age=600`.
  Must NOT do: Do NOT use slowapi or any external rate-limit lib beyond `x/time/rate`. Do NOT allow `*` origin with credentials.

  Parallelization: Wave 2 | Blocked by: 2 | Blocks: 11

  References:
  - Python rate_limit: `/home/duckviet/lrclib-upload/backend/core/rate_limit.py:1-32` — _get_real_ip.
  - Python main.py CORS: `/home/duckviet/lrclib-upload/backend/main.py:37-53` — origins, methods, headers, expose, max_age.
  - Python 429 handler: `/home/duckviet/lrclib-upload/backend/main.py:59-69` — error body shape.
  - Go x/time/rate: `https://pkg.go.dev/golang.org/x/time/rate` — Limiter, Wait.

  Acceptance criteria:
  - [ ] RED test first: `TestGetRealIP` (CF > XFF > RemoteAddr), `TestRateLimit429` (6th request to 5/min route → 429 + Retry-After), `TestCORSPreflight` (OPTIONS returns correct headers).
  - [ ] `go test ./internal/server/middleware -run Test -v` exits 0.

  QA scenarios:
  ```
  Scenario: rate limit 429 parity
    Tool: bash
    Steps: go test ./internal/server/middleware -run TestRateLimit429 -v | tee .omo/evidence/task-9-go-backend-middleware.txt
    Expected: PASS; 6th request returns 429 with Retry-After: 60 header and JSON error body.
    Evidence: .omo/evidence/task-9-go-backend-middleware.txt
  ```

  Commit: YES | `feat(server): port rate limit and cors middleware` | Files: [`internal/server/middleware/ratelimit.go`, `internal/server/middleware/cors.go`, `internal/server/middleware/middleware_test.go`]

- [ ] 10. Port lrclib proxy (challenge + publish)

  What to do: Create `internal/server/lrclib/proxy.go` with `RequestChallenge(ctx context.Context) (io.ReadCloser, http.Header, int, error)` and `Publish(ctx context.Context, token string, body io.Reader) (io.ReadCloser, http.Header, int, error)` — both proxy to `https://lrclib.net/api/request-challenge` and `/api/publish` using `net/http` client (30s timeout, `User-Agent: LyricsStudio/1.0.0`, `X-Publish-Token` header for publish). Return the upstream `resp.Body` (io.ReadCloser) for the chi handler to stream via `io.Copy` — do NOT buffer the entire response. The Publish handler forwards the raw request body (io.Reader) to upstream, not re-serialized JSON.
  Must NOT do: Do NOT solve PoW server-side. Do NOT buffer the entire upstream response in `[]byte` — return `io.ReadCloser` for streaming. Do NOT re-serialize the publish body — forward raw bytes. Do NOT modify the LRCLIB API contract.

  Parallelization: Wave 2 | Blocked by: 3,4 | Blocks: 11

  References:
  - Python lrclib_proxy: `/home/duckviet/lrclib-upload/backend/routes/lrclib_proxy.py:1-52` — challenge + publish proxy.
  - TUI contract: `/home/duckviet/lyrike-studio-tui/internal/integrations/backend/types.go:271-290` — ChallengeResponse{prefix,target}, PublishPayload, PublishToken.

  Acceptance criteria:
  - [ ] RED test first: `TestRequestChallengeProxy` (httptest upstream → proxied response), `TestPublishProxy` (forwards X-Publish-Token + body), `TestProxyTimeout` (upstream slow → error).
  - [ ] `go test ./internal/server/lrclib -run Test -v` exits 0.

  QA scenarios:
  ```
  Scenario: challenge proxy forwards correctly
    Tool: bash
    Steps: go test ./internal/server/lrclib -run TestRequestChallengeProxy -v | tee .omo/evidence/task-10-go-backend-lrclib.txt
    Expected: PASS; upstream receives POST with correct UA; client receives upstream status + body.
    Evidence: .omo/evidence/task-10-go-backend-lrclib.txt
  ```

  Commit: YES | `feat(server): port lrclib proxy` | Files: [`internal/server/lrclib/proxy.go`, `internal/server/lrclib/proxy_test.go`]

- [ ] 11. Wire all HTTP routes on chi router + port peaks-cache test

  What to do: Create `internal/server/http.go` wiring chi router with all routes: `POST /local-api/fetch` (rate 60; validate url or videoId present→400; videoId≠URL→400; cache-miss no URL→404; audio re-download if metadata exists but audio missing and URL provided), `POST /local-api/transcribe` (rate 5; cached transcript + mode match→completed; job running→running; job completed+mode match→completed; audio not cached→404; invalid videoId→400; force=true bypasses cache; `validated_mode()` normalizes mode to normal|karaoke default normal; start goroutine→queued), `GET /local-api/transcribe/stream/{id}` (SSE via `http.Flusher`, `data: <json>\n\n`; on connect, replay current job state first; if completed/failed, close stream immediately; otherwise subscribe for future events), `GET /local-api/audio/{id}` (Range streaming, content-type hardcoded `audio/mpeg`; 404 if not found), `GET /local-api/peaks/{id}` (samples 64-4000 else 400; `source=original|demucs` where demucs→ALWAYS 404 since Demucs dropped — intentional behavior change from Python which served cached demucs; `force=true` bypasses cache; cacheHit flag; save to cache on generate), `GET /cache/audio/{id}` (disk-only, no CDN; accept `source=original|vocal` where vocal→404 since Demucs dropped; content-type by extension: `.m4a`→`audio/mp4`, `.wav`→`audio/wav`, else `audio/mpeg`; `Cache-Control: public, max-age=86400, stale-while-revalidate=3600`; Range support), `GET /cache/peaks/{id}` (disk-only; `Cache-Control: public, max-age=3600, stale-while-revalidate=300`), `GET /cache/transcript/{id}` (disk-only; `Cache-Control: public, max-age=600, stale-while-revalidate=60`), `POST /api/request-challenge` (proxy to lrclib.net, stream response via `io.Copy`), `POST /api/publish` (proxy to lrclib.net with `X-Publish-Token`, forward raw body bytes), `GET+HEAD /` and `GET+HEAD /health` (health), `GET /healthz`. Request models: port `FetchRequest{URL?, VideoID?}` and `TranscribeRequest{VideoID, Force, EnableRefinement, Mode}` with `ValidatedMode()` method — do NOT port `ExtractRequest` (dead code). The transcript JSON shape saved by task 8 and loaded here for reuse: `{videoId, status, provider, language, plain, synced, is_ai_refined, model, mode, updatedAt}`. SSE running broadcast payload = `{videoId, status, startedAt, updatedAt}` (NO `job` field, matching Python `_broadcast`); the POST /transcribe running response includes `job` wrapper separately.
  Must NOT do: Do NOT add CDN branches to cache routes — disk-only. Do NOT change JSON response shapes from the TUI client contract. Do NOT block SSE on a single goroutine — one goroutine per subscriber. Do NOT send `queued` status over SSE (Python only returns queued from POST /transcribe, never broadcasts it). Do NOT port `ExtractRequest`. Do NOT add `LOCAL_COOKIES_PATH` fallback. Do NOT generate demucs/vocal sources (always 404).

  Parallelization: Wave 3 | Blocked by: 3,4,5,6,7,8,9,10 | Blocks: 12,13

  References:
  - Python local_api: `/home/duckviet/lrclib-upload/backend/routes/local_api.py:1-269` — all route logic.
  - Python cache_proxy: `/home/duckviet/lrclib-upload/backend/routes/cache_proxy.py:1-166` — disk-only fallback (remove CDN branches).
  - Python main.py: `/home/duckviet/lrclib-upload/backend/main.py:85-97` — router include + health.
  - Python peaks test: `/home/duckviet/lrclib-upload/backend/tests/test_local_api_peaks_cache.py:1-42` — peaks saves to cache, cacheHit false.
  - Python utils fetch test: `/home/duckviet/lrclib-upload/backend/tests/test_utils.py:31-60` — fetch sanitizes URL.
  - TUI contract: `/home/duckviet/lyrike-studio-tui/internal/integrations/backend/types.go:1-290` — exact JSON shapes.
  - chi: `https://pkg.go.dev/github.com/go-chi/chi/v5` — Router, Route, Middleware.

  Acceptance criteria:
  - [ ] RED test first: `TestFetchHandler` (URL sanitize + metadata + audioReady), `TestFetchMissingParams→400`, `TestFetchCacheMissNoURL→404`, `TestFetchVideoIdMismatch→400`, `TestPeaksHandler` (saves to cache, cacheHit false, samples bounds 400), `TestPeaksForceBypass`, `TestPeaksBadSamples→400`, `TestPeaksDemucsAlways404`, `TestAudioRange` (206 + Content-Range, `audio/mpeg` content-type), `TestAudioNotFound→404`, `TestTranscribePOST` (cached+mode match→completed; cached+mode mismatch→starts new; running→returns running; audio not cached→404; force→re-runs; invalid videoId→400), `TestSSEStream` (running then completed events in order — NOT queued; replay current state on connect; if completed, stream closes), `TestCacheRoutesNotFound→404`, `TestCacheControlHeaders`, `TestHealthEndpoints` (GET+HEAD `/`, `/health`, GET `/healthz`).
  - [ ] `go test ./internal/server -run TestHandler -v` exits 0.
  - [ ] `go vet ./internal/server` exits 0.

  QA scenarios:
  ```
  Scenario: fetch + peaks + audio parity via httptest
    Tool: bash
    Steps: go test ./internal/server -run 'TestFetchHandler|TestPeaksHandler|TestAudioRange' -v | tee .omo/evidence/task-11-go-backend-http.txt
    Expected: PASS; fetch returns videoId/trackName/duration; peaks saves to cache with cacheHit=false; audio range returns 206 + Content-Range.
    Evidence: .omo/evidence/task-11-go-backend-http.txt

  Scenario: SSE stream events match TUI contract
    Tool: bash
    Steps: go test ./internal/server -run TestSSEStream -v | tee .omo/evidence/task-11-go-backend-sse.txt
    Expected: PASS; events are `data: {"status":"running",...}\n\n` then `data: {"status":"completed",...}\n\n` (NO queued over SSE); replay current state on connect; decode via backend.DecodeTranscribeResponse succeeds.
    Evidence: .omo/evidence/task-11-go-backend-sse.txt

  Scenario: transcribe POST cached/reuse logic
    Tool: bash
    Steps: go test ./internal/server -run TestTranscribePOST -v | tee .omo/evidence/task-11-go-backend-transcribe.txt
    Expected: PASS; cached+mode match returns completed (no job started); running returns running; audio not cached returns 404; force re-runs.
    Evidence: .omo/evidence/task-11-go-backend-transcribe.txt
  ```

  Commit: YES | `feat(server): wire http routes and sse` | Files: [`internal/server/http.go`, `internal/server/http_test.go`, `internal/server/models.go`]

- [x] 12. Move FileStore into server + draft REST endpoints

  What to do: Create `internal/server/drafts/store.go` by COPYING the `FileStore` logic from `internal/storage/store.go` (atomic write, load, list, delete) into the server package, rooted at `LYRIKE_DRAFT_DIR`. The original `internal/storage/store.go` remains UNCHANGED (demo/offline/test still use it). Create `internal/server/drafts/handlers.go` with REST handlers: `GET /local-api/projects` → `[]storedProjectSummary` JSON (sorted by UpdatedAt desc, then ID asc), `GET /local-api/projects/{id}` → `storedSnapshot` JSON (`{id, metadata:{videoID,trackName,artistName,albumName,duration,updatedAt}, syncedLyrics}`), `PUT /local-api/projects/{id}` → save `storedSnapshot` (body = same JSON shape; preserve client-supplied `updatedAt`, do NOT overwrite with server time), `DELETE /local-api/projects/{id}` → 204 or 404. Define a `storedProjectSummary` type with explicit json tags: `{"id":"...","metadata":{...storedMetadata...}}` (camelCase, matching `storedMetadata`). Wire these routes into the chi router from task 11.
  Must NOT do: Do NOT change the `storedSnapshot`/`storedMetadata` JSON shape — the TUI's `conversions.go` must round-trip. Do NOT move/remove `internal/storage/store.go` — COPY only. Do NOT store drafts in XDG on the server — use `LYRIKE_DRAFT_DIR`. Do NOT overwrite `updatedAt` on PUT — preserve client-supplied value. Note: Draft JSON uses `"videoID"` (capital ID) while fetch/transcribe JSON uses `"videoId"` (camelCase) — do NOT unify these.

  Parallelization: Wave 3 | Blocked by: 11 | Blocks: 14

  References:
  - TUI FileStore: `/home/duckviet/lyrike-studio-tui/internal/storage/store.go:1-242` — Save/Load/ListProjects/Delete + atomic.
  - TUI conversions: `/home/duckviet/lyrike-studio-tui/internal/storage/conversions.go:1-82` — storedSnapshot/storedMetadata JSON shape.
  - TUI draft types: `/home/duckviet/lyrike-studio-tui/internal/domain/draft/types.go:1-86` — ProjectID, DraftID, Metadata, Snapshot, ProjectSummary.
  - TUI atomic: `/home/duckviet/lyrike-studio-tui/internal/storage/atomic.go:1-48` — writeFileAtomic, syncDir.
  - TUI errors: `/home/duckviet/lyrike-studio-tui/internal/storage/errors.go:1-42` — typed StorageError.

  Acceptance criteria:
  - [ ] RED test first: `TestDraftHandlersRoundTrip` (PUT then GET returns same snapshot), `TestDraftListSorted` (3 projects → sorted by UpdatedAt desc), `TestDraftDelete404`, `TestDraftCorruptReturnsError`.
  - [ ] `go test ./internal/server/drafts -run Test -v` exits 0.
  - [ ] JSON shape matches `storedSnapshot`/`storedMetadata` from `conversions.go` exactly.

  QA scenarios:
  ```
  Scenario: draft REST round-trip via httptest
    Tool: bash
    Steps: go test ./internal/server/drafts -run TestDraftHandlersRoundTrip -v | tee .omo/evidence/task-12-go-backend-drafts.txt
    Expected: PASS; PUT /local-api/projects/song1 with snapshot body; GET returns same syncedLyrics + metadata; DELETE removes it; GET after delete → 404.
    Evidence: .omo/evidence/task-12-go-backend-drafts.txt
  ```

  Commit: YES | `feat(server): move draft store and add rest endpoints` | Files: [`internal/server/drafts/store.go`, `internal/server/drafts/handlers.go`, `internal/server/drafts/handlers_test.go`, `internal/server/http.go`]

- [x] 13. Add RemoteStore (HTTP-backed) on TUI side + backend client draft methods

  What to do: Create `internal/storage/remote.go` with `RemoteStore` implementing `storage.Store` (Save/Load/ListProjects/Delete) via HTTP calls to `/local-api/projects*`. `RemoteStore.Save` serializes via `toStored(snapshot)` from `internal/storage/conversions.go` (which calls `lyrics.FormatLRCWithEnd` to produce `syncedLyrics`) then PUTs the JSON body. `RemoteStore.Load` GETs the body, decodes via `fromStored(stored)` (which calls `lyrics.ParseLRC` to parse `syncedLyrics` into `lyrics.Document`). `RemoteStore.ListDrafts` decodes `[]storedProjectSummary` (camelCase json tags matching `storedMetadata`). Create `internal/integrations/backend/drafts.go` with `SaveDraft(ctx, id, snapshot) error`, `LoadDraft(ctx, id) (Snapshot, error)`, `ListDrafts(ctx) ([]ProjectSummary, error)`, `DeleteDraft(ctx, id) error` — using the same `Client` and `do`/`expectStatus` helpers from `client.go`. `RemoteStore` wraps these client methods. Update `cmd/lyrike-studio-tui/main.go` `runReal` to ALWAYS use `RemoteStore` (via `backend.Client`) instead of `storage.NewDefaultStore()` — `--backend` always has a default value so "when configured" is always true; if `RemoteStore` operations fail (backend unreachable), surface the error to the TUI status line (do NOT silently fall back to `FileStore`, which would split state). `FileStore` is used ONLY by `runDemo` and tests. Replace the hardcoded cache path `/home/duckviet/lrclib-upload/backend/.cache/audio/{id}/original.mp4` with the backend's audio URL (`{backendURL}/local-api/audio/{videoID}`) — the TUI never accesses the server's filesystem directly; remove the `os.Stat` local-cache check.
  Must NOT do: Do NOT change existing `client.go`/`types.go`/`sse.go`/`fixture.go` methods. Do NOT remove `FileStore` from `internal/storage/` — it stays for demo/offline/test. Do NOT break existing `storage.Store` interface. Do NOT write `RemoteStore.Save` from scratch without reusing `toStored`/`fromStored` — reuse `internal/storage/conversions.go` directly. Do NOT silently fall back to `FileStore` in real mode on RemoteStore errors — surface the error. Do NOT use a local filesystem path for audio — use the backend audio URL.

  Note: RemoteStore tests use a local httptest mock of the draft endpoints; they do NOT depend on task 12's real handlers. Integration with the real server is verified in task 15.

  Parallelization: Wave 3 | Blocked by: 11 | Blocks: 14

  References:
  - TUI Store interface: `/home/duckviet/lyrike-studio-tui/internal/storage/store.go:15-21` — Save/Load/ListProjects/Delete.
  - TUI client: `/home/duckviet/lyrike-studio-tui/internal/integrations/backend/client.go:18-169` — Client, do, expectStatus, NewClientWithHTTPClient.
  - TUI conversions: `/home/duckviet/lyrike-studio-tui/internal/storage/conversions.go:11-82` — JSON shape RemoteStore must parse.
  - TUI main.go: `/home/duckviet/lyrike-studio-tui/cmd/lyrike-studio-tui/main.go:80-189` — runReal wiring to update.
  - Server draft endpoints: task 12 handlers.

  Acceptance criteria:
  - [ ] RED test first: `TestRemoteStoreRoundTrip` (httptest server serving draft endpoints → RemoteStore Save/Load/List/Delete), `TestRemoteStoreNotFound` (Load missing → typed error).
  - [ ] `go test ./internal/storage -run TestRemoteStore -v` exits 0.
  - [ ] `go test ./internal/integrations/backend -run TestDraft -v` exits 0.
  - [ ] Existing `go test ./...` stays green (no regressions).

  QA scenarios:
  ```
  Scenario: RemoteStore against httptest server
    Tool: bash
    Steps: go test ./internal/storage -run TestRemoteStoreRoundTrip -v | tee .omo/evidence/task-13-go-backend-remote.txt
    Expected: PASS; RemoteStore.Save then Load returns same snapshot; List returns sorted summaries; Delete removes.
    Evidence: .omo/evidence/task-13-go-backend-remote.txt

  Scenario: no TUI test regressions
    Tool: bash
    Steps: go test ./... 2>&1 | tee .omo/evidence/task-13-go-backend-regression.txt
    Expected: all packages PASS; existing TUI tests unaffected.
    Evidence: .omo/evidence/task-13-go-backend-regression.txt
  ```

  Commit: YES | `feat(tui): add remote draft store and backend draft client` | Files: [`internal/storage/remote.go`, `internal/storage/remote_test.go`, `internal/integrations/backend/drafts.go`, `internal/integrations/backend/drafts_test.go`, `cmd/lyrike-studio-tui/main.go`]

- [x] 14. Add `serve` subcommand + Dockerfile + fly.toml + docs

  What to do: Add `serve` subcommand to `cmd/lyrike-studio-tui/main.go`: `lyrike-studio-tui serve [--port 8080] [--cache-dir ./.cache]` starts the chi server from task 11+12. Create `Dockerfile` (multi-stage: `golang:1.25` build → `debian:slim` runtime with `ffmpeg` installed via apt + `yt-dlp` standalone binary downloaded from GitHub releases `https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp` into `/usr/local/bin/yt-dlp` chmod +x — no Python runtime needed; `EXPOSE 8080`; `CMD ["./lyrike-studio-tui", "serve"]`). Reuse `fly.toml` (same app/region/port). Update `README.md` with serve instructions, env vars (OPENAI_API_KEY, OPENAI_TRANSCRIPTION_MODEL, YOUTUBE_COOKIES, LYRIKE_CACHE_DIR, LYRIKE_DRAFT_DIR, etc.), prerequisites (ffmpeg, yt-dlp). Update `docs/troubleshooting.md` with missing yt-dlp/ffmpeg, OpenAI key missing, backend unavailable, draft migration note (existing XDG drafts not auto-migrated — now live server-side). Add `docs/server.md` documenting the Go backend architecture and route map.
  Must NOT do: Do NOT copy secrets into Dockerfile. Do NOT hardcode the OpenAI key. Do NOT install yt-dlp via pip (use the standalone binary to avoid Python runtime in the Docker image).

  Parallelization: Wave 4 | Blocked by: 12,13 | Blocks: 15

  References:
  - Python Dockerfile: `/home/duckviet/lrclib-upload/backend/Dockerfile:1-25` — pattern to adapt (ffmpeg, port 8080).
  - Python fly.toml: `/home/duckviet/lrclib-upload/backend/fly.toml:1-15` — reuse.
  - TUI README: `/home/duckviet/lyrike-studio-tui/README.md:1-44` — update.
  - TUI main.go: `/home/duckviet/lyrike-studio-tui/cmd/lyrike-studio-tui/main.go:26-58` — flag parsing pattern.
  - TUI docs: `/home/duckviet/lyrike-studio-tui/docs/` — existing docs dir.

  Acceptance criteria:
  - [ ] `go run ./cmd/lyrike-studio-tui serve --port 18080 &` starts; `curl http://127.0.0.1:18080/health` returns `{"status":"ok"}`; kill.
  - [ ] `docker build -t lyrike-studio-tui .` succeeds.
  - [ ] `README.md` lists serve command + env vars + prerequisites.
  - [ ] `docs/server.md` documents all routes.

  QA scenarios:
  ```
  Scenario: serve subcommand health check
    Tool: bash
    Steps: go run ./cmd/lyrike-studio-tui serve --port 18080 & sleep 2; curl -sf http://127.0.0.1:18080/health | tee .omo/evidence/task-14-go-backend-serve.txt; kill %1
    Expected: HTTP 200 + {"status":"ok","message":"..."}.
    Evidence: .omo/evidence/task-14-go-backend-serve.txt

  Scenario: docker build succeeds
    Tool: bash
    Steps: docker build -t lyrike-studio-tui . 2>&1 | tail -5 | tee .omo/evidence/task-14-go-backend-docker.txt
    Expected: build exits 0; final image tagged.
    Evidence: .omo/evidence/task-14-go-backend-docker.txt
  ```

  Commit: YES | `feat(server): add serve subcommand and docker` | Files: [`cmd/lyrike-studio-tui/main.go`, `Dockerfile`, `fly.toml`, `README.md`, `docs/server.md`, `docs/troubleshooting.md`]

- [x] 15. Final verification: full gate, tmux TUI+serve smoke, cleanup

  What to do: Run the full automated gate: `go test ./...`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty). Run a tmux smoke: start `lyrike-studio-tui serve --port 18080` in background, then run `lyrike-studio-tui --backend http://127.0.0.1:18080 --video-id dQw4w9WgXcQ` (NOT `--demo` — demo mode ignores `--backend`; use `runReal` mode to actually test TUI+serve integration) in tmux, verify fetch/peaks render and draft save via RemoteStore, quit. Also run the `--demo --backend-fixture` smoke for the fixture-only path. Confirm no leftover tmux sessions, no bound ports, no temp dirs.
  Must NOT do: Do NOT declare complete from unit tests alone. Do NOT leave QA state running. Do NOT skip the race detector. Do NOT use `--demo --backend` for the serve integration smoke — demo ignores `--backend`; use `runReal` mode.

  Parallelization: Wave 4 | Blocked by: all prior | Blocks: none

  References:
  - This plan: `.omo/plans/go-backend.md`.
  - Evidence dir: `.omo/evidence/`.
  - TUI demo: `/home/duckviet/lyrike-studio-tui/cmd/lyrike-studio-tui/main.go:60-78` — runDemo pattern.

  Acceptance criteria:
  - [ ] `go test ./...` exits 0.
  - [ ] `go test -race ./...` exits 0.
  - [ ] `go vet ./...` exits 0.
  - [ ] `gofmt -l .` outputs nothing.
  - [ ] tmux smoke capture contains fetch, peaks, publish, and draft-saved evidence.
  - [ ] `tmux ls` shows no `ulw-qa-*` sessions.
  - [ ] `git status --short` clean after commits.

  QA scenarios:
  ```
  Scenario: full Go gate
    Tool: bash
    Steps: go test ./... && go test -race ./... && go vet ./... && gofmt -l . | tee .omo/evidence/task-15-go-backend-gate.txt
    Expected: all exit 0; gofmt output empty.
    Evidence: .omo/evidence/task-15-go-backend-gate.txt

  Scenario: TUI+serve tmux smoke (runReal mode, NOT demo)
    Tool: tmux
    Steps: go run ./cmd/lyrike-studio-tui serve --port 18080 & sleep 2; tmux new-session -d -s ulw-qa-task-15 'go run ./cmd/lyrike-studio-tui --backend http://127.0.0.1:18080 --video-id dQw4w9WgXcQ'; sleep 3; tmux send-keys -t ulw-qa-task-15 q; tmux capture-pane -t ulw-qa-task-15 -pS -200 > .omo/evidence/task-15-go-backend-tmux.txt; tmux kill-session -t ulw-qa-task-15; kill %1
    Expected: capture shows fetch/peaks milestones from the real Go backend and graceful quit; no panic. (Demo mode ignores --backend, so runReal mode is required for real integration.)
    Evidence: .omo/evidence/task-15-go-backend-tmux.txt

  Scenario: no leftover QA state
    Tool: bash
    Steps: tmux ls 2>/dev/null | grep 'ulw-qa-' > .omo/evidence/task-15-go-backend-leftover.txt || true
    Expected: evidence file is empty.
    Evidence: .omo/evidence/task-15-go-backend-leftover.txt
  ```

  Commit: YES | `chore(release): verify go backend` | Files: [all completed task files]

## Final verification wave
> Runs in parallel after ALL todos. ALL must APPROVE. Surface results and wait for the user's explicit okay before declaring complete.
- [x] F1. Plan compliance audit — every task done, every acceptance criterion met, every evidence file exists.
- [x] F2. Code quality review — `go vet`/`gofmt` clean, no oversized file (>250 LOC pure without `SIZE_OK`), no dead code, idiomatic Go, chi middleware order correct.
- [x] F3. Real manual QA — tmux TUI+serve smoke, httptest route parity, docker build, all evidence captured.
- [x] F4. Scope fidelity — no WhisperX/Demucs/CDN/R2/boto3/aws-sdk introduced; TUI client existing contract unchanged; drafts moved to server; no secrets committed.

## Commit strategy
- One logical change per commit. Conventional Commits: `<type>(<scope>): <imperative summary>`.
- Every commit must build and pass its task-specific tests before the next task starts.
- Do not auto-commit unless the session explicitly authorizes commits.
- Final commit footer, if authorized: `Plan: .omo/plans/go-backend.md`.

## Success criteria
- The plan file exists at `.omo/plans/go-backend.md` with 15 implementation tasks plus 4 final verification checks.
- The Go backend in `internal/server/` reimplements every Python backend route with matching JSON contract, minus WhisperX/Demucs/CDN.
- Draft storage is owned by the server (`internal/server/drafts/`) and accessed by the TUI via `RemoteStore` over HTTP.
- The executor can start at Task 1 without further interview.
- Every task has references, acceptance criteria, concrete QA scenarios, evidence paths, and commit instructions.
- `go test ./...` and `go test -race ./...` pass; `go vet` clean; `gofmt -l` empty.
- Docker build succeeds; `serve` subcommand responds on /health.
- No secrets committed; no regression in existing TUI tests.
