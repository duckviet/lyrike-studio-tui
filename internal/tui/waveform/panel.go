package waveform

import (
	"fmt"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

type Panel struct {
	Title       string
	peaks       []float64
	positionMS  int64
	durationMS  int64
	loopStartMS int64
	loopEndMS   int64

	// Viewport bounds
	viewStartMS int64
	viewEndMS   int64

	// Last known width
	width int

	hoverCol int
	lines    []lyrics.Line
	follow   bool
}

var (
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3366"))
	lyricActiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#7D56F4")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)
	lyricInactiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#2C2C2C")).
				Foreground(lipgloss.Color("#9E9E9E"))
)

func NewPanel() Panel {
	p := Panel{
		Title:       "Waveform",
		peaks:       []float64{0.15, 0.4, 0.8, 1, 0.6, 0.3},
		durationMS:  10_000,
		viewStartMS: 0,
		viewEndMS:   10_000,
		hoverCol:    -1,
		follow:      true,
	}
	return p.updateTitle()
}

func NewPanelWithPeaks(peaks []float64, durationMS int64) Panel {
	panel := NewPanel()
	panel.peaks = append([]float64(nil), peaks...)
	panel.durationMS = max(durationMS, 1)
	panel.viewStartMS = 0
	panel.viewEndMS = panel.durationMS
	return panel.updateTitle()
}

func (p Panel) WithPosition(positionMS int64) Panel {
	p.positionMS = clamp(positionMS, 0, p.durationMS)
	if p.follow {
		p = p.centerViewportOnPosition()
	}
	return p.updateTitle()
}

func (p Panel) WithLoop(startMS int64, endMS int64) Panel {
	start := clamp(startMS, 0, p.durationMS)
	end := clamp(endMS, start, p.durationMS)
	p.loopStartMS = start
	p.loopEndMS = end
	return p.updateTitle()
}

func (p Panel) WithWidth(w int) Panel {
	p.width = w
	return p.updateTitle()
}

func (p Panel) WithHover(col int) Panel {
	p.hoverCol = col
	return p
}

func (p Panel) WithLines(lines []lyrics.Line) Panel {
	p.lines = lines
	return p
}

func (p Panel) viewSpanMS() int64 {
	span := p.viewEndMS - p.viewStartMS
	if span < 1 {
		return 1
	}
	return span
}

func (p Panel) SeekForColumn(column int, width int) int64 {
	if width <= 1 {
		return p.viewStartMS
	}
	c := clamp(int64(column), 0, int64(width-1))
	return p.viewStartMS + c*p.viewSpanMS()/int64(width-1)
}

func (p Panel) columnFor(ms int64, width int) int {
	if width <= 1 || p.viewSpanMS() <= 0 {
		return 0
	}
	rel := ms - p.viewStartMS
	return int(rel * int64(width-1) / p.viewSpanMS())
}

func (p Panel) peakForColumn(column int, width int) float64 {
	if len(p.peaks) == 0 {
		return 0
	}
	if len(p.peaks) == 1 {
		return p.peaks[0]
	}
	ms := p.SeekForColumn(column, width)

	// Map ms -> float index in peaks
	frac := float64(ms) / float64(p.durationMS)
	floatIdx := frac * float64(len(p.peaks)-1)

	idx := int(math.Floor(floatIdx))
	if idx < 0 {
		return p.peaks[0]
	}
	if idx >= len(p.peaks)-1 {
		return p.peaks[len(p.peaks)-1]
	}

	// Linear interpolation to make zoom smooth (no steps)
	nextFrac := floatIdx - float64(idx)
	return p.peaks[idx]*(1.0-nextFrac) + p.peaks[idx+1]*nextFrac
}

const minSpanMS = 1000 // 1 second minimum zoom

func (p Panel) zoomAt(mouseCol int, factor float64) Panel {
	if p.width <= 1 {
		return p
	}
	if mouseCol < 0 {
		mouseCol = 0
	} else if mouseCol > p.width-1 {
		mouseCol = p.width - 1
	}

	// Time at mouse position remains fixed
	anchorMS := p.SeekForColumn(mouseCol, p.width)
	anchorFrac := float64(mouseCol) / float64(p.width-1)

	newSpan := int64(float64(p.viewSpanMS()) * factor)
	newSpan = clamp(newSpan, minSpanMS, p.durationMS)

	newStart := anchorMS - int64(anchorFrac*float64(newSpan))
	newEnd := newStart + newSpan

	if newStart < 0 {
		newStart = 0
		newEnd = newSpan
	}
	if newEnd > p.durationMS {
		newEnd = p.durationMS
		newStart = newEnd - newSpan
	}
	p.viewStartMS = clamp(newStart, 0, p.durationMS)
	p.viewEndMS = clamp(newEnd, p.viewStartMS+1, p.durationMS)
	return p.updateTitle()
}

func (p Panel) pan(deltaMS int64) Panel {
	p.follow = false
	span := p.viewSpanMS()
	newStart := p.viewStartMS + deltaMS
	if newStart < 0 {
		newStart = 0
	}
	if newStart+span > p.durationMS {
		newStart = p.durationMS - span
	}
	p.viewStartMS = clamp(newStart, 0, p.durationMS)
	p.viewEndMS = clamp(newStart+span, p.viewStartMS+1, p.durationMS)
	return p.updateTitle()
}

func (p Panel) HandleMouseLocal(localX int, button tea.MouseButton, mod tea.KeyMod) Panel {
	if p.width <= 0 {
		return p
	}

	switch button {
	case tea.MouseWheelUp, tea.MouseWheelDown:
		if mod&tea.ModCtrl != 0 || mod&tea.ModAlt != 0 {
			factor := 0.85
			if button == tea.MouseWheelDown {
				factor = 1.0 / 0.85
			}
			p = p.zoomAt(localX, factor)
		} else {
			panBy := p.viewSpanMS() / 8
			if button == tea.MouseWheelDown {
				p = p.pan(panBy)
			} else {
				p = p.pan(-panBy)
			}
		}
	case tea.MouseWheelLeft, tea.MouseWheelRight:
		panBy := p.viewSpanMS() / 8
		if button == tea.MouseWheelRight {
			p = p.pan(panBy)
		} else {
			p = p.pan(-panBy)
		}
	}
	return p
}

func (p Panel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Code {
		case '+', '=':
			p = p.zoomAt(p.columnFor(p.positionMS, p.width), 0.85)
		case '-':
			p = p.zoomAt(p.columnFor(p.positionMS, p.width), 1/0.85)
		case '0':
			p.viewStartMS = 0
			p.viewEndMS = p.durationMS
			p = p.updateTitle()
		case 'f', 'F':
			p = p.ToggleFollow()
		}
	}
	return p, nil
}

