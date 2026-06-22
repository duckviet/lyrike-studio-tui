package storage

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
	"github.com/duckviet/lyrike-studio-tui/internal/integrations/backend"
)

// RemoteStore is an HTTP-backed Store that delegates to a backend Client.
type RemoteStore struct {
	client *backend.Client
}

// NewRemoteStore creates a Store backed by the supplied backend client.
func NewRemoteStore(client *backend.Client) *RemoteStore {
	return &RemoteStore{client: client}
}

// Save serializes snapshot and PUTs it to /local-api/projects/{id}.
func (s *RemoteStore) Save(snapshot draft.Snapshot) error {
	stored := toStored(snapshot)
	id := stored.ID
	if id == "" {
		return &StorageError{
			Code: CodeInvalidDraftID,
			Op:   "save",
			Err:  draft.ErrInvalidProjectID,
		}
	}

	body, err := json.Marshal(stored)
	if err != nil {
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: id,
			Op:      "save",
			Err:     err,
		}
	}

	ctx := context.Background()
	if err := s.client.SaveDraftRaw(ctx, id, body); err != nil {
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: id,
			Op:      "save",
			Err:     err,
		}
	}
	return nil
}

// Load GETs /local-api/projects/{id} and decodes the stored snapshot.
func (s *RemoteStore) Load(id draft.ProjectID) (draft.Snapshot, error) {
	if id == "" {
		return draft.Snapshot{}, &StorageError{
			Code: CodeInvalidDraftID,
			Op:   "load",
			Err:  draft.ErrInvalidProjectID,
		}
	}

	ctx := context.Background()
	body, err := s.client.LoadDraftRaw(ctx, id.String())
	if err != nil {
		if errors.Is(err, backend.ErrDraftNotFound) {
			return draft.Snapshot{}, &StorageError{
				Code:    CodeDraftNotFound,
				DraftID: id.String(),
				Op:      "load",
				Err:     err,
			}
		}
		return draft.Snapshot{}, &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: id.String(),
			Op:      "load",
			Err:     err,
		}
	}

	var stored storedSnapshot
	if err := json.Unmarshal(body, &stored); err != nil {
		return draft.Snapshot{}, &StorageError{
			Code:    CodeCorruptDraft,
			DraftID: id.String(),
			Op:      "load",
			Err:     err,
		}
	}

	snapshot, err := fromStored(stored)
	if err != nil {
		return draft.Snapshot{}, &StorageError{
			Code:    CodeCorruptDraft,
			DraftID: id.String(),
			Op:      "load",
			Err:     err,
		}
	}
	return snapshot, nil
}

// ListProjects returns summaries from GET /local-api/projects.
func (s *RemoteStore) ListProjects() ([]draft.ProjectSummary, error) {
	ctx := context.Background()
	summaries, err := s.client.ListDrafts(ctx)
	if err != nil {
		return nil, &StorageError{
			Code: CodeDraftWriteFailed,
			Op:   "list",
			Err:  err,
		}
	}
	return summaries, nil
}

// Delete removes a draft via DELETE /local-api/projects/{id}.
func (s *RemoteStore) Delete(id draft.ProjectID) error {
	if id == "" {
		return &StorageError{
			Code: CodeInvalidDraftID,
			Op:   "delete",
			Err:  draft.ErrInvalidProjectID,
		}
	}

	ctx := context.Background()
	if err := s.client.DeleteDraft(ctx, id.String()); err != nil {
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: id.String(),
			Op:      "delete",
			Err:     err,
		}
	}
	return nil
}
