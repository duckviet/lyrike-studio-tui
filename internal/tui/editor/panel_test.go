package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
)

func TestLyricsKeyboardEditUpdatesDocumentState(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))

	// Enter editing mode
	panel, _ = panel.Update(tea.KeyPressMsg{Code: 'e'})

	// Send Backspace 10 times to clear "First line"
	for i := 0; i < 10; i++ {
		panel, _ = panel.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}

	// Type "Edited line"
	for _, char := range "Edited line" {
		panel, _ = panel.Update(tea.KeyPressMsg{Text: string(char)})
	}

	// Press Enter to submit
	updated, _ := panel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if got := updated.Document.Lines()[0].Text().String(); got != "Edited line" {
		t.Fatalf("edited line text = %q, want Edited line", got)
	}
}

func TestLyricsCursorBasedEditing(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))

	// Enter editing mode (cursor starts at end: index 10)
	panel, _ = panel.Update(tea.KeyPressMsg{Code: 'e'})

	// Move cursor left 4 times (to index 6, after "First ")
	for i := 0; i < 4; i++ {
		panel, _ = panel.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	}

	// Insert "New " at cursor
	for _, char := range "New " {
		panel, _ = panel.Update(tea.KeyPressMsg{Text: string(char)})
	}

	// Press Enter to submit
	updated, _ := panel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if got := updated.Document.Lines()[0].Text().String(); got != "First New line" {
		t.Fatalf("edited text = %q, want 'First New line'", got)
	}
}

func TestUndoRedoKeyboardRestoresDocumentState(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))
	// Enter editing mode
	edited, _ := panel.Update(tea.KeyPressMsg{Code: 'e'})
	// Send Backspace 10 times to clear "First line"
	for i := 0; i < 10; i++ {
		edited, _ = edited.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	// Type "Edited line"
	for _, char := range "Edited line" {
		edited, _ = edited.Update(tea.KeyPressMsg{Text: string(char)})
	}
	// Press Enter to submit
	edited, _ = edited.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	undone, _ := edited.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	redone, _ := undone.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})

	if got := undone.Document.Lines()[0].Text().String(); got != "First line" {
		t.Fatalf("undo line text = %q, want First line", got)
	}
	if got := redone.Document.Lines()[0].Text().String(); got != "Edited line" {
		t.Fatalf("redo line text = %q, want Edited line", got)
	}
}

func TestTapKeyboardUsesPlaybackPosition(t *testing.T) {
	t.Parallel()

	position, err := playback.NewPosition(3_500)
	if err != nil {
		t.Fatalf("NewPosition() error = %v", err)
	}
	panel := NewPanel(testDocument(t)).WithTapPosition(position)
	panel, _ = panel.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	updated, _ := panel.Update(tea.KeyPressMsg{Code: 't'})

	if got := updated.Document.Lines()[1].Start().Milliseconds(); got != 3_500 {
		t.Fatalf("tap start timestamp = %d, want 3500", got)
	}
}

func TestSplitAndMergeKeyboard(t *testing.T) {
	t.Parallel()

	position, _ := playback.NewPosition(1_500)
	panel := NewPanel(testDocument(t)).WithTapPosition(position)

	// Split first line at 1500ms
	panel, _ = panel.Update(tea.KeyPressMsg{Code: 's'})

	if len(panel.Document.Lines()) != 3 {
		t.Fatalf("lines count after split = %d, want 3", len(panel.Document.Lines()))
	}

	// Merge back
	panel, _ = panel.Update(tea.KeyPressMsg{Code: 'm'})

	if len(panel.Document.Lines()) != 2 {
		t.Fatalf("lines count after merge = %d, want 2", len(panel.Document.Lines()))
	}
}

