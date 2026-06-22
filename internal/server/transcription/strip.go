package transcription

// StripWords returns a copy of result with every segment's `Words`
// slice emptied, so the formatter falls back to line-LRC. The input
// is never mutated; segments are deep-copied so callers (and the
// job manager in task 8) can keep the original around for the
// karaoke mode.
//
// Mirrors the Python reference
// (`backend/services/transcription_service.py::_strip_words`).
func StripWords(result TranscriptionResult) TranscriptionResult {
	segments := make([]TranscribedSegment, len(result.Segments))
	for i, seg := range result.Segments {
		segments[i] = TranscribedSegment{
			Text:  seg.Text,
			Start: seg.Start,
			End:   seg.End,
			Words: nil,
		}
	}
	return TranscriptionResult{
		Provider:  result.Provider,
		Language:  result.Language,
		Segments:  segments,
		PlainText: result.PlainText,
		Raw:       result.Raw,
	}
}
