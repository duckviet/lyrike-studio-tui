package waveform

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

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
		colColor := p.colorFor(peak, isPlayed, inLoop)
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
					cellStr = p.theme.Knob.Render(cellStr)
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

		colColor := p.colorFor(peak, isPlayed, inLoop)
		cellStr := string(glyph)
		if isHover {
			cellStr = p.theme.Knob.Render(cellStr)
		} else {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(colColor))
			cellStr = style.Render(cellStr)
		}
		cells[col] = cellStr
	}
	return strings.Join(cells, "")
}

func (p Panel) colorFor(peak float64, isPlayed, inLoop bool) string {
	accentHex := colorToHex(p.theme.P.Accent, "#7263E1")
	accent2Hex := colorToHex(p.theme.P.Accent2, "#FF3366")
	unplayedHex := colorToHex(p.theme.P.Unplayed, "#333333")
	subduedHex := colorToHex(p.theme.P.Subdued, "#4A4A4A")
	surfaceHex := colorToHex(p.theme.P.Surface, "#161616")
	selFgHex := colorToHex(p.theme.P.SelFg, "#FFFFFF")

	if isPlayed {
		if inLoop {
			start := lerpColor(surfaceHex, accent2Hex, 0.5)
			end := lerpColor(accent2Hex, selFgHex, 0.3)
			return lerpColor(start, end, peak)
		} else {
			start := lerpColor(surfaceHex, accentHex, 0.5)
			end := lerpColor(accentHex, selFgHex, 0.3)
			return lerpColor(start, end, peak)
		}
	} else {
		if inLoop {
			start := lerpColor(unplayedHex, accent2Hex, 0.1)
			end := lerpColor(subduedHex, accent2Hex, 0.2)
			return lerpColor(start, end, peak)
		} else {
			return lerpColor(unplayedHex, subduedHex, peak)
		}
	}
}
