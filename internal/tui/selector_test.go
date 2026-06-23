package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSelectorFuzzyFiltering(t *testing.T) {
	th := DefaultTheme()
	sel := newSelector(th)
	items := []selItem{
		{"Project Alpha", "rick", "alpha"},
		{"Project Beta", "astley", "beta"},
		{"Gamma Draft", "other", "gamma"},
	}

	sel.open(selResource, "Select Project", "Search...", items, false)

	if len(sel.match) != 3 {
		t.Fatalf("expected 3 initial matches, got %d", len(sel.match))
	}

	// 1. Simulate keypresses to filter the list
	// We type 'b' which matches Alpha (contains 'a') and Beta (contains 'b')
	// Wait, 'b' is sub-sequence of 'Project Alpha' (has a but not b? Wait, 'Beta' has 'b').
	// Let's filter by typing 'Beta'
	for _, char := range "Beta" {
		var cmd tea.Cmd
		sel, _, cmd = sel.Update(tea.KeyPressMsg{Text: string(char)})
		if cmd != nil {
			// consume or ignore tick
		}
	}

	if len(sel.match) != 1 {
		t.Fatalf("expected 1 match for 'Beta', got %d: %v", len(sel.match), sel.match)
	}

	selected, ok := sel.current()
	if !ok {
		t.Fatalf("expected current item to be selected")
	}
	if selected.id != "beta" {
		t.Fatalf("expected selected item ID 'beta', got %q", selected.id)
	}

	// 2. Select with Enter
	var result selResult
	sel, result, _ = sel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !result.accepted {
		t.Fatalf("expected Enter to accept the selection")
	}
	if result.id != "beta" {
		t.Fatalf("expected result ID 'beta', got %q", result.id)
	}
}

func TestSelectorScrolling(t *testing.T) {
	th := DefaultTheme()
	sel := newSelector(th)
	var items []selItem
	for i := 0; i < 20; i++ {
		items = append(items, selItem{title: "Item", id: string(rune('a' + i))})
	}

	sel.open(selResource, "Title", "Search...", items, false)
	if sel.cursor != 0 {
		t.Fatalf("expected initial cursor 0, got %d", sel.cursor)
	}

	// Scroll down 3 times
	for i := 0; i < 3; i++ {
		sel, _, _ = sel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if sel.cursor != 3 {
		t.Fatalf("expected cursor 3 after 3 scroll downs, got %d", sel.cursor)
	}

	// Scroll up 1 time
	sel, _, _ = sel.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if sel.cursor != 2 {
		t.Fatalf("expected cursor 2 after scroll up, got %d", sel.cursor)
	}
}
