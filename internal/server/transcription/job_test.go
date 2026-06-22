package transcription

import (
	"encoding/json"
	"errors"
	"runtime"
	"testing"

	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
	"github.com/duckviet/lyrike-studio-tui/internal/server/cache"
)

type fakeProvider struct {
	result TranscriptionResult
	err    error
}

func (f *fakeProvider) Transcribe(audioPath string) (TranscriptionResult, error) {
	return f.result, f.err
}

type blockingProvider struct {
	startCh chan struct{}
	doneCh  chan struct{}
	result  TranscriptionResult
}

func (b *blockingProvider) Transcribe(audioPath string) (TranscriptionResult, error) {
	close(b.startCh)
	<-b.doneCh
	return b.result, nil
}

type fakeRefiner struct {
	result RefineResult
	err    error
}

func (f *fakeRefiner) RefineLyrics(synced, plain, trackName, artistName string, duration float64) (RefineResult, error) {
	return f.result, f.err
}

func sampleResult() TranscriptionResult {
	return TranscriptionResult{
		Provider: "openai-whisper-1",
		Language: "en",
		Segments: []TranscribedSegment{
			{
				Text:  "Hello world",
				Start: 0.0,
				End:   2.0,
				Words: []TranscribedWord{
					{Word: "Hello", Start: 0.0, End: 1.0},
					{Word: "world", Start: 1.0, End: 2.0},
				},
			},
		},
		PlainText: "Hello world",
	}
}

func sampleMetadata() map[string]any {
	return map[string]any{
		"trackName":  "Test Track",
		"artistName": "Test Artist",
		"duration":   123.45,
	}
}

func mustDecodeEvent(t *testing.T, ev Event) backend.TranscribeResponse {
	t.Helper()
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	resp, err := backend.DecodeTranscribeResponse(data)
	if err != nil {
		t.Fatalf("decode event %s: %v", string(data), err)
	}
	return resp
}

