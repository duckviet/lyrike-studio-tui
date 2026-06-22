// Package ytdlp shells out to the yt-dlp CLI to fetch video metadata and
// download audio. It is the Go port of the Python backend's
// services/audio_service.py.
//
// All external calls go through the runner function so unit tests can
// inspect command construction without invoking a real binary. The
// YouTubeCookiesPath package variable is overridable in tests for the same
// reason: production points at /tmp/yt_cookies.txt, tests point at a
// t.TempDir() file.
package ytdlp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// YouTubeCookiesPath is the cookies file consulted at fetch and download
// time. It is a package-level variable so tests can point it at a temp file
// without writing to the real /tmp.
var YouTubeCookiesPath = "/tmp/yt_cookies.txt"

// smartTVUserAgent is the User-Agent string used for both fetch and
// download. A TV client is the strongest against YouTube's cloud IP blocks.
const smartTVUserAgent = "Mozilla/5.0 (SMART-TV; LINUX; Tizen 5.0) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/2.2 Chrome/63.0.3239.111 TV Safari/537.36"

// runner executes the yt-dlp binary. tests can replace it via
// SetRunnerForTest; production uses defaultRunner which shells out via
// os/exec. The args slice is the full argv including the program name at
// index 0 (matching exec.Cmd.Args), so callers may pass the result of
// BuildFetchArgs / BuildDownloadArgs unchanged.
type runner func(args []string) (stdout, stderr []byte, err error)

var ytRunner runner = defaultRunner

// SetRunnerForTest replaces the package runner for the duration of a test
// and returns a restore function. Use t.Cleanup with the returned function
// to restore the default.
func SetRunnerForTest(r runner) func() {
	old := ytRunner
	ytRunner = r
	return func() { ytRunner = old }
}

