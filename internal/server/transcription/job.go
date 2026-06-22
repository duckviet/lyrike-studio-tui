package transcription

import (
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/duckviet/lyrike-studio-tui/internal/server/cache"
)

// Provider transcribes an audio file into a TranscriptionResult.
type Provider interface {
	Transcribe(audioPath string) (TranscriptionResult, error)
}

// LyricRefiner improves synced and plain lyrics with an AI model.
type LyricRefiner interface {
	RefineLyrics(synced, plain, trackName, artistName string, duration float64) (RefineResult, error)
}

// Job holds the mutable state of a single transcription job.
type Job struct {
	VideoID     string
	Status      string
	StartedAt   string
	UpdatedAt   string
	Error       string
	Provider    string
	Language    string
	Plain       string
	Synced      string
	IsAIRefined bool
	Model       string
	Mode        string
}

// Event is the JSON payload broadcast to SSE subscribers and returned
// by CurrentState for replay-on-connect. Its shape mirrors the TUI
// contract in internal/integrations/backend/types.go.
type Event struct {
	Status      string `json:"status"`
	VideoID     string `json:"videoId"`
	Provider    string `json:"provider,omitempty"`
	Language    string `json:"language,omitempty"`
	Plain       string `json:"plain,omitempty"`
	Synced      string `json:"synced,omitempty"`
	IsAIRefined bool   `json:"is_ai_refined"`
	Model       string `json:"model,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Error       string `json:"error,omitempty"`
	StartedAt   string `json:"startedAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

// Manager coordinates asynchronous transcription jobs and broadcasts
// status events to per-video SSE subscribers.
type Manager struct {
	store    *cache.Store
	provider Provider
	refiner  LyricRefiner

	mu   sync.Mutex
	jobs map[string]*Job
	subs map[string]map[chan Event]struct{}
}

// NewManager creates a Manager rooted at the given cache store.
func NewManager(store *cache.Store, provider Provider, refiner LyricRefiner) *Manager {
	return &Manager{
		store:    store,
		provider: provider,
		refiner:  refiner,
		jobs:     make(map[string]*Job),
		subs:     make(map[string]map[chan Event]struct{}),
	}
}

// RunTranscriptionJob starts a transcription job for videoID unless one
// is already running. It returns the existing Job in that case.
func (m *Manager) RunTranscriptionJob(videoID, audioPath, enableRefinement, mode string) *Job {
	enable, _ := strconv.ParseBool(enableRefinement)

	m.mu.Lock()
	if job, ok := m.jobs[videoID]; ok && job.Status == "running" {
		m.mu.Unlock()
		return job
	}

	now := time.Now().UTC().Format(time.RFC3339)
	job := &Job{
		VideoID:   videoID,
		Status:    "running",
		StartedAt: now,
		UpdatedAt: now,
		Mode:      mode,
	}
	m.jobs[videoID] = job
	m.mu.Unlock()

	m.broadcast(Event{
		Status:    "running",
		VideoID:   videoID,
		StartedAt: now,
		UpdatedAt: now,
	})

	go m.run(job, audioPath, enable, mode)
	return job
}

func (m *Manager) run(job *Job, audioPath string, enableRefinement bool, mode string) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := m.provider.Transcribe(audioPath)
	if err != nil {
		m.mu.Lock()
		job.Status = "failed"
		job.Error = err.Error()
		job.UpdatedAt = now
		m.mu.Unlock()
		m.broadcast(jobToEvent(job))
		return
	}

	var input TranscriptionResult
	if mode == "karaoke" {
		input = result
	} else {
		input = StripWords(result)
	}
	synced, plain := BuildSyncedLyrics(input)

	if enableRefinement && m.refiner != nil {
		trackName, artistName, duration := m.metadataFor(job.VideoID)
		refined, refineErr := m.refiner.RefineLyrics(synced, plain, trackName, artistName, duration)
		if refineErr != nil {
			slog.Default().Info("transcription refinement failed, using unrefined lyrics", "error", refineErr)
		} else {
			synced = refined.SyncedLyrics
			plain = refined.PlainLyrics
			job.IsAIRefined = refined.IsAIRefined
			job.Model = refined.Model
		}
	}

	now = time.Now().UTC().Format(time.RFC3339)
	m.mu.Lock()
	job.Provider = result.Provider
	job.Language = result.Language
	job.Plain = plain
	job.Synced = synced
	job.Mode = mode
	job.Status = "completed"
	job.UpdatedAt = now
	m.mu.Unlock()

	payload := map[string]any{
		"videoId":       job.VideoID,
		"status":        "completed",
		"provider":      result.Provider,
		"language":      result.Language,
		"plain":         plain,
		"synced":        synced,
		"is_ai_refined": job.IsAIRefined,
		"model":         job.Model,
		"mode":          mode,
		"updatedAt":     now,
	}
	if err := m.store.SaveTranscript(job.VideoID, payload); err != nil {
		slog.Default().Error("failed to save transcript", "error", err)
	}

	m.broadcast(jobToEvent(job))
}