func TestJobRunningCompleted(t *testing.T) {
	cases := []struct {
		name         string
		mode         string
		wantSynced   string
		wantPlain    string
		enableRefine string
		refiner      LyricRefiner
		wantRefined  bool
		wantModel    string
	}{
		{
			name:         "normal mode without refinement",
			mode:         "normal",
			wantSynced:   "[00:00.00] Hello world",
			wantPlain:    "Hello world",
			enableRefine: "false",
			refiner:      &fakeRefiner{},
			wantRefined:  false,
			wantModel:    "",
		},
		{
			name:         "karaoke mode keeps word timings",
			mode:         "karaoke",
			wantSynced:   "[00:00.00]<00:00.00>Hello <00:01.00>world",
			wantPlain:    "Hello world",
			enableRefine: "false",
			refiner:      &fakeRefiner{},
			wantRefined:  false,
			wantModel:    "",
		},
		{
			name:         "normal mode with refinement",
			mode:         "normal",
			wantSynced:   "refined synced",
			wantPlain:    "refined plain",
			enableRefine: "true",
			refiner: &fakeRefiner{
				result: RefineResult{
					SyncedLyrics: "refined synced",
					PlainLyrics:  "refined plain",
					IsAIRefined:  true,
					Model:        "gpt-4o-mini",
				},
			},
			wantRefined: true,
			wantModel:   "gpt-4o-mini",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := cache.NewStore(t.TempDir())
			videoID := "vid1"
			if err := store.SaveMetadata(videoID, sampleMetadata()); err != nil {
				t.Fatalf("save metadata: %v", err)
			}

			mgr := NewManager(store, &fakeProvider{result: sampleResult()}, tc.refiner)
			ch := mgr.Subscribe(videoID)
			defer mgr.Unsubscribe(videoID, ch)

			mgr.RunTranscriptionJob(videoID, "dummy.mp3", tc.enableRefine, tc.mode)

			running := <-ch
			if running.Status != "running" {
				t.Fatalf("first event status = %q, want running", running.Status)
			}
			resp := mustDecodeEvent(t, running)
			if resp.Status() != backend.TranscriptionRunning {
				t.Fatalf("decoded running status = %q, want running", resp.Status())
			}
			rn, ok := resp.AsRunning()
			if !ok {
				t.Fatalf("decoded event is not running")
			}
			if rn.Job != nil {
				t.Fatalf("running broadcast included job: %+v", rn.Job)
			}
			if running.StartedAt == "" || running.UpdatedAt == "" {
				t.Fatalf("running event missing timestamps: %+v", running)
			}

			completed := <-ch
			if completed.Status != "completed" {
				t.Fatalf("second event status = %q, want completed", completed.Status)
			}
			resp = mustDecodeEvent(t, completed)
			c, ok := resp.AsCompleted()
			if !ok {
				t.Fatalf("decoded event is not completed")
			}
			if c.VideoID != videoID {
				t.Errorf("completed videoId = %q, want %q", c.VideoID, videoID)
			}
			if c.Plain != tc.wantPlain {
				t.Errorf("plain = %q, want %q", c.Plain, tc.wantPlain)
			}
			if c.Synced != tc.wantSynced {
				t.Errorf("synced = %q, want %q", c.Synced, tc.wantSynced)
			}
			if c.Mode != tc.mode {
				t.Errorf("mode = %q, want %q", c.Mode, tc.mode)
			}
			if c.IsAIRefined != tc.wantRefined {
				t.Errorf("is_ai_refined = %v, want %v", c.IsAIRefined, tc.wantRefined)
			}
			if c.Model != tc.wantModel {
				t.Errorf("model = %q, want %q", c.Model, tc.wantModel)
			}
			if c.Provider != "openai-whisper-1" {
				t.Errorf("provider = %q, want openai-whisper-1", c.Provider)
			}
			if c.Language != "en" {
				t.Errorf("language = %q, want en", c.Language)
			}

			cached, err := store.LoadTranscript(videoID)
			if err != nil {
				t.Fatalf("load transcript: %v", err)
			}
			if cached["status"] != "completed" {
				t.Errorf("cached status = %v, want completed", cached["status"])
			}
			if cached["plain"] != tc.wantPlain {
				t.Errorf("cached plain = %v, want %q", cached["plain"], tc.wantPlain)
			}
			if cached["synced"] != tc.wantSynced {
				t.Errorf("cached synced = %v, want %q", cached["synced"], tc.wantSynced)
			}
			if cached["is_ai_refined"] != tc.wantRefined {
				t.Errorf("cached is_ai_refined = %v, want %v", cached["is_ai_refined"], tc.wantRefined)
			}
			if cached["mode"] != tc.mode {
				t.Errorf("cached mode = %v, want %q", cached["mode"], tc.mode)
			}
		})
	}
}

func TestJobFailed(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{
			name: "transcription error",
			err:  errors.New("openai transcription failed"),
		},
		{
			name: "file not found",
			err:  errors.New("open audio file: no such file"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := cache.NewStore(t.TempDir())
			videoID := "vid-fail"
			mgr := NewManager(store, &fakeProvider{err: tc.err}, &fakeRefiner{})
			ch := mgr.Subscribe(videoID)
			defer mgr.Unsubscribe(videoID, ch)

			mgr.RunTranscriptionJob(videoID, "dummy.mp3", "false", "normal")

			running := <-ch
			if running.Status != "running" {
				t.Fatalf("first event status = %q, want running", running.Status)
			}
			mustDecodeEvent(t, running)

			failed := <-ch
			if failed.Status != "failed" {
				t.Fatalf("second event status = %q, want failed", failed.Status)
			}
			resp := mustDecodeEvent(t, failed)
			f, ok := resp.AsFailed()
			if !ok {
				t.Fatalf("decoded event is not failed")
			}
			if f.Error == "" {
				t.Fatalf("failed event missing error")
			}
			if f.VideoID != videoID {
				t.Errorf("videoId = %q, want %q", f.VideoID, videoID)
			}
		})
	}
}

