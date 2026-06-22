---
slug: go-backend
status: plan-written-metis-folded
intent: unclear
pending-action: run dual high-accuracy Momus review, then present to user
approach: Port the lrclib-upload FastAPI backend into a new Go module under internal/server/ inside lyrike-studio-tui. Reimplement the same HTTP contract (routes, status codes, JSON shapes, SSE) so the existing TUI backend client (internal/integrations/backend) works for existing endpoints. Replace yt-dlp with a Go yt-dlp CLI wrapper (shell out to the yt-dlp binary) and replace WhisperX with OpenAI whisper-1 only. Drop Demucs/whisperx/torch/numpy/boto3 AND drop Cloudflare R2 CDN entirely (user explicit). Keep ffmpeg shell-out for peaks. cache_proxy routes become disk-only (no CDN branch). Move draft storage (FileStore, currently internal/storage/) INTO the server: server owns draft persistence on disk and exposes REST endpoints /local-api/projects* ; TUI keeps the Store interface + draft domain types but uses a new RemoteStore (HTTP-backed) implementation instead of FileStore. Serve via net/http with chi router. Run backend and TUI as one Go binary with subcommands.
---

# Draft: go-backend

## Components (topology ledger)
- C1 | Go HTTP server exposing the same /local-api/* and /api/* contract the TUI client expects | active | internal/server/
- C2 | YouTube media layer (fetch info + download audio) shelling out to yt-dlp binary | active | internal/server/media/ytdlp
- C3 | Peaks computation shelling out to ffmpeg | active | internal/server/media/peaks
- C4 | Transcription layer: OpenAI whisper-1 only (no WhisperX, no Demucs) | active | internal/server/transcription
- C5 | Metadata + cache CRUD on disk (mirror .cache layout) | active | internal/server/cache
- C6 | Publish proxy to LRCLIB (request-challenge + publish) | active | internal/server/lrclib
- C7 | ~~Optional CDN (Cloudflare R2)~~ | **DROPPED** (user explicit: bỏ cdn luôn) | n/a
- C8 | Rate limiting + CORS middleware | active | internal/server/middleware
- C9 | Config + env loading (drop-in for config.py) | active | internal/server/config
- C10 | TUI↔backend wiring: single binary subcommand `serve` + keep client as-is for existing endpoints | active | cmd/lyrike-studio-tui
- C11 | Tests: unit (handlers, utils, formatter) + integration (httptest) + port parity vs Python tests | active | internal/server/..._test.go
- C12 | Docs + Dockerfile/fly.toml for Go binary + AGENTS.md for internal/server | active | docs/, Dockerfile, internal/server/AGENTS.md
- C13 | Draft storage moved INTO server (FileStore relocates from internal/storage to internal/server/drafts; server exposes REST /local-api/projects*) | active | internal/server/drafts, internal/storage/remote.go
- C14 | TUI RemoteStore (HTTP-backed) implementing existing storage.Store interface, replacing FileStore usage in TUI | active | internal/storage/remote.go

## Open assumptions (announced defaults)
- A1 | Web framework | chi v2 (github.com/go-chi/chi/v5) | idiomatic Go, minimal, middleware-friendly, matches net/http handler shape; reversible (swap router) | reversible
- A2 | yt-dlp integration | shell out to the yt-dlp binary via exec.Command, not a Go reimplementation | the Python uses yt-dlp as a library; a Go native port is infeasible; yt-dlp binary must be installed (documented prereq). Same extractor_args/headers/cookiefile flags are passed through. | reversible (could later use a library binding)
- A3 | Transcription provider | OpenAI whisper-1 ONLY via the official openai-go SDK | user explicitly said drop WhisperX, use whisper-1; removes torch/whisperx/demucs/numpy deps entirely. No Demucs vocal separation, no whisperx alignment. | NOT reversible to WhisperX without a big re-plan (intended by user)
- A4 | ffmpeg peaks | shell out to ffmpeg exactly as audio_service.compute_peaks does, parse f32le PCM, bucket max-abs, normalize | preserves the existing peaks contract; avoids a Go audio decoder dependency | reversible
- A5 | No Demucs | drop demucs peaks source entirely; /local-api/peaks?source=demucs returns 404 as the Python already does when not cached | removes a heavy Python+torch dependency and a vocal.wav path; user's simplification intent | reversible later
- A6 | ~~R2 CDN optional~~ | **DROPPED entirely** | user explicit "bỏ cdn luôn"; no R2, no aws-sdk-go-v2, no cdn_service, cache_proxy routes are disk-only (or dropped if TUI doesn't use them) | NOT reversible (intended by user)
- A6b | Draft storage | Move FileStore from internal/storage/ into internal/server/drafts/; server owns draft persistence and exposes REST endpoints GET/PUT/DELETE /local-api/projects/{id} + GET /local-api/projects; TUI keeps storage.Store interface + draft domain types but uses new RemoteStore (HTTP-backed) instead of FileStore | user explicit "chuyển phần store vào backend/server"; centralizes drafts with media cache in one binary; FileStore logic (atomic write) reused server-side | reversible
- A7 | Rate limiting | token-bucket per-IP using CF-Connecting-IP/X-Forwarded-For, same per-minute values (60 fetch, 5 transcribe, 120 cache) | parity with slowapi config; implement with golang.org/x/time/rate | reversible
- A8 | CORS | same allowed origins logic (localhost dev + FRONTEND_URL env) | parity with main.py CORSMiddleware | reversible
- A9 | SSE for transcribe stream | implement with http.Flusher + text/event-stream, same `data: <json>\n\n` frames | parity with local_api.py event_generator; the TUI's sse.go reader already expects this | reversible
- A10 | Concurrency for transcription jobs | goroutine per job + sync.Map for JOB state + channels for event broadcast (replaces threading + asyncio.Queue) | idiomatic Go; the Python uses a daemon thread + asyncio.Queue per SSE subscriber | reversible
- A11 | Binary layout | ONE Go binary, `lyrike-studio-tui serve` starts the backend, default `lyrike-studio-tui` runs the TUI; flags `--backend http://127.0.0.1:8000` still point the TUI client at any backend (local Go serve, the old Python, or remote) | single-binary deploy, matches user's "IMPL backend bằng Go ngay trong dự án này"; keeps the existing TUI client contract 100% | reversible
- A12 | Cookie handling | write YOUTUBE_COOKIES env (base64-decoded if needed) to /tmp/yt_cookies.txt at startup, pass --cookies-file to yt-dlp | parity with config.write_cookies_from_env | reversible
- A13 | Go version | 1.25 (go.mod already says go 1.25.0) | current module baseline | reversible
- A14 | Secrets | never commit .env; the lrclib-upload/backend/.env contains a live OPENAI_API_KEY and R2 keys — these MUST NOT be copied into this repo; load from env at runtime | security: the existing .env is a committed secret in the sibling repo; do not replicate | reversible
- A15 | Existing TUI backend client | keep internal/integrations/backend/* UNCHANGED for EXISTING endpoints (fetch/peaks/publish/challenge/SSE); ADD new draft methods (SaveDraft/LoadDraft/ListDrafts/DeleteDraft) for the store move | the TUI already works against the Python backend; port parity is the acceptance bar; the store move is an additive change authorized by the user | NOT reversible for existing contract; additive for drafts

## Findings (cited - path:lines)

Python backend surface (to port):
- main.py:1-122 — FastAPI app, CORS, SlowAPI, routers (local_api, lrclib_proxy, cache_proxy), /health, /healthz, startup writes cookies, uvicorn entry.
- core/config.py:1-97 — env loading, CACHE_ROOT layout (.cache/{media,audio,transcripts,peaks}), OPENAI_*, FRONTEND_URL, CDN_* (R2), RATE_LIMIT_* (60/5), YOUTUBE_COOKIES_PATH=/tmp/yt_cookies.txt, write_cookies_from_env (base64 decode).
- core/models.py:1-25 — ExtractRequest, FetchRequest{url?,videoId?}, TranscribeRequest{videoId,force,enableRefinement,mode normal|karaoke,validated_mode()}.
- core/utils.py:1-41 — utc_now_iso, normalize_video_id (strip non [A-Za-z0-9_-]), sanitize_youtube_url (drop list/index params on youtube.com/youtu.be/music.youtube.com), load_json/save_json.
- core/rate_limit.py:1-32 — _get_real_ip (CF-Connecting-IP > X-Forwarded-For[0] > remote), shared limiter.
- routes/local_api.py:1-269 — POST /local-api/fetch (rate 60/min), POST /local-api/transcribe (rate 5/min, cached/mode/reuse), GET /local-api/transcribe/stream/{id} (SSE), GET /local-api/audio/{id} (Range), GET /local-api/peaks/{id} (samples 64-4000, source original|demucs, demucs returns 404 unless cached, cacheHit).
- routes/cache_proxy.py:1-166 — GET /cache/audio/{id}, /cache/peaks/{id}, /cache/transcript/{id} with CDN-first then disk fallback + Cache-Control.
- routes/lrclib_proxy.py:1-52 — POST /api/request-challenge, POST /api/publish (X-Publish-Token header, proxies to lrclib.net).
- services/audio_service.py:1-179 — find_cached_audio (.cache/audio/{id}/original.* fallback .cache/media/{id}.*), fetch_video_info (yt-dlp extractor_args player_client tv/android/mweb/web, IPv4, smart-TV UA, cookiefile), download_audio (bestaudio[ext=m4a], outtmpl original.%(ext)s), iter_file_range (64KB), parse_range_header (bytes=, suffix, 416 on invalid), compute_peaks (ffmpeg f32le pcm, numpy bucket max-abs, normalize >1.0).
- services/transcription_service.py:1-213 — JOB_LOCK+TRANSCRIBE_JOBS+EVENT_QUEUES+MAIN_LOOP, _broadcast (call_soon_threadsafe), _cache_demucs_peaks, _strip_words, process_audio (demucs vs openai-skip-demucs), run_transcription_job (status running/completed/failed, save_transcript, broadcast). OpenAI provider already skips Demucs.
- services/metadata_service.py:1-83 — path helpers + load/save metadata/peaks/transcript.
- services/cdn_service.py:1-183 — R2 key helpers, boto3 client, upload_*, exists_in_cdn (HEAD), presigned_url, stream_object.
- services/lyrics_service.py:1-109 — extract_vocals_demucs (python -m demucs.separate), get_whisper_model, detect_primary_language. **TO DROP** (WhisperX/Demucs only).
- services/lyrics_refinement_service.py:1-83 — AsyncOpenAI gpt-4o-mini refinement, JSON response, system prompt. Keep optional refinement.
- services/transcription/* — base, factory (whisperx|openai-whisper-1), formatter (build_synced_lyrics: [mm:ss.xx]<mm:ss.xx>word or [mm:ss.xx] text), types (TranscribedWord/Segment/Result), openai_whisper_service (whisper-1 verbose_json, word+segment granularity, map global words to segments), whisperx_service. **Drop whisperx_service; port formatter + openai_whisper_service + types.**

TUI backend client contract (the spec to satisfy — DO NOT CHANGE):
- internal/integrations/backend/client.go:18-169 — Client, Fetch (POST /local-api/fetch), Peaks (GET /local-api/peaks/{id}), RequestChallenge (POST /api/request-challenge), Publish (POST /api/publish, X-Publish-Token), do, expectStatus.
- internal/integrations/backend/sse.go:15-42 — TranscribeStream (GET /local-api/transcribe/stream/{id}, SSE text/event-stream, `data: <json>\n\n`).
- internal/integrations/backend/types.go:1-290 — FetchResponse{videoId,trackName,artistName,duration,audioReady,audioUrl,cachedAt*,sourceUrl*}, Source{original|demucs}, PeaksResponse{videoId,samples,duration,peaks,sourceFile,generatedAt,source,cacheHit}, TranscriptionStatus{queued|running|completed|failed}, sealed events (Queued/Running/Completed{provider,language,plain,synced,is_ai_refined,model,mode,updatedAt}/Failed), TranscribeResponse decode switch on status, ChallengeResponse{prefix,target}, PublishPayload{trackName,artistName,albumName,duration,plainLyrics,syncedLyrics}, PublishToken(prefix:nonce).
- internal/integrations/backend/fixture.go:1-152 — exact fixture shapes the Go server must produce.
- cmd/lyrike-studio-tui/main.go:80-189 — runReal wires backend.NewClient(backendURL); cache path hardcoded /home/duckviet/lrclib-upload/backend/.cache/audio/{id}/original.mp4 (needs updating to the Go server's cache dir or an env var).

Existing Python tests to port to Go (parity bar):
- tests/test_utils.py — sanitize_youtube_url (watch/short/music/non-youtube/no-list), fetch endpoint sanitizes URL.
- tests/test_transcription_formatter.py — enhanced LRC when words exist, line LRC when words absent, skip malformed timing.
- tests/test_transcription_service_mode.py — _strip_words removes word timings (normal mode).
- tests/test_local_api_peaks_cache.py — peaks endpoint saves generated peaks to cache, cacheHit false on fresh.

Deployment artifacts:
- backend/Dockerfile:1-25 — python:3.11-slim, ffmpeg, pip install -e ., uvicorn. Port to multi-stage Go build + ffmpeg runtime.
- backend/fly.toml:1-15 — app=lyrike-studio-1, region nrt, 1cpu/1gb, port 8080. Reuse for Go binary.
- backend/.env — contains LIVE secrets (OPENAI_API_KEY, R2 keys). MUST NOT be copied; document env vars instead.

Module/git:
- lyrike-studio-tui go.mod: github.com/duckviet/lyrike-studio-tui, go 1.25.0, bubbletea/lipgloss v2, beep.
- .gitignore already ignores /.cache/, /media/, /transcripts/, /peaks/, *.sock, /.omo/evidence/.
- lrclib-upload git has uncommitted: pyproject.toml, routes/local_api.py, tests/test_local_api_peaks_cache.py (the peaks fix from the prior plan). Do NOT touch the sibling repo.

## Decisions (with rationale)
- D1: Keep the TUI backend client unchanged; the Go server targets its exact JSON contract. Rationale: the client is already tested and working against Python; changing it would break the TUI and double the work.
- D2: One Go binary, `serve` subcommand runs the backend. Rationale: user asked to put the backend "ngay trong dự án này"; single binary is the simplest deploy and lets the TUI and backend share the cache dir.
- D3: yt-dlp via exec.Command (shell out), not a Go rewrite. Rationale: yt-dlp's extractor logic is infeasible to reimplement; the binary is already a documented prereq. Pass through the same extractor_args, IPv4 source, smart-TV UA, cookiefile.
- D4: OpenAI whisper-1 only, via openai-go SDK. Drop WhisperX, Demucs, torch, numpy, boto3 (replaced by aws-sdk-go-v2 if CDN on). User explicit.
- D5: ffmpeg shell-out for peaks, parse f32le, bucket max-abs, normalize — byte-for-byte parity with compute_peaks.
- D6: Optional R2 CDN via aws-sdk-go-v2 S3-compatible client, off by default. Keeps cache_proxy routes deployable.
- D7: SSE via http.Flusher + `data: <json>\n\n` frames, same as Python. The TUI sse.go reader already parses this.
- D8: Transcription jobs: goroutine + sync.Map state + per-video broadcast channels. Replaces Python threading + asyncio.Queue.
- D9: Never copy .env / secrets. Document env vars in README + docs/troubleshooting.md.
- D10: Update main.go's hardcoded cache path to an env var (LYRIKE_CACHE_DIR) defaulting to XDG or ./lyrike-cache, so the TUI and backend can share one cache when run as one binary.

## Scope IN
- New Go package internal/server/ with: config, middleware (rate limit + CORS), cache (metadata/peaks/transcript CRUD), media/ytdlp (fetch info + download audio), media/peaks (ffmpeg), transcription (OpenAI whisper-1 + formatter + job manager + SSE), lrclib (challenge + publish proxy), http handlers wiring all routes.
- internal/server/drafts/: FileStore (moved from internal/storage/), draft REST handlers (GET/PUT/DELETE /local-api/projects/{id}, GET /local-api/projects).
- internal/storage/remote.go: RemoteStore (HTTP-backed) implementing storage.Store; TUI uses RemoteStore instead of FileStore in real mode.
- internal/integrations/backend/drafts.go: new client methods SaveDraft/LoadDraft/ListDrafts/DeleteDraft calling the new endpoints.
- `serve` subcommand in cmd/lyrike-studio-tui/main.go; update TUI wiring to use RemoteStore when backend URL is set.
- Port all 4 Python test files to Go table-driven tests + httptest integration tests + new draft endpoint tests.
- Dockerfile (multi-stage Go + ffmpeg) + fly.toml reuse + README/docs updates + internal/server/AGENTS.md.
- New go.mod deps: chi, openai-go, golang.org/x/time/rate, godotenv (optional). NO aws-sdk-go-v2 (CDN dropped).

## Scope OUT (Must NOT have)
- Do NOT change existing internal/integrations/backend/* contracts for existing endpoints (fetch/peaks/publish/challenge/SSE) — only ADD draft methods.
- Do NOT port WhisperX, Demucs, torch, numpy, boto3, slowapi (replaced by Go equivalents).
- Do NOT implement Cloudflare R2 / CDN / aws-sdk-go-v2 — dropped entirely (user explicit).
- Do NOT copy the sibling repo's .env or any secrets.
- Do NOT touch /home/duckviet/lrclib-upload/* (the Python backend stays as-is; this is a port, not a migration of the sibling repo's files).
- Do NOT add word-level tap-sync to the TUI (out of scope, unchanged from prior plan).
- Do NOT implement a native Go YouTube extractor (shell out to yt-dlp).
- Do NOT implement server-side PoW solving (the TUI solves PoW; server only proxies publish).
- Do NOT add browser UI, WaveSurfer, iframe, or Windows named-pipe support.
- Do NOT regress existing TUI tests (go test ./... must stay green).
- Do NOT keep FileStore in use by the TUI in real mode — real mode uses RemoteStore; FileStore lives only in the server and in demo/offline test fixtures.

## Open questions
None remaining — all forks resolved via announced defaults above. The user already clarified: drop WhisperX, use whisper-1 only.

## Approval gate
status: awaiting-approval
The user's two clarifications (Go backend in this repo; whisper-1 only) plus exploration resolve every fork. Adopted defaults A1-A15 are surfaced in the plan's human TL;DR for veto. On explicit approval, write .omo/plans/go-backend.md with ~10-14 todos across 4 waves, run Metis gap analysis, append todos, fill TL;DR last, then auto-run the dual high-accuracy Momus review (UNCLEAR path).
