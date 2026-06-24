package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type confirmView struct {
	th      Theme
	title   string
	message string
	danger  bool
	action  tea.Cmd
	cancel  tea.Cmd
}

func (c confirmView) View(width, height int) string {
	titleStyle := c.th.ModalTitle
	border := c.th.ModalBorder
	if c.danger {
		titleStyle = c.th.Modal(lipgloss.NewStyle().Foreground(c.th.P.Bad).Bold(true))
		border = c.th.ModalBorder.BorderForeground(c.th.P.Bad)
	}

	keyStyle := c.th.Modal(c.th.FooterKey)
	descStyle := c.th.Modal(c.th.FooterDesc)

	body := titleStyle.Render(c.title) + "\n\n" +
		c.th.SelItem.Render(c.message) + "\n\n" +
		keyStyle.Render("y") + descStyle.Render(" confirm") + "    " +
		keyStyle.Render("n") + descStyle.Render("/") +
		keyStyle.Render("esc") + descStyle.Render(" cancel")

	boxW := lipgloss.Width(c.message) + 6
	if boxW < 40 {
		boxW = 40
	}
	if boxW > width-6 {
		boxW = width - 6
	}
	if boxW < 8 {
		boxW = 8
	}
	box := border.Width(boxW).Render(c.th.PaintModal(body))
	return box
}

func (m Model) confirmAction(title, message string, danger bool, action tea.Cmd, cancel tea.Cmd) Model {
	m.confirm = confirmView{
		th:      m.theme,
		title:   title,
		message: message,
		danger:  danger,
		action:  action,
		cancel:  cancel,
	}
	m.overlay = overlayConfirm
	m.setStatus(strings.ToLower(title) + ": y/n")
	return m
}
