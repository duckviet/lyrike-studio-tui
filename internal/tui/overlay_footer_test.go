package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/tui/publish"
)

var ansiSequence = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestOverlayCenterPlacesBoxOverBase(t *testing.T) {
	base := strings.Join([]string{
		"aaaaaa",
		"bbbbbb",
		"cccccc",
		"dddddd",
	}, "\n")
	box := "XX\nYY"

	got := overlayCenter(base, box, 6, 4)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %q", len(lines), got)
	}

	if !strings.Contains(lines[1], "XX") || !strings.Contains(lines[2], "YY") {
		t.Fatalf("expected centered overlay box in middle rows, got %q", got)
	}
	if !strings.Contains(lines[0], "aaaaaa") || !strings.Contains(lines[3], "dddddd") {
		t.Fatalf("expected untouched top/bottom base rows, got %q", got)
	}
}

func TestStyledOverlayDoesNotLeakANSICodes(t *testing.T) {
	th := DefaultTheme()
	base := th.StatusOK.Render("base")
	box := overlayBlock(th.ModalTitle.Render("Help")+"\n\n"+renderHints([]hint{{key: "Esc", desc: "close"}}, th), 40, th)

	got := overlayCenter(base, box, 80, 12)
	plain := ansiSequence.ReplaceAllString(got, "")
	if strings.Contains(plain, ";") {
		t.Fatalf("expected stripped styled overlay to contain no leaked ANSI fragments, got %q", plain)
	}
	if !strings.Contains(plain, "Help") || !strings.Contains(plain, "Esc close") {
		t.Fatalf("expected styled overlay content after ANSI stripping, got %q", plain)
	}
}

func TestFooterViewRendersHintsAndStatus(t *testing.T) {
	m := Model{
		width:     80,
		height:    24,
		focus:     focusMedia,
		status:    "ready",
		statusErr: false,
		theme:     DefaultTheme(),
	}

	got := footerView(m, 80)
	if !strings.Contains(got, "Tab") {
		t.Fatalf("expected footer to include focus navigation hint, got %q", got)
	}
	if !strings.Contains(got, "ready") {
		t.Fatalf("expected footer to include status, got %q", got)
	}
	if lipgloss.Width(got) > 80 {
		t.Fatalf("expected footer to fit width 80, got width %d: %q", lipgloss.Width(got), got)
	}
}

func TestRenderLayoutIncludesFooter(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
		focus:  focusMedia,
		status: "ready",
		theme:  DefaultTheme(),
	}

	got := renderLayout(m)
	if !strings.Contains(got, "Tab") || !strings.Contains(got, "ready") {
		t.Fatalf("expected rendered layout to include footer hints and status, got %q", got)
	}
}

func TestModelViewIncludesFooter(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
		focus:  focusMedia,
		status: "ready",
		theme:  DefaultTheme(),
	}

	got := fmt.Sprint(m.View())
	if !strings.Contains(got, "Tab") || !strings.Contains(got, "ready") {
		t.Fatalf("expected model view to include footer hints and status, got %q", got)
	}
}

func TestHelpOverlayKeyLifecycle(t *testing.T) {
	m := Model{width: 80, height: 24, focus: focusMedia, theme: DefaultTheme()}

	updated, cmd := m.Update(keyPress('?', 0))
	if cmd != nil {
		t.Fatalf("help overlay open returned unexpected command")
	}
	model, ok := updated.(Model)
	if !ok {
		t.Fatalf("updated model type = %T, want Model", updated)
	}
	if model.overlay != overlayHelp {
		t.Fatalf("overlay = %v, want help", model.overlay)
	}
	got := renderLayout(model)
	if !strings.Contains(got, "Keybindings") {
		t.Fatalf("expected help overlay in rendered layout, got %q", got)
	}

	updated, cmd = model.Update(keyPress(tea.KeyEscape, 0))
	if cmd != nil {
		t.Fatalf("help overlay close returned unexpected command")
	}
	model, ok = updated.(Model)
	if !ok {
		t.Fatalf("updated model type = %T, want Model", updated)
	}
	if model.overlay != overlayNone {
		t.Fatalf("overlay = %v, want none", model.overlay)
	}
}