func testDocument(t *testing.T) lyrics.Document {
	t.Helper()

	firstStart, err := lyrics.ParseTimestamp("00:01.00")
	if err != nil {
		t.Fatalf("ParseTimestamp(firstStart) error = %v", err)
	}
	firstEnd, err := lyrics.ParseTimestamp("00:02.00")
	if err != nil {
		t.Fatalf("ParseTimestamp(firstEnd) error = %v", err)
	}
	secondStart, err := lyrics.ParseTimestamp("00:04.00")
	if err != nil {
		t.Fatalf("ParseTimestamp(secondStart) error = %v", err)
	}
	secondEnd, err := lyrics.ParseTimestamp("00:06.00")
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

func TestLyricsKeyboardImportToggleAndCancel(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))

	// Enter importing mode
	panel, _ = panel.Update(tea.KeyPressMsg{Code: 'I'})
	if !panel.Importing {
		t.Fatal("expected Importing to be true after pressing 'I'")
	}

	// Type some characters
	for _, char := range "some/path.txt" {
		panel, _ = panel.Update(tea.KeyPressMsg{Text: string(char)})
	}

	if panel.InputText != "some/path.txt" {
		t.Fatalf("expected InputText to be 'some/path.txt', got %q", panel.InputText)
	}

	// Press escape to cancel
	panel, _ = panel.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if panel.Importing {
		t.Fatal("expected Importing to be false after escape")
	}
	if panel.InputText != "" {
		t.Fatalf("expected InputText to be empty after escape, got %q", panel.InputText)
	}
}

func TestLyricsKeyboardNavigationDoesNotSeek(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))

	// Move selection down
	panel, cmd := panel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if panel.selected != 1 {
		t.Fatalf("expected selected to be 1, got %d", panel.selected)
	}
	if cmd != nil {
		t.Fatal("expected no command (no seek) on KeyDown navigation")
	}

	// Press 'g' to explicitly seek to selected line
	_, cmd = panel.Update(tea.KeyPressMsg{Code: 'g'})
	if cmd == nil {
		t.Fatal("expected seek command when pressing 'g'")
	}
}

func TestLyricsKeyboardSnapToActive(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t)).WithPlaybackPosition(5000)

	if panel.selected != 0 {
		t.Fatalf("expected initial selected to be 0, got %d", panel.selected)
	}

	panel, _ = panel.Update(tea.KeyPressMsg{Code: 'f'})
	if panel.selected != 1 {
		t.Fatalf("expected selected to snap to active line index 1, got %d", panel.selected)
	}

	panel, _ = panel.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if panel.selected != 0 {
		t.Fatalf("expected selected to be 0, got %d", panel.selected)
	}

	panel, _ = panel.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	if panel.selected != 1 {
		t.Fatalf("expected selected to snap to active line index 1 via Ctrl-F, got %d", panel.selected)
	}
}

func TestLyricsScrolling(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))
	panel = panel.WithHeight(1)

	if panel.viewport.YOffset != 0 {
		t.Fatalf("expected initial scrollOffset to be 0, got %d", panel.viewport.YOffset)
	}

	panel = panel.WithSelected(1)
	if panel.viewport.YOffset != 1 {
		t.Fatalf("expected scrollOffset to follow selection to 1, got %d", panel.viewport.YOffset)
	}

	panel = panel.HandleMouseScroll(tea.MouseWheelUp)
	if panel.viewport.YOffset != 0 {
		t.Fatalf("expected scrollOffset to be 0 after mouse scroll up, got %d", panel.viewport.YOffset)
	}

	panel = panel.HandleMouseScroll(tea.MouseWheelDown)
	if panel.viewport.YOffset != 1 {
		t.Fatalf("expected scrollOffset to be 1 after mouse scroll down, got %d", panel.viewport.YOffset)
	}
}

func TestLyricsKeyboardEditPaste(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))

	// Enter editing mode
	panel, _ = panel.Update(tea.KeyPressMsg{Code: 'e'})

	// Send Backspace 10 times to clear "First line"
	for i := 0; i < 10; i++ {
		panel, _ = panel.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}

	// Paste text
	panel, _ = panel.Update(tea.PasteMsg{Content: "Pasted text"})

	// Press Enter to submit
	updated, _ := panel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if got := updated.Document.Lines()[0].Text().String(); got != "Pasted text" {
		t.Fatalf("edited line text = %q, want Pasted text", got)
	}
}

func TestLyricsKeyboardImportPaste(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))

	// Enter importing mode
	panel, _ = panel.Update(tea.KeyPressMsg{Code: 'I'})

	// Paste text
	panel, _ = panel.Update(tea.PasteMsg{Content: "ImportedPath/file.txt"})

	if panel.InputText != "ImportedPath/file.txt" {
		t.Fatalf("expected InputText to be 'ImportedPath/file.txt', got %q", panel.InputText)
	}
}
