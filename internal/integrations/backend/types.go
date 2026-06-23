package backend

import (
	"encoding/json"
	"errors"
	"fmt"
)

// FetchResponse is the JSON body returned by POST /local-api/fetch.
type FetchResponse struct {
	VideoID    string  `json:"videoId"`
	TrackName  string  `json:"trackName"`
	ArtistName string  `json:"artistName"`
	Duration   float64 `json:"duration"`
	AudioReady bool    `json:"audioReady"`
	AudioURL   string  `json:"audioUrl"`
	CachedAt   *string `json:"cachedAt"`
	SourceURL  *string `json:"sourceUrl"`
}

// Source identifies which audio source peaks were computed from.
type Source string

const (
	SourceOriginal Source = "original"
	SourceDemucs   Source = "demucs"
)

// PeaksResponse is the JSON body returned by GET /local-api/peaks/{video_id}.
type PeaksResponse struct {
	VideoID     string    `json:"videoId"`
	Samples     int       `json:"samples"`
	Duration    float64   `json:"duration"`
	Peaks       []float64 `json:"peaks"`
	SourceFile  string    `json:"sourceFile"`
	GeneratedAt string    `json:"generatedAt"`
	Source      Source    `json:"source"`
	CacheHit    bool      `json:"cacheHit"`
}

// TranscriptionStatus is the discriminator for transcribe responses and SSE events.
type TranscriptionStatus string

const (
	TranscriptionQueued    TranscriptionStatus = "queued"
	TranscriptionRunning   TranscriptionStatus = "running"
	TranscriptionCompleted TranscriptionStatus = "completed"
	TranscriptionFailed    TranscriptionStatus = "failed"
)

// TranscribeEvent is a sealed union of the possible transcribe event payloads.
type TranscribeEvent interface {
	sealed()
}

// TranscriptionQueuedEvent is returned when a new transcription job is accepted.
type TranscriptionQueuedEvent struct {
	Status  TranscriptionStatus `json:"status"`
	VideoID string              `json:"videoId"`
	Message string              `json:"message"`
}

func (TranscriptionQueuedEvent) sealed() {}

// TranscriptionRunningEvent is returned while a transcription job is active.
type TranscriptionRunningEvent struct {
	Status  TranscriptionStatus `json:"status"`
	VideoID string              `json:"videoId"`
	Job     *TranscriptionJob   `json:"job,omitempty"`
}

func (TranscriptionRunningEvent) sealed() {}

