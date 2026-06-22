package drafts

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/draft"
)

var ErrInvalidInput = errors.New("draft: invalid input")

type Store struct {
	dir string
}

type Snapshot struct {
	ID           string          `json:"id"`
	Metadata     Metadata        `json:"metadata"`
	SyncedLyrics string          `json:"syncedLyrics"`
	raw          json.RawMessage `json:"-"`
}

type Metadata struct {
	VideoID    string          `json:"videoID"`
	TrackName  string          `json:"trackName"`
	ArtistName string          `json:"artistName"`
	AlbumName  string          `json:"albumName"`
	Duration   json.RawMessage `json:"duration"`
	UpdatedAt  string          `json:"updatedAt"`
}

type ProjectSummary struct {
	ID       string   `json:"id"`
	Metadata Metadata `json:"metadata"`
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) SaveRaw(id string, body []byte) error {
	projectID, err := draft.NewProjectID(id)
	if err != nil {
		return err
	}
	var snap Snapshot
	if err := json.Unmarshal(body, &snap); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if snap.ID != projectID.String() {
		return ErrInvalidInput
	}
	if snap.Metadata.UpdatedAt == "" || len(snap.Metadata.Duration) == 0 || snap.SyncedLyrics == "" {
		return ErrInvalidInput
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	tmp, err := os.CreateTemp(s.dir, projectID.String()+".*.tmp")
	if err != nil {
		return fmt.Errorf("temp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close: %w", err)
	}
	if err := os.Rename(tmpPath, s.path(projectID)); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}
	_ = syncDir(s.dir)
	return nil
}

func (s *Store) LoadRaw(id string) ([]byte, error) {
	projectID, err := draft.NewProjectID(id)
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(s.path(projectID))
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (s *Store) ListProjects() ([]ProjectSummary, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []ProjectSummary{}, nil
		}
		return nil, err
	}
	projects := make([]ProjectSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var snap Snapshot
		if err := json.Unmarshal(body, &snap); err != nil {
			return nil, err
		}
		projects = append(projects, ProjectSummary{ID: snap.ID, Metadata: snap.Metadata})
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Metadata.UpdatedAt == projects[j].Metadata.UpdatedAt {
			return projects[i].ID < projects[j].ID
		}
		return projects[i].Metadata.UpdatedAt > projects[j].Metadata.UpdatedAt
	})
	return projects, nil
}

func (s *Store) Delete(id string) error {
	projectID, err := draft.NewProjectID(id)
	if err != nil {
		return err
	}
	return os.Remove(s.path(projectID))
}

func IsNotFound(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

func IsInvalidID(err error) bool {
	return errors.Is(err, draft.ErrInvalidProjectID) || errors.Is(err, draft.ErrInvalidDraftID)
}

func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

func (s *Store) path(id draft.ProjectID) string {
	return filepath.Join(s.dir, id.String()+".json")
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
