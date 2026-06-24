// allow: SIZE_OK — single-file chi router wiring all backend routes per task 11.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
	"github.com/duckviet/lyrike-studio-tui/internal/server/cache"
	"github.com/duckviet/lyrike-studio-tui/internal/server/drafts"
	"github.com/duckviet/lyrike-studio-tui/internal/server/lrclib"
	"github.com/duckviet/lyrike-studio-tui/internal/server/media/peaks"
	"github.com/duckviet/lyrike-studio-tui/internal/server/media/ytdlp"
	sm "github.com/duckviet/lyrike-studio-tui/internal/server/middleware"
	"github.com/duckviet/lyrike-studio-tui/internal/server/transcription"
)

// Server wires all backend HTTP routes to the supporting services.
type Server struct {
	cfg     *Config
	store   *cache.Store
	manager *transcription.Manager
	proxy   *lrclib.Proxy
	drafts  *drafts.Store

	fetchInfo        func(context.Context, string) (map[string]any, error)
	downloadAudio    func(context.Context, string, string, string) (string, error)
	computePeaks     func(context.Context, string, int) ([]float64, error)
	findCachedAudio  func(string, string) (string, bool)
	requestChallenge func(context.Context) (io.ReadCloser, http.Header, int, error)
	publish          func(context.Context, string, io.Reader) (io.ReadCloser, http.Header, int, error)
}

// NewServer returns a Server with production dependencies.
func NewServer(cfg *Config, store *cache.Store, manager *transcription.Manager, proxy *lrclib.Proxy) *Server {
	s := &Server{
		cfg:       cfg,
		store:     store,
		manager:   manager,
		proxy:     proxy,
		fetchInfo: func(ctx context.Context, url string) (map[string]any, error) { return ytdlp.FetchVideoInfo(url) },
		downloadAudio: func(ctx context.Context, url, videoID, cacheDir string) (string, error) {
			return ytdlp.DownloadAudio(url, videoID, cacheDir)
		},
		computePeaks: func(ctx context.Context, path string, samples int) ([]float64, error) {
			return peaks.ComputePeaks(path, samples)
		},
		findCachedAudio: ytdlp.FindCachedAudio,
		drafts:          drafts.NewStore(draftDir(cfg)),
	}
	if proxy != nil {
		s.requestChallenge = proxy.RequestChallenge
		s.publish = proxy.Publish
	}
	return s
}

// Routes returns the configured chi router.
func (s *Server) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(sm.NewCORS(s.cfg.FrontendURL).Handler)
	r.Use(sm.NewRateLimiter().Handler)
	r.Use(chimiddleware.Recoverer)

	r.Get("/", s.health)
	r.Head("/", s.health)
	r.Get("/health", s.health)
	r.Head("/health", s.health)
	r.Get("/healthz", s.health)

	r.Post("/local-api/fetch", s.fetch)
	r.Post("/local-api/transcribe", s.transcribe)
	r.Get("/local-api/transcribe/stream/{id}", s.transcribeStream)
	r.Get("/local-api/projects", s.listProjects)
	r.Put("/local-api/projects/{id}", s.saveProject)
	r.Get("/local-api/projects/{id}", s.loadProject)
	r.Delete("/local-api/projects/{id}", s.deleteProject)
	r.Get("/local-api/audio/{id}", s.audio)
	r.Get("/local-api/peaks/{id}", s.peaks)

	r.Get("/cache/audio/{id}", s.cacheAudio)
	r.Get("/cache/peaks/{id}", s.cachePeaks)
	r.Get("/cache/transcript/{id}", s.cacheTranscript)

	r.Post("/api/request-challenge", s.requestChallengeHandler)
	r.Post("/api/publish", s.publishHandler)
	return r
}

// Handler returns the router as an http.Handler.
func (s *Server) Handler() http.Handler { return s.Routes() }

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
}

