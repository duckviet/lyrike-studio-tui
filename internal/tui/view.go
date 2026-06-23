package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/tui/editor"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/media"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/publish"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/waveform"
)

func calculateLayout(width, height int, statusLen int) (topHeight, wfHeight, leftW, rightW, availableHeight int) {
	availableHeight = height
	if availableHeight > 0 {
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

	topHeight, wfHeight, leftW, rightW, _ := calculateLayout(m.width, m.height, len(m.status))

	var left string
	if m.focus == focusPublish {
		left = renderPublishPanel(m.publish, leftW, topHeight, true, m.theme)
	} else {
		left = renderMediaPanel(m.media, leftW, topHeight, m.focus == focusMedia, m.theme)
	}

	var right string
	right = renderLyricsPanel(m.editor, rightW, topHeight, m.focus == focusEditor, m.theme)

	bottom := renderWaveformPanel(m.waveform.WithLines(m.editor.Document.Lines()), m.width, wfHeight, m.focus == focusWaveform, m.theme)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	body := lipgloss.JoinVertical(lipgloss.Left, topRow, bottom)
	if m.overlay != overlayNone {
		if overlay := renderOverlay(m, m.width, topHeight+wfHeight); overlay != "" {
			body = overlayCenter(body, overlay, m.width, topHeight+wfHeight)
		}
	}
	return body + "\n" + footerView(m, m.width)
}

func renderOverlay(m Model, width, height int) string {
	switch m.overlay {
	case overlayHelp:
		return m.help.View(width, height)
	case overlaySelector:
		return m.picker.View(width, height)
	case overlayConfirm:
		return m.confirm.View(width, height)
	case overlayInput:
		return renderFetchInput(m.fetchInput, width, height, m.theme)
	case overlayMetadata:
		return renderMetadataEditor(m.metadataEditor, width, height, m.theme)
	case overlayPublish:
		return renderPublishOverlay(m.publish, width, height, m.theme)
	case overlayNone:
		return ""
	default:
		return ""
	}
}

func renderPublishOverlay(p publish.Panel, width, height int, th Theme) string {
	boxWidth := width - 8
	if boxWidth > 56 {
		boxWidth = 56
	}
	if boxWidth < 20 {
		boxWidth = width
	}

	var sb strings.Builder
	titleStyle := th.Title
	keyStyle := th.FooterKey
	footerDescStyle := th.FooterDesc

	sb.WriteString(titleStyle.Render("Publishing Lyrics to LRCLIB") + "\n\n")
	switch p.State() {
	case publish.StateConfirm:
		sb.WriteString(th.Text.Render("Are you sure you want to publish lyrics?") + "\n\n")
		sb.WriteString(fmt.Sprintf("  %s %s\n", th.Dim.Render("Track:"), th.Value.Render(p.TrackName())))
		sb.WriteString(fmt.Sprintf("  %s %s\n\n", th.Dim.Render("Artist:"), th.Value.Render(p.ArtistName())))
		sb.WriteString(keyStyle.Render("y") + " " + footerDescStyle.Render("confirm & publish") + "   " +
			keyStyle.Render("Esc") + " " + footerDescStyle.Render("cancel"))
	case publish.StateValidate:
		sb.WriteString(th.Text.Render("  [ ] Validating lyrics...") + "\n")
	case publish.StatePoW:
		sb.WriteString(th.Good.Render("  [x] Lyrics validated") + "\n")
		sb.WriteString(th.Text.Render("  [>] Requesting challenge & solving PoW...") + "\n")
	case publish.StatePublish:
		sb.WriteString(th.Good.Render("  [x] Lyrics validated") + "\n")
		sb.WriteString(th.Good.Render("  [x] Proof-of-work solved") + "\n")
		sb.WriteString(th.Text.Render("  [>] Submitting to LRCLIB...") + "\n")
	case publish.StateDone:
		sb.WriteString(th.Good.Render("  [x] Lyrics validated") + "\n")
		sb.WriteString(th.Good.Render("  [x] Proof-of-work solved") + "\n")
		sb.WriteString(th.Good.Render("  [x] Published successfully!") + "\n\n")
		sb.WriteString(keyStyle.Render("Enter") + " " + footerDescStyle.Render("return to editor"))
	case publish.StateFailed:
		sb.WriteString(th.Bad.Render(fmt.Sprintf("  [!] Error: %v", p.Err())) + "\n\n")
		sb.WriteString(keyStyle.Render("r") + " " + footerDescStyle.Render("retry") + "   " +
			keyStyle.Render("Esc") + " " + footerDescStyle.Render("return to editor"))
	}

	return overlayBlock(sb.String(), boxWidth, th)
}

func renderMediaPanel(p media.Panel, width, height int, focused bool, th Theme) string {
	style := th.PaneInactive
	if focused {
		style = th.PaneActive
	}

	contentWidth := paneContentWidth(width)
	contentHeight := paneContentHeight(height)
	rows := strings.Split(p.View(contentWidth, contentHeight-1), "\n")
	rows = append([]string{p.Title}, rows...)
	rows = fitRows(rows, contentHeight, contentWidth)

	return renderBox(style, width, rows)
}

func renderWaveformPanel(p waveform.Panel, width, height int, focused bool, th Theme) string {
	style := th.PaneInactive
	if focused {
		style = th.PaneActive
	}

	contentWidth := paneContentWidth(width)
	contentHeight := paneContentHeight(height)
	rows := strings.Split(p.View(contentWidth, contentHeight-1), "\n")
	rows = append([]string{p.Title}, rows...)
	rows = fitRows(rows, contentHeight, contentWidth)

	return renderBox(style, width, rows)
}

func renderLyricsPanel(p editor.Panel, width, height int, focused bool, th Theme) string {
	style := th.PaneInactive
	if focused {
		style = th.PaneActive
	}

	contentWidth := paneContentWidth(width)
	contentHeight := paneContentHeight(height)
	rows := strings.Split(p.View(contentWidth, contentHeight-1), "\n")
	rows = append([]string{p.Title}, rows...)
	rows = fitRows(rows, contentHeight, contentWidth)

	return renderBox(style, width, rows)
}

func renderPublishPanel(p publish.Panel, width, height int, focused bool, th Theme) string {
	style := th.PaneInactive
	if focused {
		style = th.PaneActive
	}

	contentWidth := paneContentWidth(width)
	contentHeight := paneContentHeight(height)
	rows := strings.Split(p.View(contentWidth, contentHeight), "\n")
	rows = fitRows(rows, contentHeight, contentWidth)

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
