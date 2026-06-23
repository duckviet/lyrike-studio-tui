package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestHelpView(t *testing.T) {
	th := DefaultTheme()
	h := newHelpView(th)

	// Test View rendering
	rendered := h.View(80, 24)
	if !strings.Contains(rendered, "Keybindings") {
		t.Errorf("expected title 'Keybindings' in view, got: %s", rendered)
	}
	if !strings.Contains(rendered, "Global Keys") {
		t.Errorf("expected section 'Global Keys' in view, got: %s", rendered)
	}

	// Test interactive scrolling keys
	h, _ = h.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if h.offset != 1 {
		t.Errorf("expected offset to be 1 after KeyDown, got %d", h.offset)
	}

	h, _ = h.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if h.offset != 0 {
		t.Errorf("expected offset to be 0 after KeyUp, got %d", h.offset)
	}

	h, _ = h.Update(tea.KeyPressMsg{Code: 'j'})
	if h.offset != 1 {
		t.Errorf("expected offset to be 1 after 'j', got %d", h.offset)
	}

	h.reset()
	if h.offset != 0 {
		t.Errorf("expected offset to be 0 after reset, got %d", h.offset)
	}
}
