package history

import (
	"errors"
	"testing"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
)

func TestCommand_SetStart_thenUndo(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	newTS, err := lyrics.NewTimestamp(3_000)
	if err != nil {
		t.Fatalf("NewTimestamp() error = %v", err)
	}

	next, err := manager.Apply(doc, SetStart{Index: 1, Start: newTS})
	if err != nil {
		t.Fatalf("Apply(SetStart) error = %v", err)
	}

	if next.Lines()[1].Start().Milliseconds() != 3_000 {
		t.Fatalf("line[1] start = %d, want 3000", next.Lines()[1].Start().Milliseconds())
	}

	undone, err := manager.Undo(next)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}

	if undone.Lines()[1].Start().Milliseconds() != 4_000 {
		t.Fatalf("after undo line[1] start = %d, want 4000", undone.Lines()[1].Start().Milliseconds())
	}
}

func TestCommand_SetEnd_thenUndo(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	newTS, err := lyrics.NewTimestamp(3_000)
	if err != nil {
		t.Fatalf("NewTimestamp() error = %v", err)
	}

	next, err := manager.Apply(doc, SetEnd{Index: 0, End: newTS})
	if err != nil {
		t.Fatalf("Apply(SetEnd) error = %v", err)
	}

	if next.Lines()[0].End().Milliseconds() != 3_000 {
		t.Fatalf("line[0] end = %d, want 3000", next.Lines()[0].End().Milliseconds())
	}

	undone, err := manager.Undo(next)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}

	if undone.Lines()[0].End().Milliseconds() != 2_000 {
		t.Fatalf("after undo line[0] end = %d, want 2000", undone.Lines()[0].End().Milliseconds())
	}
}

func TestCommand_EditText_thenUndo(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	newText, err := lyrics.NewText("Changed line")
	if err != nil {
		t.Fatalf("NewText() error = %v", err)
	}

	next, err := manager.Apply(doc, EditText{Index: 0, Text: newText})
	if err != nil {
		t.Fatalf("Apply(EditText) error = %v", err)
	}

	if next.Lines()[0].Text().String() != "Changed line" {
		t.Fatalf("line[0] text = %q, want Changed line", next.Lines()[0].Text().String())
	}

	undone, err := manager.Undo(next)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}

	if undone.Lines()[0].Text().String() != "First line" {
		t.Fatalf("after undo line[0] text = %q, want First line", undone.Lines()[0].Text().String())
	}
}

func TestCommand_InsertLine_thenUndo(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	start, _ := lyrics.NewTimestamp(2_500)
	end, _ := lyrics.NewTimestamp(3_500)
	text, _ := lyrics.NewText("Inserted line")
	line, err := lyrics.NewLine(start, end, text)
	if err != nil {
		t.Fatalf("NewLine() error = %v", err)
	}

	next, err := manager.Apply(doc, InsertLine{Index: 1, Line: line})
	if err != nil {
		t.Fatalf("Apply(InsertLine) error = %v", err)
	}

	if len(next.Lines()) != 3 {
		t.Fatalf("len(lines) = %d, want 3", len(next.Lines()))
	}
	if next.Lines()[1].Text().String() != "Inserted line" {
		t.Fatalf("line[1] text = %q, want Inserted line", next.Lines()[1].Text().String())
	}

	undone, err := manager.Undo(next)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}

	if len(undone.Lines()) != 2 {
		t.Fatalf("after undo len(lines) = %d, want 2", len(undone.Lines()))
	}
}

func TestCommand_DeleteLine_thenUndo(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	next, err := manager.Apply(doc, DeleteLine{Index: 0})
	if err != nil {
		t.Fatalf("Apply(DeleteLine) error = %v", err)
	}

	if len(next.Lines()) != 1 {
		t.Fatalf("len(lines) = %d, want 1", len(next.Lines()))
	}
	if next.Lines()[0].Text().String() != "Second line" {
		t.Fatalf("line[0] text = %q, want Second line", next.Lines()[0].Text().String())
	}

	undone, err := manager.Undo(next)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}

	if len(undone.Lines()) != 2 {
		t.Fatalf("after undo len(lines) = %d, want 2", len(undone.Lines()))
	}
	if undone.Lines()[0].Text().String() != "First line" {
		t.Fatalf("after undo line[0] text = %q, want First line", undone.Lines()[0].Text().String())
	}
}

