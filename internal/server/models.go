package server

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

// queuedResponse is returned when a new transcription job is accepted.
type queuedResponse struct {
	Status  string `json:"status"`
	VideoID string `json:"videoId"`
	Message string `json:"message"`
}

// healthResponse is returned by the health check endpoints.
type healthResponse struct {
	Status string `json:"status"`
}
