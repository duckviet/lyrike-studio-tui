package tui

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
	"github.com/duckviet/lyrike-studio-tui/internal/storage"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/editor"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/media"
)

type fetchInput struct {
	input       textinput.Model
	activeState bool
}

func (f fetchInput) active() bool {
	return f.activeState
}

func (m Model) openFetchInput() Model {
	m.fetchInput.activeState = true
	m.fetchInput.input.SetValue("")
	m.fetchInput.input.Placeholder = ""
	m.fetchInput.input.Focus()
	m.overlay = overlayInput
	m.setStatus("fetch media: enter URL or video ID")
	return m
}

func isURL(s string) bool {
	return strings.Contains(s, "://")
}

func parseVideoIDInput(raw string) (videoID, sourceURL string, ok bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", false
	}

	if !isURL(trimmed) {
		if _, err := draft.NewProjectID(trimmed); err != nil {
			return "", "", false
		}
		return trimmed, "", true
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return "", "", false
	}

	host := strings.ToLower(u.Hostname())
	switch {
	case strings.Contains(host, "youtu.be"):
		id := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")[0]
		if id == "" {
			return "", "", false
		}
		return id, trimmed, true
	case strings.Contains(host, "youtube.com"):
		if v := u.Query().Get("v"); v != "" {
			return v, trimmed, true
		}
		for _, prefix := range []string{"/embed/", "/v/"} {
			if strings.HasPrefix(u.Path, prefix) {
				id := strings.TrimPrefix(u.Path, prefix)
				if id == "" {
					return "", "", false
				}
				return id, trimmed, true
			}
		}
	}

	return "", "", false
}

func renderFetchInput(f fetchInput, width int, height int, th Theme) string {
	boxWidth := width - 8
	if boxWidth > 56 {
		boxWidth = 56
	}
	if boxWidth < 20 {
		boxWidth = width
	}

	var content strings.Builder
	content.WriteString(th.ModalTitle.Render("Fetch Media") + "\n\n")
	content.WriteString(th.Text.Render("YouTube URL or video ID:") + "\n")
	content.WriteString(f.input.View() + "\n\n")
	content.WriteString(th.FooterKey.Render("Enter") + " " + th.FooterDesc.Render("fetch") + "   " +
		th.FooterKey.Render("Esc") + " " + th.FooterDesc.Render("cancel"))

	return overlayBlock(content.String(), boxWidth, th)
}

func newDefaultDocument() lyrics.Document {
	ts, _ := lyrics.NewTimestamp(0)
	te, _ := lyrics.NewTimestamp(10_000)
	txt, _ := lyrics.NewText("Type lyrics here...")
	line, _ := lyrics.NewLine(ts, te, txt)
	doc, _ := lyrics.NewDocument([]lyrics.Line{line})
	return doc
}

func (m Model) updateFetchInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyEscape:
			m.overlay = overlayNone
			m.fetchInput.activeState = false
			m.setStatus("fetch canceled")
			return m, nil
		case tea.KeyEnter:
			return m.submitFetchInput()
		}
	}
	m.fetchInput.input, cmd = m.fetchInput.input.Update(msg)
	return m, cmd
}

func (m Model) submitFetchInput() (tea.Model, tea.Cmd) {
	videoID, sourceURL, ok := parseVideoIDInput(m.fetchInput.input.Value())
	if !ok {
		m.setErrorStatus("invalid url or video id")
		return m, nil
	}

	if m.dirty && m.projectID != "" && draft.ProjectID(videoID) != m.projectID {
		m.fetchInput.activeState = false
		return m.confirmAction(
			"Unsaved changes",
			"Discard current work and fetch new project?",
			true,
			func() tea.Msg {
				return confirmFetchMsg{videoID: videoID, sourceURL: sourceURL}
			},
			func() tea.Msg {
				return cancelFetchMsg{}
			},
		), nil
	}

	m.overlay = overlayNone
	m.fetchInput.activeState = false
	return m.applyFetch(videoID, sourceURL)
}

func (m Model) applyFetch(videoID, sourceURL string) (tea.Model, tea.Cmd) {
	pid, err := draft.NewProjectID(videoID)
	if err != nil {
		m.setErrorStatus("invalid video id: " + err.Error())
		return m, nil
	}

	snapshot, err := m.draftStore.Load(pid)
	if err == nil && snapshot.ProjectID != "" {
		m.sourceURL = sourceURL
		m, cmd := m.loadProject(pid)
		m.fetchInput.activeState = false
		if m.client == nil {
			m.setErrorStatus("backend unavailable")
			return m, nil
		}
		return m, cmd
	}

	if !isNotFoundError(err) {
		m.setErrorStatus("project load failed: " + err.Error())
		m.fetchInput.activeState = false
		return m, nil
	}

	m.projectID = pid
	m.videoID = videoID
	m.sourceURL = sourceURL
	m.trackName = ""
	m.artistName = ""
	m.albumName = ""
	m.editor = editor.NewPanel(newDefaultDocument()).WithTheme(m.theme)
	m.media = media.NewPanel().WithTheme(m.theme)
	m.dirty = true
	m.fetchInput.activeState = false
	m.setStatus("new project: " + pid.String())
	if m.client == nil {
		m.setErrorStatus("backend unavailable")
		return m, nil
	}
	return m, m.fetchCmd(videoID, sourceURL)
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var storageErr *storage.StorageError
	if errors.As(err, &storageErr) {
		return storageErr.Code == storage.CodeDraftNotFound
	}
	return errors.Is(err, os.ErrNotExist)
}

func (m Model) fetchCmd(videoID, sourceURL string) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.Fetch(context.Background(), backend.FetchRequest{VideoID: videoID, URL: sourceURL})
		return fetchMediaMsg{resp: resp, err: err}
	}
}

type confirmFetchMsg struct {
	videoID   string
	sourceURL string
}

type cancelFetchMsg struct{}
