package draft

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

// ErrInvalidDraftID is returned when a draft ID cannot be constructed.
var ErrInvalidDraftID = errors.New("draft: invalid draft id")

// ErrInvalidProjectID is returned when a project ID cannot be constructed.
var ErrInvalidProjectID = errors.New("draft: invalid project id")

// DraftID identifies a saved draft. It is usually derived from a normalized video ID.
type DraftID string

// NewDraftID validates and constructs a DraftID.
func NewDraftID(videoID string) (DraftID, error) {
	if videoID == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidDraftID)
	}
	return DraftID(videoID), nil
}

// String returns the raw draft identifier.
func (id DraftID) String() string {
	return string(id)
}

// ProjectID identifies the single current draft for a project.
type ProjectID string

// NewProjectID validates and constructs a ProjectID safe for filesystem-backed storage.
func NewProjectID(raw string) (ProjectID, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidProjectID)
	}
	if id == "." || id == ".." {
		return "", fmt.Errorf("%w: reserved", ErrInvalidProjectID)
	}
	for _, r := range id {
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return "", fmt.Errorf("%w: unsafe character %q", ErrInvalidProjectID, r)
	}
	return ProjectID(id), nil
}

// String returns the raw project identifier.
func (id ProjectID) String() string {
	return string(id)
}

// Metadata holds the editable non-lyric fields associated with a draft.
type Metadata struct {
	VideoID    string
	TrackName  string
	ArtistName string
	AlbumName  string
	Duration   int
	UpdatedAt  time.Time
}

// Snapshot is the serializable unit of a draft: identity, metadata, and lyrics.
type Snapshot struct {
	ProjectID ProjectID
	ID        DraftID
	Metadata  Metadata
	Document  lyrics.Document
}

// ProjectSummary is the lightweight record shown by project pickers.
type ProjectSummary struct {
	ID       ProjectID
	Metadata Metadata
}