func (s *Server) fetch(w http.ResponseWriter, r *http.Request) {
	var req FetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	urlStr := SanitizeYouTubeURL(req.URL)
	videoID := NormalizeVideoID(req.VideoID)

	if urlStr == "" && videoID == "" {
		writeError(w, http.StatusBadRequest, "missing_params", "url or videoId required")
		return
	}
	if isURL(req.VideoID) {
		writeError(w, http.StatusBadRequest, "invalid_video_id", "videoId must not be a URL")
		return
	}
	if urlStr != "" && videoID != "" {
		if extracted := NormalizeVideoID(extractYouTubeVideoID(urlStr)); extracted != "" && extracted != videoID {
			writeError(w, http.StatusBadRequest, "video_id_mismatch", "videoId does not match URL")
			return
		}
	}
	if urlStr != "" && videoID == "" {
		videoID = NormalizeVideoID(extractYouTubeVideoID(urlStr))
	}
	if videoID == "" {
		writeError(w, http.StatusBadRequest, "invalid_video_id", "could not determine videoId")
		return
	}

	meta, err := s.store.LoadMetadata(videoID)
	if err != nil {
		if !errors.Is(err, cache.ErrNotFound) {
			writeError(w, http.StatusInternalServerError, "cache_read_failed", err.Error())
			return
		}
		if urlStr == "" {
			writeError(w, http.StatusNotFound, "not_found", "video not cached and no URL provided")
			return
		}
		info, err := s.fetchInfo(r.Context(), urlStr)
		if err != nil {
			var yte *ytdlp.YtdlpError
			if errors.As(err, &yte) {
				writeError(w, yte.StatusCode, "fetch_failed", yte.Message)
				return
			}
			writeError(w, http.StatusBadGateway, "fetch_failed", err.Error())
			return
		}
		meta = map[string]any{
			"videoId":    videoID,
			"trackName":  firstString(info["title"]),
			"artistName": firstString(info["uploader"], info["artist"], info["channel"]),
			"duration":   durationFloat(info["duration"]),
			"sourceUrl":  urlStr,
			"cachedAt":   UTCNowISO(),
		}
		if err := s.store.SaveMetadata(videoID, meta); err != nil {
			writeError(w, http.StatusInternalServerError, "cache_write_failed", err.Error())
			return
		}
	}

	_, audioReady := s.findCachedAudio(s.cfg.CacheDir, videoID)
	if !audioReady {
		downloadURL := urlStr
		if downloadURL == "" {
			if sVal, ok := meta["sourceUrl"].(string); ok {
				downloadURL = sVal
			}
		}
		if downloadURL != "" {
			_, err := s.downloadAudio(r.Context(), downloadURL, videoID, s.cfg.CacheDir)
			if err != nil {
				var yte *ytdlp.YtdlpError
				if errors.As(err, &yte) {
					writeError(w, yte.StatusCode, "download_failed", yte.Message)
					return
				}
				writeError(w, http.StatusBadGateway, "download_failed", err.Error())
				return
			}
			audioReady = true
		}
	}

	resp := backend.FetchResponse{
		VideoID:    videoID,
		TrackName:  stringValue(meta["trackName"]),
		ArtistName: stringValue(meta["artistName"]),
		Duration:   durationFloat(meta["duration"]),
		AudioReady: audioReady,
	}
	if audioReady {
		resp.AudioURL = "/cache/audio/" + videoID + "?source=original"
	}
	if v, ok := meta["cachedAt"].(string); ok {
		resp.CachedAt = &v
	}
	if urlStr != "" {
		resp.SourceURL = &urlStr
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) transcribe(w http.ResponseWriter, r *http.Request) {
	var req TranscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}
	videoID := NormalizeVideoID(req.VideoID)
	if videoID == "" || isURL(req.VideoID) {
		writeError(w, http.StatusBadRequest, "invalid_video_id", "invalid videoId")
		return
	}
	mode := validatedMode(req.Mode)

	if _, audioReady := s.findCachedAudio(s.cfg.CacheDir, videoID); !audioReady {
		writeError(w, http.StatusNotFound, "audio_not_cached", "audio not cached")
		return
	}

	if !req.Force {
		cached, err := s.store.LoadTranscript(videoID)
		if err == nil {
			if status, _ := cached["status"].(string); status == "completed" {
				if m, _ := cached["mode"].(string); m == mode {
					writeJSON(w, http.StatusOK, completedEventFromCache(videoID, cached))
					return
				}
			}
		} else if !errors.Is(err, cache.ErrNotFound) {
			writeError(w, http.StatusInternalServerError, "cache_read_failed", err.Error())
			return
		}
	}

	if ev, ok := s.manager.CurrentState(videoID); ok && ev.Status == "running" {
		writeJSON(w, http.StatusOK, backend.TranscriptionRunningEvent{Status: backend.TranscriptionRunning, VideoID: videoID})
		return
	}

	audioPath, _ := s.findCachedAudio(s.cfg.CacheDir, videoID)
	enableRefinement := "false"
	if req.EnableRefinement {
		enableRefinement = "true"
	}
	go s.manager.RunTranscriptionJob(videoID, audioPath, enableRefinement, mode)
	writeJSON(w, http.StatusAccepted, queuedResponse{Status: "queued", VideoID: videoID, Message: "Transcription job queued"})
}

