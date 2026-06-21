package storage

import (
	"testing"
	"time"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
)

func TestProjectStore_loadPopulatesProjectID(t *testing.T) {
	store, _ := newTestStore(t)
	snapshot := newTestSnapshot(t, "project-a")

	if err := store.Save(snapshot); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	got, err := store.Load(snapshot.ProjectID)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if got.ProjectID != snapshot.ProjectID {
		t.Fatalf("Load() ProjectID = %q, want %q", got.ProjectID, snapshot.ProjectID)
	}
	if got.ID.String() != snapshot.ProjectID.String() {
		t.Fatalf("Load() legacy ID = %q, want %q", got.ID, snapshot.ProjectID)
	}
}

func TestProjectStore_listProjectsSortsByUpdatedAtThenID(t *testing.T) {
	store, _ := newTestStore(t)
	first := newTestSnapshot(t, "project-a")
	first.Metadata.UpdatedAt = time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	second := newTestSnapshot(t, "project-b")
	second.Metadata.UpdatedAt = time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC)
	third := newTestSnapshot(t, "project-c")
	third.Metadata.UpdatedAt = first.Metadata.UpdatedAt

	for _, snapshot := range []draft.Snapshot{first, second, third} {
		if err := store.Save(snapshot); err != nil {
			t.Fatalf("Save(%q) error = %v, want nil", snapshot.ProjectID, err)
		}
	}

	got, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v, want nil", err)
	}
	want := []draft.ProjectID{second.ProjectID, first.ProjectID, third.ProjectID}
	if len(got) != len(want) {
		t.Fatalf("ListProjects() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("ListProjects()[%d] = %q, want %q", i, got[i].ID, want[i])
		}
	}
}
