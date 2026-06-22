package tui

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
	"github.com/duckviet/lyrike-studio-tui/internal/storage"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/editor"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/media"
)

type fetchInputMode uint8

const (
	fetchInputClosed fetchInputMode = iota
	fetchInputEnter
	fetchInputConfirmReplace
)

type fetchInput struct {
	input           string
	mode            fetchInputMode
	targetVideoID   string
	targetSourceURL string
}

func (f fetchInput) active() bool {
	return f.mode != fetchInputClosed
}

func (m Model) openFetchInput() Model {
	m.fetchInput = fetchInput{mode: fetchInputEnter}
	m.status = []string{"fetch media: enter URL or video ID"}
	return m
}

func isURL(s string) bool {
	return strings.Contains(s, "://")
}

// parseVideoIDInput extracts a YouTube video ID and optional source URL from
// user input. It accepts watch URLs, youtu.be short links, music.youtube.com
// URLs, embed/v paths, or a bare video ID. The logic mirrors the backend's
// extractYouTubeVideoID without importing internal/server.
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

func renderFetchInput(f fetchInput, width int, height int) string {
	var content strings.Builder
	content.WriteString("Fetch Media\n")

	switch f.mode {
	case fetchInputConfirmReplace:
		content.WriteString("Unsaved changes will be replaced.\n")
		content.WriteString("Enter: fetch ")
		content.WriteString(f.targetVideoID)
		content.WriteString(" | Esc: cancel")
	default:
		content.WriteString("YouTube URL or video ID:\n")
		maxInput := max(0, width-4)
		input := f.input
		if len(input) > maxInput {
			input = input[:maxInput]
		}
		content.WriteString(input)
		content.WriteString("\nEnter: fetch | Esc: cancel")
	}

	return focusedBorder.
		Width(max(0, width-2)).
		Height(max(0, height-2)).
		Render(lipgloss.NewStyle().Width(max(0, width-4)).Render(content.String()))
}

// newDefaultDocument returns a one-line placeholder document used when
// starting a new project from a URL or video ID.
// dup ok: TUI default doc (see cmd/lyrike-studio-tui/main.go defaultDocument).
func newDefaultDocument() lyrics.Document {
	ts, _ := lyrics.NewTimestamp(0)
	te, _ := lyrics.NewTimestamp(10_000)
	txt, _ := lyrics.NewText("Type lyrics here...")
	line, _ := lyrics.NewLine(ts, te, txt)
	doc, _ := lyrics.NewDocument([]lyrics.Line{line})
	return doc
}

func (m Model) updateFetchInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		m.fetchInput.input += msg.Content
	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyEscape:
			m.fetchInput = fetchInput{}
			m.status = []string{"fetch canceled"}
		case tea.KeyBackspace:
			if m.fetchInput.input != "" {
				runes := []rune(m.fetchInput.input)
				m.fetchInput.input = string(runes[:len(runes)-1])
			}
		case tea.KeyEnter:
			return m.submitFetchInput()
		default:
			if msg.Text != "" {
				m.fetchInput.input += msg.Text
			}
		}
	}
	return m, nil
}

func (m Model) submitFetchInput() (tea.Model, tea.Cmd) {
	if m.fetchInput.mode == fetchInputConfirmReplace {
		return m.applyFetch(m.fetchInput.targetVideoID, m.fetchInput.targetSourceURL)
	}

	videoID, sourceURL, ok := parseVideoIDInput(m.fetchInput.input)
	if !ok {
		m.status = []string{"invalid url or video id"}
		return m, nil
	}

	if m.dirty && m.projectID != "" && draft.ProjectID(videoID) != m.projectID {
		m.fetchInput.mode = fetchInputConfirmReplace
		m.fetchInput.targetVideoID = videoID
		m.fetchInput.targetSourceURL = sourceURL
		m.status = []string{"unsaved changes: Enter confirms fetch"}
		return m, nil
	}

	return m.applyFetch(videoID, sourceURL)
}

func (m Model) applyFetch(videoID, sourceURL string) (tea.Model, tea.Cmd) {
	pid, err := draft.NewProjectID(videoID)
	if err != nil {
		m.status = []string{"invalid video id: " + err.Error()}
		return m, nil
	}

	snapshot, err := m.draftStore.Load(pid)
	if err == nil && snapshot.ProjectID != "" {
		m = m.loadProject(pid)
		if sourceURL != "" {
			m.sourceURL = sourceURL
		}
		m.fetchInput = fetchInput{}
		if m.client == nil {
			m.status = []string{"backend unavailable"}
			return m, nil
		}
		return m, m.fetchCmd(videoID, sourceURL)
	}

	if !isNotFoundError(err) {
		m.status = []string{"project load failed: " + err.Error()}
		m.fetchInput = fetchInput{}
		return m, nil
	}

	m.projectID = pid
	m.videoID = videoID
	m.sourceURL = sourceURL
	m.trackName = ""
	m.artistName = ""
	m.albumName = ""
	m.editor = editor.NewPanel(newDefaultDocument())
	m.media = media.NewPanel()
	m.dirty = true
	m.fetchInput = fetchInput{}
	m.status = []string{"new project: " + pid.String()}
	if m.client == nil {
		m.status = []string{"backend unavailable"}
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
