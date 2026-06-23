package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type hint struct {
	key  string
	desc string
}

func (m Model) hints() []hint {
	if m.overlay == overlayMetadata {
		return []hint{
			{key: "Tab/↑/↓", desc: "field"},
			{key: "Enter", desc: "save"},
			{key: "Esc", desc: "cancel"},
		}
	}
	if m.overlay == overlayPublish {
		switch m.publish.State() {
		case "confirm":
			return []hint{
				{key: "y", desc: "confirm"},
				{key: "Esc", desc: "cancel"},
			}
		case "done":
			return []hint{
				{key: "Enter", desc: "close"},
			}
		case "failed":
			return []hint{
				{key: "r", desc: "retry"},
				{key: "Esc", desc: "close"},
			}
		default:
			return []hint{}
		}
	}
	if m.overlay != overlayNone {
		return []hint{
			{key: "Esc", desc: "close"},
			{key: "Enter", desc: "confirm"},
		}
	}

	common := []hint{
		{key: "Tab", desc: "focus"},
		{key: "Ctrl+O", desc: "fetch"},
		{key: "Ctrl+P", desc: "projects"},
		{key: "q", desc: "quit"},
	}

	switch m.focus {
	case focusMedia:
		return append([]hint{{key: "Space", desc: "play/pause"}}, common...)
	case focusWaveform:
		return append([]hint{{key: "←/→", desc: "seek"}}, common...)
	case focusEditor:
		return append([]hint{{key: "Enter", desc: "edit line"}}, common...)
	case focusPublish:
		return append([]hint{{key: "Enter", desc: "publish"}}, common...)
	default:
		return common
	}
}

func footerView(m Model, width int) string {
	if width <= 0 {
		return ""
	}

	left := renderHints(m.hints(), m.theme)
	right := m.status
	if right != "" {
		style := m.theme.StatusOK
		if m.statusErr {
			style = m.theme.StatusErr
		}
		right = style.Render(right)
	}

	footer := spread(left, right, width)
	if lipgloss.Width(footer) > width {
		footer = truncate(footer, width)
	}
	return footer
}

func renderHints(hints []hint, th Theme) string {
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		if h.key == "" || h.desc == "" {
			continue
		}
		parts = append(parts, th.FooterKey.Render(h.key)+" "+th.FooterDesc.Render(h.desc))
	}
	return strings.Join(parts, "  ")
}
