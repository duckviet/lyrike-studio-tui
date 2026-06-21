package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

func mustDemoDocument(t *testing.T) lyrics.Document {
	t.Helper()
	doc, err := demoDocument()
	if err != nil {
		t.Fatalf("demo document: %v", err)
	}
	return doc
}

func updateModel(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	newModel, cmd := m.Update(msg)
	if cmd != nil {
		t.Fatalf("expected nil command, got %v", cmd)
	}
	cast, ok := newModel.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", newModel)
	}
	return cast
}

func keyPress(code rune, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Mod: mod}
}

func Test_FocusAdvances_when_TabIsPressed(t *testing.T) {
	// Given: a fresh model focused on the media panel
	doc := mustDemoDocument(t)
	m := NewModel(doc, nil, nil, "", "")
	if m.focus != focusMedia {
		t.Fatalf("expected initial focus media, got %d", m.focus)
	}

	// When: Tab is pressed three times
	m = updateModel(t, m, keyPress(tea.KeyTab, 0))
	if m.focus != focusWaveform {
		t.Fatalf("after first Tab expected waveform focus, got %d", m.focus)
	}
	m = updateModel(t, m, keyPress(tea.KeyTab, 0))
	if m.focus != focusEditor {
		t.Fatalf("after second Tab expected editor focus, got %d", m.focus)
	}
	m = updateModel(t, m, keyPress(tea.KeyTab, 0))
	if m.focus != focusMedia {
		t.Fatalf("after third Tab expected wrap to media focus, got %d", m.focus)
	}
}

func Test_FocusReverses_when_ShiftTabIsPressed(t *testing.T) {
	// Given: a fresh model focused on the media panel
	doc := mustDemoDocument(t)
	m := NewModel(doc, nil, nil, "", "")

	// When: Shift+Tab is pressed
	m = updateModel(t, m, keyPress(tea.KeyTab, tea.ModShift))
	if m.focus != focusEditor {
		t.Fatalf("after Shift+Tab expected editor focus, got %d", m.focus)
	}

	// And again
	m = updateModel(t, m, keyPress(tea.KeyTab, tea.ModShift))
	if m.focus != focusWaveform {
		t.Fatalf("after second Shift+Tab expected waveform focus, got %d", m.focus)
	}
}

func Test_ModelStoresTerminalSize_when_WindowSizeMsgArrives(t *testing.T) {
	// Given: a fresh model with zero dimensions
	doc := mustDemoDocument(t)
	m := NewModel(doc, nil, nil, "", "")
	if m.width != 0 || m.height != 0 {
		t.Fatalf("expected initial dimensions 0,0, got %dx%d", m.width, m.height)
	}

	// When: a terminal resize is reported
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Then: the model records the new size
	if m.width != 80 || m.height != 24 {
		t.Fatalf("expected dimensions 80x24, got %dx%d", m.width, m.height)
	}
}

func Test_LayoutFits80x24_when_ViewRendersAfterResize(t *testing.T) {
	// Given: a model sized to 80x24
	doc := mustDemoDocument(t)
	m := NewModel(doc, nil, nil, "", "")
	m = updateModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// When: the view is rendered
	view := m.View()
	content := view.Content

	// Then: all three panel labels are present
	for _, label := range []string{"Media", "Waveform", "Lyrics"} {
		if !strings.Contains(content, label) {
			t.Fatalf("expected view to contain %q, got:\n%s", label, content)
		}
	}

	// Then: no rendered line exceeds the terminal width
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w > 80 {
			t.Fatalf("line %d exceeds 80 cols: width=%d\n%q", i, w, line)
		}
	}
}
