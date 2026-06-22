package transcription

import (
	"fmt"
	"math"
	"strings"
)

// FormatTime renders seconds as `mm:ss.xx` (always 2 decimal places,
// total width 5 for the seconds part). Negative inputs are clamped to
// 0 to match the Python reference
// (`backend/services/transcription/formatter.py::format_time`).
func FormatTime(seconds float64) string {
	if seconds < 0 || math.IsNaN(seconds) {
		seconds = 0
	}
	// Positive infinity is not meaningful as a timecode; clamp to 0.
	if math.IsInf(seconds, 1) {
		seconds = 0
	}
	minutes := int(seconds / 60)
	wholeSeconds := math.Mod(seconds, 60)
	// The seconds part must always be 5 chars wide (incl. the decimal
	// point and the leading zero on values <10). `%05.2f` gives exactly
	// that: `1.0` -> `01.00`, `60.0` -> `60.00`.
	return fmt.Sprintf("%02d:%05.2f", minutes, wholeSeconds)
}

// FormatWords renders a slice of words as `<mm:ss.xx>word` tokens
// joined by single spaces, skipping words whose stripped text is
// empty. Mirrors the Python reference
// (`backend/services/transcription/formatter.py::format_words`).
func FormatWords(words []TranscribedWord) string {
	parts := make([]string, 0, len(words))
	for _, w := range words {
		text := strings.TrimSpace(w.Word)
		if text == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("<%s>%s", FormatTime(w.Start), text))
	}
	return strings.Join(parts, " ")
}

// BuildSyncedLyrics turns a TranscriptionResult into the LRC strings
// stored on disk and returned over the API. For each segment:
//
//   - empty text or non-finite start/end -> the segment is skipped
//     from BOTH synced and plain output (matches the Python
//     `_strip_words` + `build_synced_lyrics` contract);
//   - one or more non-empty words -> enhanced LRC line
//     `[start]<w1-time>w1 <w2-time>w2 ...` (no space after the
//     closing `]`);
//   - no words -> line-LRC `[start] text` (single space after `]`).
//
// Plain output is the segment text per line, in segment order.
func BuildSyncedLyrics(result TranscriptionResult) (synced string, plain string) {
	var syncedLines, plainLines []string
	for _, seg := range result.Segments {
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			continue
		}
		// Match the Python `float(seg.start)` / `float(seg.end)` guard:
		// a non-finite value (NaN / +-Inf) is treated as malformed and
		// the whole segment is dropped.
		if !isFinite(seg.Start) || !isFinite(seg.End) {
			continue
		}
		words := FormatWords(seg.Words)
		if words != "" {
			syncedLines = append(syncedLines, fmt.Sprintf("[%s]%s", FormatTime(seg.Start), words))
		} else {
			syncedLines = append(syncedLines, fmt.Sprintf("[%s] %s", FormatTime(seg.Start), text))
		}
		plainLines = append(plainLines, text)
	}
	return strings.Join(syncedLines, "\n"), strings.Join(plainLines, "\n")
}

// isFinite reports whether v is a usable number for time formatting.
// NaN and both infinities are rejected; everything else (including
// negatives, which FormatTime then clamps) is accepted.
func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
