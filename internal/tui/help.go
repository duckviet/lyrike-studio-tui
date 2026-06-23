package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type helpView struct {
	th     Theme
	offset int
}

func newHelpView(th Theme) helpView {
	return helpView{th: th}
}

func (h *helpView) reset() {
	h.offset = 0
}

func (h helpView) Update(msg tea.Msg) (helpView, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "up", "k", "ctrl+p":
			if h.offset > 0 {
				h.offset--
			}
		case "down", "j", "ctrl+n":
			h.offset++
		case "pgup", "ctrl+u":
			h.offset -= 5
			if h.offset < 0 {
				h.offset = 0
			}
		case "pgdown", "ctrl+d":
			h.offset += 5
		case "g", "home":
			h.offset = 0
		}
	}
	return h, nil
}

type helpGroup struct {
	title string
	keys  [][2]string
}

func (h helpView) View(width, height int) string {
	groups := []helpGroup{
		{
			title: "Global Keys",
			keys: [][2]string{
				{"Tab", "Switch focus next"},
				{"S-Tab", "Switch focus prev"},
				{"Ctrl+O", "Fetch project from URL/ID"},
				{"Ctrl+P", "Open project selector"},
				{"Ctrl+S", "Save current draft"},
				{"?", "Toggle help menu"},
				{"q / Esc", "Quit application"},
			},
		},
		{
			title: "Playback & Waveform",
			keys: [][2]string{
				{"Space", "Play / Pause playback"},
				{"← / →", "Seek backward/forward 1s"},
				{"f", "Toggle waveform follow"},
			},
		},
		{
			title: "Lyrics Editor",
			keys: [][2]string{
				{"↑ / ↓", "Move line cursor (j/k)"},
				{"Enter", "Edit selected lyric line"},
				{"t", "Tap-sync line to playback"},
				{"Ctrl+T", "Start audio transcription"},
				{"Ctrl+Z", "Undo last lyric edit"},
				{"Ctrl+Y", "Redo last lyric edit"},
				{"Ctrl+E", "Edit project metadata"},
			},
		},
		{
			title: "Publishing",
			keys: [][2]string{
				{"Ctrl+P", "Enter publish flow (focused)"},
				{"y", "Confirm LRCLIB publish"},
			},
		},
	}

	colW := 26
	if w := width - 6; w < colW {
		colW = w
	}
	if colW < 12 {
		colW = 12
	}

	blocks := make([]string, len(groups))
	for i, g := range groups {
		var b strings.Builder
		b.WriteString(h.th.Prompt.Render(g.title))
		b.WriteString("\n")
		for _, k := range g.keys {
			b.WriteString(h.th.FooterKey.Render(fmt.Sprintf("%-8s", k[0])))
			b.WriteString(" ")
			b.WriteString(h.th.FooterDesc.Render(k[1]))
			b.WriteString("\n")
		}
		blocks[i] = lipgloss.NewStyle().Width(colW).Render(strings.TrimRight(b.String(), "\n"))
	}

	perRow := 3
	switch {
	case width < 62:
		perRow = 1
	case width < 92:
		perRow = 2
	}

	var rows []string
	for i := 0; i < len(blocks); i += perRow {
		end := i + perRow
		if end > len(blocks) {
			end = len(blocks)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, blocks[i:end]...))
	}
	grid := lipgloss.JoinVertical(lipgloss.Left, rows...)
	gridLines := strings.Split(grid, "\n")

	border := h.th.ModalBorder
	spacer := "\n\n"
	frameRows := 6
	if height < 7 {
		border = border.Padding(0, 1)
		spacer = "\n"
		frameRows = 3
	}

	visible := height - frameRows
	if visible < 1 {
		visible = 1
	}
	maxOffset := len(gridLines) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if h.offset > maxOffset {
		h.offset = maxOffset
	}
	end := h.offset + visible
	if end > len(gridLines) {
		end = len(gridLines)
	}
	grid = strings.Join(gridLines[h.offset:end], "\n")

	title := h.th.ModalTitle.Render("Keybindings")
	hint := h.th.Dim.Render("  ? or esc to close")
	if maxOffset > 0 {
		hint = h.th.Dim.Render(fmt.Sprintf("  ↑↓ scroll %d/%d · esc close", h.offset+1, maxOffset+1))
	}
	content := title + hint + spacer + grid

	box := border.Render(content)
	return box
}
