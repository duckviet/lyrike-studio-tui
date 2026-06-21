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
	if msg.Code == tea.KeyTab {
		if msg.Mod == tea.ModShift {
			m.focus = m.prevFocus()
		} else {
			m.focus = m.nextFocus()
		}
		return m, nil
	}

	if msg.Code == tea.KeySpace || msg.Code == ' ' {
		if m.focus == focusEditor && m.editor.Editing {
			var cmd tea.Cmd
			snap := m.player.Snapshot()
			m.editor = m.editor.WithTapPosition(snap.Position)
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		}
		if m.player != nil {
			snap := m.player.Snapshot()
			if snap.State == playback.StatePlaying {
				_, _ = m.player.Pause()
				m.status = []string{"playback paused"}
			} else {
				_, _ = m.player.Play()
				m.status = []string{"playback playing"}
			}
		}
		return m, nil
	}

	if msg.Code == tea.KeyLeft {
		if m.focus == focusEditor && m.editor.Editing {
			var cmd tea.Cmd
			snap := m.player.Snapshot()
			m.editor = m.editor.WithTapPosition(snap.Position)
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		}
		if m.player != nil {
			snap := m.player.Snapshot()
			newPos := max(0, snap.Position.Milliseconds()-1000)
			pos, _ := playback.NewPosition(newPos)
			_, _ = m.player.Seek(pos)
			m.editor = m.editor.WithPlaybackPosition(newPos)
			m.status = []string{fmt.Sprintf("seek: %dms", newPos)}
		}
		return m, nil
	}

	if msg.Code == tea.KeyRight {
		if m.focus == focusEditor && m.editor.Editing {
			var cmd tea.Cmd
			snap := m.player.Snapshot()
			m.editor = m.editor.WithTapPosition(snap.Position)
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		}
		if m.player != nil {
			snap := m.player.Snapshot()
			newPos := min(snap.Duration.Milliseconds(), snap.Position.Milliseconds()+1000)
			pos, _ := playback.NewPosition(newPos)
			_, _ = m.player.Seek(pos)
			m.editor = m.editor.WithPlaybackPosition(newPos)
			m.status = []string{fmt.Sprintf("seek: %dms", newPos)}
		}
		return m, nil
	}

	if msg.Code == 's' && msg.Mod == tea.ModCtrl {
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
		
		snap := draft.Snapshot{
			ID: id,
			Metadata: draft.Metadata{
				VideoID:    m.videoID,
				TrackName:  track,
				ArtistName: artist,
				Duration:   int(m.player.Snapshot().Duration.Milliseconds() / 1000),
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
		return m, nil
	}

	if msg.Code == 'q' {
		if m.focus == focusEditor && m.editor.Editing {
			var cmd tea.Cmd
			snap := m.player.Snapshot()
			m.editor = m.editor.WithTapPosition(snap.Position)
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		}
		m.status = []string{"quit ready"}
		return m, tea.Quit
	}

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
