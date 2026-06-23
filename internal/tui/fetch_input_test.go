package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
)

var ansiEscapeRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

func TestParseVideoIDInput(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantOK  bool
	}{
		{
			name:    "watch URL",
			raw:     "https://www.youtube.com/watch?v=P0N0h_EOS-c",
			wantID:  "P0N0h_EOS-c",
			wantURL: "https://www.youtube.com/watch?v=P0N0h_EOS-c",
			wantOK:  true,
		},
		{
			name:    "short youtu.be URL",
			raw:     "https://youtu.be/P0N0h_EOS-c",
			wantID:  "P0N0h_EOS-c",
			wantURL: "https://youtu.be/P0N0h_EOS-c",
			wantOK:  true,
		},
		{
			name:    "music.youtube.com URL",
			raw:     "https://music.youtube.com/watch?v=P0N0h_EOS-c",
			wantID:  "P0N0h_EOS-c",
			wantURL: "https://music.youtube.com/watch?v=P0N0h_EOS-c",
			wantOK:  true,
		},
		{
			name:   "bare video ID",
			raw:    "P0N0h_EOS-c",
			wantID: "P0N0h_EOS-c",
			wantOK: true,
		},
		{
			name:   "bare ID with surrounding whitespace",
			raw:    "  P0N0h_EOS-c  ",
			wantID: "P0N0h_EOS-c",
			wantOK: true,
		},
		{
			name:   "invalid empty",
			raw:    "",
			wantOK: false,
		},
		{
			name:   "invalid bare ID with spaces",
			raw:    "abc def",
			wantOK: false,
		},
		{
			name:   "non-YouTube URL",
			raw:    "https://example.com/video",
			wantOK: false,
		},
		{
			name:    "embed URL",
			raw:     "https://www.youtube.com/embed/P0N0h_EOS-c",
			wantID:  "P0N0h_EOS-c",
			wantURL: "https://www.youtube.com/embed/P0N0h_EOS-c",
			wantOK:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotURL, gotOK := parseVideoIDInput(tc.raw)
			if gotOK != tc.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if gotID != tc.wantID {
				t.Fatalf("videoID = %q, want %q", gotID, tc.wantID)
			}
			if gotURL != tc.wantURL {
				t.Fatalf("sourceURL = %q, want %q", gotURL, tc.wantURL)
			}
		})
	}
}

func TestCtrlOOpensFetchInput(t *testing.T) {
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, &memoryDraftStore{}, "", "", "")

	next, _ := model.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
	got := next.(Model)

	if !got.fetchInput.active() {
		t.Fatalf("fetchInput.active = false, want true")
	}
	if got.fetchInput.mode != fetchInputEnter {
		t.Fatalf("fetchInput.mode = %d, want enter", got.fetchInput.mode)
	}
}

func TestRenderFetchInput(t *testing.T) {
	th := DefaultTheme()
	enter := renderFetchInput(fetchInput{mode: fetchInputEnter, input: "abc"}, 80, 24, th)
	if !strings.Contains(enter, "YouTube URL or video ID:") {
		t.Fatalf("enter render missing prompt: %q", enter)
	}
	if !strings.Contains(enter, "abc") {
		t.Fatalf("enter render missing input: %q", enter)
	}

	confirm := renderFetchInput(fetchInput{mode: fetchInputConfirmReplace, targetVideoID: "newid"}, 80, 24, th)
	if !strings.Contains(confirm, "Unsaved changes") {
		t.Fatalf("confirm render missing warning: %q", confirm)
	}
	if !strings.Contains(confirm, "newid") {
		t.Fatalf("confirm render missing target id: %q", confirm)
	}
}

func typeIntoFetchInput(t *testing.T, m Model, text string) Model {
	t.Helper()
	for _, r := range text {
		next, _ := m.Update(tea.KeyPressMsg{Text: string(r)})
		m = next.(Model)
	}
	return m
}

func TestRenderFetchInputFits80x24(t *testing.T) {
	longInput := strings.Repeat("a", 120)
	out := renderFetchInput(fetchInput{mode: fetchInputEnter, input: longInput}, 80, 24, DefaultTheme())
	lines := strings.Split(out, "\n")
	if len(lines) > 24 {
		t.Fatalf("rendered %d lines, want <= 24", len(lines))
	}
	for i, line := range lines {
		stripped := ansiEscapeRe.ReplaceAllString(line, "")
		if w := utf8.RuneCountInString(stripped); w > 80 {
			t.Fatalf("line %d width %d, want <= 80: %q", i, w, stripped)
		}
	}
}

