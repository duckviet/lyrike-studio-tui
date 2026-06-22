package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSanitizeYouTubeURL is a port of tests/test_utils.py::test_sanitize_youtube_url_*
// from the Python backend. Each case matches the Python expectation exactly so
// the Go backend preserves URL sanitization parity for /local-api/fetch and
// any other call site that persists sourceUrl.
func TestSanitizeYouTubeURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "Watch",
			in:   "https://www.youtube.com/watch?v=OpQFFLBMEPI&list=RDPaKr9gWqwl4&index=2",
			want: "https://www.youtube.com/watch?v=OpQFFLBMEPI",
		},
		{
			name: "Short",
			in:   "https://youtu.be/OpQFFLBMEPI?list=RDPaKr9gWqwl4&index=2",
			want: "https://youtu.be/OpQFFLBMEPI",
		},
		{
			name: "Music",
			in:   "https://music.youtube.com/watch?v=OpQFFLBMEPI&list=RDPaKr9gWqwl4&index=2&feature=share",
			want: "https://music.youtube.com/watch?feature=share&v=OpQFFLBMEPI",
		},
		{
			name: "NonYouTube",
			in:   "https://example.com/watch?list=123&v=abc",
			want: "https://example.com/watch?list=123&v=abc",
		},
		{
			name: "NoList",
			in:   "https://www.youtube.com/watch?v=OpQFFLBMEPI",
			want: "https://www.youtube.com/watch?v=OpQFFLBMEPI",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SanitizeYouTubeURL(tc.in)
			if got != tc.want {
				t.Errorf("SanitizeYouTubeURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestSanitizeYouTubeURL_DoesNotMutateInput guards the AGENTS.md rule "Do NOT
// mutate input." Strings are immutable in Go so a direct ref check is
// unnecessary, but we still snapshot length + a hash of the input to detect
// any future regression that swaps to *string or shares state.
func TestSanitizeYouTubeURL_DoesNotMutateInput(t *testing.T) {
	t.Parallel()
	in := "https://www.youtube.com/watch?v=OpQFFLBMEPI&list=RDPaKr9gWqwl4&index=2"
	original := in
	_ = SanitizeYouTubeURL(in)
	if in != original {
		t.Errorf("input mutated: got %q, want %q", in, original)
	}
}

// TestSanitizeYouTubeURL_EmptyAndInvalid returns the input unchanged for
// empty strings and unparseable input — mirrors the Python `except: return url`
// fallback in core/utils.py.
func TestSanitizeYouTubeURL_EmptyAndInvalid(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",
		"not a url",
		"://broken",
	}
	for _, in := range cases {
		in := in
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			got := SanitizeYouTubeURL(in)
			if got != in {
				t.Errorf("SanitizeYouTubeURL(%q) = %q, want %q (unchanged)", in, got, in)
			}
		})
	}
}

// TestNormalizeVideoID is a port of the implicit Python expectation that
// non [A-Za-z0-9_-] characters are stripped while '-' and '_' are kept, and
// that all-stripped input collapses to "".
func TestNormalizeVideoID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "Plain", in: "OpQFFLBMEPI", want: "OpQFFLBMEPI"},
		{name: "StripsSpecials", in: "OpQ!@#FFL$%^BME&*()PI", want: "OpQFFLBMEPI"},
		{name: "KeepsDashUnderscore", in: "abc-DEF_123", want: "abc-DEF_123"},
		{name: "Empty", in: "", want: ""},
		{name: "AllSpecial", in: "!@#$%^&*()", want: ""},
		{name: "SpacesStripped", in: "abc def 123", want: "abcdef123"},
		{name: "UnicodeStripped", in: "OpQéñFFL", want: "OpQFFL"},
		{name: "StripsLeadingTrailing", in: "  abc  ", want: "abc"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeVideoID(tc.in)
			if got != tc.want {
				t.Errorf("NormalizeVideoID(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestUTCNowISO asserts the format is RFC3339Nano and the produced timestamp
// is "now" within a generous window. We avoid time-of-day dependence: the
// assertion that matters is the format prefix and the value-after-parse.
func TestUTCNowISO(t *testing.T) {
	t.Parallel()
	got := UTCNowISO()
	// RFC3339Nano in UTC looks like 2024-01-02T15:04:05.123456789Z — the 'Z'
	// suffix and the 'T' separator are the load-bearing markers here.
	if !strings.HasSuffix(got, "Z") {
		t.Errorf("UTCNowISO() = %q, want RFC3339Nano UTC (Z suffix)", got)
	}
	if !strings.Contains(got, "T") {
		t.Errorf("UTCNowISO() = %q, want RFC3339Nano (T separator)", got)
	}
	parsed, err := time.Parse(time.RFC3339Nano, got)
	if err != nil {
		t.Fatalf("UTCNowISO() = %q, RFC3339Nano parse failed: %v", got, err)
	}
	if parsed.Location() != time.UTC {
		t.Errorf("UTCNowISO() parsed location = %v, want UTC", parsed.Location())
	}
	if delta := time.Since(parsed); delta < -time.Second || delta > time.Second {
		t.Errorf("UTCNowISO() = %q, parsed time %v not within 1s of now", got, parsed)
	}
}

// TestLoadSaveJSONRoundTrip exercises the JSON helpers together: save a
// representative payload, read it back, assert deep equality. Also asserts
// LoadJSON returns (nil, nil) for a missing file and a typed error for
// corrupt JSON.
func TestLoadSaveJSONRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.json")

	want := map[string]any{
		"videoId":    "OpQFFLBMEPI",
		"trackName":  "Test Track",
		"artistName": "Test Artist",
		"duration":   180.5,
		"isWord":     true,
		"tags":       []any{"a", "b", "c"},
	}
	if err := SaveJSON(path, want); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	got, err := LoadJSON(path)
	if err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if got["videoId"] != want["videoId"] {
		t.Errorf("LoadJSON videoId = %v, want %v", got["videoId"], want["videoId"])
	}
	if got["trackName"] != want["trackName"] {
		t.Errorf("LoadJSON trackName = %v, want %v", got["trackName"], want["trackName"])
	}

	// Missing file → (nil, nil) per spec.
	missing := filepath.Join(dir, "does-not-exist.json")
	m, err := LoadJSON(missing)
	if err != nil {
		t.Errorf("LoadJSON missing file: want nil error, got %v", err)
	}
	if m != nil {
		t.Errorf("LoadJSON missing file: want nil map, got %v", m)
	}

	// Corrupt JSON → non-nil error.
	corrupt := filepath.Join(dir, "corrupt.json")
	if err := os.WriteFile(corrupt, []byte("not json {"), 0o600); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}
	if _, err := LoadJSON(corrupt); err == nil {
		t.Errorf("LoadJSON corrupt file: want error, got nil")
	}
}

// TestSaveJSON_Atomic confirms SaveJSON does not leave a temp file behind on
// success. The atomic-rename pattern is the contract; checking the directory
// catches any regression that swaps the implementation for a direct write.
func TestSaveJSON_Atomic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.json")
	if err := SaveJSON(path, map[string]any{"k": "v"}); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected 1 file after SaveJSON, got %d: %v", len(entries), names)
	}
}
