package waveform

import "strings"

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