func defaultRunner(args []string) ([]byte, []byte, error) {
	if len(args) == 0 {
		return nil, nil, fmt.Errorf("ytdlp: empty argv")
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// YtdlpError carries an HTTP status code for a yt-dlp failure so callers
// can map it directly to an HTTP response.
type YtdlpError struct {
	StatusCode int
	Message    string
}

// Error implements the error interface.
func (e *YtdlpError) Error() string { return e.Message }

// FindCachedAudio returns the path to a previously downloaded audio file
// for videoID, or "" and false if none exists. The audio directory
// {cacheDir}/audio/{videoID}/original.* is checked first; if that directory
// does not exist, the legacy flat layout {cacheDir}/media/{videoID}.* is
// scanned, skipping .json sidecars.
//
// This mirrors the Python find_cached_audio() helper.
func FindCachedAudio(cacheDir, videoID string) (string, bool) {
	audioDir := filepath.Join(cacheDir, "audio", videoID)
	if info, err := os.Stat(audioDir); err == nil && info.IsDir() {
		matches, err := filepath.Glob(filepath.Join(audioDir, "original.*"))
		if err == nil {
			for _, p := range matches {
				if fi, statErr := os.Stat(p); statErr == nil && !fi.IsDir() {
					return p, true
				}
			}
		}
		return "", false
	}

	matches, err := filepath.Glob(filepath.Join(cacheDir, "media", videoID+".*"))
	if err != nil {
		return "", false
	}
	for _, p := range matches {
		if strings.EqualFold(filepath.Ext(p), ".json") {
			continue
		}
		if fi, statErr := os.Stat(p); statErr == nil && !fi.IsDir() {
			return p, true
		}
	}
	return "", false
}

// BuildFetchArgs constructs the yt-dlp argv used to fetch a video's
// metadata (no media downloaded). The returned slice includes the program
// name at index 0 so it can be passed straight to exec.Command. The
// cookies-file flag is only added when YouTubeCookiesPath exists on disk.
func BuildFetchArgs(url string) []string {
	args := []string{
		"yt-dlp",
		"--quiet",
		"--no-warnings",
		"--no-playlist",
		"--skip-download",
		"--no-check-certificate",
		"--extractor-args", "youtube:player_client=tv,android,mweb,web",
		"--extractor-args", "youtube:player_skip=web",
		"--source-address", "0.0.0.0",
		"--user-agent", smartTVUserAgent,
		"--add-header", "Accept:*/*",
		"--add-header", "Accept-Language:en-US,en;q=0.9",
	}
	if _, err := os.Stat(YouTubeCookiesPath); err == nil {
		args = append(args, "--cookies-file", YouTubeCookiesPath)
	}
	args = append(args, "--dump-json", url)
	return args
}

// BuildDownloadArgs constructs the yt-dlp argv used to download the best
// available audio to {cacheDir}/audio/{videoID}/original.%(ext)s. The
// download and fetch flag sets differ on purpose: download omits the "web"
// player client, omits player_skip, and omits the Accept/Accept-Language
// headers (those break some extractor clients and are only useful for
// metadata fetch).
func BuildDownloadArgs(url, videoID, cacheDir string) []string {
	outtmpl := filepath.Join(cacheDir, "audio", videoID, "original.%(ext)s")
	args := []string{
		"yt-dlp",
		"--format", "bestaudio[ext=m4a]/bestaudio/best",
		"--output", outtmpl,
		"--extractor-args", "youtube:player_client=tv,android,mweb",
		"--source-address", "0.0.0.0",
		"--user-agent", smartTVUserAgent,
	}
	if _, err := os.Stat(YouTubeCookiesPath); err == nil {
		args = append(args, "--cookies-file", YouTubeCookiesPath)
	}
	args = append(args, url)
	return args
}

// MapYtdlpError maps a yt-dlp stderr string to an HTTP status code, per
// the Python audio_service.fetch_video_info() rules:
//
//	video unavailable / private / removed -> 404
//	sign in / bot / captcha                -> 403
//	otherwise                              -> 502
//
// The comparison is case-insensitive.
func MapYtdlpError(stderr string) int {
	msg := strings.ToLower(stderr)
	switch {
	case strings.Contains(msg, "video unavailable"),
		strings.Contains(msg, "private"),
		strings.Contains(msg, "removed"):
		return 404
	case strings.Contains(msg, "sign in"),
		strings.Contains(msg, "bot"),
		strings.Contains(msg, "captcha"):
		return 403
	default:
		return 502
	}
}

// FetchVideoInfo runs yt-dlp in dump-json mode and returns the parsed
// metadata. Failures are wrapped in *YtdlpError with a status code derived
// from the yt-dlp stderr; a non-parseable JSON body is reported as 502.
func FetchVideoInfo(url string) (map[string]any, error) {
	stdout, stderr, err := ytRunner(BuildFetchArgs(url))
	if err != nil {
		return nil, &YtdlpError{
			StatusCode: MapYtdlpError(string(stderr)),
			Message:    strings.TrimSpace(string(stderr)),
		}
	}
	var info map[string]any
	if err := json.Unmarshal(stdout, &info); err != nil {
		return nil, &YtdlpError{
			StatusCode: 502,
			Message:    fmt.Sprintf("yt-dlp: failed to parse JSON: %v", err),
		}
	}
	return info, nil
}

// DownloadAudio downloads the best available audio for url into
// {cacheDir}/audio/{videoID}/original.<ext> and returns the resolved file
// path via FindCachedAudio. The audio dir is created by yt-dlp itself; if
// the download succeeds but the file is not found on disk, a 500 is
// returned.
func DownloadAudio(url, videoID, cacheDir string) (string, error) {
	_, stderr, err := ytRunner(BuildDownloadArgs(url, videoID, cacheDir))
	if err != nil {
		return "", &YtdlpError{
			StatusCode: MapYtdlpError(string(stderr)),
			Message:    strings.TrimSpace(string(stderr)),
		}
	}
	if path, ok := FindCachedAudio(cacheDir, videoID); ok {
		return path, nil
	}
	return "", &YtdlpError{
		StatusCode: 500,
		Message:    "yt-dlp download completed but cached file not found",
	}
}
