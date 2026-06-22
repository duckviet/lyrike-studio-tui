// Package server — Task 2 of the go-backend plan. Ported verbatim from
// /home/duckviet/lrclib-upload/backend/core/utils.py. The two HTTP-bound
// helpers (SanitizeYouTubeURL, NormalizeVideoID) are the contract the TUI's
// /local-api/fetch path depends on; the JSON helpers back the cache CRUD
// tasks (3+) and the draft store (12+).
package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// youtubeHostContains lists the host suffixes that SanitizeYouTubeURL treats
// as YouTube-family. Membership is a substring check on the lower-cased
// hostname, matching the Python backend's `"youtube.com" in hostname or
// "youtu.be" in hostname` branch.
var youtubeHostContains = []string{"youtube.com", "youtu.be"}

// isYouTubeHost reports whether the lower-cased hostname belongs to a
// YouTube-family domain. Empty hostnames are not YouTube.
func isYouTubeHost(host string) bool {
	if host == "" {
		return false
	}
	for _, needle := range youtubeHostContains {
		if strings.Contains(host, needle) {
			return true
		}
	}
	return false
}

// NormalizeVideoID returns raw with every byte that is not in
// [A-Za-z0-9_-] removed. Whitespace at the edges is stripped first to match
// the Python `re.sub(..., raw_video_id.strip())` semantics. The empty
// string maps to "" and an all-special input collapses to "".
func NormalizeVideoID(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	// Build the output in a pre-sized builder. Strings are immutable in Go
	// and strings.Map avoids an intermediate []rune allocation by
	// dispatching rune-at-a-time.
	var b strings.Builder
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		switch {
		case r >= 'A' && r <= 'Z',
			r >= 'a' && r <= 'z',
			r >= '0' && r <= '9',
			r == '-' || r == '_':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SanitizeYouTubeURL drops the `list` and `index` query parameters on
// YouTube-family URLs and preserves every other parameter and the original
// scheme/host/path. Non-YouTube URLs and unparseable input are returned
// unchanged. The input string is never mutated.
func SanitizeYouTubeURL(raw string) string {
	if raw == "" {
		return raw
	}
	trimmed := strings.TrimSpace(raw)
	parsed, err := url.Parse(trimmed)
	if err != nil {
		// Mirror the Python `except Exception: return url` fallback.
		return raw
	}
	host := parsed.Hostname()
	if !isYouTubeHost(strings.ToLower(host)) {
		return raw
	}

	// Re-build the query from the parsed values so we preserve order, blank
	// values, and original encoding of the params we keep. parseQuery keeps
	// the first occurrence of each key; the Python parse_qsl with
	// keep_blank_values=True drops duplicates too, so this is parity.
	q := parsed.Query()
	q.Del("list")
	q.Del("index")
	parsed.RawQuery = q.Encode()

	return parsed.String()
}

// UTCNowISO returns the current time in UTC formatted as RFC3339Nano. The
// format carries nanosecond precision and a trailing 'Z' so the string is
// unambiguous across timezones — the wire format the TUI contract expects.
func UTCNowISO() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// LoadJSON reads the file at path and unmarshals it into a map. A missing
// file returns (nil, nil) so callers can treat "absent" as a normal
// empty-cache state without sprinkling os.IsNotExist checks at every call
// site. A present-but-corrupt file returns a wrapped json error.
func LoadJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return out, nil
}

// SaveJSON marshals v as JSON (compact, UTF-8, no HTML escaping) and writes
// it atomically to path: a temp file in the same directory, fsync via Close,
// rename into place, then fsync the parent directory so the rename survives
// a crash. The destination directory is created if missing.
func SaveJSON(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	temp, err := os.CreateTemp(dir, ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temp in %s: %w", dir, err)
	}
	tempPath := temp.Name()
	cleanup := func() { _ = os.Remove(tempPath) }

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		cleanup()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename %s -> %s: %w", tempPath, path, err)
	}
	// Best-effort directory fsync; not all filesystems support it and a
	// failure here is non-fatal (the data is already on disk via the
	// rename).
	_ = syncDir(dir)
	return nil
}

// syncDir issues an fsync on the directory so the rename above is durable
// across power loss. Errors are returned to the caller, which can choose
// whether to surface them. (SaveJSON logs-and-continues because a directory
// fsync failure is a soft warning on most filesystems.)
func syncDir(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open dir %s: %w", dir, err)
	}
	defer func() { _ = handle.Close() }()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("sync dir %s: %w", dir, err)
	}
	return nil
}
