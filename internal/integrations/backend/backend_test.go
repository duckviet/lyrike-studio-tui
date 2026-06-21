package backend

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDecodeFetchResponse_whenFixtureValid(t *testing.T) {
	t.Parallel()

	got, err := DecodeFetchResponse(FetchFixture())
	if err != nil {
		t.Fatalf("DecodeFetchResponse() error = %v, want nil", err)
	}

	if got.VideoID != "dQw4w9WgXcQ" {
		t.Fatalf("VideoID = %q, want dQw4w9WgXcQ", got.VideoID)
	}
	if got.TrackName != "Never Gonna Give You Up" {
		t.Fatalf("TrackName = %q, want Never Gonna Give You Up", got.TrackName)
	}
	if got.ArtistName != "Rick Astley" {
		t.Fatalf("ArtistName = %q, want Rick Astley", got.ArtistName)
	}
	if got.Duration != 212.0 {
		t.Fatalf("Duration = %f, want 212", got.Duration)
	}
	if !got.AudioReady {
		t.Fatalf("AudioReady = false, want true")
	}
	if got.AudioURL != "/local-api/audio/dQw4w9WgXcQ" {
		t.Fatalf("AudioURL = %q, want /local-api/audio/dQw4w9WgXcQ", got.AudioURL)
	}
}

func TestDecodeFetchResponse_whenInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := DecodeFetchResponse([]byte(`{not json`))
	if err == nil {
		t.Fatalf("DecodeFetchResponse() error = nil, want typed decode error")
	}

	var decodeErr *DecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("error type = %T, want *backend.DecodeError", err)
	}
	if decodeErr.Kind != DecodeKindFetch {
		t.Fatalf("Kind = %q, want fetch", decodeErr.Kind)
	}
}

func TestDecodePeaksResponse_whenFixtureValid(t *testing.T) {
	t.Parallel()

	got, err := DecodePeaksResponse(PeaksFixture())
	if err != nil {
		t.Fatalf("DecodePeaksResponse() error = %v, want nil", err)
	}

	if got.VideoID != "dQw4w9WgXcQ" {
		t.Fatalf("VideoID = %q, want dQw4w9WgXcQ", got.VideoID)
	}
	if got.Samples != 2000 {
		t.Fatalf("Samples = %d, want 2000", got.Samples)
	}
	if len(got.Peaks) != 2000 {
		t.Fatalf("len(Peaks) = %d, want 2000", len(got.Peaks))
	}
	if got.Source != SourceOriginal {
		t.Fatalf("Source = %q, want original", got.Source)
	}
}

func TestDecodeTranscribeCompletedResponse_whenFixtureValid(t *testing.T) {
	t.Parallel()

	got, err := DecodeTranscribeResponse(TranscribeCompletedFixture())
	if err != nil {
		t.Fatalf("DecodeTranscribeResponse() error = %v, want nil", err)
	}

	completed, ok := got.AsCompleted()
	if !ok {
		t.Fatalf("expected completed event, got status %q", got.Status())
	}
	if completed.VideoID != "dQw4w9WgXcQ" {
		t.Fatalf("VideoID = %q, want dQw4w9WgXcQ", completed.VideoID)
	}
	if completed.Status != TranscriptionCompleted {
		t.Fatalf("Status = %q, want completed", completed.Status)
	}
	if completed.Plain == "" {
		t.Fatalf("Plain is empty, want non-empty")
	}
	if completed.Synced == "" {
		t.Fatalf("Synced is empty, want non-empty")
	}
}

func TestDecodeTranscribeResponse_whenInvalidStatus(t *testing.T) {
	t.Parallel()

	body := []byte(`{"videoId":"x","status":"unknown"}`)
	got, err := DecodeTranscribeResponse(body)
	if err != nil {
		t.Fatalf("DecodeTranscribeResponse() error = %v, want nil", err)
	}

	if got.Status() != TranscriptionStatus("unknown") {
		t.Fatalf("Status() = %q, want unknown", got.Status())
	}

	if _, ok := got.AsCompleted(); ok {
		t.Fatalf("AsCompleted() = true for unknown status, want false")
	}
}

func TestChallengeFixture_decodes(t *testing.T) {
	t.Parallel()

	var got ChallengeResponse
	if err := json.Unmarshal(ChallengeFixture(), &got); err != nil {
		t.Fatalf("json.Unmarshal(ChallengeFixture()) error = %v", err)
	}
	if got.Prefix == "" {
		t.Fatalf("Prefix is empty")
	}
	if got.Target == "" {
		t.Fatalf("Target is empty")
	}
}

func TestPublishFixture_decodes(t *testing.T) {
	t.Parallel()

	var got PublishPayload
	if err := json.Unmarshal(PublishFixture(), &got); err != nil {
		t.Fatalf("json.Unmarshal(PublishFixture()) error = %v", err)
	}
	if got.TrackName == "" {
		t.Fatalf("TrackName is empty")
	}
	if got.Duration <= 0 {
		t.Fatalf("Duration = %d, want > 0", got.Duration)
	}
}