func (p Panel) zoomStatus() string {
	if p.durationMS <= 0 {
		return "100%"
	}

	const mapSize = 8
	startFrac := float64(p.viewStartMS) / float64(p.durationMS)
	spanFrac := float64(p.viewSpanMS()) / float64(p.durationMS)

	startIdx := int(math.Round(startFrac * mapSize))
	spanLen := int(math.Round(spanFrac * mapSize))
	if spanLen < 1 {
		spanLen = 1
	}
	endIdx := startIdx + spanLen
	if endIdx > mapSize {
		endIdx = mapSize
		startIdx = endIdx - spanLen
	}

	var sb strings.Builder
	for i := 0; i < mapSize; i++ {
		if i >= startIdx && i < endIdx {
			sb.WriteRune('█')
		} else {
			sb.WriteRune('░')
		}
	}

	startStr := formatTimelineMS(p.viewStartMS)
	endStr := formatTimelineMS(p.viewEndMS)
	sec := p.viewSpanMS() / 1000

	return fmt.Sprintf("%s %s–%s (%ds)", sb.String(), startStr, endStr, sec)
}

func (p Panel) updateTitle() Panel {
	followStr := ""
	if p.follow {
		followStr = " [Follow]"
	}
	if p.viewSpanMS() == p.durationMS {
		p.Title = "Waveform" + followStr
	} else {
		p.Title = fmt.Sprintf("Waveform [%s]%s", p.zoomStatus(), followStr)
	}
	return p
}

func (p Panel) ToggleFollow() Panel {
	p.follow = !p.follow
	p = p.updateTitle()
	if p.follow {
		p = p.centerViewportOnPosition()
	}
	return p
}

func (p Panel) centerViewportOnPosition() Panel {
	span := p.viewSpanMS()
	newStart := p.positionMS - span/2
	newEnd := newStart + span

	if newStart < 0 {
		newStart = 0
		newEnd = span
	}
	if newEnd > p.durationMS {
		newEnd = p.durationMS
		newStart = newEnd - span
	}

	p.viewStartMS = clamp(newStart, 0, p.durationMS)
	p.viewEndMS = clamp(newEnd, p.viewStartMS+1, p.durationMS)
	return p
}

