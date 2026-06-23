package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/duckviet/lyrike-studio-tui/internal/tui/theme"
)

// truncate shortens s to at most w display columns, adding an ellipsis when
// cut. It is display-width and ANSI aware.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}

// spread lays out left and right text on one line of the given width, padding
// the gap between them (minimum one space).
func spread(left, right string, width int) string {
	if width <= 0 {
		return ""
	}
	rw := lipgloss.Width(right)
	if rw >= width {
		return ansi.Truncate(right, width, "")
	}
	if lipgloss.Width(left)+1+rw > width {
		budget := width - rw - 1
		if budget <= 0 {
			left = ""
		} else {
			left = ansi.Truncate(left, budget, "…")
		}
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// paneStyleWidth/paneStyleHeight are the values to pass to a bordered pane
// style's Width/Height. In lipgloss v2 those set the box's total size (border
// and padding included), so they are simply the outer size; the content area
// then works out to paneContentWidth/paneContentHeight.
func paneStyleWidth(outer int) int {
	return outer
}

func paneStyleHeight(outer int) int {
	return outer
}

func paneContentWidth(outer int) int {
	return paneInnerSize(outer, 2+2*theme.PanePaddingX)
}

func paneContentHeight(outer int) int {
	return paneInnerSize(outer, 2+2*theme.PanePaddingY)
}

func paneInnerSize(outer, frame int) int {
	if n := outer - frame; n > 0 {
		return n
	}
	return 1
}
