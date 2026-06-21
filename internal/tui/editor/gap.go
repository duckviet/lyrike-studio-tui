package editor

import (
	"github.com/duckviet/lyrike-studio-tui/internal/domain/history"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

func (p Panel) makeInsertGap(idx int) (Panel, int64, int64) {
	lines := p.Document.Lines()
	if len(lines) == 0 {
		start := p.tapPosition.Milliseconds()
		return p, start, start + 3000
	}

	// Case 1: Inserting at the beginning (idx == 0)
	if idx == 0 {
		nextStart := lines[0].Start().Milliseconds()
		if nextStart >= 1000 {
			return p, nextStart - 1000, nextStart
		}
		// Shift nextStart to 1000 to make room
		ts, _ := lyrics.NewTimestamp(1000)
		p = p.apply(history.SetStart{Index: 0, Start: ts})
		return p, 0, 1000
	}

	// Case 2: Inserting at the end (idx == len(lines))
	if idx == len(lines) {
		prevEnd := lines[idx-1].End().Milliseconds()
		return p, prevEnd, prevEnd + 3000
	}

	// Case 3: Inserting in between (0 < idx < len(lines))
	minStart := lines[idx-1].End().Milliseconds()
	maxEnd := lines[idx].Start().Milliseconds()

	if maxEnd-minStart >= 1000 {
		// Gap is already large enough
		return p, minStart, maxEnd
	}

	// We need to create a gap of at least 1000ms.
	// Try to shrink previous line's end first.
	prevStart := lines[idx-1].Start().Milliseconds()
	if minStart-prevStart > 1000 {
		newEnd := minStart - 1000
		ts, _ := lyrics.NewTimestamp(newEnd)
		p = p.apply(history.SetEnd{Index: idx - 1, End: ts})
		return p, newEnd, maxEnd
	}

	// Shift next line's start if possible.
	nextEnd := lines[idx].End().Milliseconds()
	if nextEnd-maxEnd > 1000 {
		newStart := maxEnd + 1000
		ts, _ := lyrics.NewTimestamp(newStart)
		p = p.apply(history.SetStart{Index: idx, Start: ts})
		return p, minStart, newStart
	}

	// Split the difference or create a tiny gap.
	newEnd := minStart - 200
	if newEnd < prevStart {
		newEnd = prevStart + 1
	}
	newStart := maxEnd + 200
	if newStart > nextEnd {
		newStart = nextEnd - 1
	}
	ts1, _ := lyrics.NewTimestamp(newEnd)
	p = p.apply(history.SetEnd{Index: idx - 1, End: ts1})
	ts2, _ := lyrics.NewTimestamp(newStart)
	p = p.apply(history.SetStart{Index: idx, Start: ts2})
	return p, newEnd, newStart
}
