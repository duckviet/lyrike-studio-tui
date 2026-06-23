package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/tui/editor"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/media"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/publish"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/waveform"
)

func calculateLayout(width, height int, statusLen int) (topHeight, wfHeight, leftW, rightW, availableHeight int) {
	availableHeight = height
	if statusLen > 0 {
		availableHeight--
	}

	wfHeight = availableHeight * 2 / 5
	if wfHeight < 8 {
		wfHeight = 8
	}
	if wfHeight > 16 {
		wfHeight = 16
	}
	if availableHeight-wfHeight < 6 && availableHeight >= 14 {
		wfHeight = availableHeight - 6
	}
	topHeight = availableHeight - wfHeight

	leftW = width / 3
	rightW = width - leftW
	return
}

func renderLayout(m Model) string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	if m.fetchInput.active() {
		return renderFetchInput(m.fetchInput, m.width, m.height, m.theme)
	}
	if m.picker.active() {
		return renderProjectPicker(m.picker, m.width, m.height, m.theme)
	}

	topHeight, wfHeight, leftW, rightW, _ := calculateLayout(m.width, m.height, len(m.status))

	var left string
	if m.focus == focusPublish {
		left = renderPublishPanel(m.publish, leftW, topHeight, true, m.theme)
	} else if m.metadataEditor.active {
		left = renderMetadataEditor(m.metadataEditor, leftW, topHeight, m.theme)
	} else {
		left = renderMediaPanel(m.media, leftW, topHeight, m.focus == focusMedia, m.theme)
	}

	var right string
	right = renderLyricsPanel(m.editor, rightW, topHeight, m.focus == focusEditor, m.theme)

	bottom := renderWaveformPanel(m.waveform.WithLines(m.editor.Document.Lines()), m.width, wfHeight, m.focus == focusWaveform, m.theme)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	layout := lipgloss.JoinVertical(lipgloss.Left, topRow, bottom)

	status := m.status
	if len(status) == 0 {
		return layout
	}
	return layout + "\n" + strings.Join(status, " | ")
}

func renderMediaPanel(p media.Panel, width, height int, focused bool, th Theme) string {
	style := th.PaneInactive
	if focused {
		style = th.PaneActive
	}

	contentHeight := height - 2
	rows := []string{p.Title, p.View(width-2, contentHeight-1)}
	rows = fitRows(rows, contentHeight, width-2)

	return renderBox(style, width, rows)
}

func renderWaveformPanel(p waveform.Panel, width, height int, focused bool, th Theme) string {
	style := th.PaneInactive
	if focused {
		style = th.PaneActive
	}

	contentHeight := height - 2
	rows := []string{p.Title, p.View(width-2, contentHeight-1)}
	rows = fitRows(rows, contentHeight, width-2)

	return renderBox(style, width, rows)
}

func renderLyricsPanel(p editor.Panel, width, height int, focused bool, th Theme) string {
	style := th.PaneInactive
	if focused {
		style = th.PaneActive
	}

	contentHeight := height - 2
	rows := strings.Split(p.View(width-2, contentHeight-1), "\n")
	rows = append([]string{p.Title}, rows...)
	rows = fitRows(rows, contentHeight, width-2)

	return renderBox(style, width, rows)
}

func renderPublishPanel(p publish.Panel, width, height int, focused bool, th Theme) string {
	style := th.PaneInactive
	if focused {
		style = th.PaneActive
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