// View renders a symmetric, multi-row waveform mirrored around a center axis.
// height should be the total height of the panel's content area. We output exactly
// height lines. The last line is the timeline; the rest is the waveform grid
// centered vertically.
func (p Panel) View(width int, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	p.width = width
	if height == 1 {
		return p.viewSingleRow(width)
	}

	// We reserve 1 line for the timeline and 1 line for the lyric track if showing.
	showLyricTrack := height >= 4
	var wfHeight int
	if showLyricTrack {
		wfHeight = height - 2
	} else {
		wfHeight = height - 1
	}

	var topPadding int
	if wfHeight > 1 && wfHeight%2 == 0 {
		wfHeight--
		topPadding = 1
	}
	half := wfHeight / 2 // rows above (and below) the center

	// Precompute per-column data.
	cursor := p.columnFor(p.positionMS, width)
	loopStart, loopEnd := -1, -1
	if p.loopEndMS > p.loopStartMS {
		loopStart = p.columnFor(p.loopStartMS, width)
		loopEnd = p.columnFor(p.loopEndMS, width)
	}

	grid := make([][]string, wfHeight)
	for r := range grid {
		grid[r] = make([]string, width)
		for c := range grid[r] {
			grid[r][c] = " "
		}
	}

	for col := 0; col < width; col++ {
		peak := math.Max(0, math.Min(1, p.peakForColumn(col, width)))

		// Total vertical extent in "sub-cells" (8 sub-levels per cell).
		filled := peak * float64(half)
		fullCells := int(filled)
		remainder := filled - float64(fullCells)

		inLoop := loopStart >= 0 && col >= loopStart && col <= loopEnd
		isHover := col == p.hoverCol && p.hoverCol >= 0
		isPlayed := col < cursor

		// Determine base color based on played, loop, and amplitude
		var colColor string
		if isPlayed {
			if inLoop {
				colColor = lerpColor("#4338CA", "#818CF8", peak)
			} else {
				colColor = lerpColor("#5B21B6", "#C084FC", peak)
			}
		} else {
			if inLoop {
				colColor = lerpColor("#4B5563", "#9CA3AF", peak)
			} else {
				colColor = lerpColor("#3F3F46", "#8E9196", peak)
			}
		}
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(colColor))

		for r := 0; r < wfHeight; r++ {
			dist := r - half // distance from center; negative = above
			isTop := dist < 0
			if dist < 0 {
				dist = -dist
			}

			var glyph rune
			switch {
			case dist < fullCells:
				glyph = '█'
			case dist == fullCells && remainder > 0:
				if isTop {
					glyph = upGlyph(remainder)
				} else {
					glyph = downGlyph(remainder)
				}
			case r == half:
				// center axis line where there's no bar
				if inLoop {
					glyph = '━'
				} else {
					glyph = '─'
				}
			default:
				glyph = ' '
			}

			if isHover {
				glyph = '┃'
			}

			cellStr := string(glyph)
			if glyph != ' ' {
				if isHover {
					cellStr = cursorStyle.Render(cellStr)
				} else {
					cellStr = style.Render(cellStr)
				}
			}

			grid[r][col] = cellStr
		}
	}

	var b strings.Builder
	// Top padding
	for i := 0; i < topPadding; i++ {
		b.WriteString(strings.Repeat(" ", width))
		b.WriteByte('\n')
	}
	// Waveform grid
	for r := 0; r < wfHeight; r++ {
		b.WriteString(strings.Join(grid[r], ""))
		b.WriteByte('\n')
	}
	// Timeline and Lyric track
	if showLyricTrack {
		b.WriteString(p.timeline(width))
		b.WriteByte('\n')
		b.WriteString(p.renderLyricTrack(width))
	} else {
		b.WriteString(p.timeline(width))
	}
	return b.String()
}

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

func (p Panel) viewSingleRow(width int) string {
	if len(p.peaks) == 0 {
		return strings.Repeat("·", width)
	}

	cursor := p.columnFor(p.positionMS, width)
	loopStart, loopEnd := -1, -1
	if p.loopEndMS > p.loopStartMS {
		loopStart = p.columnFor(p.loopStartMS, width)
		loopEnd = p.columnFor(p.loopEndMS, width)
	}

	cells := make([]string, width)
	for col := range width {
		peak := p.peakForColumn(col, width)
		glyph := glyphForPeak(peak)

		inLoop := loopStart >= 0 && col >= loopStart && col <= loopEnd
		isHover := col == p.hoverCol && p.hoverCol >= 0
		isPlayed := col < cursor

		if inLoop && glyph == '·' {
			glyph = '─'
		}
		if isHover {
			glyph = '┃'
		}

		var colColor string
		if isPlayed {
			if inLoop {
				colColor = lerpColor("#4338CA", "#818CF8", peak)
			} else {
				colColor = lerpColor("#5B21B6", "#C084FC", peak)
			}
		} else {
			if inLoop {
				colColor = lerpColor("#4B5563", "#9CA3AF", peak)
			} else {
				colColor = lerpColor("#3F3F46", "#8E9196", peak)
			}
		}

		cellStr := string(glyph)
		if isHover {
			cellStr = cursorStyle.Render(cellStr)
		} else {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(colColor))
			cellStr = style.Render(cellStr)
		}
		cells[col] = cellStr
	}
	return strings.Join(cells, "")
}

