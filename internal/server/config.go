// Package server hosts the lyrike-studio-tui Go backend.
//
// Task 1 of the go-backend plan scaffolds the package with typed env-based
// configuration and a cookie-file writer. Higher-level HTTP routes, services,
// and middleware will live in this package in later tasks.
package server

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// DefaultYouTubeCookiesPath is the production cookie file path written by
// WriteCookiesFromEnv. Tests pass their own temp path; do NOT hardcode this
// in tests or write to it from test code.
const DefaultYouTubeCookiesPath = "/tmp/yt_cookies.txt"

// DefaultTranscriptionProvider is the only supported transcription provider.
// The plan explicitly drops WhisperX and Demucs, so the env var is read for
// logging/visibility but its value is not honored.
const DefaultTranscriptionProvider = "openai-whisper-1"

// Config holds every runtime knob the backend reads from the environment.
// Fields are exported so callers can build HTTP handlers/middleware against
// the same struct without re-parsing env vars.
type Config struct {
	// OpenAI
	OpenAIAPIKey             string
	OpenAITranscriptionModel string
	TranscriptionProvider    string // fixed to "openai-whisper-1"
	EnableLyricsRefinement   bool
	YouTubeCookies           string // raw env value, not yet decoded/written

	// HTTP
	FrontendURL                  string
	Port                         int
	RateLimitPerMinute           int
	RateLimitTranscribePerMinute int

	// Storage layout
	CacheDir string
	DraftDir string
}

// LoadConfig reads the process environment, applies defaults, and creates the
// cache directory layout. It is safe to call from tests when the caller has
// already set LYRIKE_CACHE_DIR / LYRIKE_DRAFT_DIR via t.Setenv.
//
// A missing .env file at the working directory is ignored (matches the Python
// backend's behavior).
func LoadConfig() (*Config, error) {
	// Best-effort .env load. Missing file is not an error.
	_ = godotenv.Load()

	cacheDir := strings.TrimSpace(getenv("LYRIKE_CACHE_DIR", "./.cache"))
	draftDir := strings.TrimSpace(getenv("LYRIKE_DRAFT_DIR", "./.cache/drafts"))

	rateGeneral, err := getenvInt("RATE_LIMIT_PER_MINUTE", 60)
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_PER_MINUTE: %w", err)
	}
	rateTranscribe, err := getenvInt("RATE_LIMIT_TRANSCRIBE_PER_MINUTE", 5)
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_TRANSCRIBE_PER_MINUTE: %w", err)
	}
	port, err := getenvInt("PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("PORT: %w", err)
	}

	cfg := &Config{
		OpenAIAPIKey:                 getenv("OPENAI_API_KEY", ""),
		OpenAITranscriptionModel:     getenv("OPENAI_TRANSCRIPTION_MODEL", "whisper-1"),
		TranscriptionProvider:        DefaultTranscriptionProvider,
		EnableLyricsRefinement:       getenvBool("ENABLE_LYRICS_REFINEMENT", false),
		YouTubeCookies:               getenv("YOUTUBE_COOKIES", ""),
		FrontendURL:                  getenv("FRONTEND_URL", ""),
		Port:                         port,
		RateLimitPerMinute:           rateGeneral,
		RateLimitTranscribePerMinute: rateTranscribe,
		CacheDir:                     cacheDir,
		DraftDir:                     draftDir,
	}

	if err := ensureCacheLayout(cfg.CacheDir, cfg.DraftDir); err != nil {
		return nil, fmt.Errorf("create cache layout: %w", err)
	}

	return cfg, nil
}

// WriteCookiesFromEnv decodes YOUTUBE_COOKIES (base64 if it does not start
// with '#' and is long enough) and writes the result to path. A missing or
// empty env var is not an error; no file is written in that case.
//
// The path is a parameter so tests can target t.TempDir() instead of the real
// /tmp/yt_cookies.txt.
func WriteCookiesFromEnv(path string) error {
	raw := strings.TrimSpace(os.Getenv("YOUTUBE_COOKIES"))
	if raw == "" {
		return nil
	}

	content := raw
	// Heuristic from the Python backend: only attempt base64 when the payload
	// does not already look like a Netscape header and is long enough to be
	// plausibly encoded. Short-circuits on '#' to preserve any raw cookie
	// file the user dropped in via env.
	if !strings.HasPrefix(content, "#") && len(content) > 20 {
		if decoded, err := base64.StdEncoding.DecodeString(content); err == nil {
			content = string(decoded)
		}
	}

	// 0o600 — cookies grant YouTube identity; never world-readable.
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write cookies to %s: %w", path, err)
	}
	return nil
}

// ensureCacheLayout creates the on-disk cache directories the backend uses.
// All directories use 0o755; the leaf cookie file (written separately by
// WriteCookiesFromEnv) is the only sensitive artifact.
func ensureCacheLayout(cacheDir, draftDir string) error {
	for _, d := range []string{
		filepath.Join(cacheDir, "media"),
		filepath.Join(cacheDir, "audio"),
		filepath.Join(cacheDir, "transcripts"),
		filepath.Join(cacheDir, "peaks"),
		draftDir,
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return nil
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func getenvInt(key string, def int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %q as int: %w", raw, err)
	}
	return n, nil
}

func getenvBool(key string, def bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return def
	}
}
