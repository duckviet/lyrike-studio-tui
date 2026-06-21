package waveform

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (p Panel) timeline(width int) string {
	if width <= 0 {
		return ""
	}
	cells := []rune(strings.Repeat(" ", width))
	// Mark timeline label every 12 columns
	step := 12
	for col := 0; col < width; col += step {
		ms := p.SeekForColumn(col, width)
		label := formatTimelineMS(ms)
		for i, r := range label {
			if col+i < width {
				cells[col+i] = r
			}
		}
	}
	timelineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	return timelineStyle.Render(string(cells))
}

func formatTimelineMS(ms int64) string {
	totalSec := ms / 1000
	min := totalSec / 60
	sec := totalSec % 60
	return fmt.Sprintf("%d:%02d", min, sec)
}
