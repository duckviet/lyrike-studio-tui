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
	m.overlay = overlayMetadata
	m.setStatus("metadata editor open")
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
			m.overlay = overlayNone
			m.setStatus("metadata editing canceled")
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
			m.overlay = overlayNone
			m.dirty = true
			m.setStatus("metadata updated")
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
	boxWidth := width - 8
	if boxWidth > 56 {
		boxWidth = 56
	}
	if boxWidth < 20 {
		boxWidth = width
	}

	titleStyle := th.Modal(th.Title)
	labelStyle := th.Modal(th.Dim)
	activeStyle := th.Prompt
	valueStyle := th.Modal(th.Value)
	hintStyle := th.Modal(th.Dim)

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

	lines = append(lines, hintStyle.Render("Tab/↑↓: switch field"))
	lines = append(lines, hintStyle.Render("Enter: save  Esc: cancel"))

	content := strings.Join(lines, "\n")
	return overlayBlock(content, boxWidth, th)
}
