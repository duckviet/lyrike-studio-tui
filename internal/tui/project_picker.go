package tui

import (
	"io"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/editor"
)

func (m Model) openProjectPicker() Model {
	projects, err := m.draftStore.ListProjects()
	if err != nil {
		m.setErrorStatus("project list failed: " + err.Error())
		return m
	}

	var items []selItem
	// Add virtual [New Project] item at index 0
	items = append(items, selItem{
		title: "[New Project]",
		desc:  "Start a new project from a URL or video ID",
		id:    "__new_project__",
	})

	for _, p := range projects {
		title := p.Metadata.TrackName
		if title == "" {
			title = p.ID.String()
		}
		desc := p.Metadata.ArtistName
		if desc == "" {
			desc = "No Artist"
		}
		items = append(items, selItem{
			title: title,
			desc:  desc,
			id:    p.ID.String(),
		})
	}

	m.picker.open(selResource, "Select Project", "Search...", items, false)
	m.overlay = overlaySelector
	m.setStatus("project selector open")
	return m
}

func (m Model) OpenProjectPickerOnStartup() Model {
	return m.openProjectPicker()
}

func (m Model) loadProject(id draft.ProjectID) (Model, tea.Cmd) {
	snapshot, err := m.draftStore.Load(id)
	if err != nil {
		m.setErrorStatus("project load failed: " + err.Error())
		return m, nil
	}
	m.projectID = snapshot.ProjectID
	m.videoID = snapshot.Metadata.VideoID
	m.trackName = snapshot.Metadata.TrackName
	m.artistName = snapshot.Metadata.ArtistName
	m.albumName = snapshot.Metadata.AlbumName
	m.media = m.media.WithMetadata(m.trackName, m.artistName, m.albumName)
	m.editor = editor.NewPanel(snapshot.Document).WithTheme(m.theme)
	m.dirty = false
	m.focus = focusEditor

	var cmd tea.Cmd
	if m.playerFactory != nil && m.videoID != "" {
		if closer, ok := m.player.(io.Closer); ok {
			_ = closer.Close()
		}
		newPlayer, status := m.playerFactory(m.videoID)
		m.player = newPlayer
		if status != "" {
			m.setStatus("project loaded: " + id.String() + " | " + status)
		} else {
			m.setStatus("project loaded: " + id.String())
		}
	} else {
		m.setStatus("project loaded: " + id.String())
	}

	if m.client != nil && (m.videoID != "" || m.sourceURL != "") {
		return m, m.fetchCmd(m.videoID, m.sourceURL)
	}
	return m, cmd
}
