package history

import (
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
)

// ---------- SetStart ----------

type SetStart struct {
	Index int
	Start lyrics.Timestamp
}

func (c SetStart) Name() string { return "set_start" }

func (c SetStart) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	old := doc.Lines()[c.Index].Start()
	next, err := doc.WithLineStart(c.Index, c.Start)
	if err != nil {
		return doc, nil, err
	}
	return next, SetStart{Index: c.Index, Start: old}, nil
}

// ---------- SetEnd ----------

type SetEnd struct {
	Index int
	End   lyrics.Timestamp
}

func (c SetEnd) Name() string { return "set_end" }

func (c SetEnd) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	old := doc.Lines()[c.Index].End()
	next, err := doc.WithLineEnd(c.Index, c.End)
	if err != nil {
		return doc, nil, err
	}
	return next, SetEnd{Index: c.Index, End: old}, nil
}

// ---------- EditText ----------

type EditText struct {
	Index int
	Text  lyrics.Text
}

func (c EditText) Name() string { return "edit_text" }

func (c EditText) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	old := doc.Lines()[c.Index].Text()
	next, err := doc.WithLineText(c.Index, c.Text)
	if err != nil {
		return doc, nil, err
	}
	return next, EditText{Index: c.Index, Text: old}, nil
}

// ---------- InsertLine ----------

type InsertLine struct {
	Index int
	Line  lyrics.Line
}

func (c InsertLine) Name() string { return "insert_line" }

func (c InsertLine) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	next, err := doc.WithInsertedLine(c.Index, c.Line)
	if err != nil {
		return doc, nil, err
	}
	return next, DeleteLine{Index: c.Index}, nil
}

// ---------- DeleteLine ----------

type DeleteLine struct {
	Index int
}

func (c DeleteLine) Name() string { return "delete_line" }

func (c DeleteLine) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	deleted := doc.Lines()[c.Index]
	next, err := doc.WithDeletedLine(c.Index)
	if err != nil {
		return doc, nil, err
	}
	return next, InsertLine{Index: c.Index, Line: deleted}, nil
}

// ---------- ReorderLines ----------

type ReorderLines struct {
	From int
	To   int
}

func (c ReorderLines) Name() string { return "reorder_lines" }

func (c ReorderLines) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	next, err := doc.WithReorderedLine(c.From, c.To)
	if err != nil {
		return doc, nil, err
	}
	return next, ReorderLines{From: c.To, To: c.From}, nil
}

// ---------- TapSync ----------

type TapSync struct {
	Index    int
	Position playback.Position
}

func (c TapSync) Name() string { return "tap_sync" }

func (c TapSync) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	newTS, err := lyrics.NewTimestamp(c.Position.Milliseconds())
	if err != nil {
		return doc, nil, err
	}
	lines := doc.Lines()
	old := lines[c.Index].Start()

	next, err := doc.WithLineStart(c.Index, newTS)
	if err != nil {
		return doc, nil, err
	}

	// Also close previous segment's End to match this Start.
	var oldPrevEnd lyrics.Timestamp
	if c.Index > 0 {
		oldPrevEnd = lines[c.Index-1].End()
		next, err = next.WithLineEnd(c.Index-1, newTS)
		if err != nil {
			// If closing previous fails, still return the start-only change.
			next2, _ := doc.WithLineStart(c.Index, newTS)
			return next2, SetStart{Index: c.Index, Start: old}, nil
		}
	}

	// Compound inverse: restore old start and old previous end.
	return next, &tapSyncInverse{
		Index:      c.Index,
		OldStart:   old,
		OldPrevEnd: oldPrevEnd,
		HasPrev:    c.Index > 0,
	}, nil
}

type tapSyncInverse struct {
	Index      int
	OldStart   lyrics.Timestamp
	OldPrevEnd lyrics.Timestamp
	HasPrev    bool
}

func (c *tapSyncInverse) Name() string { return "tap_sync_inverse" }

func (c *tapSyncInverse) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	lines := doc.Lines()
	curStart := lines[c.Index].Start()
	pos, _ := playback.NewPosition(curStart.Milliseconds())

	next, err := doc.WithLineStart(c.Index, c.OldStart)
	if err != nil {
		return doc, nil, err
	}
	if c.HasPrev {
		next, err = next.WithLineEnd(c.Index-1, c.OldPrevEnd)
		if err != nil {
			return doc, nil, err
		}
	}
	return next, TapSync{Index: c.Index, Position: pos}, nil
}
