package editor

import (
	"testing"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
)

func TestMakeInsertGapAtBeginning(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))
	panel, start, end := panel.makeInsertGap(0)

	if start != 0 || end != 1000 {
		t.Fatalf("makeInsertGap() = [%d, %d], want [0, 1000]", start, end)
	}
}

func TestMakeInsertGapAtEnd(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))
	panel, start, end := panel.makeInsertGap(2)

	if start != 6000 || end != 9000 {
		t.Fatalf("makeInsertGap() = [%d, %d], want [6000, 9000]", start, end)
	}
}

func TestMakeInsertGapWhenGapAlreadyLargeEnough(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))
	panel.Document = mustDocument(t, []int64{0, 4000, 8000})
	panel, start, end := panel.makeInsertGap(1)

	if start != 1000 || end != 4000 {
		t.Fatalf("makeInsertGap() = [%d, %d], want [1000, 4000]", start, end)
	}
}

func TestMakeInsertGapShrinksPreviousWhenNeeded(t *testing.T) {
	t.Parallel()

	panel := NewPanel(testDocument(t))
	firstStart, _ := lyrics.NewTimestamp(0)
	firstEnd, _ := lyrics.NewTimestamp(3000)
	firstText, _ := lyrics.NewText("first")
	firstLine, _ := lyrics.NewLine(firstStart, firstEnd, firstText)

	secondStart, _ := lyrics.NewTimestamp(5000)
	secondEnd, _ := lyrics.NewTimestamp(6000)
	secondText, _ := lyrics.NewText("second")
	secondLine, _ := lyrics.NewLine(secondStart, secondEnd, secondText)

	doc, err := lyrics.NewDocument([]lyrics.Line{firstLine, secondLine})
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	panel.Document = doc

	panel, start, end := panel.makeInsertGap(1)

	if start != 3000 || end != 5000 {
		t.Fatalf("makeInsertGap() = [%d, %d], want [3000, 5000]", start, end)
	}
}

func mustDocument(t *testing.T, starts []int64) lyrics.Document {
	t.Helper()

	lines := make([]lyrics.Line, 0, len(starts))
	for i, start := range starts {
		startTS, _ := lyrics.NewTimestamp(start)
		endTS, _ := lyrics.NewTimestamp(start + 1000)
		text, _ := lyrics.NewText("line")
		line, _ := lyrics.NewLine(startTS, endTS, text)
		if i == len(starts)-1 {
			endTS, _ = lyrics.NewTimestamp(start + 2000)
			line, _ = lyrics.NewLine(startTS, endTS, text)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return lyrics.Document{}
	}
	doc, err := lyrics.NewDocument(lines)
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	return doc
}

func TestMakeInsertGapUsesTapPositionWhenNoLines(t *testing.T) {
	t.Parallel()

	pos, _ := playback.NewPosition(1234)
	panel := NewPanel(testDocument(t)).WithTapPosition(pos)
	panel.Document = lyrics.Document{}

	panel, start, end := panel.makeInsertGap(0)

	if start != 1234 || end != 4234 {
		t.Fatalf("makeInsertGap() = [%d, %d], want [1234, 4234]", start, end)
	}
}