func TestCommand_ReorderLines_thenUndo(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	next, err := manager.Apply(doc, ReorderLines{From: 0, To: 1})
	if err != nil {
		t.Fatalf("Apply(ReorderLines) error = %v", err)
	}

	if next.Lines()[0].Start().Milliseconds() != 0 {
		t.Fatalf("line[0] start = %d, want 0", next.Lines()[0].Start().Milliseconds())
	}
	if next.Lines()[0].Text().String() != "Second line" {
		t.Fatalf("line[0] text = %q, want Second line", next.Lines()[0].Text().String())
	}
	if next.Lines()[1].Start().Milliseconds() != 4_000 {
		t.Fatalf("line[1] start = %d, want 4000", next.Lines()[1].Start().Milliseconds())
	}
	if next.Lines()[1].Text().String() != "First line" {
		t.Fatalf("line[1] text = %q, want First line", next.Lines()[1].Text().String())
	}

	undone, err := manager.Undo(next)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}

	if undone.Lines()[0].Text().String() != "First line" {
		t.Fatalf("after undo line[0] text = %q, want First line", undone.Lines()[0].Text().String())
	}
}

func TestCommand_TapSync_usesPlaybackPosition(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	pos, _ := playback.NewPosition(3_500)

	next, err := manager.Apply(doc, TapSync{Index: 1, Position: pos})
	if err != nil {
		t.Fatalf("Apply(TapSync) error = %v", err)
	}

	if next.Lines()[1].Start().Milliseconds() != 3_500 {
		t.Fatalf("line[1] start = %d, want 3500", next.Lines()[1].Start().Milliseconds())
	}
	// TapSync also closes previous segment.
	if next.Lines()[0].End().Milliseconds() != 3_500 {
		t.Fatalf("line[0] end = %d, want 3500", next.Lines()[0].End().Milliseconds())
	}
}

func TestCommand_SplitLine_thenUndo(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	// Split first line at 1500ms (time-only split, A keeps text).
	next, err := manager.Apply(doc, SplitLine{Index: 0, SplitAtMS: 1_500, TextPos: -1})
	if err != nil {
		t.Fatalf("Apply(SplitLine) error = %v", err)
	}

	if len(next.Lines()) != 3 {
		t.Fatalf("len(lines) = %d, want 3", len(next.Lines()))
	}
	if next.Lines()[0].Start().Milliseconds() != 0 {
		t.Fatalf("line[0] start = %d, want 0", next.Lines()[0].Start().Milliseconds())
	}
	if next.Lines()[0].End().Milliseconds() != 1_500 {
		t.Fatalf("line[0] end = %d, want 1500", next.Lines()[0].End().Milliseconds())
	}
	if next.Lines()[0].Text().String() != "First line" {
		t.Fatalf("line[0] text = %q, want First line", next.Lines()[0].Text().String())
	}
	if next.Lines()[1].Start().Milliseconds() != 1_500 {
		t.Fatalf("line[1] start = %d, want 1500", next.Lines()[1].Start().Milliseconds())
	}
	if next.Lines()[1].End().Milliseconds() != 2_000 {
		t.Fatalf("line[1] end = %d, want 2000", next.Lines()[1].End().Milliseconds())
	}

	// Undo should merge back to original.
	undone, err := manager.Undo(next)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}

	if len(undone.Lines()) != 2 {
		t.Fatalf("after undo len(lines) = %d, want 2", len(undone.Lines()))
	}
}

func TestCommand_MergeLines_thenUndo(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	next, err := manager.Apply(doc, MergeLines{Index: 0})
	if err != nil {
		t.Fatalf("Apply(MergeLines) error = %v", err)
	}

	if len(next.Lines()) != 1 {
		t.Fatalf("len(lines) = %d, want 1", len(next.Lines()))
	}
	if next.Lines()[0].Text().String() != "First line Second line" {
		t.Fatalf("line[0] text = %q, want 'First line Second line'", next.Lines()[0].Text().String())
	}
	if next.Lines()[0].Start().Milliseconds() != 0 {
		t.Fatalf("line[0] start = %d, want 0", next.Lines()[0].Start().Milliseconds())
	}
	if next.Lines()[0].End().Milliseconds() != 6_000 {
		t.Fatalf("line[0] end = %d, want 6000", next.Lines()[0].End().Milliseconds())
	}

	// Undo: split back.
	undone, err := manager.Undo(next)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}

	if len(undone.Lines()) != 2 {
		t.Fatalf("after undo len(lines) = %d, want 2", len(undone.Lines()))
	}
	if undone.Lines()[0].Text().String() != "First line" {
		t.Fatalf("after undo line[0] text = %q, want 'First line'", undone.Lines()[0].Text().String())
	}
	if undone.Lines()[1].Text().String() != "Second line" {
		t.Fatalf("after undo line[1] text = %q, want 'Second line'", undone.Lines()[1].Text().String())
	}
}