func TestFetchInputEnterLoadsExistingDraft(t *testing.T) {
	videoID := "P0N0h_EOS-c"
	projectID := draft.ProjectID(videoID)
	loaded := draft.Snapshot{
		ProjectID: projectID,
		ID:        draft.DraftID(projectID.String()),
		Metadata:  draft.Metadata{VideoID: videoID, TrackName: "Loaded Track", AlbumName: "Loaded Album"},
		Document:  mustDemoDocument(t),
	}
	store := &memoryDraftStore{
		loads: map[draft.ProjectID]draft.Snapshot{projectID: loaded},
	}
	model := NewModelWithDraftStore(newDefaultDocument(), nil, nil, store, "", "", "").openFetchInput()
	model = typeIntoFetchInput(t, model, videoID)

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)

	if got.projectID != projectID {
		t.Fatalf("projectID = %q, want %q", got.projectID, projectID)
	}
	if got.dirty {
		t.Fatalf("dirty = true, want false after load")
	}
	if got.fetchInput.active() {
		t.Fatalf("fetchInput.active = true, want false")
	}
	if fmt.Sprint(got.editor.Document) != fmt.Sprint(loaded.Document) {
		t.Fatalf("editor document was not loaded from snapshot")
	}
}

func TestFetchInputEnterNewProject(t *testing.T) {
	videoID := "P0N0h_EOS-c"
	store := &memoryDraftStore{}
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, "", "", "").openFetchInput()
	model.trackName = "Old Track"
	model.albumName = "Old Album"
	model = typeIntoFetchInput(t, model, videoID)

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)

	if got.projectID != draft.ProjectID(videoID) {
		t.Fatalf("projectID = %q, want %q", got.projectID, videoID)
	}
	if !got.dirty {
		t.Fatalf("dirty = false, want true")
	}
	if got.trackName != "" {
		t.Fatalf("trackName = %q, want empty", got.trackName)
	}
	if got.albumName != "" {
		t.Fatalf("albumName = %q, want empty", got.albumName)
	}
	if got.fetchInput.active() {
		t.Fatalf("fetchInput.active = true, want false")
	}
}

func TestFetchInputDirtyConfirm(t *testing.T) {
	store := &memoryDraftStore{}
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, draft.ProjectID("oldid"), "", "")
	model.dirty = true
	model = model.openFetchInput()
	model = typeIntoFetchInput(t, model, "newid")

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	confirm := next.(Model)
	if confirm.fetchInput.mode != fetchInputConfirmReplace {
		t.Fatalf("fetchInput.mode = %d, want confirm replace", confirm.fetchInput.mode)
	}
	if confirm.fetchInput.targetVideoID != "newid" {
		t.Fatalf("targetVideoID = %q, want newid", confirm.fetchInput.targetVideoID)
	}
	if confirm.projectID != draft.ProjectID("oldid") {
		t.Fatalf("projectID = %q, want oldid before confirm", confirm.projectID)
	}

	next, _ = confirm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)
	if got.projectID != draft.ProjectID("newid") {
		t.Fatalf("projectID = %q, want newid", got.projectID)
	}
}

func TestFetchInputInvalidInput(t *testing.T) {
	store := &memoryDraftStore{}
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, store, "", "", "").openFetchInput()
	model = typeIntoFetchInput(t, model, "abc def")

	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := next.(Model)

	if !got.fetchInput.active() {
		t.Fatalf("fetchInput.active = false, want true")
	}
	if len(got.status) == 0 || !strings.Contains(strings.ToLower(got.status[0]), "invalid") {
		t.Fatalf("status = %q, want invalid message", got.status)
	}
}

func TestFetchInputPaste(t *testing.T) {
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, &memoryDraftStore{}, "", "", "").openFetchInput()
	next, _ := model.Update(tea.PasteMsg{Content: "https://www.youtube.com/watch?v=P0N0h_EOS-c"})
	got := next.(Model)

	if got.fetchInput.input != "https://www.youtube.com/watch?v=P0N0h_EOS-c" {
		t.Fatalf("fetchInput.input = %q, want https://www.youtube.com/watch?v=P0N0h_EOS-c", got.fetchInput.input)
	}
}
