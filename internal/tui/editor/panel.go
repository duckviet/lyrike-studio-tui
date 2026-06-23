package editor

import (
	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/history"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/theme"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/viewport"
)

type Panel struct {
	Title              string
	Document           lyrics.Document
	manager            *history.Manager
	selected           int
	tapPosition        playback.Position
	playbackPositionMS int64
	lastErr            error
	Editing            bool
	Importing          bool
	InputText          string
	cursorPos          int
	ShowHelp           bool
	helpScroll         int
	viewport           viewport.Model
	theme              theme.Theme
}

func NewPanel(doc lyrics.Document) Panel {
	return Panel{
		Title:    "Lyrics",
		Document: doc,
		manager:  history.NewManager(),
		viewport: viewport.New(0, 10).WithTotalLines(len(doc.Lines())),
	}
}

func (p Panel) WithTheme(t theme.Theme) Panel {
	p.theme = t
	return p
}

func (p Panel) Selected() int {
	return p.selected
}

func (p Panel) LastError() error {
	return p.lastErr
}

func (p Panel) WithTapPosition(position playback.Position) Panel {
	p.tapPosition = position
	return p
}

func (p Panel) WithPlaybackPosition(pos int64) Panel {
	p.playbackPositionMS = pos
	return p
}

func (p Panel) activeLineIndex() int {
	lines := p.Document.Lines()
	index := -1
	for i, line := range lines {
		if p.playbackPositionMS >= line.Start().Milliseconds() {
			index = i
		}
	}
	return index
}

func (p Panel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		return p.handlePaste(msg)
	case tea.KeyPressMsg:
		return p.handleKey(msg)
	}
	return p, nil
}

type StartPublishMsg struct {
	Lyrics string
}

func (p Panel) WithSelected(index int) Panel {
	if index >= 0 && index < len(p.Document.Lines()) {
		p.selected = index
		p.viewport = p.viewport.WithTotalLines(len(p.Document.Lines())).EnsureVisible(index)
	}
	return p
}

func (p Panel) WithHeight(h int) Panel {
	p.viewport = p.viewport.WithHeight(h).EnsureVisible(p.selected)
	return p
}

func (p Panel) HandleMouseScroll(button tea.MouseButton) Panel {
	p.viewport = p.viewport.WithTotalLines(len(p.Document.Lines())).WithHeight(p.viewport.Height)
	if button == tea.MouseWheelUp {
		p.viewport = p.viewport.ScrollUp()
	} else if button == tea.MouseWheelDown {
		p.viewport = p.viewport.ScrollDown()
	}
	return p
}
