package waveform

import "charm.land/lipgloss/v2"

var (
	cursorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3366"))
	lyricActiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#7D56F4")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)
	lyricInactiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#2C2C2C")).
				Foreground(lipgloss.Color("#9E9E9E"))
)
