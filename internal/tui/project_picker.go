package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/editor"
)

type projectPickerMode uint8

const (
	projectPickerClosed projectPickerMode = iota
	projectPickerChoose
	projectPickerCreate
	projectPickerConfirmLoad
)

type projectPicker struct {
	mode     projectPickerMode
	projects []draft.ProjectSummary
	selected int
	input    string
	target   draft.ProjectID
}

func (p projectPicker) active() bool {
	return p.mode != projectPickerClosed
}

func (p projectPicker) withProjects(projects []draft.ProjectSummary) projectPicker {
	p.projects = append([]draft.ProjectSummary(nil), projects...)
	if p.selected >= len(p.projects) {
		p.selected = max(0, len(p.projects)-1)
	}
	if len(p.projects) == 0 {
		p.mode = projectPickerCreate
	}
	return p
}

func (m Model) openProjectPicker() Model {
	projects, err := m.draftStore.ListProjects()
	if err != nil {
		m.status = []string{"project list failed: " + err.Error()}
		return m
	}
	m.picker = projectPicker{mode: projectPickerChoose}.withProjects(projects)
	m.status = []string{"project picker open"}
	return m
}

func (m Model) OpenProjectPickerOnStartup() Model {
	return m.openProjectPicker()
}

func (m Model) updateProjectPicker(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	msg = normalizeKeyPress(msg)
	switch m.picker.mode {
	case projectPickerChoose:
		return m.updateProjectPickerChoose(msg), nil
	case projectPickerCreate:
		return m.updateProjectPickerCreate(msg), nil
	case projectPickerConfirmLoad:
		return m.updateProjectPickerConfirm(msg), nil
	default:
		return m, nil
	}
}

func (m Model) updateProjectPickerChoose(msg tea.KeyPressMsg) Model {
	switch {
	case msg.Code == tea.KeyEscape:
		m.picker = projectPicker{}
		m.status = []string{"project picker canceled"}
	case msg.Code == 'n':
		m.picker.mode = projectPickerCreate
		m.picker.input = ""
		m.status = []string{"new project id"}
	case msg.Code == 'j' || msg.Code == tea.KeyDown:
		if len(m.picker.projects) > 0 {
			m.picker.selected = min(len(m.picker.projects)-1, m.picker.selected+1)
		}
	case msg.Code == 'k' || msg.Code == tea.KeyUp:
		m.picker.selected = max(0, m.picker.selected-1)
	case msg.Code == tea.KeyEnter:
		if len(m.picker.projects) == 0 {
			m.picker.mode = projectPickerCreate
			return m
		}
		selected := m.picker.projects[m.picker.selected].ID
		if m.dirty && selected != m.projectID {
			m.picker.mode = projectPickerConfirmLoad
			m.picker.target = selected
			m.status = []string{"unsaved changes: Enter confirms project load"}
			return m
		}
		m = m.loadProject(selected)
	}
	return m
}

func (m Model) updateProjectPickerCreate(msg tea.KeyPressMsg) Model {
	switch msg.Code {
	case tea.KeyEscape:
		m.picker = projectPicker{}
		m.status = []string{"new project canceled"}
	case tea.KeyBackspace:
		if m.picker.input != "" {
			runes := []rune(m.picker.input)
			m.picker.input = string(runes[:len(runes)-1])
		}
	case tea.KeyEnter:
		id, err := draft.NewProjectID(m.picker.input)
		if err != nil {
			m.status = []string{"invalid project id: " + err.Error()}
			return m
		}
		m.projectID = id
		m.picker = projectPicker{}
		m.dirty = true
		m.status = []string{"project selected: " + id.String()}
	default:
		if msg.Text != "" {
			m.picker.input += msg.Text
		}
	}
	return m
}

func (m Model) updateProjectPickerConfirm(msg tea.KeyPressMsg) Model {
	switch msg.Code {
	case tea.KeyEscape:
		m.picker.mode = projectPickerChoose
		m.picker.target = ""
		m.status = []string{"project load canceled"}
	case tea.KeyEnter:
		m = m.loadProject(m.picker.target)
	}
	return m
}

func (m Model) loadProject(id draft.ProjectID) Model {
	snapshot, err := m.draftStore.Load(id)
	if err != nil {
		m.status = []string{"project load failed: " + err.Error()}
		return m
	}
	m.projectID = snapshot.ProjectID
	m.videoID = snapshot.Metadata.VideoID
	m.trackName = snapshot.Metadata.TrackName
	m.artistName = snapshot.Metadata.ArtistName
	m.albumName = snapshot.Metadata.AlbumName
	m.media = m.media.WithMetadata(m.trackName, m.artistName, m.albumName)
	m.editor = editor.NewPanel(snapshot.Document)
	m.picker = projectPicker{}
	m.dirty = false
	m.focus = focusEditor
	m.status = []string{"project loaded: " + id.String()}
	return m
}

func renderProjectPicker(p projectPicker, width int, height int) string {
	var builder strings.Builder
	builder.WriteString("Projects\n")
	switch p.mode {
	case projectPickerCreate:
		builder.WriteString("New project ID: ")
		builder.WriteString(p.input)
		builder.WriteString("\nEnter: create | Esc: cancel")
	case projectPickerConfirmLoad:
		builder.WriteString("Unsaved changes will be replaced.\n")
		builder.WriteString("Enter: load ")
		builder.WriteString(p.target.String())
		builder.WriteString(" | Esc: cancel")
	default:
		if len(p.projects) == 0 {
			builder.WriteString("No projects saved.\n")
			builder.WriteString("n: new project | Esc: cancel")
			break
		}
		for i, project := range p.projects {
			prefix := "  "
			if i == p.selected {
				prefix = "> "
			}
			title := project.Metadata.TrackName
			if title == "" {
				title = project.ID.String()
			}
			builder.WriteString(fmt.Sprintf("%s%s  %s\n", prefix, project.ID.String(), title))
		}
		builder.WriteString("Enter: load | n: new | Esc: cancel")
	}
	content := "Project Picker\n" + builder.String()
	return focusedBorder.
		Width(max(0, width-2)).
		Height(max(0, height-2)).
		Render(lipgloss.NewStyle().Width(max(0, width-4)).Render(content))
}
