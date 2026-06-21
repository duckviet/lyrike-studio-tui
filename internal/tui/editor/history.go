package editor

import (
	"github.com/duckviet/lyrike-studio-tui/internal/domain/history"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

func (p Panel) applyEditText(value string) Panel {
	text, err := lyrics.NewText(value)
	if err != nil {
		p.lastErr = err
		return p
	}
	return p.apply(history.EditText{Index: p.selected, Text: text})
}

func (p Panel) apply(command history.Command) Panel {
	next, err := p.manager.Apply(p.Document, command)
	if err != nil {
		p.lastErr = err
		return p
	}
	p.Document = next
	p.lastErr = nil
	return p
}

func (p Panel) undo() Panel {
	next, err := p.manager.Undo(p.Document)
	if err != nil {
		p.lastErr = err
		return p
	}
	p.Document = next
	p.lastErr = nil
	return p
}

func (p Panel) redo() Panel {
	next, err := p.manager.Redo(p.Document)
	if err != nil {
		p.lastErr = err
		return p
	}
	p.Document = next
	p.lastErr = nil
	return p
}
