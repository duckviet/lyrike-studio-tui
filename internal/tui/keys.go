package tui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
	"github.com/duckviet/lyrike-studio-tui/internal/tui/publish"
)

func (m Model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	msg = normalizeKeyPress(msg)

	if action := globalKeyAction(msg); action != keyActionNone {
		return m.applyRootKeyAction(action)
	}
	if m.editorEditModeOwnsKey() {
		return m.updateFocusedPanel(msg)
	}
	if m.focus == focusEditor && (msg.Code == 'f' || (msg.Code == 'f' && msg.Mod == tea.ModCtrl)) {
		return m.updateFocusedPanel(msg)
	}
	if action := nonEditingRootKeyAction(msg); action != keyActionNone {
		return m.applyRootKeyAction(action)
	}
	return m.updateFocusedPanel(msg)
}

func (m Model) applyRootKeyAction(action keyAction) (tea.Model, tea.Cmd) {
	switch action {
	case keyActionFocusNext:
		m.focus = m.nextFocus()
	case keyActionFocusPrev:
		m.focus = m.prevFocus()
	case keyActionSaveDraft:
		m = m.saveDraft()
	case keyActionOpenProjects:
		m = m.openProjectPicker()
	case keyActionOpenFetch:
		m = m.openFetchInput()
	case keyActionEditMetadata:
		m = m.openMetadataEditor()
	case keyActionTogglePlayback:
		m = m.togglePlayback()
	case keyActionSeekBackward:
		m = m.seekPlayback(-1000)
	case keyActionSeekForward:
		m = m.seekPlayback(1000)
	case keyActionToggleFollow:
		m.waveform = m.waveform.ToggleFollow()
	case keyActionQuit:
		m.status = []string{"quit ready"}
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) updateFocusedPanel(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focus {
	case focusMedia:
		m.media, cmd = m.media.Update(msg)
	case focusWaveform:
		m.waveform = m.waveform.WithWidth(m.width - 2)
		m.waveform, cmd = m.waveform.Update(msg)
	case focusEditor:
		before := m.editor.Document
		snap := m.player.Snapshot()
		m.editor = m.editor.WithTapPosition(snap.Position)
		m.editor, cmd = m.editor.Update(msg)
		if fmt.Sprint(before) != fmt.Sprint(m.editor.Document) {
			m.dirty = true
		}
		if key, ok := msg.(tea.KeyPressMsg); ok && key.Code == 't' {
			m.status = []string{"tap-sync applied"}
		}
	case focusPublish:
		if key, ok := msg.(tea.KeyPressMsg); ok {
			if key.Code == tea.KeyEscape || (m.publish.State() == publish.StateDone && key.Code == tea.KeyEnter) {
				m.focus = focusEditor
				return m, nil
			}
		}
		m.publish, cmd = m.publish.Update(msg)
	}
	return m, cmd
}

func (m Model) togglePlayback() Model {
	if m.player == nil {
		return m
	}
	snap := m.player.Snapshot()
	if snap.State == playback.StatePlaying {
		_, _ = m.player.Pause()
		m.status = []string{"playback paused"}
	} else {
		_, _ = m.player.Play()
		m.status = []string{"playback playing"}
	}
	return m
}

func (m Model) seekPlayback(deltaMS int64) Model {
	if m.player == nil {
		return m
	}
	snap := m.player.Snapshot()
	newPos := snap.Position.Milliseconds() + deltaMS
	newPos = max(0, min(snap.Duration.Milliseconds(), newPos))
	pos, _ := playback.NewPosition(newPos)
	_, _ = m.player.Seek(pos)
	m.editor = m.editor.WithPlaybackPosition(newPos)
	m.status = []string{fmt.Sprintf("seek: %dms", newPos)}
	return m
}

func (m Model) saveDraft() Model {
	if m.projectID == "" {
		m = m.openFetchInput()
		m.status = []string{"fetch a video before saving"}
		return m
	}
	doc := m.editor.Document

	track := m.trackName
	if track == "" {
		track = "Unknown Track"
	}
	artist := m.artistName
	if artist == "" {
		artist = "Unknown Artist"
	}
	album := m.albumName
	if album == "" {
		album = "Unknown Album"
	}

	durationSeconds := 0
	if m.player != nil {
		durationSeconds = int(m.player.Snapshot().Duration.Milliseconds() / 1000)
	}
	snap := draft.Snapshot{
		ProjectID: m.projectID,
		ID:        draft.DraftID(m.projectID.String()),
		Metadata: draft.Metadata{
			VideoID:    m.videoID,
			TrackName:  track,
			ArtistName: artist,
			AlbumName:  album,
			Duration:   durationSeconds,
			UpdatedAt:  time.Now(),
		},
		Document: doc,
	}
	err := m.draftStore.Save(snap)
	if err == nil {
		m.dirty = false
		m.status = []string{"project saved: " + m.projectID.String()}
	} else {
		m.status = []string{"draft save failed: " + err.Error()}
	}
	return m
}

func (m Model) nextFocus() focus {
	switch m.focus {
	case focusMedia:
		return focusWaveform
	case focusWaveform:
		return focusEditor
	case focusEditor:
		return focusMedia
	default:
		return focusMedia
	}
}

func (m Model) prevFocus() focus {
	switch m.focus {
	case focusEditor:
		return focusWaveform
	case focusWaveform:
		return focusMedia
	case focusMedia:
		return focusEditor
	default:
		return focusEditor
	}
}

func (m Model) WithProjectMetadata(track, artist, album string) Model {
	m.trackName = track
	m.artistName = artist
	m.albumName = album
	m.media = m.media.WithMetadata(track, artist, album)
	return m
}
