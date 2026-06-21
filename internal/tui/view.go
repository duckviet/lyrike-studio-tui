package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/tui/editor"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/media"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/publish"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/waveform"
)

var (
	focusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4"))

	normalBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#555555"))
)

func renderLayout(m Model) string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	availableHeight := m.height
	if len(m.status) > 0 {
		availableHeight--
	}

	wfHeight := availableHeight / 3
	if wfHeight < 6 {
		wfHeight = 6
	}
	if wfHeight > 12 {
		wfHeight = 12
	}
	topHeight := availableHeight - wfHeight

	leftW := m.width / 3
	rightW := m.width - leftW

	left := renderMediaPanel(m.media, leftW, topHeight, m.focus == focusMedia)

	var right string
	if m.focus == focusPublish {
		right = renderPublishPanel(m.publish, rightW, topHeight, true)
	} else {
		right = renderLyricsPanel(m.editor, rightW, topHeight, m.focus == focusEditor)
	}

	bottom := renderWaveformPanel(m.waveform, m.width, wfHeight, m.focus == focusWaveform)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	layout := lipgloss.JoinVertical(lipgloss.Left, topRow, bottom)

	status := m.status
	if len(status) == 0 {
		return layout
	}
	return layout + "\n" + strings.Join(status, " | ")
}

func renderMediaPanel(p media.Panel, width, height int, focused bool) string {
	style := normalBorder
	if focused {
		style = focusedBorder
	}

	contentHeight := height - 2
	rows := []string{p.Title, p.View(width-2, contentHeight-1)}
	rows = fitRows(rows, contentHeight, width-2)

	return renderBox(style, width, rows)
}

func renderWaveformPanel(p waveform.Panel, width, height int, focused bool) string {
	style := normalBorder
	if focused {
		style = focusedBorder
	}

	contentHeight := height - 2
	rows := []string{p.Title, p.View(width-2, contentHeight-1)}
	rows = fitRows(rows, contentHeight, width-2)

	return renderBox(style, width, rows)
}

func renderLyricsPanel(p editor.Panel, width, height int, focused bool) string {
	style := normalBorder
	if focused {
		style = focusedBorder
	}

	contentHeight := height - 2
	rows := strings.Split(p.View(width-2, contentHeight-1), "\n")
	rows = append([]string{p.Title}, rows...)
	rows = fitRows(rows, contentHeight, width-2)

	return renderBox(style, width, rows)
}

func renderPublishPanel(p publish.Panel, width, height int, focused bool) string {
	style := normalBorder
	if focused {
		style = focusedBorder
	}

	contentHeight := height - 2
	rows := strings.Split(p.View(width-2, contentHeight), "\n")
	rows = fitRows(rows, contentHeight, width-2)

	return renderBox(style, width, rows)
}

func renderBox(style lipgloss.Style, width int, rows []string) string {
	return style.
		Width(width).
		Render(strings.Join(rows, "\n"))
}

func fitRows(rows []string, maxRows, maxWidth int) []string {
	var fitted []string
	for _, row := range rows {
		if len(fitted) >= maxRows {
			break
		}
		wrapped := lipgloss.Wrap(row, maxWidth, "")
		for _, line := range strings.Split(wrapped, "\n") {
			if len(fitted) >= maxRows {
				break
			}
			fitted = append(fitted, line)
		}
	}
	for len(fitted) < maxRows {
		fitted = append(fitted, "")
	}
	return fitted
}
