package storage

import (
	"fmt"
	"time"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

type storedSnapshot struct {
	ID           string         `json:"id"`
	Metadata     storedMetadata `json:"metadata"`
	SyncedLyrics string         `json:"syncedLyrics"`
}

type storedMetadata struct {
	VideoID    string `json:"videoID"`
	TrackName  string `json:"trackName"`
	ArtistName string `json:"artistName"`
	AlbumName  string `json:"albumName"`
	Duration   int    `json:"duration"`
	UpdatedAt  string `json:"updatedAt"`
}

func toStored(snapshot draft.Snapshot) storedSnapshot {
	id := snapshot.ProjectID.String()
	if id == "" {
		id = snapshot.ID.String()
	}
	return storedSnapshot{
		ID:           id,
		Metadata:     toStoredMetadata(snapshot.Metadata),
		SyncedLyrics: lyrics.FormatLRCWithEnd(snapshot.Document),
	}
}

func toStoredMetadata(m draft.Metadata) storedMetadata {
	return storedMetadata{
		VideoID:    m.VideoID,
		TrackName:  m.TrackName,
		ArtistName: m.ArtistName,
		AlbumName:  m.AlbumName,
		Duration:   m.Duration,
		UpdatedAt:  m.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func fromStored(stored storedSnapshot) (draft.Snapshot, error) {
	projectID, err := draft.NewProjectID(stored.ID)
	if err != nil {
		return draft.Snapshot{}, fmt.Errorf("projectID: %w", err)
	}
	id, err := draft.NewDraftID(stored.ID)
	if err != nil {
		return draft.Snapshot{}, fmt.Errorf("invalid stored id %q: %w", stored.ID, err)
	}

	updatedAt, err := time.Parse(time.RFC3339Nano, stored.Metadata.UpdatedAt)
	if err != nil {
		return draft.Snapshot{}, fmt.Errorf("invalid updatedAt %q: %w", stored.Metadata.UpdatedAt, err)
	}

	doc, err := lyrics.ParseLRC(stored.SyncedLyrics)
	if err != nil {
		return draft.Snapshot{}, fmt.Errorf("parse lyrics: %w", err)
	}

	return draft.Snapshot{
		ProjectID: projectID,
		ID:        id,
		Metadata: draft.Metadata{
			VideoID:    stored.Metadata.VideoID,
			TrackName:  stored.Metadata.TrackName,
			ArtistName: stored.Metadata.ArtistName,
			AlbumName:  stored.Metadata.AlbumName,
			Duration:   stored.Metadata.Duration,
			UpdatedAt:  updatedAt,
		},
		Document: doc,
	}, nil
}
