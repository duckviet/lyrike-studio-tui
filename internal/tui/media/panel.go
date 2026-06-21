package media

import (
	"fmt"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type TransportState string

const (
	TransportPaused  TransportState = "paused"
	TransportPlaying TransportState = "playing"
)

type Panel struct {
	Title      string
	state      TransportState
	positionMS int64
	durationMS int64
	trackName  string
	artistName string
	albumName  string
}

func NewPanel() Panel {
	return Panel{
		Title:      "Media",
		state:      TransportPaused,
		durationMS: 10_000,
	}
}

func (p Panel) WithTransport(state TransportState, positionMS int64, durationMS int64) Panel {
	p.state = state
	p.durationMS = max(durationMS, 1)
	p.positionMS = clamp(positionMS, 0, p.durationMS)
	return p
}

func (p Panel) WithMetadata(track, artist, album string) Panel {
	p.trackName = track
	p.artistName = artist
	p.albumName = album
	return p
}

// ProgressBarRow returns the 0-based row index of the progress bar within the
// panel's content area (excluding the border). The caller adds the border offset.
// Layout (each item = 1 row): Title, Track, Artist, Album, Spacing, Status, ProgressBar
// Index inside the View() string:  0=Track,1=Artist,2=Album,3="",4=Status,5=ProgressBar
// In renderMediaPanel rows[0]=Title, rows[1]=View content starting at Track.
// So progress bar is at row index 1 (title) + 5 (lines before bar in View) + 1 (border top) = 7.
const ProgressBarRow = 7 // absolute row inside panel border

// SeekForX converts an X position (0-based, within the panel inner content)
// to a playback position in milliseconds.
func (p Panel) SeekForX(x, contentWidth int) int64 {
	if contentWidth <= 1 || p.durationMS <= 0 {
		return 0
	}
	// The progress bar layout: "HH:MM.SS " (startLabel + space) ... bar ... " HH:MM.SS"
	startLabel := formatMillis(p.positionMS)
	endLabel := formatMillis(p.durationMS)
	labelOffset := len(startLabel) + 1 // startLabel + one space
	barWidth := contentWidth - len(startLabel) - len(endLabel) - 2
	if barWidth < 1 {
		return 0
	}
	barX := x - labelOffset
	if barX < 0 {
		barX = 0
	}
	if barX > barWidth {
		barX = barWidth
	}
	return p.durationMS * int64(barX) / int64(barWidth)
}

func (p Panel) Update(_ tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

func (p Panel) View(width int, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	if height < 6 {
		// single line fallback
		return fmt.Sprintf("%s %s / %s  [space] play/pause  [←/→] seek  [l] loop", p.icon(), formatMillis(p.positionMS), formatMillis(p.durationMS))
	}

	track := p.trackName
	if track == "" {
		track = "No Track Loaded"
	}
	artist := p.artistName
	if artist == "" {
		artist = "Unknown Artist"
	}
	album := p.albumName
	if album == "" {
		album = "Unknown Album"
	}

	trackStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	artistStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	albumStyle := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#666666"))

	stateStr := "Paused"
	if p.state == TransportPlaying {
		stateStr = "Playing"
	}
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3366"))
	statusLine := fmt.Sprintf("%s %s", p.icon(), statusStyle.Render(stateStr))

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	shortcuts := []string{
		fmt.Sprintf("%s %s", keyStyle.Render("[Space]"), descStyle.Render("Play/Pause")),
		fmt.Sprintf("%s %s", keyStyle.Render("[←] / [→]"), descStyle.Render("Seek 1s")),
		fmt.Sprintf("%s %s", keyStyle.Render("[l]"), descStyle.Render("Toggle Loop")),
	}

	var lines []string
	lines = append(lines, trackStyle.Render(track))
	lines = append(lines, artistStyle.Render(artist))
	lines = append(lines, albumStyle.Render(album))
	lines = append(lines, "") // spacing
	lines = append(lines, statusLine)
	lines = append(lines, p.progressBar(width))

	contentRowsCount := len(lines) + len(shortcuts) + 1
	paddingNeeded := height - contentRowsCount
	for i := 0; i < paddingNeeded; i++ {
		lines = append(lines, "")
	}

	lines = append(lines, descStyle.Render("Shortcuts:"))
	lines = append(lines, shortcuts...)

	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func (p Panel) progressBar(width int) string {
	if p.durationMS <= 0 || width <= 10 {
		return ""
	}
	startLabel := formatMillis(p.positionMS)
	endLabel := formatMillis(p.durationMS)

	// space for the progress bar itself: subtract len of labels and padding spaces
	barWidth := width - len(startLabel) - len(endLabel) - 2
	if barWidth < 3 {
		return fmt.Sprintf("%s / %s", startLabel, endLabel)
	}

	percent := float64(p.positionMS) / float64(p.durationMS)
	filledWidth := int(math.Round(percent * float64(barWidth)))
	if filledWidth < 0 {
		filledWidth = 0
	}
	if filledWidth > barWidth {
		filledWidth = barWidth
	}

	var playedPart, knob, unplayedPart string

	playedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	unplayedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	knobStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3366"))

	if filledWidth > 0 {
		playedPart = playedStyle.Render(strings.Repeat("━", filledWidth-1))
		knob = knobStyle.Render("●")
		unfilledWidth := barWidth - filledWidth
		if unfilledWidth > 0 {
			unplayedPart = unplayedStyle.Render(strings.Repeat("─", unfilledWidth))
		}
	} else {
		playedPart = ""
		knob = knobStyle.Render("○")
		unplayedPart = unplayedStyle.Render(strings.Repeat("─", barWidth-1))
	}

	return fmt.Sprintf("%s %s%s%s %s", startLabel, playedPart, knob, unplayedPart, endLabel)
}

func (p Panel) icon() string {
	if p.state == TransportPlaying {
		return "▶"
	}
	return "⏸"
}

func formatMillis(milliseconds int64) string {
	totalSeconds := milliseconds / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	centiseconds := (milliseconds % 1000) / 10
	return fmt.Sprintf("%02d:%02d.%02d", minutes, seconds, centiseconds)
}

func clamp(value int64, minimum int64, maximum int64) int64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

