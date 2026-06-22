package transcription

import (
	"math"
	"testing"
)

// TestFormatter_EnhancedLRC_WordsExist covers the enhanced LRC path:
// when a segment has at least one non-empty word, the line is
// `[start]<word1-time>word1 <word2-time>word2 ...` with no space
// between the bracket and the first `<` marker, matching the Python
// reference (`test_transcription_formatter.py::test_emits_enhanced_lrc_when_words_exist`).
func TestFormatter_EnhancedLRC_WordsExist(t *testing.T) {
	// Given
	result := TranscriptionResult{
		Provider: "test",
		Language: "en",
		Segments: []TranscribedSegment{
			{
				Text:  "Hello world",
				Start: 1.0,
				End:   2.0,
				Words: []TranscribedWord{
					{Word: "Hello", Start: 1.1, End: 1.4},
					{Word: "world", Start: 1.5, End: 1.9},
				},
			},
		},
		PlainText: "Hello world",
	}

	// When
	synced, plain := BuildSyncedLyrics(result)

	// Then
	const wantSynced = "[00:01.00]<00:01.10>Hello <00:01.50>world"
	const wantPlain = "Hello world"
	if synced != wantSynced {
		t.Fatalf("synced = %q, want %q", synced, wantSynced)
	}
	if plain != wantPlain {
		t.Fatalf("plain = %q, want %q", plain, wantPlain)
	}
}

// TestFormatter_LineLRC_NoWords covers the line-LRC fallback path:
// when a segment has no words, the line is `[start] text` (single
// space between bracket and text), matching the Python reference
// (`test_transcription_formatter.py::test_keeps_line_lrc_when_words_are_absent`).
func TestFormatter_LineLRC_NoWords(t *testing.T) {
	// Given
	result := TranscriptionResult{
		Provider: "test",
		Language: "en",
		Segments: []TranscribedSegment{
			{
				Text:  "Hello world",
				Start: 1.0,
				End:   2.0,
			},
		},
		PlainText: "Hello world",
	}

	// When
	synced, plain := BuildSyncedLyrics(result)

	// Then
	const wantSynced = "[00:01.00] Hello world"
	const wantPlain = "Hello world"
	if synced != wantSynced {
		t.Fatalf("synced = %q, want %q", synced, wantSynced)
	}
	if plain != wantPlain {
		t.Fatalf("plain = %q, want %q", plain, wantPlain)
	}
}

// TestFormatter_SkipsMalformedTiming covers the malformed-timing skip
// path: if a segment's start is not a finite, non-NaN number, the
// segment is omitted from both synced and plain output. The Python
// reference sets `start = "bad"` to force float() to raise; in Go
// the field is `float64`, so a NaN start is the typed equivalent
// that `format_time` cannot produce a valid time string for.
func TestFormatter_SkipsMalformedTiming(t *testing.T) {
	// Given
	result := TranscriptionResult{
		Provider: "test",
		Language: "en",
		Segments: []TranscribedSegment{
			{
				Text:  "Hello world",
				Start: math.NaN(),
				End:   2.0,
			},
		},
		PlainText: "Hello world",
	}

	// When
	synced, plain := BuildSyncedLyrics(result)

	// Then: the unsafe timed segment is skipped entirely.
	if synced != "" {
		t.Fatalf("synced = %q, want empty", synced)
	}
	if plain != "" {
		t.Fatalf("plain = %q, want empty", plain)
	}
}
