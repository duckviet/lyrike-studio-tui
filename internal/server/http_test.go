// allow: SIZE_OK — comprehensive handler test suite required by task 11 acceptance criteria.
package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
	"github.com/duckviet/lyrike-studio-tui/internal/server/cache"
	"github.com/duckviet/lyrike-studio-tui/internal/server/transcription"
)

// testProvider implements transcription.Provider for tests.
type testProvider struct {
	result  transcription.TranscriptionResult
	err     error
	startCh chan struct{}
	doneCh  chan struct{}
}

func (p *testProvider) Transcribe(audioPath string) (transcription.TranscriptionResult, error) {
	if p.startCh != nil {
		close(p.startCh)
		<-p.doneCh
	}
	return p.result, p.err
}

// testRefiner implements transcription.LyricRefiner for tests.
type testRefiner struct {
	result transcription.RefineResult
	err    error
}

func (r *testRefiner) RefineLyrics(synced, plain, trackName, artistName string, duration float64) (transcription.RefineResult, error) {
	return r.result, r.err
}

func sampleTranscriptionResult() transcription.TranscriptionResult {
	return transcription.TranscriptionResult{
		Provider: "openai-whisper-1",
		Language: "en",
		Segments: []transcription.TranscribedSegment{
			{
				Text:  "Hello world",
				Start: 0.0,
				End:   2.0,
				Words: []transcription.TranscribedWord{
					{Word: "Hello", Start: 0.0, End: 1.0},
					{Word: "world", Start: 1.0, End: 2.0},
				},
			},
		},
		PlainText: "Hello world",
	}
}

func newTestServer(t *testing.T) (*Server, *cache.Store, *transcription.Manager) {
	t.Helper()
	cfg := &Config{CacheDir: t.TempDir()}
	store := cache.NewStore(cfg.CacheDir)
	manager := transcription.NewManager(store, &testProvider{result: sampleTranscriptionResult()}, &testRefiner{})
	s := NewServer(cfg, store, manager, nil)
	s.fetchInfo = func(ctx context.Context, url string) (map[string]any, error) {
		return map[string]any{
			"title":    "Test Track",
			"uploader": "Test Artist",
			"duration": 123.45,
		}, nil
	}
	s.downloadAudio = func(ctx context.Context, url, videoID, cacheDir string) (string, error) {
		return writeTestAudio(t, cacheDir, videoID), nil
	}
	s.computePeaks = func(ctx context.Context, path string, samples int) ([]float64, error) {
		return make([]float64, samples), nil
	}
	return s, store, manager
}

func writeTestAudio(t *testing.T, cacheDir, videoID string) string {
	t.Helper()
	path := filepath.Join(cacheDir, "audio", videoID, "original.mp3")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir audio: %v", err)
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte{0xff}, 100), 0o644); err != nil {
		t.Fatalf("write audio: %v", err)
	}
	return path
}

func writeTestMetadata(t *testing.T, store *cache.Store, videoID string) {
	t.Helper()
	if err := store.SaveMetadata(videoID, map[string]any{
		"trackName":  "Test Track",
		"artistName": "Test Artist",
		"duration":   123.45,
	}); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
}

