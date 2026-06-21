package backend

import (
	"encoding/json"
	"fmt"
	"math"
)

// FetchFixture returns a sample 200 response body from POST /local-api/fetch.
func FetchFixture() []byte {
	cachedAt := "2025-01-15T08:30:00Z"
	sourceURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	resp := FetchResponse{
		VideoID:    "dQw4w9WgXcQ",
		TrackName:  "Never Gonna Give You Up",
		ArtistName: "Rick Astley",
		Duration:   212,
		AudioReady: true,
		AudioURL:   "/local-api/audio/dQw4w9WgXcQ",
		CachedAt:   &cachedAt,
		SourceURL:  &sourceURL,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		// Fixtures are compiled; marshalling must never fail.
		panic(fmt.Sprintf("backend: FetchFixture marshal: %v", err))
	}
	return b
}

// PeaksFixture returns a sample 200 response body from GET /local-api/peaks/{video_id}.
func PeaksFixture() []byte {
	samples := 2000
	peaks := make([]float64, samples)
	for i := range samples {
		t := float64(i) / float64(samples) * 4 * math.Pi
		peaks[i] = math.Round(math.Sin(t)*1000) / 1000
	}

	resp := PeaksResponse{
		VideoID:     "dQw4w9WgXcQ",
		Samples:     samples,
		Duration:    212.0,
		Peaks:       peaks,
		SourceFile:  "audio.m4a",
		GeneratedAt: "2025-01-15T08:35:00Z",
		Source:      SourceOriginal,
		CacheHit:    true,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Sprintf("backend: PeaksFixture marshal: %v", err))
	}
	return b
}

// TranscribeQueuedFixture returns a sample "queued" transcribe response.
func TranscribeQueuedFixture() []byte {
	resp := TranscriptionQueuedEvent{
		Status:  TranscriptionQueued,
		VideoID: "dQw4w9WgXcQ",
		Message: "Transcription started.",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Sprintf("backend: TranscribeQueuedFixture marshal: %v", err))
	}
	return b
}

// TranscribeRunningFixture returns a sample "running" transcribe response.
func TranscribeRunningFixture() []byte {
	resp := TranscriptionRunningEvent{
		Status:  TranscriptionRunning,
		VideoID: "dQw4w9WgXcQ",
		Job: &TranscriptionJob{
			Status:    "running",
			StartedAt: "2025-01-15T08:31:00Z",
			UpdatedAt: "2025-01-15T08:31:05Z",
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Sprintf("backend: TranscribeRunningFixture marshal: %v", err))
	}
	return b
}

// TranscribeCompletedFixture returns a sample "completed" transcribe response.
func TranscribeCompletedFixture() []byte {
	resp := TranscriptionCompletedEvent{
		Status:      TranscriptionCompleted,
		VideoID:     "dQw4w9WgXcQ",
		Provider:    "whisper",
		Language:    "en",
		Plain:       "We're no strangers to love\nYou know the rules and so do I",
		Synced:      "[00:12.34] We're no strangers to love\n[00:18.00] You know the rules and so do I",
		IsAIRefined: false,
		Mode:        "normal",
		UpdatedAt:   "2025-01-15T08:32:00Z",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Sprintf("backend: TranscribeCompletedFixture marshal: %v", err))
	}
	return b
}

// TranscribeFailedFixture returns a sample "failed" transcribe response.
func TranscribeFailedFixture() []byte {
	resp := TranscriptionFailedEvent{
		Status:    TranscriptionFailed,
		VideoID:   "dQw4w9WgXcQ",
		Error:     "transcription worker failed",
		UpdatedAt: "2025-01-15T08:32:00Z",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Sprintf("backend: TranscribeFailedFixture marshal: %v", err))
	}
	return b
}

// ChallengeFixture returns a sample 200 response body from POST /api/request-challenge.
func ChallengeFixture() []byte {
	resp := ChallengeResponse{
		Prefix: "a1b2c3d4e5f6",
		Target: "00000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Sprintf("backend: ChallengeFixture marshal: %v", err))
	}
	return b
}

// PublishFixture returns a sample request body for POST /api/publish.
func PublishFixture() []byte {
	resp := PublishPayload{
		TrackName:    "Never Gonna Give You Up",
		ArtistName:   "Rick Astley",
		AlbumName:    "Whenever You Need Somebody",
		Duration:     212,
		PlainLyrics:  "We're no strangers to love\nYou know the rules and so do I",
		SyncedLyrics: "[00:12.34] We're no strangers to love\n[00:18.00] You know the rules and so do I",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		panic(fmt.Sprintf("backend: PublishFixture marshal: %v", err))
	}
	return b
}
