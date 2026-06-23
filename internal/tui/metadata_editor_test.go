package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestMetadataEditorOverlay(t *testing.T) {
	model := NewModelWithDraftStore(mustDemoDocument(t), nil, nil, &memoryDraftStore{}, "", "", "")
	model.trackName = "Original Track"
	model.artistName = "Original Artist"
	model.albumName = "Original Album"

	// 1. Open metadata editor
	model = model.openMetadataEditor()
	if model.overlay != overlayMetadata {
		t.Fatalf("expected overlay to be overlayMetadata, got %v", model.overlay)
	}
	if !model.metadataEditor.active {
		t.Fatalf("expected metadataEditor to be active")
	}

	// 2. Type some text into the track name field (focus = 0)
	next, _ := model.Update(tea.KeyPressMsg{Text: " New"})
	model = next.(Model)
	if model.metadataEditor.trackName != "Original Track New" {
		t.Fatalf("expected trackName to append, got %q", model.metadataEditor.trackName)
	}

	// 3. Switch to artist field using Tab
	next, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	model = next.(Model)
	if model.metadataEditor.focus != 1 {
		t.Fatalf("expected focus to be 1 (Artist), got %d", model.metadataEditor.focus)
	}

	// 4. Type some text into artist
	next, _ = model.Update(tea.KeyPressMsg{Text: "2"})
	model = next.(Model)
	if model.metadataEditor.artistName != "Original Artist2" {
		t.Fatalf("expected artistName to append, got %q", model.metadataEditor.artistName)
	}

	// 5. Cancel with Escape
	cancelModel, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	canceled := cancelModel.(Model)
	if canceled.overlay != overlayNone {
		t.Fatalf("expected overlay to be overlayNone after Escape, got %v", canceled.overlay)
	}
	if canceled.trackName != "Original Track" {
		t.Fatalf("expected trackName to remain unchanged, got %q", canceled.trackName)
	}

	// 6. Save with Enter
	saveModel, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	saved := saveModel.(Model)
	if saved.overlay != overlayNone {
		t.Fatalf("expected overlay to be overlayNone after Enter, got %v", saved.overlay)
	}
	if saved.trackName != "Original Track New" {
		t.Fatalf("expected trackName to be saved, got %q", saved.trackName)
	}
	if saved.artistName != "Original Artist2" {
		t.Fatalf("expected artistName to be saved, got %q", saved.artistName)
	}

	// 7. Verify rendering
	rendered := renderMetadataEditor(model.metadataEditor, 80, 24, DefaultTheme())
	if !strings.Contains(rendered, "Edit Metadata") {
		t.Fatalf("expected 'Edit Metadata' in view: %q", rendered)
	}
}
