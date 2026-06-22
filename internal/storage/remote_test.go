package storage

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
)

func TestRemoteStoreRoundTrip(t *testing.T) {
	t.Parallel()

	projects := make(map[string]storedSnapshot)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			id := r.URL.Path[len("/local-api/projects/"):]
			var stored storedSnapshot
			if err := json.NewDecoder(r.Body).Decode(&stored); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			projects[id] = stored
			w.WriteHeader(http.StatusOK)

		case http.MethodGet:
			if r.URL.Path == "/local-api/projects" {
				var list []storedProjectSummary
				for id, stored := range projects {
					list = append(list, storedProjectSummary{
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

	client := backend.NewClientWithHTTPClient(server.URL, server.Client())
	store := NewRemoteStore(client)
	snapshot := newTestSnapshot(t, "dQw4w9WgXcQ")

	if err := store.Save(snapshot); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	loaded, err := store.Load(snapshot.ProjectID)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	assertRemoteSnapshotsEqual(t, snapshot, loaded)

	summaries, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v, want nil", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %d, want 1", len(summaries))
	}
	if summaries[0].ID != snapshot.ProjectID {
		t.Fatalf("summary.ID = %q, want %q", summaries[0].ID, snapshot.ProjectID)
	}

	if err := store.Delete(snapshot.ProjectID); err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}

	_, err = store.Load(snapshot.ProjectID)
	if err == nil {
		t.Fatalf("Load() after Delete error = nil, want not found")
	}
	var storageErr *StorageError
	if !errors.As(err, &storageErr) {
		t.Fatalf("error type = %T, want *StorageError", err)
	}
	if storageErr.Code != CodeDraftNotFound {
		t.Fatalf("Code = %q, want %q", storageErr.Code, CodeDraftNotFound)
	}
}

func TestRemoteStoreNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := backend.NewClientWithHTTPClient(server.URL, server.Client())
	store := NewRemoteStore(client)

	_, err := store.Load(draft.ProjectID("missing"))
	if err == nil {
		t.Fatalf("Load() error = nil, want not found")
	}
	var storageErr *StorageError
	if !errors.As(err, &storageErr) {
		t.Fatalf("error type = %T, want *StorageError", err)
	}
	if storageErr.Code != CodeDraftNotFound {
		t.Fatalf("Code = %q, want %q", storageErr.Code, CodeDraftNotFound)
	}
}

func assertRemoteSnapshotsEqual(t *testing.T, want, got draft.Snapshot) {
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
