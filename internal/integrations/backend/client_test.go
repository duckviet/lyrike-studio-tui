package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_FetchSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/local-api/fetch" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(FetchFixture())
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.Fetch(context.Background(), FetchRequest{VideoID: "dQw4w9WgXcQ"})
	if err != nil {
		t.Fatalf("Fetch() error = %v, want nil", err)
	}
	if resp.VideoID != "dQw4w9WgXcQ" {
		t.Fatalf("VideoID = %q, want dQw4w9WgXcQ", resp.VideoID)
	}
	if resp.TrackName != "Never Gonna Give You Up" {
		t.Fatalf("TrackName = %q", resp.TrackName)
	}
}

func TestClient_FetchInvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Fetch(context.Background(), FetchRequest{VideoID: "x"})
	if err == nil {
		t.Fatalf("Fetch() error = nil, want decode error")
	}
	if !IsDecodeError(err) {
		t.Fatalf("error type = %T, want decode error", err)
	}
}

func TestClient_FetchHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Fetch(context.Background(), FetchRequest{VideoID: "x"})
	if err == nil {
		t.Fatalf("Fetch() error = nil, want HTTP error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestClient_PeaksSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/local-api/peaks/dQw4w9WgXcQ"
		if r.URL.Path != wantPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, wantPath)
		}
		if r.URL.Query().Get("source") != "original" {
			t.Fatalf("source = %q, want original", r.URL.Query().Get("source"))
		}
		if r.URL.Query().Get("samples") != "2000" {
			t.Fatalf("samples = %q, want 2000", r.URL.Query().Get("samples"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(PeaksFixture())
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.Peaks(context.Background(), "dQw4w9WgXcQ", SourceOriginal, 2000)
	if err != nil {
		t.Fatalf("Peaks() error = %v", err)
	}
	if len(resp.Peaks) != 2000 {
		t.Fatalf("len(Peaks) = %d, want 2000", len(resp.Peaks))
	}
}

func TestClient_TranscribeSSE(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/local-api/transcribe/stream/dQw4w9WgXcQ"
		if r.URL.Path != wantPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", TranscribeRunningFixture())
		_, _ = fmt.Fprintf(w, "data: %s\n\n", TranscribeCompletedFixture())
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var statuses []TranscriptionStatus
	err := client.TranscribeStream(ctx, "dQw4w9WgXcQ", func(event TranscribeResponse) {
		statuses = append(statuses, event.Status())
	})
	if err != nil {
		t.Fatalf("TranscribeStream() error = %v", err)
	}

	if len(statuses) != 2 {
		t.Fatalf("len(statuses) = %d, want 2", len(statuses))
	}
	if statuses[0] != TranscriptionRunning {
		t.Fatalf("status[0] = %q, want running", statuses[0])
	}
	if statuses[1] != TranscriptionCompleted {
		t.Fatalf("status[1] = %q, want completed", statuses[1])
	}
}

func TestClient_RequestChallenge(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/request-challenge" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(ChallengeFixture())
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.RequestChallenge(context.Background())
	if err != nil {
		t.Fatalf("RequestChallenge() error = %v", err)
	}
	if resp.Prefix == "" {
		t.Fatalf("Prefix is empty")
	}
	if resp.Target == "" {
		t.Fatalf("Target is empty")
	}
}

func TestClient_PublishSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/publish" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("X-Publish-Token") != "prefix:123" {
			t.Fatalf("X-Publish-Token = %q", r.Header.Get("X-Publish-Token"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("published"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	var payload PublishPayload
	if err := json.Unmarshal(PublishFixture(), &payload); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	err := client.Publish(context.Background(), "prefix:123", payload)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
}

func TestClient_PublishUpstreamError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("bad pow"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Publish(context.Background(), "prefix:123", PublishPayload{})
	if err == nil {
		t.Fatalf("Publish() error = nil, want HTTP error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
}

func TestClient_ContextTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.Fetch(ctx, FetchRequest{VideoID: "x"})
	if err == nil {
		t.Fatalf("Fetch() error = nil, want timeout")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Fatalf("error = %v, want context timeout", err)
	}
}
