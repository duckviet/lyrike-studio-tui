// Package audiourl downloads remote audio files to temporary local files so
// playback packages that only understand filesystem paths (such as the beep
// player) can play HTTP/HTTPS audio URLs.
package audiourl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// httpClient is a tuned client used for every audio download. It is shared
// across calls so connections can be reused, and it carries an explicit total
// timeout so a stalled backend cannot hang the TUI indefinitely.
var httpClient = &http.Client{
	Timeout: 120 * time.Second,
}

// DownloadToTemp downloads the resource at rawURL to a temporary file and
// returns the local file path. The returned cleanup function removes the
// temporary file; callers should invoke it after playback has finished and the
// file is no longer open.
func DownloadToTemp(ctx context.Context, rawURL string) (path string, cleanup func(), err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", nil, fmt.Errorf("parse audio url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", nil, fmt.Errorf("unsupported url scheme %q", u.Scheme)
	}

	ext := extensionFromURL(u)

	tempFile, err := os.CreateTemp("", "lyrike-audio-*"+ext)
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}
	path = tempFile.Name()
	cleanup = func() { _ = os.Remove(path) }

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("download audio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cleanup()
		return "", nil, fmt.Errorf("download audio: unexpected status %d", resp.StatusCode)
	}

	if ext == "" {
		ext = extensionFromContentType(resp.Header.Get("Content-Type"))
		if ext != "" && !strings.HasSuffix(path, ext) {
			renamed := path + ext
			if err := os.Rename(path, renamed); err == nil {
				path = renamed
				cleanup = func() { _ = os.Remove(renamed) }
			}
		}
	}

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("write audio temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close audio temp file: %w", err)
	}

	return path, cleanup, nil
}

func extensionFromURL(u *url.URL) string {
	base := filepath.Base(u.Path)
	if ext := filepath.Ext(base); ext != "" && ext != "." {
		return ext
	}
	return ""
}

func extensionFromContentType(contentType string) string {
	// Drop parameters such as "audio/mpeg; charset=utf-8".
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	switch strings.ToLower(contentType) {
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/mp4":
		return ".m4a"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return ".wav"
	case "audio/webm":
		return ".webm"
	case "audio/ogg", "audio/vorbis", "audio/opus":
		return ".ogg"
	case "audio/flac":
		return ".flac"
	case "audio/aac":
		return ".aac"
	default:
		return ""
	}
}
