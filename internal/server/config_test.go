package server

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

// envKeys lists every env var LoadConfig reads. The helpers below clear and
// restore these around each test so a developer's shell cannot leak state into
// the test run.
var envKeys = []string{
	"OPENAI_API_KEY",
	"OPENAI_TRANSCRIPTION_MODEL",
	"TRANSCRIPTION_PROVIDER",
	"ENABLE_LYRICS_REFINEMENT",
	"FRONTEND_URL",
	"RATE_LIMIT_PER_MINUTE",
	"RATE_LIMIT_TRANSCRIBE_PER_MINUTE",
	"YOUTUBE_COOKIES",
	"PORT",
	"LYRIKE_CACHE_DIR",
	"LYRIKE_DRAFT_DIR",
}

// clearEnv removes every tracked env var for the duration of t and restores
// whatever values were present when the test started. Use this when a test
// needs to assert default values.
func clearEnv(t *testing.T) {
	t.Helper()
	type saved struct {
		key string
		val string
		ok  bool
	}
	previous := make([]saved, 0, len(envKeys))
	for _, k := range envKeys {
		v, ok := os.LookupEnv(k)
		previous = append(previous, saved{key: k, val: v, ok: ok})
		_ = os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, p := range previous {
			if p.ok {
				_ = os.Setenv(p.key, p.val)
			} else {
				_ = os.Unsetenv(p.key)
			}
		}
	})
}

// withTempCache points LYRIKE_CACHE_DIR and LYRIKE_DRAFT_DIR at fresh
// directories under t.TempDir() so tests never touch the real ./cache.
func withTempCache(t *testing.T) (cacheDir, draftDir string) {
	t.Helper()
	base := t.TempDir()
	cacheDir = filepath.Join(base, "cache")
	draftDir = filepath.Join(base, "drafts")
	t.Setenv("LYRIKE_CACHE_DIR", cacheDir)
	t.Setenv("LYRIKE_DRAFT_DIR", draftDir)
	return cacheDir, draftDir
}

func TestConfigDefaults(t *testing.T) {
	clearEnv(t)
	withTempCache(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	// Sensible defaults from the plan.
	if got, want := cfg.OpenAITranscriptionModel, "whisper-1"; got != want {
		t.Errorf("OpenAITranscriptionModel = %q, want %q", got, want)
	}
	if got, want := cfg.TranscriptionProvider, "openai-whisper-1"; got != want {
		t.Errorf("TranscriptionProvider = %q, want %q (fixed provider)", got, want)
	}
	if cfg.EnableLyricsRefinement {
		t.Errorf("EnableLyricsRefinement = true, want false (default)")
	}
	if got, want := cfg.RateLimitPerMinute, 60; got != want {
		t.Errorf("RateLimitPerMinute = %d, want %d", got, want)
	}
	if got, want := cfg.RateLimitTranscribePerMinute, 5; got != want {
		t.Errorf("RateLimitTranscribePerMinute = %d, want %d", got, want)
	}
	if got, want := cfg.Port, 8080; got != want {
		t.Errorf("Port = %d, want %d", got, want)
	}
	if got, want := cfg.CacheDir, filepath.Join(t.TempDir(), "cache"); got == want {
		// Sanity: the actual configured cache dir should be the temp dir we set.
		// (t.TempDir() in withTempCache already passed; this just guards that we
		// did not accidentally fall back to "./.cache".)
	}
	// Cache and draft dirs should be the temp paths we configured.
	if cfg.CacheDir == "" || cfg.CacheDir == "./.cache" {
		t.Errorf("CacheDir = %q, expected the test temp dir override", cfg.CacheDir)
	}
	if cfg.DraftDir == "" || cfg.DraftDir == "./.cache/drafts" {
		t.Errorf("DraftDir = %q, expected the test temp dir override", cfg.DraftDir)
	}
	// Optional fields default to empty when env not set.
	if cfg.OpenAIAPIKey != "" {
		t.Errorf("OpenAIAPIKey = %q, want empty (no env set)", cfg.OpenAIAPIKey)
	}
	if cfg.FrontendURL != "" {
		t.Errorf("FrontendURL = %q, want empty (no env set)", cfg.FrontendURL)
	}
	if cfg.YouTubeCookies != "" {
		t.Errorf("YouTubeCookies = %q, want empty (no env set)", cfg.YouTubeCookies)
	}
}

func TestConfigEnvOverride(t *testing.T) {
	clearEnv(t)
	withTempCache(t)

	t.Setenv("OPENAI_API_KEY", "sk-test-abc123")
	t.Setenv("OPENAI_TRANSCRIPTION_MODEL", "gpt-4o-transcribe")
	t.Setenv("ENABLE_LYRICS_REFINEMENT", "true")
	t.Setenv("FRONTEND_URL", "https://app.example.com")
	t.Setenv("RATE_LIMIT_PER_MINUTE", "120")
	t.Setenv("RATE_LIMIT_TRANSCRIBE_PER_MINUTE", "10")
	t.Setenv("YOUTUBE_COOKIES", "# Netscape raw cookie content")
	t.Setenv("PORT", "9090")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	if got, want := cfg.OpenAIAPIKey, "sk-test-abc123"; got != want {
		t.Errorf("OpenAIAPIKey = %q, want %q", got, want)
	}
	if got, want := cfg.OpenAITranscriptionModel, "gpt-4o-transcribe"; got != want {
		t.Errorf("OpenAITranscriptionModel = %q, want %q", got, want)
	}
	if !cfg.EnableLyricsRefinement {
		t.Errorf("EnableLyricsRefinement = false, want true (env=true)")
	}
	if got, want := cfg.FrontendURL, "https://app.example.com"; got != want {
		t.Errorf("FrontendURL = %q, want %q", got, want)
	}
	if got, want := cfg.RateLimitPerMinute, 120; got != want {
		t.Errorf("RateLimitPerMinute = %d, want %d", got, want)
	}
	if got, want := cfg.RateLimitTranscribePerMinute, 10; got != want {
		t.Errorf("RateLimitTranscribePerMinute = %d, want %d", got, want)
	}
	if got, want := cfg.YouTubeCookies, "# Netscape raw cookie content"; got != want {
		t.Errorf("YouTubeCookies = %q, want %q", got, want)
	}
	if got, want := cfg.Port, 9090; got != want {
		t.Errorf("Port = %d, want %d", got, want)
	}
	// TranscriptionProvider remains fixed regardless of env input.
	if got, want := cfg.TranscriptionProvider, "openai-whisper-1"; got != want {
		t.Errorf("TranscriptionProvider = %q, want %q (provider is fixed)", got, want)
	}
}

func TestCacheDirsCreated(t *testing.T) {
	clearEnv(t)
	cacheDir, draftDir := withTempCache(t)

	if _, err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	for _, sub := range []string{"media", "audio", "transcripts", "peaks"} {
		path := filepath.Join(cacheDir, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected dir %s to exist: %v", path, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory, got file", path)
		}
	}
	info, err := os.Stat(draftDir)
	if err != nil {
		t.Errorf("expected draft dir %s to exist: %v", draftDir, err)
	} else if !info.IsDir() {
		t.Errorf("expected %s to be a directory, got file", draftDir)
	}
}

