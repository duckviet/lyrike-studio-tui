package history

import (
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

// ---------- SplitLine ----------

// SplitLine splits the segment at Index into two segments at SplitAtMS.
// Line A: [Start, SplitAtMS), Line B: [SplitAtMS, End).
// A keeps text, B gets empty text. If TextPos >= 0, text is split at that rune.
type SplitLine struct {
	Index     int
	SplitAtMS int64
	TextPos   int // rune index; -1 for time-only split (A keeps all text)
}

func (c SplitLine) Name() string { return "split_line" }

func (c SplitLine) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	lines := doc.Lines()
	line := lines[c.Index]

	splitTS, err := lyrics.NewTimestamp(c.SplitAtMS)
	if err != nil {
		return doc, nil, err
	}

	// Build line A: [Start, splitTS)
	lineA, err := lyrics.NewLine(line.Start(), splitTS, splitText(line.Text(), c.TextPos, true))
	if err != nil {
		return doc, nil, err
	}

	// Build line B: [splitTS, End)
	lineB, err := lyrics.NewLine(splitTS, line.End(), splitText(line.Text(), c.TextPos, false))
	if err != nil {
		return doc, nil, err
	}

	// Replace line at Index with lineA, insert lineB after.
	newLines := make([]lyrics.Line, 0, len(lines)+1)
	newLines = append(newLines, lines[:c.Index]...)
	newLines = append(newLines, lineA, lineB)
	newLines = append(newLines, lines[c.Index+1:]...)

	next, err := lyrics.NewDocument(newLines)
	if err != nil {
		return doc, nil, err
	}
	next = next.WithMetadata(doc.Metadata())
	return next, MergeLines{Index: c.Index}, nil
}

func splitText(text lyrics.Text, pos int, first bool) lyrics.Text {
	if pos < 0 {
		if first {
			return text
		}
		empty, _ := lyrics.NewText("")
		return empty
	}
	runes := []rune(text.String())
	if pos > len(runes) {
		pos = len(runes)
	}
	var part string
	if first {
		part = string(runes[:pos])
	} else {
		part = string(runes[pos:])
	}
	t, _ := lyrics.NewText(part)
	return t
}

// ---------- MergeLines ----------

// MergeLines merges line at Index with line at Index+1.
// Result: [Start(A), End(B)], text = A.text + " " + B.text.
type MergeLines struct {
	Index int
}

func (c MergeLines) Name() string { return "merge_lines" }

func (c MergeLines) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	lines := doc.Lines()
	if c.Index < 0 || c.Index+1 >= len(lines) {
		return doc, nil, lyrics.NewValidationErrorPublic(lyrics.CodeInvalidIndex, "merge index out of range")
	}

	a := lines[c.Index]
	b := lines[c.Index+1]

	mergedText := mergeTexts(a.Text(), b.Text())
	merged, err := lyrics.NewLine(a.Start(), b.End(), mergedText)
	if err != nil {
		return doc, nil, err
	}

	newLines := make([]lyrics.Line, 0, len(lines)-1)
	newLines = append(newLines, lines[:c.Index]...)
	newLines = append(newLines, merged)
	newLines = append(newLines, lines[c.Index+2:]...)

	next, err := lyrics.NewDocument(newLines)
	if err != nil {
		return doc, nil, err
	}
	next = next.WithMetadata(doc.Metadata())

	// Inverse: split back at original boundary.
	return next, SplitLine{
		Index:     c.Index,
		SplitAtMS: b.Start().Milliseconds(),
		TextPos:   len([]rune(a.Text().String())),
	}, nil
}

func mergeTexts(a, b lyrics.Text) lyrics.Text {
	as := a.String()
	bs := b.String()
	if as == "" {
		t, _ := lyrics.NewText(bs)
		return t
	}
	if bs == "" {
		return a
	}
	t, _ := lyrics.NewText(as + " " + bs)
	return t
}

// ---------- SwapText ----------

// SwapText swaps text (and word timings) between lines at Index and Index+1.
// Timestamps stay in place.
type SwapText struct {
	Index int
}

func (c SwapText) Name() string { return "swap_text" }

func (c SwapText) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	next, err := doc.WithReorderedLine(c.Index, c.Index+1)
	if err != nil {
		return doc, nil, err
	}
	return next, SwapText{Index: c.Index}, nil
}

// ---------- NudgeStart ----------

// NudgeStart shifts Start of line at Index by DeltaMS.
type NudgeStart struct {
	Index   int
	DeltaMS int64
}

func (c NudgeStart) Name() string { return "nudge_start" }

func (c NudgeStart) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	lines := doc.Lines()
	line := lines[c.Index]
	oldStart := line.Start()
	newMS := line.Start().Milliseconds() + c.DeltaMS
	if newMS < 0 {
		newMS = 0
	}
	if newMS >= line.End().Milliseconds() {
		newMS = line.End().Milliseconds() - 1
	}
	newTS, err := lyrics.NewTimestamp(newMS)
	if err != nil {
		return doc, nil, err
	}
	next, err := doc.WithLineStart(c.Index, newTS)
	if err != nil {
		return doc, nil, err
	}
	return next, SetStart{Index: c.Index, Start: oldStart}, nil
}

// ---------- NudgeEnd ----------

// NudgeEnd shifts End of line at Index by DeltaMS.
type NudgeEnd struct {
	Index   int
	DeltaMS int64
}

func (c NudgeEnd) Name() string { return "nudge_end" }

func (c NudgeEnd) Apply(doc lyrics.Document) (lyrics.Document, Command, error) {
	lines := doc.Lines()
	line := lines[c.Index]
	oldEnd := line.End()
	newMS := line.End().Milliseconds() + c.DeltaMS
	if newMS <= line.Start().Milliseconds() {
		newMS = line.Start().Milliseconds() + 1
	}
	// Clamp to not exceed next line's start.
	if c.Index+1 < len(lines) {
		nextStart := lines[c.Index+1].Start().Milliseconds()
		if newMS > nextStart {
			newMS = nextStart
		}
	}
	newTS, err := lyrics.NewTimestamp(newMS)
	if err != nil {
		return doc, nil, err
	}
	next, err := doc.WithLineEnd(c.Index, newTS)
	if err != nil {
		return doc, nil, err
	}
	return next, SetEnd{Index: c.Index, End: oldEnd}, nil
}
