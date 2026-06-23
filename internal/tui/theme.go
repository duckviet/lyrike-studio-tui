package tui

import (
	"github.com/duckviet/lyrike-studio-tui/internal/tui/theme"
)

// Re-export theme helpers so callers can write tui.DefaultTheme().

type (
	// Theme is the canonical style bundle used by the TUI.
	Theme = theme.Theme
	// Palette is the small set of semantic colors every style is derived from.
	Palette = theme.Palette
)

// DefaultTheme returns the built-in lyrike-studio-tui theme.
func DefaultTheme() Theme {
	return theme.DefaultTheme()
}

// NewTheme builds all styles from a palette.
func NewTheme(name string, p Palette) Theme {
	return theme.NewTheme(name, p)
}

// DefaultPalette returns the built-in lyrike-studio-tui palette.
func DefaultPalette() Palette {
	return theme.DefaultPalette()
}
