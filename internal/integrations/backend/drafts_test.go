package backend

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

func TestDraftClientRoundTrip(t *testing.T) {
	t.Parallel()

	projects := make(map[string]draftSnapshot)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			id := r.URL.Path[len("/local-api/projects/"):]
			var stored draftSnapshot
			if err := json.NewDecoder(r.Body).Decode(&stored); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			projects[id] = stored
			w.WriteHeader(http.StatusOK)

		case http.MethodGet:
			if r.URL.Path == "/local-api/projects" {
				var list []draftProjectSummary
				for id, stored := range projects {
					list = append(list, draftProjectSummary{
						ID:       id,
						Metadata: stored.Metadata,
					})
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(list)
				return
			}
			id := r.URL.Path[len("/local-api/projects/"):]
			stored, ok := projects[id]
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(stored)

		case http.MethodDelete:
			id := r.URL.Path[len("/local-api/projects/"):]
			delete(projects, id)
			w.WriteHeader(http.StatusOK)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	snapshot := newBackendTestSnapshot(t, "dQw4w9WgXcQ")

	if err := client.SaveDraft(ctx, "dQw4w9WgXcQ", snapshot); err != nil {
		t.Fatalf("SaveDraft() error = %v, want nil", err)
	}

	loaded, err := client.LoadDraft(ctx, "dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("LoadDraft() error = %v, want nil", err)
	}
	assertSnapshotsEqual(t, snapshot, loaded)

	summaries, err := client.ListDrafts(ctx)
	if err != nil {
		t.Fatalf("ListDrafts() error = %v, want nil", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %d, want 1", len(summaries))
	}
	if summaries[0].ID.String() != "dQw4w9WgXcQ" {
		t.Fatalf("summary.ID = %q, want dQw4w9WgXcQ", summaries[0].ID)
	}

	if err := client.DeleteDraft(ctx, "dQw4w9WgXcQ"); err != nil {
		t.Fatalf("DeleteDraft() error = %v, want nil", err)
	}

	_, err = client.LoadDraft(ctx, "dQw4w9WgXcQ")
	if !errors.Is(err, ErrDraftNotFound) {
		t.Fatalf("LoadDraft() after Delete error = %v, want ErrDraftNotFound", err)
	}
}

func TestDraftClientNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.LoadDraft(context.Background(), "missing")
	if !errors.Is(err, ErrDraftNotFound) {
		t.Fatalf("LoadDraft() error = %v, want ErrDraftNotFound", err)
	}
}

func newBackendTestSnapshot(t *testing.T, videoID string) draft.Snapshot {
	t.Helper()

	id, err := draft.NewDraftID(videoID)
	if err != nil {
		t.Fatalf("NewDraftID(%q) error = %v", videoID, err)
	}

	text1, err := lyrics.NewText("First line")
	if err != nil {
		t.Fatalf("NewText() error = %v", err)
	}
	ts1, err := lyrics.NewTimestamp(0)
	if err != nil {
		t.Fatalf("NewTimestamp() error = %v", err)
	}
	te1, err := lyrics.NewTimestamp(10_000)
	if err != nil {
		t.Fatalf("NewTimestamp() error = %v", err)
	}
	line1, err := lyrics.NewLine(ts1, te1, text1)
	if err != nil {
		t.Fatalf("NewLine() error = %v", err)
	}

	text2, err := lyrics.NewText("Second line")
	if err != nil {
		t.Fatalf("NewText() error = %v", err)
	}
	ts2, err := lyrics.NewTimestamp(12_340)
	if err != nil {
		t.Fatalf("NewTimestamp() error = %v", err)
	}
	te2, err := lyrics.NewTimestamp(20_000)
	if err != nil {
		t.Fatalf("NewTimestamp() error = %v", err)
	}
	line2, err := lyrics.NewLine(ts2, te2, text2)
	if err != nil {
		t.Fatalf("NewLine() error = %v", err)
	}

	doc, err := lyrics.NewDocument([]lyrics.Line{line1, line2})
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	return draft.Snapshot{
		ProjectID: draft.ProjectID(id.String()),
		ID:        id,
		Metadata: draft.Metadata{
			VideoID:    videoID,
			TrackName:  "Test Track",
			ArtistName: "Test Artist",
			AlbumName:  "Test Album",
			Duration:   212,
			UpdatedAt:  time.Date(2025, 1, 15, 8, 30, 0, 0, time.UTC),
		},
		Document: doc,
	}
}

func assertSnapshotsEqual(t *testing.T, want, got draft.Snapshot) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.Metadata.VideoID != want.Metadata.VideoID {
		t.Fatalf("Metadata.VideoID = %q, want %q", got.Metadata.VideoID, want.Metadata.VideoID)
	}
	if got.Metadata.Duration != want.Metadata.Duration {
		t.Fatalf("Metadata.Duration = %d, want %d", got.Metadata.Duration, want.Metadata.Duration)
	}
	if !got.Metadata.UpdatedAt.Equal(want.Metadata.UpdatedAt) {
		t.Fatalf("Metadata.UpdatedAt = %v, want %v", got.Metadata.UpdatedAt, want.Metadata.UpdatedAt)
	}

	wantLines := want.Document.Lines()
	gotLines := got.Document.Lines()
	if len(gotLines) != len(wantLines) {
		t.Fatalf("len(Document.Lines()) = %d, want %d", len(gotLines), len(wantLines))
	}
	for i := range wantLines {
		if gotLines[i].Start().Milliseconds() != wantLines[i].Start().Milliseconds() {
			t.Fatalf("line[%d] start = %d, want %d", i, gotLines[i].Start().Milliseconds(), wantLines[i].Start().Milliseconds())
		}
		if gotLines[i].End().Milliseconds() != wantLines[i].End().Milliseconds() {
			t.Fatalf("line[%d] end = %d, want %d", i, gotLines[i].End().Milliseconds(), wantLines[i].End().Milliseconds())
		}
		if gotLines[i].Text().String() != wantLines[i].Text().String() {
			t.Fatalf("line[%d] text = %q, want %q", i, gotLines[i].Text().String(), wantLines[i].Text().String())
		}
	}
}
