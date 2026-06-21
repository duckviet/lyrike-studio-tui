package editor

import (
	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/history"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

func (p Panel) handleKey(key tea.KeyPressMsg) (Panel, tea.Cmd) {
	key = normalizeKeyPress(key)

	if p.ShowHelp {
		if key.Code == tea.KeyDown || key.Code == 'j' {
			p.helpScroll++
			return p, nil
		}
		if key.Code == tea.KeyUp || key.Code == 'k' {
			p.helpScroll = max(p.helpScroll-1, 0)
			return p, nil
		}
		if key.Code == tea.KeyEscape || key.Code == 'h' || key.Code == '?' {
			p.ShowHelp = false
			p.Title = "Lyrics"
			return p, nil
		}
		return p, nil
	}

	if p.Editing {
		runes := []rune(p.InputText)
		if key.Code == tea.KeyEnter {
			p = p.applyEditText(p.InputText)
			p.Editing = false
			p.InputText = ""
			return p, nil
		}
		if key.Code == tea.KeyEscape {
			p.Editing = false
			p.InputText = ""
			return p, nil
		}
		if key.Code == tea.KeyLeft {
			p.cursorPos = max(0, p.cursorPos-1)
			return p, nil
		}
		if key.Code == tea.KeyRight {
			p.cursorPos = min(len(runes), p.cursorPos+1)
			return p, nil
		}
		if key.Code == tea.KeyBackspace {
			if p.cursorPos > 0 {
				runes = append(runes[:p.cursorPos-1], runes[p.cursorPos:]...)
				p.InputText = string(runes)
				p.cursorPos--
			}
			return p, nil
		}
		if key.Code == tea.KeyDelete {
			if p.cursorPos < len(runes) {
				runes = append(runes[:p.cursorPos], runes[p.cursorPos+1:]...)
				p.InputText = string(runes)
			}
			return p, nil
		}
		// Split at cursor: Alt+S or Ctrl+T
		if (key.Code == 's' && key.Mod == tea.ModAlt) || (key.Code == 't' && key.Mod == tea.ModCtrl) {
			if len(p.Document.Lines()) > 0 {
				splitMS := p.tapPosition.Milliseconds()
				line := p.Document.Lines()[p.selected]
				if splitMS <= line.Start().Milliseconds() || splitMS >= line.End().Milliseconds() {
					splitMS = line.Start().Milliseconds() + (line.End().Milliseconds()-line.Start().Milliseconds())/2
				}
				p = p.apply(history.SplitLine{
					Index:     p.selected,
					SplitAtMS: splitMS,
					TextPos:   p.cursorPos,
				})
				p.Editing = false
				p.InputText = ""
			}
			return p, nil
		}
		if key.Text != "" && (key.Mod&(tea.ModCtrl|tea.ModAlt)) == 0 {
			insertRunes := []rune(key.Text)
			runes = append(runes[:p.cursorPos], append(insertRunes, runes[p.cursorPos:]...)...)
			p.InputText = string(runes)
			p.cursorPos += len(insertRunes)
		}
		return p, nil
	}

	// Normal Mode
	switch {
	case key.Code == tea.KeyDown || key.Code == 'j':
		if len(p.Document.Lines()) > 0 {
			p.selected = min(p.selected+1, len(p.Document.Lines())-1)
			return p, p.seekToSelectedCmd()
		}
	case key.Code == tea.KeyUp || key.Code == 'k':
		if len(p.Document.Lines()) > 0 {
			p.selected = max(p.selected-1, 0)
			return p, p.seekToSelectedCmd()
		}
	case key.Code == 'J' && key.Mod == tea.ModShift: // Swap text down
		if p.selected+1 < len(p.Document.Lines()) {
			p = p.apply(history.SwapText{Index: p.selected})
			p.selected++
		}
	case key.Code == 'K' && key.Mod == tea.ModShift: // Swap text up
		if p.selected > 0 {
			p = p.apply(history.SwapText{Index: p.selected - 1})
			p.selected--
		}
	case key.Code == 'e':
		if len(p.Document.Lines()) > 0 {
			p.Editing = true
			p.InputText = p.Document.Lines()[p.selected].Text().String()
			p.cursorPos = len([]rune(p.InputText))
		}
		return p, nil
	case key.Code == 'i': // Insert before selected
		idx := p.selected
		var startMS, endMS int64
		p, startMS, endMS = p.makeInsertGap(idx)
		ts, _ := lyrics.NewTimestamp(startMS)
		te, _ := lyrics.NewTimestamp(endMS)
		txt, _ := lyrics.NewText("")
		line, _ := lyrics.NewLine(ts, te, txt)
		p = p.apply(history.InsertLine{Index: idx, Line: line})
		p.selected = idx
		p.Editing = true
		p.InputText = ""
		p.cursorPos = 0
		return p, nil
	case key.Code == 'a' || key.Code == 'o' || key.Code == tea.KeyEnter: // Insert after selected
		idx := p.selected + 1
		if len(p.Document.Lines()) == 0 {
			idx = 0
		}
		var startMS, endMS int64
		p, startMS, endMS = p.makeInsertGap(idx)
		ts, _ := lyrics.NewTimestamp(startMS)
		te, _ := lyrics.NewTimestamp(endMS)
		txt, _ := lyrics.NewText("")
		line, _ := lyrics.NewLine(ts, te, txt)
		p = p.apply(history.InsertLine{Index: idx, Line: line})
		p.selected = idx
		p.Editing = true
		p.InputText = ""
		p.cursorPos = 0
		return p, nil
	case key.Code == 'd':
		if len(p.Document.Lines()) > 0 {
			p = p.apply(history.DeleteLine{Index: p.selected})
			p.selected = max(0, min(p.selected, len(p.Document.Lines())-1))
		}
		return p, nil
	case key.Code == 's': // Split line at playhead
		if len(p.Document.Lines()) > 0 {
			splitMS := p.tapPosition.Milliseconds()
			line := p.Document.Lines()[p.selected]
			if splitMS <= line.Start().Milliseconds() || splitMS >= line.End().Milliseconds() {
				splitMS = line.Start().Milliseconds() + (line.End().Milliseconds()-line.Start().Milliseconds())/2
			}
			p = p.apply(history.SplitLine{
				Index:     p.selected,
				SplitAtMS: splitMS,
				TextPos:   -1,
			})
		}
		return p, nil
	case key.Code == 'm': // Merge lines
		if len(p.Document.Lines()) > 0 {
			p = p.apply(history.MergeLines{Index: p.selected})
		}
		return p, nil
	case key.Code == 't': // Tap Sync
		if len(p.Document.Lines()) > 0 {
			p = p.apply(history.TapSync{Index: p.selected, Position: p.tapPosition})
		}
		return p, nil
	case key.Code == 'z' && key.Mod == tea.ModCtrl:
		return p.undo(), nil
	case key.Code == 'y' && key.Mod == tea.ModCtrl:
		return p.redo(), nil
	case key.Code == 'p':
		return p, func() tea.Msg {
			return StartPublishMsg{
				Lyrics: lyrics.FormatLRC(p.Document),
			}
		}
	case key.Code == 'h' || key.Code == '?':
		p.ShowHelp = true
		p.helpScroll = 0
		p.Title = "Help & Keybindings"
		return p, nil
	case key.Code == '[':
		if key.Mod == tea.ModCtrl {
			// fine nudge Start -10ms
			p = p.apply(history.NudgeStart{Index: p.selected, DeltaMS: -10})
		} else {
			// nudge Start -100ms
			p = p.apply(history.NudgeStart{Index: p.selected, DeltaMS: -100})
		}
	case key.Code == ']':
		if key.Mod == tea.ModCtrl {
			// fine nudge Start +10ms
			p = p.apply(history.NudgeStart{Index: p.selected, DeltaMS: 10})
		} else {
			// nudge Start +100ms
			p = p.apply(history.NudgeStart{Index: p.selected, DeltaMS: 100})
		}
	case key.Code == '{':
		p = p.apply(history.NudgeEnd{Index: p.selected, DeltaMS: -100})
	case key.Code == '}':
		p = p.apply(history.NudgeEnd{Index: p.selected, DeltaMS: 100})
	case key.Code == ',' && key.Mod == tea.ModCtrl:
		// fine nudge End -10ms
		p = p.apply(history.NudgeEnd{Index: p.selected, DeltaMS: -10})
	case key.Code == '.' && key.Mod == tea.ModCtrl:
		// fine nudge End +10ms
		p = p.apply(history.NudgeEnd{Index: p.selected, DeltaMS: 10})
	}
	return p, nil
}

func normalizeKeyPress(key tea.KeyPressMsg) tea.KeyPressMsg {
	if key.Code == 0 && key.Mod == 0 {
		runes := []rune(key.Text)
		if len(runes) == 1 {
			key.Code = runes[0]
		}
	}
	return key
}

type SeekToMSMsg int64

func (p Panel) seekToSelectedCmd() tea.Cmd {
	if p.selected >= 0 && p.selected < len(p.Document.Lines()) {
		line := p.Document.Lines()[p.selected]
		return func() tea.Msg {
			return SeekToMSMsg(line.Start().Milliseconds())
		}
	}
	return nil
}
