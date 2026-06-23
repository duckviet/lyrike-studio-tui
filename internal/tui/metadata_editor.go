package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

type metadataEditor struct {
	active     bool
	trackName  string
	artistName string
	albumName  string
	focus      int // 0: Track Name, 1: Artist, 2: Album
}

func (m Model) openMetadataEditor() Model {
	m.metadataEditor = metadataEditor{
		active:     true,
		trackName:  m.trackName,
		artistName: m.artistName,
		albumName:  m.albumName,
		focus:      0,
	}
	m.status = []string{"metadata editor open"}
	return m
}

func (m Model) updateMetadataEditor(msg tea.Msg) (Model, tea.Cmd) {
	me := m.metadataEditor

	switch msg := msg.(type) {
	case tea.PasteMsg:
		if msg.Content != "" {
			switch me.focus {
			case 0:
				me.trackName += msg.Content
			case 1:
				me.artistName += msg.Content
			case 2:
				me.albumName += msg.Content
			}
			m.metadataEditor = me
		}
		return m, nil

	case tea.KeyPressMsg:
		msg = normalizeKeyPress(msg)
		switch {
		case msg.Code == tea.KeyEscape:
			m.metadataEditor = metadataEditor{}
			m.status = []string{"metadata editing canceled"}
			return m, nil

		case msg.Code == tea.KeyTab && msg.Mod != tea.ModShift, msg.Code == tea.KeyDown:
			me.focus = (me.focus + 1) % 3
			m.metadataEditor = me
			return m, nil

		case (msg.Code == tea.KeyTab && msg.Mod == tea.ModShift), msg.Code == tea.KeyUp:
			me.focus = (me.focus - 1 + 3) % 3
			m.metadataEditor = me
			return m, nil

		case msg.Code == tea.KeyEnter:
			m.trackName = strings.TrimSpace(me.trackName)
			m.artistName = strings.TrimSpace(me.artistName)
			m.albumName = strings.TrimSpace(me.albumName)
			m.media = m.media.WithMetadata(m.trackName, m.artistName, m.albumName)
			m.metadataEditor = metadataEditor{}
			m.dirty = true
			m.status = []string{"metadata updated"}
			return m, nil

		case msg.Code == tea.KeyBackspace:
			switch me.focus {
			case 0:
				if len(me.trackName) > 0 {
					runes := []rune(me.trackName)
					me.trackName = string(runes[:len(runes)-1])
				}
			case 1:
				if len(me.artistName) > 0 {
					runes := []rune(me.artistName)
					me.artistName = string(runes[:len(runes)-1])
				}
			case 2:
				if len(me.albumName) > 0 {
					runes := []rune(me.albumName)
					me.albumName = string(runes[:len(runes)-1])
				}
			}
			m.metadataEditor = me
			return m, nil

		default:
			if msg.Text != "" {
				switch me.focus {
				case 0:
					me.trackName += msg.Text
				case 1:
					me.artistName += msg.Text
				case 2:
					me.albumName += msg.Text
				}
				m.metadataEditor = me
			}
			return m, nil
		}
	}
	return m, nil
}

func renderMetadataEditor(me metadataEditor, width, height int, th Theme) string {
	innerW := max(0, width-4) // account for border (2) + padding (2)

	titleStyle := th.Title
	labelStyle := th.Dim
	activeStyle := th.Prompt
	valueStyle := th.Value
	hintStyle := th.Dim

	type field struct {
		label string
		val   string
	}
	fields := []field{
		{"Track Name", me.trackName},
		{"Artist    ", me.artistName},
		{"Album     ", me.albumName},
	}

	var lines []string
	lines = append(lines, titleStyle.Render("Edit Metadata"))
	lines = append(lines, "")

	for i, f := range fields {
		if i == me.focus {
			label := activeStyle.Render("> " + f.label + ":")
			val := valueStyle.Render(f.val) + activeStyle.Render("|")
			lines = append(lines, label)
			lines = append(lines, "  "+val)
		} else {
			label := labelStyle.Render("  " + f.label + ":")
			val := valueStyle.Render(f.val)
			lines = append(lines, label)
			lines = append(lines, "  "+val)
		}
		lines = append(lines, "")
	}

	// Fill remaining height with empty lines, leaving 2 at the bottom for hint
	contentH := height - 2 // inside border
	hintLines := 2
	fillerCount := contentH - len(lines) - hintLines
	for i := 0; i < fillerCount; i++ {
		lines = append(lines, "")
	}
	lines = append(lines, hintStyle.Render("Tab/↑↓: switch field"))
	lines = append(lines, hintStyle.Render("Enter: save  Esc: cancel"))

	content := strings.Join(lines, "\n")
	return th.PaneActive.
		Width(max(0, width-2)).
		Height(max(0, height-2)).
		Render(th.Value.Width(innerW).Render(content))
}
