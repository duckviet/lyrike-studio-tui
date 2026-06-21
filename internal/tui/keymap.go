package tui

import tea "charm.land/bubbletea/v2"

type keyAction uint8

const (
	keyActionNone keyAction = iota
	keyActionFocusNext
	keyActionFocusPrev
	keyActionSaveDraft
	keyActionOpenProjects
	keyActionEditMetadata
	keyActionTogglePlayback
	keyActionSeekBackward
	keyActionSeekForward
	keyActionToggleFollow
	keyActionQuit
)

func globalKeyAction(key tea.KeyPressMsg) keyAction {
	switch {
	case key.Code == tea.KeyTab && key.Mod == tea.ModShift:
		return keyActionFocusPrev
	case key.Code == tea.KeyTab:
		return keyActionFocusNext
	case key.Code == 's' && key.Mod == tea.ModCtrl:
		return keyActionSaveDraft
	case key.Code == 'p' && key.Mod == tea.ModCtrl:
		return keyActionOpenProjects
	case key.Code == 'e' && key.Mod == tea.ModCtrl:
		return keyActionEditMetadata
	default:
		return keyActionNone
	}
}

func normalizeKeyPress(key tea.KeyPressMsg) tea.KeyPressMsg {
	if key.Code == 0 && key.Mod == 0 {
		runes := []rune(key.Text)
		if len(runes) == 1 {
			key.Code = runes[0]
		}
	}
	return key
}

func nonEditingRootKeyAction(key tea.KeyPressMsg) keyAction {
	switch {
	case key.Code == tea.KeySpace || key.Code == ' ':
		return keyActionTogglePlayback
	case key.Code == tea.KeyLeft:
		return keyActionSeekBackward
	case key.Code == tea.KeyRight:
		return keyActionSeekForward
	case key.Code == 'f' && key.Mod == 0:
		return keyActionToggleFollow
	case key.Code == 'q' && key.Mod == 0:
		return keyActionQuit
	default:
		return keyActionNone
	}
}

func (m Model) editorEditModeOwnsKey() bool {
	return m.focus == focusEditor && (m.editor.Editing || m.editor.Importing)
}
