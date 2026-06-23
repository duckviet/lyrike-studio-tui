package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
)

func Test_GlobalKeyAction_mapsFocusAndSaveKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  tea.KeyPressMsg
		want keyAction
	}{
		{
			name: "tab moves focus next",
			key:  tea.KeyPressMsg{Code: tea.KeyTab},
			want: keyActionFocusNext,
		},
		{
			name: "shift tab moves focus previous",
			key:  tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift},
			want: keyActionFocusPrev,
		},
		{
			name: "ctrl s saves draft",
			key:  tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl},
			want: keyActionSaveDraft,
		},
		{
			name: "plain s is panel local",
			key:  tea.KeyPressMsg{Code: 's'},
			want: keyActionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := globalKeyAction(tt.key); got != tt.want {
				t.Fatalf("globalKeyAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_NonEditingRootKeyAction_mapsPlaybackAndQuitKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  tea.KeyPressMsg
		want keyAction
	}{
		{
			name: "space toggles playback",
			key:  tea.KeyPressMsg{Code: tea.KeySpace},
			want: keyActionTogglePlayback,
		},
		{
			name: "left seeks backward",
			key:  tea.KeyPressMsg{Code: tea.KeyLeft},
			want: keyActionSeekBackward,
		},
		{
			name: "right seeks forward",
			key:  tea.KeyPressMsg{Code: tea.KeyRight},
			want: keyActionSeekForward,
		},
		{
			name: "q quits",
			key:  tea.KeyPressMsg{Code: 'q'},
			want: keyActionQuit,
		},
		{
			name: "ctrl q is not quit",
			key:  tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl},
			want: keyActionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := nonEditingRootKeyAction(tt.key); got != tt.want {
				t.Fatalf("nonEditingRootKeyAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_RootRoutesSelectionKeysToFocusedEditor(t *testing.T) {
	t.Parallel()

	model := newRoutingTestModel(t)
	model.focus = focusEditor

	model = updateRoutingModel(t, model, tea.KeyPressMsg{Code: tea.KeyDown})
	if got := model.editor.Selected(); got != 1 {
		t.Fatalf("selected after Down = %d, want 1", got)
	}

	model = updateRoutingModel(t, model, tea.KeyPressMsg{Code: tea.KeyUp})
	if got := model.editor.Selected(); got != 0 {
		t.Fatalf("selected after Up = %d, want 0", got)
	}

	model = updateRoutingModel(t, model, tea.KeyPressMsg{Text: "j"})
	if got := model.editor.Selected(); got != 1 {
		t.Fatalf("selected after text-only j = %d, want 1", got)
	}

	model = updateRoutingModel(t, model, tea.KeyPressMsg{Text: "k"})
	if got := model.editor.Selected(); got != 0 {
		t.Fatalf("selected after text-only k = %d, want 0", got)
	}
}

func Test_EditorEditModeOwnsPlaybackAndQuitKeys(t *testing.T) {
	t.Parallel()

	model := newRoutingTestModel(t)
	model.focus = focusEditor

	model = updateRoutingModel(t, model, tea.KeyPressMsg{Code: 'e'})
	if !model.editor.Editing {
		t.Fatal("editor should enter edit mode")
	}

	model = updateRoutingModel(t, model, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if got := model.player.Snapshot().State; got != playback.StatePaused {
		t.Fatalf("player state after edit-mode Space = %s, want %s", got, playback.StatePaused)
	}
	if got := model.editor.InputText; got != "First line " {
		t.Fatalf("input after edit-mode Space = %q, want %q", got, "First line ")
	}

	model = updateRoutingModel(t, model, tea.KeyPressMsg{Code: 'q', Text: "q"})
	if got := model.editor.InputText; got != "First line q" {
		t.Fatalf("input after edit-mode q = %q, want %q", got, "First line q")
	}
	if model.status == "quit ready" {
		t.Fatal("edit-mode q should not trigger quit")
	}
}

func Test_DemoModelRoutesTextSelectionKeysAfterEditCommit(t *testing.T) {
	t.Parallel()

	model, err := DemoModel()
	if err != nil {
		t.Fatalf("DemoModel() error = %v", err)
	}
	model = updateRoutingModel(t, model, tea.KeyPressMsg{Code: tea.KeyTab})
	model = updateRoutingModel(t, model, tea.KeyPressMsg{Code: tea.KeyTab})
	model = updateRoutingModel(t, model, tea.KeyPressMsg{Text: "j"})
	model = updateRoutingModel(t, model, tea.KeyPressMsg{Text: "j"})
	if got := model.editor.Selected(); got != 2 {
		t.Fatalf("selected after demo j/j = %d, want 2", got)
	}

	model = updateRoutingModel(t, model, tea.KeyPressMsg{Text: "e"})
	if !model.editor.Editing {
		t.Fatal("demo editor should enter edit mode")
	}
	model = updateRoutingModel(t, model, tea.KeyPressMsg{Code: tea.KeyEnter})
	model = updateRoutingModel(t, model, tea.KeyPressMsg{Text: "k"})
	if got := model.editor.Selected(); got != 1 {
		t.Fatalf("selected after demo edit commit then k = %d, want 1", got)
	}
}

func newRoutingTestModel(t *testing.T) Model {
	t.Helper()

	duration, err := playback.NewDuration(10_000)
	if err != nil {
		t.Fatalf("NewDuration() error = %v", err)
	}
	player, err := playback.NewFakePlayer(duration)
	if err != nil {
		t.Fatalf("NewFakePlayer() error = %v", err)
	}
	doc := routingTestDocument(t)
	return NewModel(doc, nil, player, "video-1", "fixture")
}

func updateRoutingModel(t *testing.T, model Model, msg tea.Msg) Model {
	t.Helper()

	next, _ := model.Update(msg)
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update() returned %T, want tui.Model", next)
	}
	return updated
}

func routingTestDocument(t *testing.T) lyrics.Document {
	t.Helper()

	firstStart, err := lyrics.ParseTimestamp("00:01.00")
	if err != nil {
		t.Fatalf("ParseTimestamp(firstStart) error = %v", err)
	}
	firstEnd, err := lyrics.ParseTimestamp("00:02.00")
	if err != nil {
		t.Fatalf("ParseTimestamp(firstEnd) error = %v", err)
	}
	secondStart, err := lyrics.ParseTimestamp("00:03.00")
	if err != nil {
		t.Fatalf("ParseTimestamp(secondStart) error = %v", err)
	}
	secondEnd, err := lyrics.ParseTimestamp("00:04.00")
	if err != nil {
		t.Fatalf("ParseTimestamp(secondEnd) error = %v", err)
	}
	firstText, err := lyrics.NewText("First line")
	if err != nil {
		t.Fatalf("NewText(first) error = %v", err)
	}
	secondText, err := lyrics.NewText("Second line")
	if err != nil {
		t.Fatalf("NewText(second) error = %v", err)
	}
	firstLine, err := lyrics.NewLine(firstStart, firstEnd, firstText)
	if err != nil {
		t.Fatalf("NewLine(first) error = %v", err)
	}
	secondLine, err := lyrics.NewLine(secondStart, secondEnd, secondText)
	if err != nil {
		t.Fatalf("NewLine(second) error = %v", err)
	}
	doc, err := lyrics.NewDocument([]lyrics.Line{firstLine, secondLine})
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	return doc
}
