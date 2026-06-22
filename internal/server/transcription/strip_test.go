package transcription

import (
	"testing"
)

// TestStripWords_RemovesWordTimings covers the normal-mode path: after
// StripWords, every segment's `Words` slice is empty, so BuildSyncedLyrics
// falls back to line-LRC (no `<mm:ss.xx>word` markers) and the plain text
// is preserved verbatim. Mirrors the Python reference
// (`test_transcription_service_mode.py::test_strip_words_removes_word_timings`).
func TestStripWords_RemovesWordTimings(t *testing.T) {
	// Given
	original := TranscriptionResult{
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
	stripped := StripWords(original)
	synced, plain := BuildSyncedLyrics(stripped)

	// Then
	const wantSynced = "[00:01.00] Hello world"
	const wantPlain = "Hello world"
	if synced != wantSynced {
		t.Fatalf("synced = %q, want %q", synced, wantSynced)
	}
	if plain != wantPlain {
		t.Fatalf("plain = %q, want %q", plain, wantPlain)
	}
	if got := len(stripped.Segments[0].Words); got != 0 {
		t.Fatalf("len(stripped.Segments[0].Words) = %d, want 0", got)
	}
}

// TestStripWords_DoesNotMutateInput locks the non-mutation contract:
// the input result's word slice must still carry its original entries
// after StripWords returns. The Python reference uses
// `dataclasses.replace` which always copies; the Go port must match.
func TestStripWords_DoesNotMutateInput(t *testing.T) {
	// Given
	original := TranscriptionResult{
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
	_ = StripWords(original)

	// Then
	if got := len(original.Segments[0].Words); got != 2 {
		t.Fatalf("input was mutated: len(Words) = %d, want 2", got)
	}
}
