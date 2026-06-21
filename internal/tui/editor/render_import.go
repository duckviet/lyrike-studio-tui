package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (p Panel) viewImporting() string {
	var builder strings.Builder
	builder.WriteString("Import Lyrics from File\n")
	builder.WriteString("Enter path to .lrc or .txt file:\n")

	runes := []rune(p.InputText)
	if p.cursorPos > len(runes) {
		p.cursorPos = len(runes)
	}
	pathStr := string(runes[:p.cursorPos]) + "|" + string(runes[p.cursorPos:])
	builder.WriteString("> " + pathStr + "\n\n")

	if p.lastErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3366"))
		builder.WriteString(errStyle.Render("Error: "+p.lastErr.Error()) + "\n\n")
	}

	builder.WriteString("(Press Enter to import, Esc to cancel)")
	return builder.String()
}
