package history

import (
	"errors"
	"fmt"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

// ErrNothingToUndo is returned when the undo stack is empty.
var ErrNothingToUndo = errors.New("history: nothing to undo")

// ErrNothingToRedo is returned when the redo stack is empty.
var ErrNothingToRedo = errors.New("history: nothing to redo")

// Command is a reversible edit against a lyrics document.
type Command interface {
	Apply(doc lyrics.Document) (next lyrics.Document, inverse Command, err error)
	Name() string
}

// Manager stores executed commands as inverse operations for undo/redo.
type Manager struct {
	undo []Command
	redo []Command
}

// NewManager creates an empty history manager.
func NewManager() *Manager {
	return &Manager{}
}

// Apply executes cmd against doc, stores the inverse for undo, and clears redo.
func (m *Manager) Apply(doc lyrics.Document, cmd Command) (lyrics.Document, error) {
	next, inverse, err := cmd.Apply(doc)
	if err != nil {
		return doc, fmt.Errorf("%s: %w", cmd.Name(), err)
	}
	m.undo = append(m.undo, inverse)
	m.redo = nil
	return next, nil
}

// Undo applies the most recent inverse command and stores its inverse in redo.
func (m *Manager) Undo(doc lyrics.Document) (lyrics.Document, error) {
	if len(m.undo) == 0 {
		return doc, ErrNothingToUndo
	}
	inverse := m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]

	next, redoCmd, err := inverse.Apply(doc)
	if err != nil {
		m.undo = append(m.undo, inverse)
		return doc, fmt.Errorf("undo %s: %w", inverse.Name(), err)
	}
	m.redo = append(m.redo, redoCmd)
	return next, nil
}

// Redo reapplies the most recently undone command.
func (m *Manager) Redo(doc lyrics.Document) (lyrics.Document, error) {
	if len(m.redo) == 0 {
		return doc, ErrNothingToRedo
	}
	cmd := m.redo[len(m.redo)-1]
	m.redo = m.redo[:len(m.redo)-1]

	next, inverse, err := cmd.Apply(doc)
	if err != nil {
		m.redo = append(m.redo, cmd)
		return doc, fmt.Errorf("redo %s: %w", cmd.Name(), err)
	}
	m.undo = append(m.undo, inverse)
	return next, nil
}

func (m *Manager) CanUndo() bool { return len(m.undo) > 0 }
func (m *Manager) CanRedo() bool { return len(m.redo) > 0 }
func (m *Manager) Clear()        { m.undo = nil; m.redo = nil }
