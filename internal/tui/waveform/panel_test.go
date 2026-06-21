package waveform

import (
	"regexp"
	"strings"
	"testing"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsi(str string) string {
	return ansiRegexp.ReplaceAllString(str, "")
}

func TestWaveformMapsPeaksToCells(t *testing.T) {
	t.Parallel()

	panel := NewPanelWithPeaks([]float64{0, 0.2, 0.4, 0.7, 1}, 10_000).
		WithHover(0)

	got := stripAnsi(panel.View(5, 1))

	if got != "┃░▒▓█" {
		t.Fatalf("View() = %q, want %q", got, "┃░▒▓█")
	}
}

func TestWaveformMapsSeekColumnToPosition(t *testing.T) {
	t.Parallel()

	panel := NewPanelWithPeaks([]float64{1}, 10_000)

	got := panel.SeekForColumn(5, 11)

	if got != 5_000 {
		t.Fatalf("SeekForColumn() = %d, want 5000", got)
	}
}

func TestWaveformClampsLoopBounds(t *testing.T) {
	t.Parallel()

	panel := NewPanelWithPeaks([]float64{0, 0, 0, 0, 0}, 10_000).
		WithLoop(-1_000, 20_000).
		WithPosition(20_000).
		WithHover(4)

	got := stripAnsi(panel.View(5, 1))

	if !strings.HasPrefix(got, "────") {
		t.Fatalf("View() = %q, want clamped loop marker across waveform", got)
	}
	if !strings.HasSuffix(got, "┃") {
		t.Fatalf("View() = %q, want position clamped to the final cell", got)
	}
}
