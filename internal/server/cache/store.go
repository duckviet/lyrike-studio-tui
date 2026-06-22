// Package cache is the on-disk CRUD layer for the lyrike backend.
//
// Task 3 of the go-backend plan ports metadata_service.py: it stores three
// kinds of JSON artifacts rooted at a single cacheDir:
//
//	{cacheDir}/
//	├── media/        {videoID}.json
//	├── peaks/        {videoID}/{source}.json
//	└── transcripts/  {videoID}.json
//
// All payloads are map[string]any to mirror the Python `dict` shape used by
// the existing service. HTTP handlers (a later task) are responsible for
// parsing the wire JSON into typed structs — the cache layer never sees the
// HTTP boundary.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Code identifies the class of cache failure. Every *CacheError carries one.
type Code string

const (
	CodeNotFound    Code = "not_found"
	CodeCorrupt     Code = "corrupt_json"
	CodeWriteFailed Code = "write_failed"
	CodeInvalidID   Code = "invalid_id"
)

// Sentinel errors. Callers can match any of them with errors.Is.
var (
	ErrNotFound    = errors.New("cache: not found")
	ErrCorrupt     = errors.New("cache: corrupt json")
	ErrWriteFailed = errors.New("cache: write failed")
	ErrInvalidID   = errors.New("cache: invalid video id")
)

// CacheError is the typed error returned by every Store method. It wraps the
// underlying cause and reports a Code that callers can match with errors.Is
// against the package sentinels.
type CacheError struct {
	Code Code
	Path string
	Op   string
	Err  error
}

func (e *CacheError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("cache %s %s: %s: %v", e.Op, e.Path, e.Code, e.Err)
	}
	return fmt.Sprintf("cache %s: %s: %v", e.Op, e.Code, e.Err)
}

// Unwrap returns the wrapped cause.
func (e *CacheError) Unwrap() error { return e.Err }

// Is enables errors.Is matching against the package sentinels.
func (e *CacheError) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return e.Code == CodeNotFound
	case ErrCorrupt:
		return e.Code == CodeCorrupt
	case ErrWriteFailed:
		return e.Code == CodeWriteFailed
	case ErrInvalidID:
		return e.Code == CodeInvalidID
	}
	return false
}

// Store persists video metadata, peaks, and transcripts on disk. All writes
// are atomic (temp + rename + fsync parent dir) and parent directories are
// created on demand.
type Store struct {
	cacheDir string
}

// NewStore returns a Store rooted at cacheDir. The directory is NOT created
// here — the caller is responsible for the cache layout (see
// server.ensureCacheLayout). Reads return ErrNotFound when the file is
// absent; writes create missing parents on demand.
func NewStore(cacheDir string) *Store {
	return &Store{cacheDir: cacheDir}
}

// MetadataPath returns the on-disk path for a video's metadata JSON.
func (s *Store) MetadataPath(videoID string) string {
	return filepath.Join(s.cacheDir, "media", videoID+".json")
}

// PeaksPath returns the on-disk path for a video's peaks JSON for the given
// source (e.g. "original", "demucs").
func (s *Store) PeaksPath(videoID, source string) string {
	return filepath.Join(s.cacheDir, "peaks", videoID, source+".json")
}

// TranscriptPath returns the on-disk path for a video's transcript JSON.
func (s *Store) TranscriptPath(videoID string) string {
	return filepath.Join(s.cacheDir, "transcripts", videoID+".json")
}

// LoadMetadata reads and decodes the metadata JSON for videoID.
func (s *Store) LoadMetadata(videoID string) (map[string]any, error) {
	if err := validateVideoID(videoID); err != nil {
		return nil, err
	}
	return s.loadJSON(s.MetadataPath(videoID), "load metadata")
}

// SaveMetadata marshals payload and writes it atomically.
func (s *Store) SaveMetadata(videoID string, payload map[string]any) error {
	if err := validateVideoID(videoID); err != nil {
		return err
	}
	return s.saveJSON(s.MetadataPath(videoID), payload, "save metadata")
}

// LoadPeaks reads and decodes the peaks JSON for (videoID, source).
func (s *Store) LoadPeaks(videoID, source string) (map[string]any, error) {
	if err := validateVideoID(videoID); err != nil {
		return nil, err
	}
	if err := validateSource(source); err != nil {
		return nil, err
	}
	return s.loadJSON(s.PeaksPath(videoID, source), "load peaks")
}

// SavePeaks marshals payload and writes it atomically.
func (s *Store) SavePeaks(videoID, source string, payload map[string]any) error {
	if err := validateVideoID(videoID); err != nil {
		return err
	}
	if err := validateSource(source); err != nil {
		return err
	}
	return s.saveJSON(s.PeaksPath(videoID, source), payload, "save peaks")
}

// LoadTranscript reads and decodes the transcript JSON for videoID.
func (s *Store) LoadTranscript(videoID string) (map[string]any, error) {
	if err := validateVideoID(videoID); err != nil {
		return nil, err
	}
	return s.loadJSON(s.TranscriptPath(videoID), "load transcript")
}

// SaveTranscript marshals payload and writes it atomically.
func (s *Store) SaveTranscript(videoID string, payload map[string]any) error {
	if err := validateVideoID(videoID); err != nil {
		return err
	}
	return s.saveJSON(s.TranscriptPath(videoID), payload, "save transcript")
}

func (s *Store) loadJSON(path, op string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &CacheError{Code: CodeNotFound, Path: path, Op: op, Err: err}
		}
		return nil, &CacheError{Code: CodeWriteFailed, Path: path, Op: op, Err: err}
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, &CacheError{Code: CodeCorrupt, Path: path, Op: op, Err: err}
	}
	return out, nil
}

func (s *Store) saveJSON(path string, payload map[string]any, op string) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return &CacheError{Code: CodeWriteFailed, Path: path, Op: op, Err: err}
	}
	if err := writeFileAtomic(path, data); err != nil {
		return &CacheError{Code: CodeWriteFailed, Path: path, Op: op, Err: err}
	}
	return nil
}

func validateVideoID(id string) error {
	if id == "" {
		return &CacheError{Code: CodeInvalidID, Op: "validate", Err: ErrInvalidID}
	}
	if strings.ContainsAny(id, `/\`) || id == "." || id == ".." {
		return &CacheError{Code: CodeInvalidID, Op: "validate", Err: ErrInvalidID}
	}
	return nil
}

func validateSource(src string) error {
	if src == "" {
		return &CacheError{Code: CodeInvalidID, Op: "validate source", Err: ErrInvalidID}
	}
	if strings.ContainsAny(src, `/\`) || src == "." || src == ".." {
		return &CacheError{Code: CodeInvalidID, Op: "validate source", Err: ErrInvalidID}
	}
	return nil
}
