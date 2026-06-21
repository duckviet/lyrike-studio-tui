package draft

import (
	"errors"
	"fmt"
	"time"

	"github.com/duckviet/lyrike-studio-tui/internal/domain/lyrics"
)

// ErrInvalidDraftID is returned when a draft ID cannot be constructed.
var ErrInvalidDraftID = errors.New("draft: invalid draft id")

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

// Metadata holds the editable non-lyric fields associated with a draft.
type Metadata struct {
	VideoID    string
	TrackName  string
	ArtistName string
	Duration   int
	UpdatedAt  time.Time
}

// Snapshot is the serializable unit of a draft: identity, metadata, and lyrics.
type Snapshot struct {
	ID       DraftID
	Metadata Metadata
	Document lyrics.Document
}
