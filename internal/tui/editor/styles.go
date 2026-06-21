package editor

import "charm.land/lipgloss/v2"

var (
	activeLineStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	selectedLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3366"))
)
