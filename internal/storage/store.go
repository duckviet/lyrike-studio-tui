package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
)

// Store persists draft snapshots.
type Store interface {
	Save(snapshot draft.Snapshot) error
	Load(id draft.ProjectID) (draft.Snapshot, error)
	ListProjects() ([]draft.ProjectSummary, error)
	Delete(id draft.ProjectID) error
}

// FileStore is a filesystem-backed Store that writes drafts atomically.
type FileStore struct {
	dir string
}

// NewFileStore creates a Store rooted at dir. The directory is created if it
// does not already exist.

func NewFileStore(dir string) *FileStore {
	return &FileStore{dir: dir}
}

// Save writes snapshot atomically using a temp file + rename + fsync.
func (s *FileStore) Save(snapshot draft.Snapshot) error {
	projectID, err := snapshotProjectID(snapshot)
	if err != nil {
		return &StorageError{
			Code: CodeInvalidDraftID,
			Op:   "save",
			Err:  err,
		}
	}

	snapshot.ProjectID = projectID
	if snapshot.ID == "" {
		snapshot.ID = draft.DraftID(projectID.String())
	}
	stored := toStored(snapshot)
	data, err := json.Marshal(stored)
	if err != nil {
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: projectID.String(),
			Op:      "save",
			Err:     fmt.Errorf("marshal: %w", err),
		}
	}

	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: projectID.String(),
			Op:      "save",
			Err:     fmt.Errorf("mkdir: %w", err),
		}
	}

	finalPath := s.path(projectID)
	tmpPath := finalPath + ".tmp"

	if err := writeFileAtomic(tmpPath, data); err != nil {
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: projectID.String(),
			Op:      "save",
			Err:     fmt.Errorf("write temp: %w", err),
		}
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: projectID.String(),
			Op:      "save",
			Err:     fmt.Errorf("rename: %w", err),
		}
	}

	// Best-effort fsync of the parent directory so the rename is durable.
	_ = syncDir(s.dir)

	return nil
}

// Load reads and parses the draft identified by id.
func (s *FileStore) Load(id draft.ProjectID) (draft.Snapshot, error) {
	if id == "" {
		return draft.Snapshot{}, &StorageError{
			Code: CodeInvalidDraftID,
			Op:   "load",
			Err:  draft.ErrInvalidProjectID,
		}
	}

	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
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
			Err:     fmt.Errorf("read: %w", err),
		}
	}

	var stored storedSnapshot
	if err := json.Unmarshal(data, &stored); err != nil {
		return draft.Snapshot{}, &StorageError{
			Code:    CodeCorruptDraft,
			DraftID: id.String(),
			Op:      "load",
			Err:     fmt.Errorf("unmarshal: %w", err),
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

// List returns all draft IDs persisted in the store, sorted lexically.
func (s *FileStore) ListProjects() ([]draft.ProjectSummary, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, &StorageError{
			Code: CodeDraftWriteFailed,
			Op:   "list",
			Err:  fmt.Errorf("read dir: %w", err),
		}
	}

	var summaries []draft.ProjectSummary
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		id, err := draft.NewProjectID(strings.TrimSuffix(name, ".json"))
		if err != nil {
			continue
		}
		snapshot, err := s.Load(id)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, draft.ProjectSummary{
			ID:       id,
			Metadata: snapshot.Metadata,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		left := summaries[i]
		right := summaries[j]
		if !left.Metadata.UpdatedAt.Equal(right.Metadata.UpdatedAt) {
			return left.Metadata.UpdatedAt.After(right.Metadata.UpdatedAt)
		}
		return left.ID.String() < right.ID.String()
	})
	return summaries, nil
}

// Delete removes the draft identified by id.
func (s *FileStore) Delete(id draft.ProjectID) error {
	if id == "" {
		return &StorageError{
			Code: CodeInvalidDraftID,
			Op:   "delete",
			Err:  draft.ErrInvalidProjectID,
		}
	}

	if err := os.Remove(s.path(id)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &StorageError{
				Code:    CodeDraftNotFound,
				DraftID: id.String(),
				Op:      "delete",
				Err:     err,
			}
		}
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: id.String(),
			Op:      "delete",
			Err:     fmt.Errorf("remove: %w", err),
		}
	}

	return nil
}

func (s *FileStore) path(id draft.ProjectID) string {
	return filepath.Join(s.dir, id.String()+".json")
}

func snapshotProjectID(snapshot draft.Snapshot) (draft.ProjectID, error) {
	if snapshot.ProjectID != "" {
		return snapshot.ProjectID, nil
	}
	if snapshot.ID != "" {
		return draft.NewProjectID(snapshot.ID.String())
	}
	if snapshot.Metadata.VideoID != "" {
		return draft.NewProjectID(snapshot.Metadata.VideoID)
	}
	return "", draft.ErrInvalidProjectID
}