func lerpColor(startHex, endHex string, frac float64) string {
	startHex = strings.TrimPrefix(startHex, "#")
	endHex = strings.TrimPrefix(endHex, "#")

	var r1, g1, b1, r2, g2, b2 int
	fmt.Sscanf(startHex, "%02x%02x%02x", &r1, &g1, &b1)
	fmt.Sscanf(endHex, "%02x%02x%02x", &r2, &g2, &b2)

	r := int(float64(r1) + frac*float64(r2-r1))
	g := int(float64(g1) + frac*float64(g2-g1))
	b := int(float64(b1) + frac*float64(b2-b1))

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

var blocksUp = []rune{' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func upGlyph(frac float64) rune {
	idx := int(math.Round(frac * 7))
	if idx < 0 {
		idx = 0
	}
	if idx > 7 {
		idx = 7
	}
	return blocksUp[idx]
}

func downGlyph(frac float64) rune {
	switch {
	case frac >= 0.75:
		return '█'
	case frac >= 0.33:
		return '▀'
	case frac > 0:
		return '▔'
	default:
		return ' '
	}
}

func glyphForPeak(peak float64) rune {
	normalized := math.Max(0, math.Min(1, peak))
	switch {
	case normalized >= 0.75:
		return '█'
	case normalized >= 0.5:
		return '▓'
	case normalized >= 0.25:
		return '▒'
	case normalized > 0:
		return '░'
	default:
		return '·'
	}
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

func (p Panel) renderLyricTrack(width int) string {
	if width <= 0 {
		return ""
	}

	type lyricCell struct {
		r        rune
		active   bool
		hasBlock bool
	}

	cells := make([]lyricCell, width)
	for i := range cells {
		cells[i] = lyricCell{r: ' ', active: false, hasBlock: false}
	}

	for _, line := range p.lines {
		startMS := line.Start().Milliseconds()
		endMS := line.End().Milliseconds()

		if endMS <= startMS {
			continue
		}

		startCol := p.columnFor(startMS, width)
		endCol := p.columnFor(endMS, width)

		if endCol < 0 || startCol >= width {
			continue
		}

		visibleStart := startCol
		if visibleStart < 0 {
			visibleStart = 0
		}
		visibleEnd := endCol
		if visibleEnd >= width {
			visibleEnd = width - 1
		}

		if visibleStart > visibleEnd {
			continue
		}

		isActive := p.positionMS >= startMS && p.positionMS <= endMS

		hasLeftBoundary := startCol >= 0
		hasRightBoundary := endCol < width

		contentStart := visibleStart
		if hasLeftBoundary {
			contentStart = visibleStart + 1
		}

		contentEnd := visibleEnd
		if hasRightBoundary {
			contentEnd = visibleEnd - 1
		}

		for col := visibleStart; col <= visibleEnd; col++ {
			cells[col].hasBlock = true
			cells[col].active = isActive
			cells[col].r = ' '
		}

		if hasLeftBoundary && visibleStart < width {
			cells[visibleStart].r = '│'
		}
		if hasRightBoundary && visibleEnd >= 0 {
			cells[visibleEnd].r = '│'
		}

		contentWidth := contentEnd - contentStart + 1
		if contentWidth > 0 {
			textRunes := []rune(line.Text().String())
			var displayText []rune
			if len(textRunes) > contentWidth {
				if contentWidth > 3 {
					displayText = append(textRunes[:contentWidth-2], '.', '.')
				} else {
					displayText = textRunes[:contentWidth]
				}
			} else {
				padding := contentWidth - len(textRunes)
				leftPad := padding / 2
				rightPad := padding - leftPad
				displayText = make([]rune, 0, contentWidth)
				for i := 0; i < leftPad; i++ {
					displayText = append(displayText, ' ')
				}
				displayText = append(displayText, textRunes...)
				for i := 0; i < rightPad; i++ {
					displayText = append(displayText, ' ')
				}
			}

			for i, r := range displayText {
				colIdx := contentStart + i
				if colIdx >= 0 && colIdx < width {
					cells[colIdx].r = r
				}
			}
		}
	}

	var b strings.Builder
	for _, c := range cells {
		charStr := string(c.r)
		if c.hasBlock {
			if c.active {
				b.WriteString(lyricActiveStyle.Render(charStr))
			} else {
				b.WriteString(lyricInactiveStyle.Render(charStr))
			}
		} else {
			b.WriteString(charStr)
		}
	}
	return b.String()
}