func TestJobDedupRunning(t *testing.T) {
	store := cache.NewStore(t.TempDir())
	videoID := "vid-dedup"
	provider := &blockingProvider{
		startCh: make(chan struct{}),
		doneCh:  make(chan struct{}),
		result:  sampleResult(),
	}
	mgr := NewManager(store, provider, &fakeRefiner{})
	ch := mgr.Subscribe(videoID)
	defer mgr.Unsubscribe(videoID, ch)

	before := runtime.NumGoroutine()

	job1 := mgr.RunTranscriptionJob(videoID, "dummy.mp3", "false", "normal")
	<-provider.startCh

	job2 := mgr.RunTranscriptionJob(videoID, "dummy.mp3", "false", "normal")
	if job1 != job2 {
		t.Fatalf("expected same job instance, got different: %p vs %p", job1, job2)
	}

	close(provider.doneCh)

	running := <-ch
	if running.Status != "running" {
		t.Fatalf("first event status = %q, want running", running.Status)
	}
	completed := <-ch
	if completed.Status != "completed" {
		t.Fatalf("second event status = %q, want completed", completed.Status)
	}

	runtime.GC()
	after := runtime.NumGoroutine()
	if after > before {
		t.Fatalf("possible goroutine leak: before=%d after=%d", before, after)
	}
}

func TestJobRefinementFailure(t *testing.T) {
	store := cache.NewStore(t.TempDir())
	videoID := "vid-refine-fail"
	if err := store.SaveMetadata(videoID, sampleMetadata()); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	refiner := &fakeRefiner{err: errors.New("refine failed")}
	mgr := NewManager(store, &fakeProvider{result: sampleResult()}, refiner)
	ch := mgr.Subscribe(videoID)
	defer mgr.Unsubscribe(videoID, ch)

	mgr.RunTranscriptionJob(videoID, "dummy.mp3", "true", "normal")

	running := <-ch
	if running.Status != "running" {
		t.Fatalf("first event status = %q, want running", running.Status)
	}

	completed := <-ch
	if completed.Status != "completed" {
		t.Fatalf("second event status = %q, want completed", completed.Status)
	}
	if completed.IsAIRefined {
		t.Fatalf("expected is_ai_refined=false on refinement failure")
	}
	if completed.Model != "" {
		t.Fatalf("expected empty model on refinement failure, got %q", completed.Model)
	}

	resp := mustDecodeEvent(t, completed)
	c, ok := resp.AsCompleted()
	if !ok {
		t.Fatalf("decoded event is not completed")
	}
	if c.IsAIRefined {
		t.Errorf("decoded is_ai_refined = true, want false")
	}
	if c.Model != "" {
		t.Errorf("decoded model = %q, want empty", c.Model)
	}
}

func TestJobSubscribeUnsubscribe(t *testing.T) {
	store := cache.NewStore(t.TempDir())
	mgr := NewManager(store, &fakeProvider{}, &fakeRefiner{})
	videoID := "vid-sub"

	before := runtime.NumGoroutine()

	ch := mgr.Subscribe(videoID)
	mgr.Unsubscribe(videoID, ch)

	_, open := <-ch
	if open {
		t.Fatalf("channel was not closed after unsubscribe")
	}

	runtime.GC()
	after := runtime.NumGoroutine()
	if after > before {
		t.Fatalf("possible goroutine leak: before=%d after=%d", before, after)
	}
}

func TestJobCurrentStateReplay(t *testing.T) {
	store := cache.NewStore(t.TempDir())
	videoID := "vid-replay"
	mgr := NewManager(store, &fakeProvider{result: sampleResult()}, &fakeRefiner{})
	ch := mgr.Subscribe(videoID)
	defer mgr.Unsubscribe(videoID, ch)

	mgr.RunTranscriptionJob(videoID, "dummy.mp3", "false", "normal")

	<-ch // running
	<-ch // completed

	ev, ok := mgr.CurrentState(videoID)
	if !ok {
		t.Fatalf("CurrentState returned false, want true")
	}
	if ev.Status != "completed" {
		t.Fatalf("CurrentState status = %q, want completed", ev.Status)
	}
	if ev.VideoID != videoID {
		t.Errorf("CurrentState videoId = %q, want %q", ev.VideoID, videoID)
	}
	if ev.Provider != "openai-whisper-1" {
		t.Errorf("CurrentState provider = %q, want openai-whisper-1", ev.Provider)
	}
	if ev.Synced == "" || ev.Plain == "" {
		t.Fatalf("CurrentState missing synced/plain: %+v", ev)
	}

	resp := mustDecodeEvent(t, ev)
	if _, ok := resp.AsCompleted(); !ok {
		t.Fatalf("CurrentState event did not decode as completed")
	}
}
