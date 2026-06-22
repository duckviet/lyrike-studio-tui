# Server

The Go backend that replaces the Python FastAPI service for the
lyrike-studio-tui workflow. Lives in `internal/server/` and is wired into a
single binary (`lyrike-studio-tui serve`) in a later task. The HTTP contract
matches the one the TUI's `internal/integrations/backend` package already
pins, so the TUI does not change.

These rules apply to every file in this package.

- Keep external contracts typed at the boundary. Parse JSON into structs
  (matching `internal/integrations/backend/types.go`) before passing values
  to handlers; never smuggle `map[string]any` or `interface{}` across the
  HTTP boundary.
- Use `context.Context` as the first argument for any operation that can
  block (HTTP calls to LRCLIB / OpenAI, `os/exec` to yt-dlp / ffmpeg,
  file I/O on slow disks). Cancellation must propagate.
- Configure network clients with explicit timeouts. The default
  `http.Client` is forbidden — use a tuned `*http.Client` (timeouts,
  transport tuning) at every site that makes an outbound call.
- Test with local fakes or `httptest.Server`. Never call external services
  in unit tests; never write to `/tmp` paths in tests (use `t.TempDir()`
  and pass the path as a parameter so production code is testable without
  polluting the real filesystem).
- Do not add third-party HTTP, JSON, or config dependencies beyond what the
  go-backend plan explicitly authorizes (chi, openai-go, x/time/rate,
  godotenv). No viper, no cobra, no slowapi, no boto3 equivalent.
- Secrets are read from the environment, never hardcoded, never logged.
  `YOUTUBE_COOKIES` is the only sensitive value this package writes to
  disk and it must use `0o600` perms.
