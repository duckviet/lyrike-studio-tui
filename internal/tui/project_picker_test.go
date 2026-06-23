package tui

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
	"github.com/duckviet/lyrike-studio-tui/internal/storage"
)

type memoryDraftStore struct {
	saved    draft.Snapshot
	saves    int
	projects []draft.ProjectSummary
	loads    map[draft.ProjectID]draft.Snapshot
}

func (s *memoryDraftStore) Save(snapshot draft.Snapshot) error {
	s.saved = snapshot
	s.saves++
	return nil
}

func (s *memoryDraftStore) Load(id draft.ProjectID) (draft.Snapshot, error) {
	if snap, ok := s.loads[id]; ok {
		return snap, nil
	}
	return draft.Snapshot{}, &storage.StorageError{Code: storage.CodeDraftNotFound, Op: "load", Err: os.ErrNotExist}
}

func (s *memoryDraftStore) ListProjects() ([]draft.ProjectSummary, error) {
	return append([]draft.ProjectSummary(nil), s.projects...), nil
}

func (s *memoryDraftStore) Delete(id draft.ProjectID) error {
	delete(s.loads, id)
	return nil
}

func TestProjectSave_usesInjectedStoreAndProjectID(t *testing.T) {
	store := &memoryDraftStore{}
	projectID := draft.ProjectID("project-a")
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, projectID, "", "")

	next, _ := model.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	got := next.(Model)

	if store.saves != 1 {
		t.Fatalf("Save calls = %d, want 1", store.saves)
	}
	if store.saved.ProjectID != projectID {
		t.Fatalf("saved ProjectID = %q, want %q", store.saved.ProjectID, projectID)
	}
	if got.dirty {
		t.Fatalf("dirty = true, want false after save")
	}
}

func TestProjectSave_withoutProjectOpensFetchInput(t *testing.T) {
	store := &memoryDraftStore{}
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, "", "", "")

	next, _ := model.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	got := next.(Model)

	if !got.fetchInput.active() {
		t.Fatalf("fetchInput.active = false, want true")
	}
	if got.overlay == overlaySelector {
		t.Fatalf("overlay = selector, want none")
	}
	if store.saves != 0 {
		t.Fatalf("Save calls = %d, want 0", store.saves)
	}
}

func TestProjectPicker_ctrlLOpensProjectList(t *testing.T) {
	store := &memoryDraftStore{
		projects: []draft.ProjectSummary{{ID: "project-b"}, {ID: "project-a"}},
	}
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, "project-a", "", "")

	next, _ := model.Update(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl})
	got := next.(Model)

	if got.overlay != overlaySelector {
		t.Fatalf("overlay = %d, want selector (%d)", got.overlay, overlaySelector)
	}
	if len(got.picker.items) != 3 { // 1 virtual [New Project] + 2 projects
		t.Fatalf("projects len = %d, want 3", len(got.picker.items))
	}
}

func TestProjectPicker_loadRequiresConfirmationWhenDirty(t *testing.T) {
	projectA := draft.ProjectID("project-a")
	projectB := draft.ProjectID("project-b")
	loaded := draft.Snapshot{
		ProjectID: projectB,
		ID:        draft.DraftID(projectB.String()),
		Metadata:  draft.Metadata{TrackName: "Loaded Track", AlbumName: "Loaded Album"},
		Document:  mustDemoDocument(t),
	}
	store := &memoryDraftStore{
		projects: []draft.ProjectSummary{{ID: projectB}},
		loads:    map[draft.ProjectID]draft.Snapshot{projectB: loaded},
	}
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, projectA, "", "")
	model.dirty = true
	model = model.openProjectPicker()

	// Move cursor down to project-b (index 1, since index 0 is virtual [New Project])
	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model = next.(Model)

	next, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	confirm := next.(Model)
	if confirm.overlay != overlayConfirm {
		t.Fatalf("overlay = %v, want confirm", confirm.overlay)
	}
	if confirm.projectID != projectA {
		t.Fatalf("projectID = %q, want current project before confirm", confirm.projectID)
	}

	next, cmd := confirm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected non-nil command on confirm")
	}
	msg := cmd()
	next2, _ := next.Update(msg)
	got := next2.(Model)

	if got.projectID != projectB {
		t.Fatalf("projectID = %q, want %q", got.projectID, projectB)
	}
	if got.dirty {
		t.Fatalf("dirty = true, want false after load")
	}
}

func TestProjectPicker_loadReinitializesPlayerAndFetches(t *testing.T) {
	projectB := draft.ProjectID("project-b")
	loaded := draft.Snapshot{
		ProjectID: projectB,
		ID:        draft.DraftID(projectB.String()),
		Metadata:  draft.Metadata{VideoID: "vid-b", TrackName: "Loaded Track"},
		Document:  mustDemoDocument(t),
	}
	store := &memoryDraftStore{
		projects: []draft.ProjectSummary{{ID: projectB}},
		loads:    map[draft.ProjectID]draft.Snapshot{projectB: loaded},
	}
	client := backend.NewClient("http://example.com")

	factoryCalled := false
	factory := func(videoID string) (playback.Player, string) {
		factoryCalled = true
		return nil, "factory status"
	}

	model := NewModelWithDraftStore(mustDemoDocument(t), client, nil, store, "", "", "").
		WithPlayerFactory(factory)
	model = model.openProjectPicker()

	// Move cursor down to project-b (index 1)
	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model = next.(Model)

	next, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if got.projectID != projectB {
		t.Fatalf("projectID = %q, want %q", got.projectID, projectB)
	}
	if !factoryCalled {
		t.Fatalf("player factory was not called")
	}
	if cmd == nil {
		t.Fatalf("loadProject returned nil cmd, want fetch cmd")
	}
}

func TestProjectPickerNOpensFetchInput(t *testing.T) {
	store := &memoryDraftStore{
		projects: []draft.ProjectSummary{{ID: "project-a"}},
	}
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, "", "", "").openProjectPicker()

	// Ctrl-N keypress inside selector
	next, _ := model.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	got := next.(Model)

	if !got.fetchInput.active() {
		t.Fatalf("fetchInput.active = false, want true")
	}
	if got.overlay == overlaySelector {
		t.Fatalf("overlay = selector, want none")
	}
}

func TestProjectPickerNoProjectsShowsNewFromURL(t *testing.T) {
	store := &memoryDraftStore{}
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, "", "", "").openProjectPicker()

	out := model.picker.View(180, 24)
	if !strings.Contains(out, "[New Project]") {
		t.Fatalf("render missing '[New Project]': %q", out)
	}
}