func (m *Manager) metadataFor(videoID string) (trackName, artistName string, duration float64) {
	meta, err := m.store.LoadMetadata(videoID)
	if err != nil {
		return "", "", 0
	}
	if s, ok := meta["trackName"].(string); ok {
		trackName = s
	}
	if s, ok := meta["artistName"].(string); ok {
		artistName = s
	}
	switch v := meta["duration"].(type) {
	case float64:
		duration = v
	case int:
		duration = float64(v)
	case int64:
		duration = float64(v)
	}
	return
}

// Subscribe registers a new subscriber for videoID and returns a
// buffered channel that receives broadcast events.
func (m *Manager) Subscribe(videoID string) chan Event {
	ch := make(chan Event, 16)
	m.mu.Lock()
	if m.subs[videoID] == nil {
		m.subs[videoID] = make(map[chan Event]struct{})
	}
	m.subs[videoID][ch] = struct{}{}
	m.mu.Unlock()
	return ch
}

func (m *Manager) Unsubscribe(videoID string, ch chan Event) {
	m.mu.Lock()
	if subs, ok := m.subs[videoID]; ok {
		if _, ok := subs[ch]; ok {
			delete(subs, ch)
			close(ch)
		}
		if len(subs) == 0 {
			delete(m.subs, videoID)
		}
	}
	m.mu.Unlock()
}

// CurrentState returns the latest event for videoID for replay-on-connect.
func (m *Manager) CurrentState(videoID string) (Event, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[videoID]
	if !ok {
		return Event{}, false
	}
	return jobToEvent(job), true
}

func (m *Manager) broadcast(ev Event) {
	m.mu.Lock()
	for ch := range m.subs[ev.VideoID] {
		select {
		case ch <- ev:
		default:
		}
	}
	m.mu.Unlock()
}

func jobToEvent(job *Job) Event {
	switch job.Status {
	case "running":
		return Event{
			Status:    "running",
			VideoID:   job.VideoID,
			StartedAt: job.StartedAt,
			UpdatedAt: job.UpdatedAt,
		}
	case "completed":
		return Event{
			Status:      "completed",
			VideoID:     job.VideoID,
			Provider:    job.Provider,
			Language:    job.Language,
			Plain:       job.Plain,
			Synced:      job.Synced,
			IsAIRefined: job.IsAIRefined,
			Model:       job.Model,
			Mode:        job.Mode,
			UpdatedAt:   job.UpdatedAt,
		}
	case "failed":
		return Event{
			Status:    "failed",
			VideoID:   job.VideoID,
			Error:     job.Error,
			UpdatedAt: job.UpdatedAt,
		}
	default:
		return Event{
			Status:  job.Status,
			VideoID: job.VideoID,
		}
	}
}
