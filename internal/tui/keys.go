package tui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/playback"
	"github.com/duckviet/lyrike-studio-tui/internal/storage"
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

func (m Model) updateFocusedPanel(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focus {
	case focusMedia:
		m.media, cmd = m.media.Update(msg)
	case focusWaveform:
		m.waveform = m.waveform.WithWidth(m.width - 2)
		m.waveform, cmd = m.waveform.Update(msg)
	case focusEditor:
		snap := m.player.Snapshot()
		m.editor = m.editor.WithTapPosition(snap.Position)
		m.editor, cmd = m.editor.Update(msg)
		if msg.Code == 't' {
			m.status = []string{"tap-sync applied"}
		}
	case focusPublish:
		if msg.Code == tea.KeyEscape || (m.publish.State() == publish.StateDone && msg.Code == tea.KeyEnter) {
			m.focus = focusEditor
			return m, nil
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
	store := storage.NewDefaultStore()
	doc := m.editor.Document
	idStr := m.videoID
	if idStr == "" {
		idStr = "default"
	}
	id, _ := draft.NewDraftID(idStr)

	track := m.trackName
	if track == "" {
		track = "Unknown Track"
	}
	artist := m.artistName
	if artist == "" {
		artist = "Unknown Artist"
	}

	durationSeconds := 0
	if m.player != nil {
		durationSeconds = int(m.player.Snapshot().Duration.Milliseconds() / 1000)
	}
	snap := draft.Snapshot{
		ID: id,
		Metadata: draft.Metadata{
			VideoID:    m.videoID,
			TrackName:  track,
			ArtistName: artist,
			Duration:   durationSeconds,
			UpdatedAt:  time.Now(),
		},
		Document: doc,
	}
	err := store.Save(snap)
	if err == nil {
		m.status = []string{"draft save complete"}
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
