package tui

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
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
	if got.picker.active() {
		t.Fatalf("picker.active = true, want false")
	}
	if store.saves != 0 {
		t.Fatalf("Save calls = %d, want 0", store.saves)
	}
}

func TestProjectPicker_ctrlPOpensProjectList(t *testing.T) {
	store := &memoryDraftStore{
		projects: []draft.ProjectSummary{{ID: "project-b"}, {ID: "project-a"}},
	}
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, "project-a", "", "")

	next, _ := model.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	got := next.(Model)

	if got.picker.mode != projectPickerChoose {
		t.Fatalf("picker mode = %d, want choose", got.picker.mode)
	}
	if len(got.picker.projects) != 2 {
		t.Fatalf("projects len = %d, want 2", len(got.picker.projects))
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

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	confirm := next.(Model)
	if confirm.picker.mode != projectPickerConfirmLoad {
		t.Fatalf("picker mode = %d, want confirm", confirm.picker.mode)
	}
	if confirm.projectID != projectA {
		t.Fatalf("projectID = %q, want current project before confirm", confirm.projectID)
	}

	next, _ = confirm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if got.projectID != projectB {
		t.Fatalf("projectID = %q, want %q", got.projectID, projectB)
	}
	if got.dirty {
		t.Fatalf("dirty = true, want false after load")
	}
}

func TestProjectPickerNOpensFetchInput(t *testing.T) {
	store := &memoryDraftStore{
		projects: []draft.ProjectSummary{{ID: "project-a"}},
	}
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, "", "", "").openProjectPicker()

	next, _ := model.Update(tea.KeyPressMsg{Code: 'n'})
	got := next.(Model)

	if !got.fetchInput.active() {
		t.Fatalf("fetchInput.active = false, want true")
	}
	if got.picker.active() {
		t.Fatalf("picker.active = true, want false")
	}
}

func TestProjectPickerNoProjectsShowsNewFromURL(t *testing.T) {
	store := &memoryDraftStore{}
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, "", "", "").openProjectPicker()

	out := renderProjectPicker(model.picker, 80, 24)
	if !strings.Contains(out, "new from URL") {
		t.Fatalf("render missing 'new from URL': %q", out)
	}
}