func doRequest(t *testing.T, handler http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestFetchHandler(t *testing.T) {
	s, _, _ := newTestServer(t)
	videoID := "dQw4w9WgXcQ"
	writeTestAudio(t, s.cfg.CacheDir, videoID)

	payload, _ := json.Marshal(FetchRequest{URL: "https://www.youtube.com/watch?v=" + videoID})
	rr := doRequest(t, s.Handler(), http.MethodPost, "/local-api/fetch", bytes.NewReader(payload))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	resp, err := backend.DecodeFetchResponse(rr.Body.Bytes())
	if err != nil {
		t.Fatalf("decode fetch response: %v", err)
	}
	if resp.VideoID != videoID {
		t.Errorf("videoId = %q, want %q", resp.VideoID, videoID)
	}
	if resp.TrackName != "Test Track" {
		t.Errorf("trackName = %q, want %q", resp.TrackName, "Test Track")
	}
	if resp.ArtistName != "Test Artist" {
		t.Errorf("artistName = %q, want %q", resp.ArtistName, "Test Artist")
	}
	if resp.Duration != 123.45 {
		t.Errorf("duration = %v, want %v", resp.Duration, 123.45)
	}
	if !resp.AudioReady {
		t.Errorf("audioReady = false, want true")
	}
	if resp.AudioURL == "" {
		t.Errorf("audioUrl empty")
	}
}

func TestFetchMissingParams(t *testing.T) {
	s, _, _ := newTestServer(t)
	payload, _ := json.Marshal(FetchRequest{})
	rr := doRequest(t, s.Handler(), http.MethodPost, "/local-api/fetch", bytes.NewReader(payload))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestFetchCacheMissNoURL(t *testing.T) {
	s, _, _ := newTestServer(t)
	payload, _ := json.Marshal(FetchRequest{VideoID: "missingid"})
	rr := doRequest(t, s.Handler(), http.MethodPost, "/local-api/fetch", bytes.NewReader(payload))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestFetchVideoIdMismatch(t *testing.T) {
	s, _, _ := newTestServer(t)
	payload, _ := json.Marshal(FetchRequest{URL: "https://www.youtube.com/watch?v=abc123", VideoID: "xyz789"})
	rr := doRequest(t, s.Handler(), http.MethodPost, "/local-api/fetch", bytes.NewReader(payload))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestPeaksHandler(t *testing.T) {
	s, store, _ := newTestServer(t)
	videoID := "peaktest"
	writeTestMetadata(t, store, videoID)
	writeTestAudio(t, s.cfg.CacheDir, videoID)

	rr := doRequest(t, s.Handler(), http.MethodGet, "/local-api/peaks/"+videoID, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	resp, err := backend.DecodePeaksResponse(rr.Body.Bytes())
	if err != nil {
		t.Fatalf("decode peaks response: %v", err)
	}
	if resp.VideoID != videoID {
		t.Errorf("videoId = %q, want %q", resp.VideoID, videoID)
	}
	if resp.Samples != 400 {
		t.Errorf("samples = %d, want 400", resp.Samples)
	}
	if resp.CacheHit {
		t.Errorf("cacheHit = true, want false")
	}

	cached, err := store.LoadPeaks(videoID, "original")
	if err != nil {
		t.Fatalf("load peaks from cache: %v", err)
	}
	if len(toFloatSlice(cached["peaks"])) != 400 {
		t.Errorf("cached peaks length = %d, want 400", len(toFloatSlice(cached["peaks"])))
	}
}

func TestPeaksForceBypass(t *testing.T) {
	s, store, _ := newTestServer(t)
	videoID := "forcetest"
	writeTestMetadata(t, store, videoID)
	writeTestAudio(t, s.cfg.CacheDir, videoID)

	rr1 := doRequest(t, s.Handler(), http.MethodGet, "/local-api/peaks/"+videoID, nil)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", rr1.Code, http.StatusOK)
	}
	resp1, _ := backend.DecodePeaksResponse(rr1.Body.Bytes())
	if resp1.CacheHit {
		t.Fatalf("first cacheHit = true, want false")
	}

	rr2 := doRequest(t, s.Handler(), http.MethodGet, "/local-api/peaks/"+videoID+"?force=true", nil)
	if rr2.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", rr2.Code, http.StatusOK)
	}
	resp2, _ := backend.DecodePeaksResponse(rr2.Body.Bytes())
	if resp2.CacheHit {
		t.Errorf("force cacheHit = true, want false")
	}
}

func TestPeaksBadSamples(t *testing.T) {
	s, store, _ := newTestServer(t)
	videoID := "badsamples"
	writeTestMetadata(t, store, videoID)
	writeTestAudio(t, s.cfg.CacheDir, videoID)

	rr := doRequest(t, s.Handler(), http.MethodGet, "/local-api/peaks/"+videoID+"?samples=10", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestPeaksDemucsAlways404(t *testing.T) {
	s, store, _ := newTestServer(t)
	videoID := "demucstest"
	writeTestMetadata(t, store, videoID)
	writeTestAudio(t, s.cfg.CacheDir, videoID)

	rr := doRequest(t, s.Handler(), http.MethodGet, "/local-api/peaks/"+videoID+"?source=demucs", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestAudioRange(t *testing.T) {
	s, _, _ := newTestServer(t)
	videoID := "audiorange"
	path := writeTestAudio(t, s.cfg.CacheDir, videoID)
	fi, _ := os.Stat(path)

	req := httptest.NewRequest(http.MethodGet, "/local-api/audio/"+videoID, nil)
	req.Header.Set("Range", "bytes=0-9")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusPartialContent)
	}
	wantCT := "audio/mpeg"
	if ct := rr.Header().Get("Content-Type"); ct != wantCT {
		t.Errorf("content-type = %q, want %q", ct, wantCT)
	}
	cr := rr.Header().Get("Content-Range")
	wantCR := "bytes 0-9/" + strconv.FormatInt(fi.Size(), 10)
	if cr != wantCR {
		t.Errorf("content-range = %q, want %q", cr, wantCR)
	}
	if rr.Body.Len() != 10 {
		t.Errorf("body length = %d, want 10", rr.Body.Len())
	}
}

func TestAudioNotFound(t *testing.T) {
	s, _, _ := newTestServer(t)
	rr := doRequest(t, s.Handler(), http.MethodGet, "/local-api/audio/nosuch", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestTranscribePOST(t *testing.T) {
	videoID := "transcribetest"

	t.Run("cached mode match returns completed", func(t *testing.T) {
		s, store, _ := newTestServer(t)
		writeTestAudio(t, s.cfg.CacheDir, videoID)
		if err := store.SaveTranscript(videoID, map[string]any{
			"status":        "completed",
			"videoId":       videoID,
			"provider":      "openai-whisper-1",
			"language":      "en",
			"plain":         "hello",
			"synced":        "[00:00.00] hello",
			"is_ai_refined": false,
			"model":         "",
			"mode":          "normal",
			"updatedAt":     "2025-01-01T00:00:00Z",
		}); err != nil {
			t.Fatalf("save transcript: %v", err)
		}

		payload, _ := json.Marshal(TranscribeRequest{VideoID: videoID, Mode: "normal"})
		rr := doRequest(t, s.Handler(), http.MethodPost, "/local-api/transcribe", bytes.NewReader(payload))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		resp, err := backend.DecodeTranscribeResponse(rr.Body.Bytes())
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Status() != backend.TranscriptionCompleted {
			t.Errorf("status = %q, want completed", resp.Status())
		}
	})

	t.Run("running returns running", func(t *testing.T) {
		cfg := &Config{CacheDir: t.TempDir()}
		store := cache.NewStore(cfg.CacheDir)
		blocking := &testProvider{result: sampleTranscriptionResult(), startCh: make(chan struct{}), doneCh: make(chan struct{})}
		manager := transcription.NewManager(store, blocking, &testRefiner{})
		s := NewServer(cfg, store, manager, nil)
		writeTestAudio(t, s.cfg.CacheDir, videoID)

		go manager.RunTranscriptionJob(videoID, filepath.Join(cfg.CacheDir, "audio", videoID, "original.mp3"), "false", "normal")
		<-blocking.startCh

		payload, _ := json.Marshal(TranscribeRequest{VideoID: videoID, Mode: "normal"})
		rr := doRequest(t, s.Handler(), http.MethodPost, "/local-api/transcribe", bytes.NewReader(payload))
		close(blocking.doneCh)

		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if ev, ok := manager.CurrentState(videoID); ok && (ev.Status == "completed" || ev.Status == "failed") {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		resp, err := backend.DecodeTranscribeResponse(rr.Body.Bytes())
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Status() != backend.TranscriptionRunning {
			t.Errorf("status = %q, want running", resp.Status())
		}
	})

	t.Run("audio not cached returns 404", func(t *testing.T) {
		s, _, _ := newTestServer(t)
		payload, _ := json.Marshal(TranscribeRequest{VideoID: "noaudio", Mode: "normal"})
		rr := doRequest(t, s.Handler(), http.MethodPost, "/local-api/transcribe", bytes.NewReader(payload))
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("force re-runs", func(t *testing.T) {
		s, store, manager := newTestServer(t)
		writeTestAudio(t, s.cfg.CacheDir, videoID)
		if err := store.SaveTranscript(videoID, map[string]any{
			"status":  "completed",
			"videoId": videoID,
			"mode":    "normal",
			"plain":   "old",
			"synced":  "old",
		}); err != nil {
			t.Fatalf("save transcript: %v", err)
		}

		payload, _ := json.Marshal(TranscribeRequest{VideoID: videoID, Force: true, Mode: "normal"})
		rr := doRequest(t, s.Handler(), http.MethodPost, "/local-api/transcribe", bytes.NewReader(payload))
		if rr.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusAccepted, rr.Body.String())
		}
		resp, err := backend.DecodeTranscribeResponse(rr.Body.Bytes())
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Status() != backend.TranscriptionQueued {
			t.Errorf("status = %q, want queued", resp.Status())
		}

		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if ev, ok := manager.CurrentState(videoID); ok && ev.Status == "completed" {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("force job did not complete")
	})

	t.Run("invalid videoId returns 400", func(t *testing.T) {
		s, _, _ := newTestServer(t)
		payload, _ := json.Marshal(TranscribeRequest{VideoID: "http://example.com", Mode: "normal"})
		rr := doRequest(t, s.Handler(), http.MethodPost, "/local-api/transcribe", bytes.NewReader(payload))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})
}

func TestSSEStream(t *testing.T) {
	cfg := &Config{CacheDir: t.TempDir()}
	store := cache.NewStore(cfg.CacheDir)
	blocking := &testProvider{result: sampleTranscriptionResult(), startCh: make(chan struct{}), doneCh: make(chan struct{})}
	manager := transcription.NewManager(store, blocking, &testRefiner{})
	s := NewServer(cfg, store, manager, nil)
	videoID := "ssetest"
	writeTestAudio(t, s.cfg.CacheDir, videoID)

	ch := manager.Subscribe(videoID)
	defer manager.Unsubscribe(videoID, ch)
	go manager.RunTranscriptionJob(videoID, filepath.Join(cfg.CacheDir, "audio", videoID, "original.mp3"), "false", "normal")
	<-blocking.startCh
	<-ch // running

	req := httptest.NewRequest(http.MethodGet, "/local-api/transcribe/stream/"+videoID, nil)
	rr := httptest.NewRecorder()
	go func() {
		// Give the handler time to subscribe before completing the job.
		time.Sleep(50 * time.Millisecond)
		close(blocking.doneCh)
	}()
	s.Handler().ServeHTTP(rr, req)

	var events []backend.TranscribeResponse
	scanner := bufio.NewScanner(rr.Body)
	var data []byte
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data = []byte(strings.TrimPrefix(line, "data: "))
		}
		if line == "" && len(data) > 0 {
			resp, err := backend.DecodeTranscribeResponse(data)
			if err != nil {
				t.Fatalf("decode sse event: %v", err)
			}
			events = append(events, resp)
			data = nil
		}
	}

	if len(events) < 2 {
		t.Fatalf("got %d events, want at least 2: %v", len(events), events)
	}
	if events[0].Status() != backend.TranscriptionRunning {
		t.Errorf("first event status = %q, want running", events[0].Status())
	}
	if events[len(events)-1].Status() != backend.TranscriptionCompleted {
		t.Errorf("last event status = %q, want completed", events[len(events)-1].Status())
	}
	for _, ev := range events {
		if ev.Status() == backend.TranscriptionQueued {
			t.Errorf("unexpected queued event in SSE stream")
		}
	}
}

func TestDraftProjects(t *testing.T) {
	s, _, _ := newTestServer(t)
	handler := s.Handler()
	projectID := "draft-project-1"
	body := []byte(`{
		"id":"draft-project-1",
		"metadata":{
			"videoID":"dQw4w9WgXcQ",
			"trackName":"Demo Track",
			"artistName":"Demo Artist",
			"albumName":"Demo Album",
			"duration":123.45,
			"updatedAt":"2026-06-22T00:00:00Z"
		},
		"syncedLyrics":"[00:01.00] hello world"
	}`)

	put := doRequest(t, handler, http.MethodPut, "/local-api/projects/"+projectID, bytes.NewReader(body))
	if put.Code != http.StatusNoContent {
		t.Fatalf("PUT status = %d, want %d; body=%s", put.Code, http.StatusNoContent, put.Body.String())
	}

	get := doRequest(t, handler, http.MethodGet, "/local-api/projects/"+projectID, nil)
	if get.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body=%s", get.Code, http.StatusOK, get.Body.String())
	}
	var loaded map[string]any
	if err := json.Unmarshal(get.Body.Bytes(), &loaded); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if loaded["id"] != projectID {
		t.Fatalf("GET id = %q, want %q", loaded["id"], projectID)
	}
	if loaded["syncedLyrics"] != "[00:01.00] hello world" {
		t.Fatalf("GET syncedLyrics = %q, want saved lyrics", loaded["syncedLyrics"])
	}
	metadata, ok := loaded["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("GET metadata = %#v, want object", loaded["metadata"])
	}
	if metadata["duration"] != 123.45 {
		t.Fatalf("GET metadata.duration = %#v, want 123.45", metadata["duration"])
	}

	list := doRequest(t, handler, http.MethodGet, "/local-api/projects", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("LIST status = %d, want %d; body=%s", list.Code, http.StatusOK, list.Body.String())
	}
	var projects []map[string]any
	if err := json.Unmarshal(list.Body.Bytes(), &projects); err != nil {
		t.Fatalf("decode LIST: %v", err)
	}
	if len(projects) != 1 || projects[0]["id"] != projectID {
		t.Fatalf("LIST projects = %#v, want one project %q", projects, projectID)
	}
}

func TestDraftProjectsErrors(t *testing.T) {
	s, _, _ := newTestServer(t)
	handler := s.Handler()

	missingDelete := doRequest(t, handler, http.MethodDelete, "/local-api/projects/missing-project", nil)
	if missingDelete.Code != http.StatusNotFound {
		t.Fatalf("DELETE missing status = %d, want %d; body=%s", missingDelete.Code, http.StatusNotFound, missingDelete.Body.String())
	}

	invalidID := doRequest(t, handler, http.MethodGet, "/local-api/projects/bad:id", nil)
	if invalidID.Code != http.StatusBadRequest {
		t.Fatalf("GET invalid status = %d, want %d; body=%s", invalidID.Code, http.StatusBadRequest, invalidID.Body.String())
	}

	malformed := doRequest(t, handler, http.MethodPut, "/local-api/projects/draft-project-1", strings.NewReader(`{"id":`))
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("PUT malformed status = %d, want %d; body=%s", malformed.Code, http.StatusBadRequest, malformed.Body.String())
	}
}

func TestCacheRoutesNotFound(t *testing.T) {
	s, _, _ := newTestServer(t)
	for _, path := range []string{"/cache/audio/missing", "/cache/peaks/missing", "/cache/transcript/missing"} {
		rr := doRequest(t, s.Handler(), http.MethodGet, path, nil)
		if rr.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want %d", path, rr.Code, http.StatusNotFound)
		}
	}
}

func TestCacheControlHeaders(t *testing.T) {
	s, store, _ := newTestServer(t)
	videoID := "cachectrl"
	writeTestAudio(t, s.cfg.CacheDir, videoID)
	if err := store.SavePeaks(videoID, "original", map[string]any{"peaks": []float64{0.5}}); err != nil {
		t.Fatalf("save peaks: %v", err)
	}
	if err := store.SaveTranscript(videoID, map[string]any{"status": "completed"}); err != nil {
		t.Fatalf("save transcript: %v", err)
	}

	cases := []struct {
		path   string
		wantCC string
	}{
		{"/cache/audio/" + videoID + "?source=original", "public, max-age=86400, stale-while-revalidate=3600"},
		{"/cache/peaks/" + videoID, "public, max-age=3600, stale-while-revalidate=300"},
		{"/cache/transcript/" + videoID, "public, max-age=600, stale-while-revalidate=60"},
	}
	for _, tc := range cases {
		rr := doRequest(t, s.Handler(), http.MethodGet, tc.path, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", tc.path, rr.Code, http.StatusOK)
		}
		if got := rr.Header().Get("Cache-Control"); got != tc.wantCC {
			t.Errorf("%s cache-control = %q, want %q", tc.path, got, tc.wantCC)
		}
	}
}

func TestHealthEndpoints(t *testing.T) {
	s, _, _ := newTestServer(t)
	getCases := []string{"/", "/health", "/healthz"}
	for _, path := range getCases {
		rr := doRequest(t, s.Handler(), http.MethodGet, path, nil)
		if rr.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, rr.Code, http.StatusOK)
		}
		if path != "/healthz" && rr.Body.String() != `{"status":"ok"}`+"\n" {
			t.Errorf("GET %s body = %q, want {\"status\":\"ok\"}", path, rr.Body.String())
		}
	}
	headCases := []string{"/", "/health"}
	for _, path := range headCases {
		rr := doRequest(t, s.Handler(), http.MethodHead, path, nil)
		if rr.Code != http.StatusOK {
			t.Errorf("HEAD %s status = %d, want %d", path, rr.Code, http.StatusOK)
		}
		if rr.Body.Len() != 0 {
			t.Errorf("HEAD %s body length = %d, want 0", path, rr.Body.Len())
		}
	}
}
