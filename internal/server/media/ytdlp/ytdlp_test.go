package ytdlp

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// setCookiesPathForTest overrides YouTubeCookiesPath for the lifetime of the
// test and restores the original value during cleanup. The default path
// (/tmp/yt_cookies.txt) does not exist in the test environment, so subtests
// that need a real cookies file can call this to point the package at a
// temp file.
func setCookiesPathForTest(t *testing.T, path string) {
	t.Helper()
	old := YouTubeCookiesPath
	YouTubeCookiesPath = path
	t.Cleanup(func() { YouTubeCookiesPath = old })
}

// hasFlag reports whether needle appears verbatim in args.
func hasFlag(args []string, needle string) bool {
	return slices.Contains(args, needle)
}

// argValue returns the value following key (the next element) if key is
// present and has a follow-up token; ok is false otherwise.
func argValue(args []string, key string) (string, bool) {
	for i, a := range args {
		if a == key && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// argValues returns every value that follows key, in order. Useful for
// flags like --extractor-args or --add-header that may appear multiple times.
func argValues(args []string, key string) []string {
	var out []string
	for i, a := range args {
		if a == key && i+1 < len(args) {
			out = append(out, args[i+1])
		}
	}
	return out
}

func TestFindCachedAudio(t *testing.T) {
	t.Run("audio dir", func(t *testing.T) {
		dir := t.TempDir()
		audioDir := filepath.Join(dir, "audio", "abc")
		if err := os.MkdirAll(audioDir, 0o755); err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(audioDir, "original.m4a")
		if err := os.WriteFile(want, []byte("audio"), 0o644); err != nil {
			t.Fatal(err)
		}

		got, ok := FindCachedAudio(dir, "abc")
		if !ok {
			t.Fatal("expected to find cached audio in audio dir")
		}
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("media fallback", func(t *testing.T) {
		dir := t.TempDir()
		mediaDir := filepath.Join(dir, "media")
		if err := os.MkdirAll(mediaDir, 0o755); err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(mediaDir, "abc.m4a")
		if err := os.WriteFile(want, []byte("audio"), 0o644); err != nil {
			t.Fatal(err)
		}
		// .json sidecar must be skipped, not returned.
		if err := os.WriteFile(filepath.Join(mediaDir, "abc.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}

		got, ok := FindCachedAudio(dir, "abc")
		if !ok {
			t.Fatal("expected media fallback to find audio file")
		}
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("not found", func(t *testing.T) {
		dir := t.TempDir()
		got, ok := FindCachedAudio(dir, "missing")
		if ok {
			t.Errorf("expected not found, got path %q", got)
		}
		if got != "" {
			t.Errorf("expected empty path on miss, got %q", got)
		}
	})
}

func TestBuildFetchArgs(t *testing.T) {
	const url = "https://www.youtube.com/watch?v=abc"

	t.Run("without cookies", func(t *testing.T) {
		// /tmp/yt_cookies.txt is not present in the test environment.
		args := BuildFetchArgs(url)
		// Materialize into an exec.Cmd so the test mirrors "build command,
		// inspect cmd.Args" — the slice we assert against must be exactly
		// what os/exec would hand to the binary.
		cmd := exec.Command("yt-dlp", args...)
		got := cmd.Args

		requiredFlags := []string{
			"--quiet",
			"--no-warnings",
			"--noplaylist",
			"--skip-download",
			"--nocheckcertificate",
			"--dump-json",
		}
		for _, f := range requiredFlags {
			if !hasFlag(got, f) {
				t.Errorf("missing flag %q in fetch args: %v", f, got)
			}
		}
		if hasFlag(got, "--cookies-file") {
			t.Errorf("did not expect --cookies-file when cookies file is missing: %v", got)
		}

		extractorArgs := argValues(got, "--extractor-args")
		wantPlayerClient := "youtube:player_client=tv,android,mweb,web"
		wantPlayerSkip := "youtube:player_skip=web"
		if !slices.Contains(extractorArgs, wantPlayerClient) {
			t.Errorf("expected --extractor-args %q in %v", wantPlayerClient, extractorArgs)
		}
		if !slices.Contains(extractorArgs, wantPlayerSkip) {
			t.Errorf("expected --extractor-args %q in %v", wantPlayerSkip, extractorArgs)
		}

		if val, ok := argValue(got, "--source-address"); !ok || val != "0.0.0.0" {
			t.Errorf("expected --source-address 0.0.0.0, got %q (ok=%v)", val, ok)
		}
		if val, ok := argValue(got, "--user-agent"); !ok || val != smartTVUserAgent {
			t.Errorf("expected --user-agent %q, got %q (ok=%v)", smartTVUserAgent, val, ok)
		}
		headers := argValues(got, "--add-header")
		if !slices.Contains(headers, "Accept:*/*") {
			t.Errorf("expected --add-header Accept:*, got %v", headers)
		}
		if !slices.Contains(headers, "Accept-Language:en-US,en;q=0.9") {
			t.Errorf("expected --add-header Accept-Language, got %v", headers)
		}

		// URL must be present (as the final positional arg).
		if got[len(got)-1] != url {
			t.Errorf("expected url %q as last arg, got %q", url, got[len(got)-1])
		}
		// Binary name occupies index 0.
		if got[0] != "yt-dlp" {
			t.Errorf("expected binary name yt-dlp at index 0, got %q", got[0])
		}
	})

	t.Run("with cookies", func(t *testing.T) {
		dir := t.TempDir()
		cookiePath := filepath.Join(dir, "cookies.txt")
		if err := os.WriteFile(cookiePath, []byte("# Netscape cookies"), 0o600); err != nil {
			t.Fatal(err)
		}
		setCookiesPathForTest(t, cookiePath)

		args := BuildFetchArgs(url)
		cmd := exec.Command("yt-dlp", args...)
		got := cmd.Args

		val, ok := argValue(got, "--cookies-file")
		if !ok {
			t.Fatalf("expected --cookies-file in args, got %v", got)
		}
		if val != cookiePath {
			t.Errorf("expected --cookies-file %q, got %q", cookiePath, val)
		}
	})
}

func TestBuildDownloadArgs(t *testing.T) {
	const (
		url     = "https://www.youtube.com/watch?v=abc"
		videoID = "abc"
	)
	dir := t.TempDir()
	// Create the audio dir so the function's internal assumptions hold and
	// so the path is meaningful.
	audioDir := filepath.Join(dir, "audio", videoID)
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	args := BuildDownloadArgs(url, videoID, dir)
	cmd := exec.Command("yt-dlp", args...)
	got := cmd.Args

	if val, ok := argValue(got, "--format"); !ok || val != "bestaudio[ext=m4a]/bestaudio/best" {
		t.Errorf("expected --format bestaudio[ext=m4a]/bestaudio/best, got %q (ok=%v)", val, ok)
	}
	wantOutTmpl := filepath.Join(dir, "audio", videoID, "original.%(ext)s")
	if val, ok := argValue(got, "--outtmpl"); !ok || val != wantOutTmpl {
		t.Errorf("expected --outtmpl %q, got %q (ok=%v)", wantOutTmpl, val, ok)
	}

	extractorArgs := argValues(got, "--extractor-args")
	wantPlayerClient := "youtube:player_client=tv,android,mweb"
	if !slices.Contains(extractorArgs, wantPlayerClient) {
		t.Errorf("expected --extractor-args %q in %v", wantPlayerClient, extractorArgs)
	}
	// Download MUST NOT include the fetch-only "web" client.
	forbiddenClient := "youtube:player_client=tv,android,mweb,web"
	if slices.Contains(extractorArgs, forbiddenClient) {
		t.Errorf("download args must not include fetch-only client list %q", forbiddenClient)
	}
	// Download MUST NOT include player_skip (fetch-only flag).
	for _, e := range extractorArgs {
		if e == "youtube:player_skip=web" {
			t.Errorf("download args must not include player_skip=web: %v", extractorArgs)
		}
	}

	if val, ok := argValue(got, "--source-address"); !ok || val != "0.0.0.0" {
		t.Errorf("expected --source-address 0.0.0.0, got %q (ok=%v)", val, ok)
	}
	if val, ok := argValue(got, "--user-agent"); !ok || val != smartTVUserAgent {
		t.Errorf("expected --user-agent %q, got %q (ok=%v)", smartTVUserAgent, val, ok)
	}
	// Download MUST NOT add Accept / Accept-Language headers (those are
	// fetch-only and break some extractor clients).
	headers := argValues(got, "--add-header")
	if len(headers) != 0 {
		t.Errorf("download args must not include --add-header, got %v", headers)
	}

	if got[len(got)-1] != url {
		t.Errorf("expected url %q as last arg, got %q", url, got[len(got)-1])
	}
}

func TestMapYtdlpError(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   int
	}{
		{"video unavailable", "ERROR: [youtube] dQw4w9WgXcQ: Video unavailable", 404},
		{"private", "ERROR: [youtube] dQw4w9WgXcQ: Private video. Sign in if you've been granted access", 404},
		{"removed", "ERROR: This video has been removed by the uploader", 404},
		{"sign in", "ERROR: Sign in to confirm you're not a bot", 403},
		{"bot", "ERROR: Sign in to confirm you are not a bot", 403},
		{"captcha", "ERROR: Complete the captcha to continue", 403},
		{"unknown", "ERROR: unable to extract initial data; please report this issue", 502},
		{"empty", "", 502},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := MapYtdlpError(tc.stderr); got != tc.want {
				t.Errorf("MapYtdlpError(%q) = %d, want %d", tc.stderr, got, tc.want)
			}
		})
	}
}
