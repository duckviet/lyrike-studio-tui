package editor

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/history"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
)

type Panel struct {
	Title              string
	Document           lyrics.Document
	manager            *history.Manager
	selected           int
	tapPosition        playback.Position
	playbackPositionMS int64
	lastErr            error
	Editing            bool
	InputText          string
	cursorPos          int
	ShowHelp           bool
	helpScroll         int
}

var (
	activeLineStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	selectedLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3366"))
)

var helpLines = []string{
	"Navigation:",
	"  Tab / Shift-Tab : Focus next/prev panel",
	"  j / k           : Move selection down/up",
	"  Down / Up       : Move selection down/up",
	"",
	"Playback:",
	"  Space           : Play / Pause outside text edit",
	"  Left / Right    : Seek backward/forward 1s outside text edit",
	"",
	"Editing Lyrics:",
	"  e               : Edit current line text",
	"  i               : Insert blank line before",
	"  a / o / Enter   : Insert blank line after",
	"  d               : Delete current line",
	"  s               : Split at playhead (Normal)",
	"                    Split at cursor (Edit)",
	"  m               : Merge current line with next",
	"  J / K           : Swap text with next/prev line",
	"",
	"Sync & Nudge Time:",
	"  t               : Tap-sync line to playhead",
	"  [ / ]           : Nudge Start time ±100ms",
	"  { / }           : Nudge End time ±100ms",
	"  Ctrl-[ / Ctrl-] : Fine nudge Start time ±10ms",
	"  Ctrl-, / Ctrl-. : Fine nudge End time ±10ms",
	"",
	"Commands & File:",
	"  Ctrl-S          : Save draft snapshot",
	"  p               : Publish lyrics to server",
	"  h / ?           : Toggle help menu",
	"  q               : Quit application",
}

func NewPanel(doc lyrics.Document) Panel {
	return Panel{
		Title:    "Lyrics",
		Document: doc,
		manager:  history.NewManager(),
	}
}

func (p Panel) Selected() int {
	return p.selected
}

func (p Panel) LastError() error {
	return p.lastErr
}

func (p Panel) WithTapPosition(position playback.Position) Panel {
	p.tapPosition = position
	return p
}

func (p Panel) WithPlaybackPosition(pos int64) Panel {
	p.playbackPositionMS = pos
	return p
}

func (p Panel) activeLineIndex() int {
	lines := p.Document.Lines()
	index := -1
	for i, line := range lines {
		if p.playbackPositionMS >= line.Start().Milliseconds() {
			index = i
		}
	}
	return index
}

func (p Panel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, nil
	}
	return p.handleKey(key)
}

type StartPublishMsg struct {
	Lyrics string
}

func (p Panel) WithSelected(index int) Panel {
	if index >= 0 && index < len(p.Document.Lines()) {
		p.selected = index
	}
	return p
}

func (p Panel) View(_ int, height int) string {
	if p.ShowHelp {
		maxScroll := len(helpLines) - height
		if maxScroll < 0 {
			maxScroll = 0
		}
		if p.helpScroll > maxScroll {
			p.helpScroll = maxScroll
		}
		endIdx := p.helpScroll + height
		if endIdx > len(helpLines) {
			endIdx = len(helpLines)
		}
		return strings.Join(helpLines[p.helpScroll:endIdx], "\n")
	}

	lines := p.Document.Lines()
	startIdx := 0
	endIdx := len(lines)

	if len(lines) > height {
		half := height / 2
		startIdx = p.selected - half
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx = startIdx + height
		if endIdx > len(lines) {
			endIdx = len(lines)
			startIdx = endIdx - height
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	activeIdx := p.activeLineIndex()

	var builder strings.Builder
	for index := startIdx; index < endIdx; index++ {
		line := lines[index]

		marker := "  "
		if index == p.selected {
			marker = "> "
		}
		playMarker := "  "
		if index == activeIdx {
			playMarker = "▶ "
		}

		text := line.Text().String()
		if p.Editing && index == p.selected {
			runes := []rune(p.InputText)
			if p.cursorPos > len(runes) {
				p.cursorPos = len(runes)
			}
			text = "[Edit]: " + string(runes[:p.cursorPos]) + "|" + string(runes[p.cursorPos:])
		}
		timeRange := fmt.Sprintf("%s-%s", line.Start().String(), line.End().String())
		lineStr := fmt.Sprintf("%s%s%s %s", marker, playMarker, timeRange, text)

		if index == activeIdx {
			lineStr = activeLineStyle.Render(lineStr)
		} else if index == p.selected {
			lineStr = selectedLineStyle.Render(lineStr)
		}

		builder.WriteString(lineStr)
		if index < endIdx-1 {
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}

func (p Panel) applyEditText(value string) Panel {
	text, err := lyrics.NewText(value)
	if err != nil {
		p.lastErr = err
		return p
	}
	return p.apply(history.EditText{Index: p.selected, Text: text})
}

func (p Panel) apply(command history.Command) Panel {
	next, err := p.manager.Apply(p.Document, command)
	if err != nil {
		p.lastErr = err
		return p
	}
	p.Document = next
	p.lastErr = nil
	return p
}

func (p Panel) undo() Panel {
	next, err := p.manager.Undo(p.Document)
	if err != nil {
		p.lastErr = err
		return p
	}
	p.Document = next
	p.lastErr = nil
	return p
}

func (p Panel) redo() Panel {
	next, err := p.manager.Redo(p.Document)
	if err != nil {
		p.lastErr = err
		return p
	}
	p.Document = next
	p.lastErr = nil
	return p
}

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
