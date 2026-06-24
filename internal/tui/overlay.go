package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type overlayKind int

const (
	overlayNone overlayKind = iota
	overlaySelector
	overlayHelp
	overlayConfirm
	overlayInput
	overlayMetadata
	overlayPublish
)

func overlayBlock(content string, width int, th Theme) string {
	if width < 0 {
		width = 0
	}
	painted := th.PaintModal(content)
	return th.ModalBorder.Width(width).Render(painted)
}

func overlayCenter(base, box string, width, height int) string {
	x := (width - lipgloss.Width(box)) / 2
	y := (height - lipgloss.Height(box)) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return overlayBlockCompositor(base, box, x, y, width, height)
}

func clampBlock(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], width, "")
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func overlayBlockCompositor(base, overlay string, x, y, width, height int) string {
	baseLines := strings.Split(clampBlock(base, width, height), "\n")

	for i, overlayLine := range strings.Split(overlay, "\n") {
		row := y + i
		if row >= height {
			continue
		}

		line := overlayLine
		if x >= width {
			continue
		}
		if w := lipgloss.Width(line); w > width-x {
			line = ansi.Truncate(line, width-x, "")
		}
		lineW := lipgloss.Width(line)
		if lineW == 0 {
			continue
		}

		left := ansi.Cut(baseLines[row], 0, x)
		if pad := x - lipgloss.Width(left); pad > 0 {
			left += strings.Repeat(" ", pad)
		}

		right := ""
		if start := x + lineW; start < width {
			right = ansi.Cut(baseLines[row], start, width)
		}
		baseLines[row] = left + line + right
	}

	return strings.Join(baseLines, "\n")
}