func (s *Server) transcribeStream(w http.ResponseWriter, r *http.Request) {
	videoID := NormalizeVideoID(chi.URLParam(r, "id"))
	if videoID == "" {
		writeError(w, http.StatusBadRequest, "invalid_video_id", "invalid videoId")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	if ev, ok := s.manager.CurrentState(videoID); ok {
		writeSSE(w, flusher, ev)
		if ev.Status == "completed" || ev.Status == "failed" {
			return
		}
	}

	ch := s.manager.Subscribe(videoID)
	defer s.manager.Unsubscribe(videoID, ch)
	ctx := r.Context()
	for {
		select {
		case ev, open := <-ch:
			if !open {
				return
			}
			if ev.Status == "queued" {
				continue
			}
			writeSSE(w, flusher, ev)
			if ev.Status == "completed" || ev.Status == "failed" {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) audio(w http.ResponseWriter, r *http.Request) {
	videoID := NormalizeVideoID(chi.URLParam(r, "id"))
	audioPath, ok := s.findCachedAudio(s.cfg.CacheDir, videoID)
	if !ok {
		writeError(w, http.StatusNotFound, "audio_not_found", "audio not found")
		return
	}
	serveAudioFile(w, r, audioPath, audioContentType(audioPath))
}

func (s *Server) peaks(w http.ResponseWriter, r *http.Request) {
	videoID := NormalizeVideoID(chi.URLParam(r, "id"))
	if videoID == "" {
		writeError(w, http.StatusBadRequest, "invalid_video_id", "invalid videoId")
		return
	}

	source := r.URL.Query().Get("source")
	if source == "" {
		source = "original"
	}
	if source == "demucs" {
		writeError(w, http.StatusNotFound, "not_found", "demucs source not available")
		return
	}
	if source != "original" {
		writeError(w, http.StatusBadRequest, "invalid_source", "source must be original or demucs")
		return
	}

	samples := 400
	if raw := r.URL.Query().Get("samples"); raw != "" {
		var err error
		samples, err = strconv.Atoi(raw)
		if err != nil || samples < 64 || samples > 4000 {
			writeError(w, http.StatusBadRequest, "invalid_samples", "samples must be between 64 and 4000")
			return
		}
	}

	force := r.URL.Query().Get("force") == "true"
	cacheHit := false
	var peaksData []float64
	duration := 0.0
	audioPath := ""

	if !force {
		if cached, err := s.store.LoadPeaks(videoID, source); err == nil {
			cacheHit = true
			peaksData = toFloatSlice(cached["peaks"])
			duration = durationFloat(cached["duration"])
			audioPath, _ = s.findCachedAudio(s.cfg.CacheDir, videoID)
			if v, ok := cached["sourceFile"].(string); ok && v != "" {
				audioPath = v
			}
			if v := intValue(cached["samples"], 0); v > 0 {
				samples = int(v)
			}
		}
	}

	if !cacheHit {
		var ok bool
		audioPath, ok = s.findCachedAudio(s.cfg.CacheDir, videoID)
		if !ok {
			writeError(w, http.StatusNotFound, "audio_not_found", "audio not cached")
			return
		}
		meta, _ := s.store.LoadMetadata(videoID)
		duration = durationFloat(meta["duration"])
		var err error
		peaksData, err = s.computePeaks(r.Context(), audioPath, samples)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "peaks_compute_failed", err.Error())
			return
		}
		payload := map[string]any{
			"videoId":     videoID,
			"source":      source,
			"duration":    duration,
			"samples":     samples,
			"peaks":       peaksData,
			"sourceFile":  filepath.Base(audioPath),
			"generatedAt": UTCNowISO(),
		}
		if err := s.store.SavePeaks(videoID, source, payload); err != nil {
			slog.Default().Error("failed to save peaks", "error", err)
		}
	}

	writeJSON(w, http.StatusOK, backend.PeaksResponse{
		VideoID:     videoID,
		Samples:     samples,
		Duration:    duration,
		Peaks:       peaksData,
		SourceFile:  filepath.Base(audioPath),
		GeneratedAt: UTCNowISO(),
		Source:      backend.Source(source),
		CacheHit:    cacheHit,
	})
}

func (s *Server) cacheAudio(w http.ResponseWriter, r *http.Request) {
	videoID := NormalizeVideoID(chi.URLParam(r, "id"))
	source := r.URL.Query().Get("source")
	if source == "" {
		source = "original"
	}
	if source == "vocal" {
		writeError(w, http.StatusNotFound, "not_found", "vocal source not available")
		return
	}
	if source != "original" {
		writeError(w, http.StatusBadRequest, "invalid_source", "source must be original or vocal")
		return
	}
	path, ok := s.findCachedAudio(s.cfg.CacheDir, videoID)
	if !ok {
		writeError(w, http.StatusNotFound, "audio_not_found", "audio not found")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=86400, stale-while-revalidate=3600")
	serveAudioFile(w, r, path, audioContentType(path))
}

func (s *Server) cachePeaks(w http.ResponseWriter, r *http.Request) {
	videoID := NormalizeVideoID(chi.URLParam(r, "id"))
	payload, err := s.store.LoadPeaks(videoID, "original")
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "peaks not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "cache_read_failed", err.Error())
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=300")
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) cacheTranscript(w http.ResponseWriter, r *http.Request) {
	videoID := NormalizeVideoID(chi.URLParam(r, "id"))
	payload, err := s.store.LoadTranscript(videoID)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "transcript not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "cache_read_failed", err.Error())
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=600, stale-while-revalidate=60")
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.drafts.ListProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "draft_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) saveProject(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if err := s.drafts.SaveRaw(chi.URLParam(r, "id"), body); err != nil {
		writeDraftError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) loadProject(w http.ResponseWriter, r *http.Request) {
	body, err := s.drafts.LoadRaw(chi.URLParam(r, "id"))
	if err != nil {
		writeDraftError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	if err := s.drafts.Delete(chi.URLParam(r, "id")); err != nil {
		writeDraftError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) requestChallengeHandler(w http.ResponseWriter, r *http.Request) {
	if s.requestChallenge == nil {
		writeError(w, http.StatusNotFound, "not_configured", "lrclib proxy not configured")
		return
	}
	body, header, status, err := s.requestChallenge(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "proxy_failed", err.Error())
		return
	}
	defer body.Close()
	copyHeaders(w.Header(), header)
	w.WriteHeader(status)
	_, _ = io.Copy(w, body)
}

func (s *Server) publishHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Publish-Token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "missing_token", "X-Publish-Token header required")
		return
	}
	if s.publish == nil {
		writeError(w, http.StatusNotFound, "not_configured", "lrclib proxy not configured")
		return
	}
	body, header, status, err := s.publish(r.Context(), token, r.Body)
	if err != nil {
		writeError(w, http.StatusBadGateway, "proxy_failed", err.Error())
		return
	}
	defer body.Close()
	copyHeaders(w.Header(), header)
	w.WriteHeader(status)
	_, _ = io.Copy(w, body)
}

// helpers

func draftDir(cfg *Config) string {
	if cfg.DraftDir != "" {
		return cfg.DraftDir
	}
	return filepath.Join(cfg.CacheDir, "drafts")
}

func writeDraftError(w http.ResponseWriter, err error) {
	switch {
	case drafts.IsInvalidID(err):
		writeError(w, http.StatusBadRequest, "invalid_project_id", err.Error())
	case drafts.IsInvalidInput(err):
		writeError(w, http.StatusBadRequest, "invalid_draft", err.Error())
	case drafts.IsNotFound(err):
		writeError(w, http.StatusNotFound, "draft_not_found", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "draft_error", err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, detail string) {
	writeJSON(w, status, map[string]string{"error": code, "detail": detail})
}

func writeSSE(w io.Writer, f http.Flusher, ev any) {
	data, _ := json.Marshal(ev)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	f.Flush()
}

func validatedMode(mode string) string {
	if strings.ToLower(mode) == "karaoke" {
		return "karaoke"
	}
	return "normal"
}

func isURL(s string) bool { return strings.Contains(s, "://") }

func extractYouTubeVideoID(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	switch {
	case strings.Contains(host, "youtu.be"):
		return strings.Split(strings.TrimPrefix(u.Path, "/"), "/")[0]
	case strings.Contains(host, "youtube.com"):
		if v := u.Query().Get("v"); v != "" {
			return v
		}
		for _, prefix := range []string{"/embed/", "/v/"} {
			if strings.HasPrefix(u.Path, prefix) {
				return strings.TrimPrefix(u.Path, prefix)
			}
		}
	}
	return ""
}

func firstString(vals ...any) string {
	for _, v := range vals {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func boolValue(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func durationFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}

func intValue(v any, def int64) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	}
	return def
}

func toFloatSlice(v any) []float64 {
	if arr, ok := v.([]float64); ok {
		return arr
	}
	if arr, ok := v.([]any); ok {
		out := make([]float64, len(arr))
		for i, e := range arr {
			out[i] = durationFloat(e)
		}
		return out
	}
	return nil
}

func audioContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".m4a":
		return "audio/mp4"
	case ".wav":
		return "audio/wav"
	default:
		return "audio/mpeg"
	}
}

func serveAudioFile(w http.ResponseWriter, r *http.Request, path, contentType string) {
	fi, err := os.Stat(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "audio_not_found", "audio not found")
		return
	}
	size := fi.Size()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, peaks.IterFileRange(path, 0, size-1))
		return
	}
	start, end, err := peaks.ParseRangeHeader(rangeHeader, size)
	if err != nil {
		w.Header().Set("Content-Range", "bytes */"+strconv.FormatInt(size, 10))
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}
	length := end - start + 1
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
	w.WriteHeader(http.StatusPartialContent)
	_, _ = io.Copy(w, peaks.IterFileRange(path, start, end))
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func completedEventFromCache(videoID string, m map[string]any) backend.TranscriptionCompletedEvent {
	return backend.TranscriptionCompletedEvent{
		Status:      backend.TranscriptionCompleted,
		VideoID:     videoID,
		Provider:    stringValue(m["provider"]),
		Language:    stringValue(m["language"]),
		Plain:       stringValue(m["plain"]),
		Synced:      stringValue(m["synced"]),
		IsAIRefined: boolValue(m["is_ai_refined"]),
		Model:       stringValue(m["model"]),
		Mode:        stringValue(m["mode"]),
		UpdatedAt:   stringValue(m["updatedAt"]),
	}
}
