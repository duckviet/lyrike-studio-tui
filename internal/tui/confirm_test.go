package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestConfirmView(t *testing.T) {
	th := DefaultTheme()
	action := func() tea.Msg {
		return nil
	}
	cancel := func() tea.Msg {
		return nil
	}

	c := confirmView{
		th:      th,
		title:   "Test Confirm",
		message: "Are you sure you want to run this test?",
		danger:  true,
		action:  action,
		cancel:  cancel,
	}

	rendered := c.View(80, 20)
	if !strings.Contains(rendered, "Test Confirm") {
		t.Errorf("expected title 'Test Confirm' in view, got: %s", rendered)
	}
	if !strings.Contains(rendered, "Are you sure") {
		t.Errorf("expected message in view, got: %s", rendered)
	}

	// Verify confirmAction on Model
	var m Model
	m.theme = th
	m = m.confirmAction("Test Title", "Test Message", false, action, cancel)
	if m.overlay != overlayConfirm {
		t.Errorf("expected overlay to be overlayConfirm, got %v", m.overlay)
	}
	if m.confirm.title != "Test Title" {
		t.Errorf("expected title to be 'Test Title', got %s", m.confirm.title)
	}
}