func TestHelpKeyDoesNotHijackFetchInput(t *testing.T) {
	th := DefaultTheme()
	ti := textinput.New()
	m := Model{
		width:      80,
		height:     24,
		theme:      th,
		overlay:    overlayInput,
		fetchInput: fetchInput{input: ti, activeState: true},
	}

	updated, _ := m.Update(keyPress('?', 0))
	model, ok := updated.(Model)
	if !ok {
		t.Fatalf("updated model type = %T, want Model", updated)
	}
	if model.overlay != overlayInput {
		t.Fatalf("overlay = %v, want overlayInput", model.overlay)
	}
	if !model.fetchInput.active() {
		t.Fatalf("expected fetch input to stay active")
	}
}

func TestHintsChangeWithFocusAndOverlay(t *testing.T) {
	base := Model{focus: focusMedia, theme: DefaultTheme()}
	mediaHints := renderHints(base.hints(), base.theme)

	editor := base
	editor.focus = focusEditor
	editorHints := renderHints(editor.hints(), editor.theme)
	if mediaHints == editorHints {
		t.Fatalf("expected focus-specific hints to change, got %q", mediaHints)
	}

	overlay := base
	overlay.overlay = overlayHelp
	overlayHints := renderHints(overlay.hints(), overlay.theme)
	if !strings.Contains(overlayHints, "Esc") {
		t.Fatalf("expected overlay hints to include close action, got %q", overlayHints)
	}
}

func TestOverlayBlocksPasteAndMouse(t *testing.T) {
	m := Model{
		width:   80,
		height:  24,
		focus:   focusMedia,
		overlay: overlayHelp,
		theme:   DefaultTheme(),
	}

	// 1. Verify PasteMsg is blocked (model unchanged, no command)
	pasteMsg := tea.PasteMsg{Content: "test paste"}
	updatedModel, cmd := m.Update(pasteMsg)
	if cmd != nil {
		t.Fatalf("expected nil cmd when paste is blocked by overlay, got %v", cmd)
	}
	model, ok := updatedModel.(Model)
	if !ok {
		t.Fatalf("expected updated model type Model, got %T", updatedModel)
	}
	if model.overlay != overlayHelp {
		t.Fatalf("overlay changed under blocked paste")
	}

	// 2. Verify MouseMsg is blocked
	mouseMsg := tea.MouseClickMsg{X: 10, Y: 20}
	updatedModel2, cmd2 := m.Update(mouseMsg)
	if cmd2 != nil {
		t.Fatalf("expected nil cmd when mouse is blocked by overlay, got %v", cmd2)
	}
	model2, ok := updatedModel2.(Model)
	if !ok {
		t.Fatalf("expected updated model type Model, got %T", updatedModel2)
	}
	if model2.focus != focusMedia {
		t.Fatalf("focus changed under blocked mouse event")
	}
}

func TestPublishOverlayLifecycle(t *testing.T) {
	m := Model{
		width:   80,
		height:  24,
		focus:   focusEditor,
		theme:   DefaultTheme(),
		publish: publish.NewPanel(),
	}

	// 1. Trigger keyActionPublish (Ctrl+P in editor)
	updated, _ := m.Update(keyPress('p', tea.ModCtrl))
	model, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected Model")
	}
	if model.overlay != overlayPublish {
		t.Fatalf("expected overlayPublish, got %v", model.overlay)
	}
	if model.publish.State() != "confirm" {
		t.Fatalf("expected state confirm, got %s", model.publish.State())
	}

	// 2. Verify Cancel (Esc)
	canceled, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	canceledModel := canceled.(Model)
	if canceledModel.overlay != overlayNone {
		t.Fatalf("expected overlayNone after cancel")
	}

	// 3. Verify Confirm (y)
	confirmed, cmd := model.Update(keyPress('y', 0))
	confirmedModel := confirmed.(Model)
	if confirmedModel.overlay != overlayPublish {
		t.Fatalf("expected overlayPublish to stay active during confirm progress")
	}
	if cmd == nil {
		t.Fatalf("expected command for solving pow/publish")
	}
}