func TestCommand_SwapText_thenUndo(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	next, err := manager.Apply(doc, SwapText{Index: 0})
	if err != nil {
		t.Fatalf("Apply(SwapText) error = %v", err)
	}

	// Timestamps stay, text swaps.
	if next.Lines()[0].Text().String() != "Second line" {
		t.Fatalf("line[0] text = %q, want Second line", next.Lines()[0].Text().String())
	}
	if next.Lines()[1].Text().String() != "First line" {
		t.Fatalf("line[1] text = %q, want First line", next.Lines()[1].Text().String())
	}

	undone, err := manager.Undo(next)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}

	if undone.Lines()[0].Text().String() != "First line" {
		t.Fatalf("after undo line[0] text = %q, want First line", undone.Lines()[0].Text().String())
	}
}

func TestCommand_NudgeStart_thenUndo(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	next, err := manager.Apply(doc, NudgeStart{Index: 1, DeltaMS: -500})
	if err != nil {
		t.Fatalf("Apply(NudgeStart) error = %v", err)
	}

	if next.Lines()[1].Start().Milliseconds() != 3_500 {
		t.Fatalf("line[1] start = %d, want 3500", next.Lines()[1].Start().Milliseconds())
	}

	undone, err := manager.Undo(next)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}

	if undone.Lines()[1].Start().Milliseconds() != 4_000 {
		t.Fatalf("after undo line[1] start = %d, want 4000", undone.Lines()[1].Start().Milliseconds())
	}
}

func TestCommand_NudgeEnd_thenUndo(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	next, err := manager.Apply(doc, NudgeEnd{Index: 0, DeltaMS: 500})
	if err != nil {
		t.Fatalf("Apply(NudgeEnd) error = %v", err)
	}

	if next.Lines()[0].End().Milliseconds() != 2_500 {
		t.Fatalf("line[0] end = %d, want 2500", next.Lines()[0].End().Milliseconds())
	}

	undone, err := manager.Undo(next)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}

	if undone.Lines()[0].End().Milliseconds() != 2_000 {
		t.Fatalf("after undo line[0] end = %d, want 2000", undone.Lines()[0].End().Milliseconds())
	}
}

func TestManager_Undo_whenEmpty_returnsError(t *testing.T) {
	t.Parallel()

	manager := NewManager()
	doc := newTestDocument(t)

	_, err := manager.Undo(doc)
	if err == nil {
		t.Fatalf("Undo() error = nil, want error")
	}
	if !errors.Is(err, ErrNothingToUndo) {
		t.Fatalf("error is not ErrNothingToUndo: %v", err)
	}
}

func TestManager_Redo_afterApplyClearsRedoStack(t *testing.T) {
	t.Parallel()

	doc := newTestDocument(t)
	manager := NewManager()

	pos1, _ := playback.NewPosition(1_000)
	pos2, _ := playback.NewPosition(1_500)

	a, err := manager.Apply(doc, TapSync{Index: 0, Position: pos1})
	if err != nil {
		t.Fatalf("Apply(first) error = %v", err)
	}
	b, err := manager.Undo(a)
	if err != nil {
		t.Fatalf("Undo() error = %v", err)
	}
	if !manager.CanRedo() {
		t.Fatalf("CanRedo = false after undo, want true")
	}

	_, err = manager.Apply(b, TapSync{Index: 0, Position: pos2})
	if err != nil {
		t.Fatalf("Apply(second) error = %v", err)
	}

	if manager.CanRedo() {
		t.Fatalf("CanRedo = true after new apply, want false")
	}
}

// newTestDocument creates a document with two lines:
//
//	[0ms, 2000ms) "First line"
//	[4000ms, 6000ms) "Second line"
func newTestDocument(t *testing.T) lyrics.Document {
	t.Helper()

	text1, _ := lyrics.NewText("First line")
	start1, _ := lyrics.NewTimestamp(0)
	end1, _ := lyrics.NewTimestamp(2_000)
	line1, err := lyrics.NewLine(start1, end1, text1)
	if err != nil {
		t.Fatalf("NewLine() error = %v", err)
	}

	text2, _ := lyrics.NewText("Second line")
	start2, _ := lyrics.NewTimestamp(4_000)
	end2, _ := lyrics.NewTimestamp(6_000)
	line2, err := lyrics.NewLine(start2, end2, text2)
	if err != nil {
		t.Fatalf("NewLine() error = %v", err)
	}

	doc, err := lyrics.NewDocument([]lyrics.Line{line1, line2})
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	return doc
}
