package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

// ErrDraftNotFound is returned by LoadDraft when the backend reports 404.
var ErrDraftNotFound = errors.New("draft not found")

type draftSnapshot struct {
	ID           string        `json:"id"`
	Metadata     draftMetadata `json:"metadata"`
	SyncedLyrics string        `json:"syncedLyrics"`
}

type draftMetadata struct {
	VideoID    string `json:"videoID"`
	TrackName  string `json:"trackName"`
	ArtistName string `json:"artistName"`
	AlbumName  string `json:"albumName"`
	Duration   int    `json:"duration"`
	UpdatedAt  string `json:"updatedAt"`
}

type draftProjectSummary struct {
	ID       string        `json:"id"`
	Metadata draftMetadata `json:"metadata"`
}

// SaveDraft writes snapshot to the backend as PUT /local-api/projects/{id}.
func (c *Client) SaveDraft(ctx context.Context, id string, snapshot draft.Snapshot) error {
	stored := toDraftSnapshot(snapshot)
	body, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("save draft: marshal: %w", err)
	}
	return c.SaveDraftRaw(ctx, id, body)
}

// SaveDraftRaw PUTs an already-serialized draft snapshot body.
func (c *Client) SaveDraftRaw(ctx context.Context, id string, body []byte) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/local-api/projects/"+id, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("save draft: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return fmt.Errorf("save draft: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return expectStatus(resp, http.StatusOK)
	}
	return nil
}

// LoadDraft reads a snapshot from GET /local-api/projects/{id}.
func (c *Client) LoadDraft(ctx context.Context, id string) (draft.Snapshot, error) {
	body, err := c.LoadDraftRaw(ctx, id)
	if err != nil {
		return draft.Snapshot{}, err
	}
	var stored draftSnapshot
	if err := json.Unmarshal(body, &stored); err != nil {
		return draft.Snapshot{}, fmt.Errorf("load draft: decode: %w", err)
	}
	return fromDraftSnapshot(stored)
}

// LoadDraftRaw returns the raw GET body for /local-api/projects/{id}.
func (c *Client) LoadDraftRaw(ctx context.Context, id string) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/local-api/projects/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("load draft: create request: %w", err)
	}

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("load draft: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, ErrDraftNotFound
	}
	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, err
	}
	return mustReadBody(resp), nil
}

// ListDrafts returns all drafts from GET /local-api/projects.
func (c *Client) ListDrafts(ctx context.Context) ([]draft.ProjectSummary, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/local-api/projects", nil)
	if err != nil {
		return nil, fmt.Errorf("list drafts: create request: %w", err)
	}

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("list drafts: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var stored []draftProjectSummary
	if err := json.Unmarshal(mustReadBody(resp), &stored); err != nil {
		return nil, fmt.Errorf("list drafts: decode: %w", err)
	}

	summaries := make([]draft.ProjectSummary, len(stored))
	for i, item := range stored {
		summary, err := fromDraftProjectSummary(item)
		if err != nil {
			return nil, fmt.Errorf("list drafts: item %d: %w", i, err)
		}
		summaries[i] = summary
	}
	return summaries, nil
}

// DeleteDraft removes a draft via DELETE /local-api/projects/{id}.
func (c *Client) DeleteDraft(ctx context.Context, id string) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/local-api/projects/"+id, nil)
	if err != nil {
		return fmt.Errorf("delete draft: create request: %w", err)
	}

	resp, err := c.do(httpReq)
	if err != nil {
		return fmt.Errorf("delete draft: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return expectStatus(resp, http.StatusOK)
	}
	return nil
}

func toDraftSnapshot(snapshot draft.Snapshot) draftSnapshot {
	id := snapshot.ProjectID.String()
	if id == "" {
		id = snapshot.ID.String()
	}
	return draftSnapshot{
		ID:           id,
		Metadata:     toDraftMetadata(snapshot.Metadata),
		SyncedLyrics: lyrics.FormatLRCWithEnd(snapshot.Document),
	}
}

func toDraftMetadata(m draft.Metadata) draftMetadata {
	return draftMetadata{
		VideoID:    m.VideoID,
		TrackName:  m.TrackName,
		ArtistName: m.ArtistName,
		AlbumName:  m.AlbumName,
		Duration:   m.Duration,
		UpdatedAt:  m.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func fromDraftSnapshot(stored draftSnapshot) (draft.Snapshot, error) {
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

func fromDraftProjectSummary(stored draftProjectSummary) (draft.ProjectSummary, error) {
	id, err := draft.NewProjectID(stored.ID)
	if err != nil {
		return draft.ProjectSummary{}, fmt.Errorf("projectID: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, stored.Metadata.UpdatedAt)
	if err != nil {
		return draft.ProjectSummary{}, fmt.Errorf("invalid updatedAt %q: %w", stored.Metadata.UpdatedAt, err)
	}
	return draft.ProjectSummary{
		ID: id,
		Metadata: draft.Metadata{
			VideoID:    stored.Metadata.VideoID,
			TrackName:  stored.Metadata.TrackName,
			ArtistName: stored.Metadata.ArtistName,
			AlbumName:  stored.Metadata.AlbumName,
			Duration:   stored.Metadata.Duration,
			UpdatedAt:  updatedAt,
		},
	}, nil
}