func TestWriteCookiesFromEnv_Base64(t *testing.T) {
	clearEnv(t)

	// A minimal but valid-looking Netscape cookies file. Encode it as base64 so
	// the test exercises the decoding branch (no leading '#' before the
	// payload, length > 20).
	raw := "# Netscape HTTP Cookie File\n" +
		".youtube.com\tTRUE\t/\tTRUE\t0\tCONSENT\tYES+cb\n" +
		".youtube.com\tTRUE\t/\tTRUE\t0\tSID\ttest-sid-value\n"
	
	// Test standard clean base64
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	t.Setenv("YOUTUBE_COOKIES", encoded)

	path := filepath.Join(t.TempDir(), "yt_cookies.txt")
	if err := WriteCookiesFromEnv(path); err != nil {
		t.Fatalf("WriteCookiesFromEnv() returned error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != raw {
		t.Errorf("cookie file content mismatch.\n got: %q\nwant: %q", string(got), raw)
	}

	// Test base64 with newlines and spaces, and missing padding (raw encoding)
	// We'll manually insert spaces/newlines and strip trailing padding '='
	dirtyEncoded := ""
	for i, c := range encoded {
		if c == '=' {
			continue // strip padding
		}
		dirtyEncoded += string(c)
		if i%10 == 0 {
			dirtyEncoded += "\n " // add newline and space
		}
	}
	t.Setenv("YOUTUBE_COOKIES", dirtyEncoded)
	path2 := filepath.Join(t.TempDir(), "yt_cookies_dirty.txt")
	if err := WriteCookiesFromEnv(path2); err != nil {
		t.Fatalf("WriteCookiesFromEnv() dirty returned error: %v", err)
	}
	got2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatalf("read %s: %v", path2, err)
	}
	if string(got2) != raw {
		t.Errorf("dirty cookie file content mismatch.\n got: %q\nwant: %q", string(got2), raw)
	}
}

func TestWriteCookiesFromEnv_Plain(t *testing.T) {
	clearEnv(t)

	// A payload that already starts with '#' and is short: should be written
	// without attempting base64 decode. The '#' guard short-circuits the
	// decoder regardless of length. Leading/trailing whitespace is trimmed to
	// mirror the Python backend's .strip() behavior.
	raw := "# Netscape HTTP Cookie File\n.local\tTRUE\t/\tFALSE\t0\tk\tv\n"
	t.Setenv("YOUTUBE_COOKIES", raw)

	path := filepath.Join(t.TempDir(), "yt_cookies.txt")
	if err := WriteCookiesFromEnv(path); err != nil {
		t.Fatalf("WriteCookiesFromEnv() returned error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	want := "# Netscape HTTP Cookie File\n.local\tTRUE\t/\tFALSE\t0\tk\tv"
	if string(got) != want {
		t.Errorf("cookie file content mismatch.\n got: %q\nwant: %q", string(got), want)
	}
}

func TestWriteCookiesFromEnv_EmptyEnv(t *testing.T) {
	clearEnv(t)

	// With YOUTUBE_COOKIES unset, no file should be written and no error
	// returned (matches Python's "return False" semantics).
	path := filepath.Join(t.TempDir(), "yt_cookies.txt")
	if err := WriteCookiesFromEnv(path); err != nil {
		t.Fatalf("WriteCookiesFromEnv() with empty env returned error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected no file written, but stat err = %v", err)
	}
}
