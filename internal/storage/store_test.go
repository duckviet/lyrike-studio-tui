package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

func TestDraft_SaveAndLoad_roundTrip(t *testing.T) {
	t.Parallel()

	store, dir := newTestStore(t)
	snapshot := newTestSnapshot(t, "dQw4w9WgXcQ")

	if err := store.Save(snapshot); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	got, err := store.Load(snapshot.ProjectID)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if got.ID != snapshot.ID {
		t.Fatalf("ID = %q, want %q", got.ID, snapshot.ID)
	}
	if got.Metadata.VideoID != snapshot.Metadata.VideoID {
		t.Fatalf("Metadata.VideoID = %q, want %q", got.Metadata.VideoID, snapshot.Metadata.VideoID)
	}
	if got.Metadata.Duration != snapshot.Metadata.Duration {
		t.Fatalf("Metadata.Duration = %d, want %d", got.Metadata.Duration, snapshot.Metadata.Duration)
	}
	if !got.Metadata.UpdatedAt.Equal(snapshot.Metadata.UpdatedAt) {
		t.Fatalf("Metadata.UpdatedAt = %v, want %v", got.Metadata.UpdatedAt, snapshot.Metadata.UpdatedAt)
	}

	wantLines := snapshot.Document.Lines()
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

	// Verify the file is written inside the injected directory.
	expectedPath := filepath.Join(dir, "dQw4w9WgXcQ.json")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected draft file %s: %v", expectedPath, err)
	}
}

func TestDraft_Load_whenDraftMissing(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	id, err := draft.NewDraftID("missing")
	if err != nil {
		t.Fatalf("NewDraftID() error = %v", err)
	}

	_, err = store.Load(draft.ProjectID(id.String()))
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

func TestDraft_Load_whenCorrupt(t *testing.T) {
	t.Parallel()

	store, dir := newTestStore(t)
	id, err := draft.NewDraftID("corrupt")
	if err != nil {
		t.Fatalf("NewDraftID() error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte("not json"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err = store.Load(draft.ProjectID(id.String()))
	if err == nil {
		t.Fatalf("Load() error = nil, want corrupt draft error")
	}

	var storageErr *StorageError
	if !errors.As(err, &storageErr) {
		t.Fatalf("error type = %T, want *StorageError", err)
	}
	if storageErr.Code != CodeCorruptDraft {
		t.Fatalf("Code = %q, want %q", storageErr.Code, CodeCorruptDraft)
	}
}

func TestDraft_Save_ignoresStaleTempFile(t *testing.T) {
	t.Parallel()

	store, dir := newTestStore(t)
	id, err := draft.NewDraftID("recover")
	if err != nil {
		t.Fatalf("NewDraftID() error = %v", err)
	}

	// Leave a stale partial write that should not be treated as the draft.
	if err := os.WriteFile(filepath.Join(dir, "recover.json.tmp"), []byte("partial"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	snapshot := newTestSnapshot(t, "recover")
	if err := store.Save(snapshot); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load must read the new atomic .json, not the stale .tmp.
	got, err := store.Load(draft.ProjectID(id.String()))
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if got.ID != id {
		t.Fatalf("ID = %q, want %q", got.ID, id)
	}
}

func TestDraftStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	store := NewDefaultStore()
	snapshot := newTestSnapshot(t, "xdg-video")

	if err := store.Save(snapshot); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load(snapshot.ProjectID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.ID != snapshot.ID {
		t.Fatalf("ID = %q, want %q", got.ID, snapshot.ID)
	}

	expectedDir := filepath.Join(dir, "lyrike-studio-tui", "drafts")
	if _, err := os.Stat(filepath.Join(expectedDir, "xdg-video.json")); err != nil {
		t.Fatalf("expected draft file in XDG dir: %v", err)
	}
}

func TestDraft_List(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)

	for _, videoID := range []string{"aaa", "bbb"} {
		snapshot := newTestSnapshot(t, videoID)
		if err := store.Save(snapshot); err != nil {
			t.Fatalf("Save(%q) error = %v", videoID, err)
		}
	}

	ids, err := store.ListProjects()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("len(ids) = %d, want 2", len(ids))
	}
}

func TestDraft_Delete(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	snapshot := newTestSnapshot(t, "delete-me")
	if err := store.Save(snapshot); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := store.Delete(snapshot.ProjectID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := store.Load(snapshot.ProjectID)
	if err == nil {
		t.Fatalf("Load() after Delete error = nil, want not found")
	}
}

func newTestStore(t *testing.T) (*FileStore, string) {
	t.Helper()
	dir := t.TempDir()
	return NewFileStore(dir), dir
}

func newTestSnapshot(t *testing.T, videoID string) draft.Snapshot {
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
