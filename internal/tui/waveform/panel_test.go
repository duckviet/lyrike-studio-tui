package waveform

import (
	"regexp"
	"strings"
	"testing"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
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

func TestWaveformRenderLyricTrack(t *testing.T) {
	t.Parallel()

	start, _ := lyrics.NewTimestamp(2000)
	end, _ := lyrics.NewTimestamp(8000)
	txt, _ := lyrics.NewText("abc")
	line, _ := lyrics.NewLine(start, end, txt)

	panel := NewPanelWithPeaks([]float64{0.5}, 10_000).
		WithLines([]lyrics.Line{line}).
		WithPosition(5000)

	gotRaw := panel.renderLyricTrack(11)
	got := stripAnsi(gotRaw)
	want := "  │ abc │  "

	if got != want {
		t.Fatalf("renderLyricTrack() = %q, want %q", got, want)
	}

	if !strings.Contains(gotRaw, "\x1b[") {
		t.Fatalf("expected styled/ANSI color codes in output, got %q", gotRaw)
	}
}

func TestWaveformFollowMode(t *testing.T) {
	t.Parallel()

	panel := NewPanelWithPeaks([]float64{0.5}, 10_000)
	panel.viewStartMS = 0
	panel.viewEndMS = 5000
	panel.follow = true

	panel = panel.WithPosition(4000)

	if panel.viewStartMS != 1500 || panel.viewEndMS != 6500 {
		t.Fatalf("expected viewport [1500, 6500], got [%d, %d]", panel.viewStartMS, panel.viewEndMS)
	}

	panel = panel.ToggleFollow()
	if panel.follow {
		t.Fatal("expected follow to be false after toggle")
	}

	panel = panel.WithPosition(8000)
	if panel.viewStartMS != 1500 || panel.viewEndMS != 6500 {
		t.Fatalf("expected viewport to remain [1500, 6500], got [%d, %d]", panel.viewStartMS, panel.viewEndMS)
	}

	panel = panel.ToggleFollow()
	panel = panel.pan(500)
	if panel.follow {
		t.Fatal("expected follow to be disabled after panning")
	}
}
