package editor

import (
	"fmt"
	"strings"
)

func (p Panel) View(_ int, height int) string {
	if p.Importing {
		return p.viewImporting()
	}

	if p.ShowHelp {
		return p.viewHelp(height)
	}

	return p.viewLines(height)
}

func (p Panel) viewHelp(height int) string {
	maxScroll := len(helpLines) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.helpScroll > maxScroll {
		p.helpScroll = maxScroll
	}
	endIdx := p.helpScroll + height
	if endIdx > len(helpLines) {
		endIdx = len(helpLines)
	}
	return strings.Join(helpLines[p.helpScroll:endIdx], "\n")
}

func (p Panel) viewLines(height int) string {
	lines := p.Document.Lines()
	p.viewport = p.viewport.WithHeight(height).WithTotalLines(len(lines)).EnsureVisible(p.selected)

	startIdx := p.viewport.YOffset
	endIdx := startIdx + height
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	activeIdx := p.activeLineIndex()

	var builder strings.Builder
	for index := startIdx; index < endIdx; index++ {
		line := lines[index]

		marker := "  "
		if index == p.selected {
			marker = "> "
		}
		playMarker := "  "
		if index == activeIdx {
			playMarker = "▶ "
		}

		text := line.Text().String()
		if p.Editing && index == p.selected {
			runes := []rune(p.InputText)
			if p.cursorPos > len(runes) {
				p.cursorPos = len(runes)
			}
			text = "[Edit]: " + string(runes[:p.cursorPos]) + "|" + string(runes[p.cursorPos:])
		}
		timeRange := fmt.Sprintf("%s-%s", line.Start().String(), line.End().String())
		lineStr := fmt.Sprintf("%s%s%s |   %s", marker, playMarker, timeRange, text)

		if index == activeIdx {
			lineStr = p.theme.ActiveLine.Render(lineStr)
		} else if index == p.selected {
			lineStr = p.theme.SelectedLine.Render(lineStr)
		}

		builder.WriteString(lineStr)
		if index < endIdx-1 {
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}
