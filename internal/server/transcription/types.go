// Package transcription contains the typed shape of a transcription
// result and the LRC formatters that turn it into synced / plain
// lyrics. The package boundary owns these types; the OpenAI provider
// (task 7) and the job manager (task 8) build a TranscriptionResult
// and call BuildSyncedLyrics / StripWords against it.
//
// This file is the typed mirror of the Python
// `backend/services/transcription/types.py` dataclasses.
package transcription

// TranscribedWord is a single word with its start/end time in seconds.
type TranscribedWord struct {
	Word  string
	Start float64
	End   float64
}

// TranscribedSegment is a contiguous chunk of the transcript (one
// sentence / line) with its own start/end time and the per-word
// timings that fall inside it.
type TranscribedSegment struct {
	Text  string
	Start float64
	End   float64
	Words []TranscribedWord
}

// TranscriptionResult is what a provider returns: a list of timed
// segments plus the provider name, language, full plain text, and
// the raw upstream payload (for refinement / debugging).
type TranscriptionResult struct {
	Provider  string
	Language  string
	Segments  []TranscribedSegment
	PlainText string
	Raw       map[string]any
}

// RefineResult is the output of AI lyric refinement.
type RefineResult struct {
	SyncedLyrics string
	PlainLyrics  string
	IsAIRefined  bool
	Model        string
	Error        string
}
