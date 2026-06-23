package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/publish"
)

func (m Model) updateOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.overlay {
	case overlayHelp:
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			switch keyMsg.String() {
			case "esc", "q", "?":
				m.overlay = overlayNone
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.help, cmd = m.help.Update(msg)
		return m, cmd

	case overlaySelector:
		var cmd tea.Cmd
		var result selResult
		m.picker, result, cmd = m.picker.Update(msg)
		if result.canceled {
			m.overlay = overlayNone
			m.setStatus("project picker canceled")
			return m, nil
		}
		if result.accepted {
			if result.id == "__new_project__" {
				m.overlay = overlayNone
				return m.openFetchInput(), nil
			}
			selected := draft.ProjectID(result.id)
			if m.dirty && selected != m.projectID {
				return m.confirmAction(
					"Unsaved changes",
					"Discard current work and load project?",
					true,
					func() tea.Msg {
						return confirmProjectLoadMsg{id: selected}
					},
					func() tea.Msg {
						return cancelProjectLoadMsg{}
					},
				), nil
			}
			m.overlay = overlayNone
			return m.loadProject(selected)
		}
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			// Ctrl-N shortcut inside the selector to start new project from URL
			if keyMsg.Code == 'n' && keyMsg.Mod == tea.ModCtrl {
				m.overlay = overlayNone
				return m.openFetchInput(), nil
			}
		}
		return m, cmd

	case overlayConfirm:
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			switch keyMsg.String() {
			case "y", "Y", "enter":
				m.overlay = overlayNone
				return m, m.confirm.action
			case "n", "N", "esc":
				m.overlay = overlayNone
				return m, m.confirm.cancel
			}
		}
		return m, nil

	case overlayInput:
		return m.updateFetchInput(msg)
	case overlayMetadata:
		return m.updateMetadataEditor(msg)
	case overlayPublish:
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			state := m.publish.State()
			if state == publish.StateDone && keyMsg.Code == tea.KeyEnter {
				m.overlay = overlayNone
				return m, nil
			}
			if state == publish.StateFailed && keyMsg.Code == tea.KeyEscape {
				m.overlay = overlayNone
				return m, nil
			}
			if state == publish.StateConfirm && keyMsg.Code == tea.KeyEscape {
				m.overlay = overlayNone
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.publish, cmd = m.publish.Update(msg)
		return m, cmd

	default:
		m.overlay = overlayNone
		return m, nil
	}
}

type confirmProjectLoadMsg struct {
	id draft.ProjectID
}

type cancelProjectLoadMsg struct{}
