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
	Load(id draft.DraftID) (draft.Snapshot, error)
	List() ([]draft.DraftID, error)
	Delete(id draft.DraftID) error
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
	if snapshot.ID == "" {
		return &StorageError{
			Code: CodeInvalidDraftID,
			Op:   "save",
			Err:  draft.ErrInvalidDraftID,
		}
	}

	stored := toStored(snapshot)
	data, err := json.Marshal(stored)
	if err != nil {
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: snapshot.ID.String(),
			Op:      "save",
			Err:     fmt.Errorf("marshal: %w", err),
		}
	}

	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: snapshot.ID.String(),
			Op:      "save",
			Err:     fmt.Errorf("mkdir: %w", err),
		}
	}

	finalPath := s.path(snapshot.ID)
	tmpPath := finalPath + ".tmp"

	if err := writeFileAtomic(tmpPath, data); err != nil {
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: snapshot.ID.String(),
			Op:      "save",
			Err:     fmt.Errorf("write temp: %w", err),
		}
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return &StorageError{
			Code:    CodeDraftWriteFailed,
			DraftID: snapshot.ID.String(),
			Op:      "save",
			Err:     fmt.Errorf("rename: %w", err),
		}
	}

	// Best-effort fsync of the parent directory so the rename is durable.
	_ = syncDir(s.dir)

	return nil
}

// Load reads and parses the draft identified by id.
func (s *FileStore) Load(id draft.DraftID) (draft.Snapshot, error) {
	if id == "" {
		return draft.Snapshot{}, &StorageError{
			Code: CodeInvalidDraftID,
			Op:   "load",
			Err:  draft.ErrInvalidDraftID,
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
func (s *FileStore) List() ([]draft.DraftID, error) {
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

	var ids []draft.DraftID
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		id, err := draft.NewDraftID(strings.TrimSuffix(name, ".json"))
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids, nil
}

// Delete removes the draft identified by id.
func (s *FileStore) Delete(id draft.DraftID) error {
	if id == "" {
		return &StorageError{
			Code: CodeInvalidDraftID,
			Op:   "delete",
			Err:  draft.ErrInvalidDraftID,
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

func (s *FileStore) path(id draft.DraftID) string {
	return filepath.Join(s.dir, id.String()+".json")
}