// TranscriptionJob captures the running/failed job metadata.
type TranscriptionJob struct {
	Status    string `json:"status"`
	StartedAt string `json:"startedAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	Error     string `json:"error,omitempty"`
}

// TranscriptionCompletedEvent is returned when transcription has finished.
type TranscriptionCompletedEvent struct {
	Status      TranscriptionStatus `json:"status"`
	VideoID     string              `json:"videoId"`
	Provider    string              `json:"provider,omitempty"`
	Language    string              `json:"language,omitempty"`
	Plain       string              `json:"plain"`
	Synced      string              `json:"synced"`
	IsAIRefined bool                `json:"is_ai_refined"`
	Model       string              `json:"model,omitempty"`
	Mode        string              `json:"mode"`
	UpdatedAt   string              `json:"updatedAt"`
}

func (TranscriptionCompletedEvent) sealed() {}

// TranscriptionFailedEvent is returned when transcription fails.
type TranscriptionFailedEvent struct {
	Status    TranscriptionStatus `json:"status"`
	VideoID   string              `json:"videoId"`
	Error     string              `json:"error"`
	UpdatedAt string              `json:"updatedAt"`
}

func (TranscriptionFailedEvent) sealed() {}

// TranscribeResponse wraps a decoded event and exposes the common status.
type TranscribeResponse struct {
	status TranscriptionStatus
	event  TranscribeEvent
}

// Status returns the transcription status discriminator.
func (r TranscribeResponse) Status() TranscriptionStatus {
	return r.status
}

// Event returns the sealed event payload.
func (r TranscribeResponse) Event() TranscribeEvent {
	return r.event
}

// AsCompleted returns the completed payload if the event is completed.
func (r TranscribeResponse) AsCompleted() (TranscriptionCompletedEvent, bool) {
	c, ok := r.event.(TranscriptionCompletedEvent)
	return c, ok
}

// AsFailed returns the failed payload if the event is failed.
func (r TranscribeResponse) AsFailed() (TranscriptionFailedEvent, bool) {
	f, ok := r.event.(TranscriptionFailedEvent)
	return f, ok
}

// AsQueued returns the queued payload if the event is queued.
func (r TranscribeResponse) AsQueued() (TranscriptionQueuedEvent, bool) {
	q, ok := r.event.(TranscriptionQueuedEvent)
	return q, ok
}

// AsRunning returns the running payload if the event is running.
func (r TranscribeResponse) AsRunning() (TranscriptionRunningEvent, bool) {
	rn, ok := r.event.(TranscriptionRunningEvent)
	return rn, ok
}

// FetchRequest is the JSON body for POST /local-api/fetch.
type FetchRequest struct {
	URL     string `json:"url,omitempty"`
	VideoID string `json:"videoId,omitempty"`
}

// TranscribeRequest is the JSON body for POST /local-api/transcribe.
type TranscribeRequest struct {
	VideoID          string `json:"videoId"`
	Force            bool   `json:"force"`
	EnableRefinement bool   `json:"enableRefinement"`
	Mode             string `json:"mode"`
}

// DecodeKind identifies which contract failed to decode.
type DecodeKind string

const (
	DecodeKindFetch      DecodeKind = "fetch"
	DecodeKindPeaks      DecodeKind = "peaks"
	DecodeKindTranscribe DecodeKind = "transcribe"
)

// APIError reports an unexpected HTTP status from the backend.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("backend API error status=%d body=%q", e.StatusCode, e.Body)
}

// IsAPIError reports whether err is or wraps an *APIError.
func IsAPIError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr)
}

// DecodeError reports a JSON decoding failure for a backend contract.
type DecodeError struct {
	Kind DecodeKind
	Err  error
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("backend decode %s: %v", e.Kind, e.Err)
}

// Unwrap returns the underlying JSON error.
func (e *DecodeError) Unwrap() error {
	return e.Err
}

// DecodeFetchResponse parses the JSON body from POST /local-api/fetch.
func DecodeFetchResponse(body []byte) (FetchResponse, error) {
	var resp FetchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return FetchResponse{}, &DecodeError{Kind: DecodeKindFetch, Err: err}
	}
	return resp, nil
}

// DecodePeaksResponse parses the JSON body from GET /local-api/peaks/{video_id}.
func DecodePeaksResponse(body []byte) (PeaksResponse, error) {
	var resp PeaksResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return PeaksResponse{}, &DecodeError{Kind: DecodeKindPeaks, Err: err}
	}
	return resp, nil
}

// DecodeTranscribeResponse parses the JSON body from POST /local-api/transcribe
// or a single SSE data frame.
func DecodeTranscribeResponse(body []byte) (TranscribeResponse, error) {
	var envelope struct {
		Status TranscriptionStatus `json:"status"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return TranscribeResponse{}, &DecodeError{Kind: DecodeKindTranscribe, Err: err}
	}

	switch envelope.Status {
	case TranscriptionQueued:
		var event TranscriptionQueuedEvent
		if err := json.Unmarshal(body, &event); err != nil {
			return TranscribeResponse{}, &DecodeError{Kind: DecodeKindTranscribe, Err: err}
		}
		return TranscribeResponse{status: event.Status, event: event}, nil
	case TranscriptionRunning:
		var event TranscriptionRunningEvent
		if err := json.Unmarshal(body, &event); err != nil {
			return TranscribeResponse{}, &DecodeError{Kind: DecodeKindTranscribe, Err: err}
		}
		return TranscribeResponse{status: event.Status, event: event}, nil
	case TranscriptionCompleted:
		var event TranscriptionCompletedEvent
		if err := json.Unmarshal(body, &event); err != nil {
			return TranscribeResponse{}, &DecodeError{Kind: DecodeKindTranscribe, Err: err}
		}
		return TranscribeResponse{status: event.Status, event: event}, nil
	case TranscriptionFailed:
		var event TranscriptionFailedEvent
		if err := json.Unmarshal(body, &event); err != nil {
			return TranscribeResponse{}, &DecodeError{Kind: DecodeKindTranscribe, Err: err}
		}
		return TranscribeResponse{status: event.Status, event: event}, nil
	default:
		// Preserve the unknown status so callers can inspect it without losing data.
		var raw struct {
			Status  TranscriptionStatus `json:"status"`
			VideoID string              `json:"videoId"`
		}
		_ = json.Unmarshal(body, &raw)
		return TranscribeResponse{status: envelope.Status, event: rawTranscribeEvent{status: raw.Status, videoID: raw.VideoID}}, nil
	}
}

type rawTranscribeEvent struct {
	status  TranscriptionStatus
	videoID string
}

func (rawTranscribeEvent) sealed() {}

// IsDecodeError reports whether err is a *DecodeError.
func IsDecodeError(err error) bool {
	var decodeErr *DecodeError
	return errors.As(err, &decodeErr)
}

// ChallengeResponse is the JSON body from POST /api/request-challenge.
type ChallengeResponse struct {
	Prefix string `json:"prefix"`
	Target string `json:"target"`
}

// PublishPayload is the JSON body for POST /api/publish.
type PublishPayload struct {
	TrackName    string `json:"trackName"`
	ArtistName   string `json:"artistName"`
	AlbumName    string `json:"albumName"`
	Duration     int    `json:"duration"`
	PlainLyrics  string `json:"plainLyrics"`
	SyncedLyrics string `json:"syncedLyrics"`
}

// PublishToken formats a prefix and solved nonce for the X-Publish-Token header.
func PublishToken(prefix, nonce string) string {
	return prefix + ":" + nonce
}
