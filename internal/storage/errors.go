package storage

import (
	"errors"
	"fmt"
)

// ErrorCode identifies the class of storage failure.
type ErrorCode string

const (
	CodeDraftNotFound    ErrorCode = "draft_not_found"
	CodeInvalidDraftID   ErrorCode = "invalid_draft_id"
	CodeCorruptDraft     ErrorCode = "corrupt_draft"
	CodeDraftWriteFailed ErrorCode = "draft_write_failed"
)

// StorageError reports a domain storage failure with a typed code.
type StorageError struct {
	Code    ErrorCode
	DraftID string
	Op      string
	Err     error
}

func (e *StorageError) Error() string {
	if e.DraftID == "" {
		return fmt.Sprintf("storage %s: %s: %v", e.Op, e.Code, e.Err)
	}
	return fmt.Sprintf("storage %s draft %s: %s: %v", e.Op, e.DraftID, e.Code, e.Err)
}

// Unwrap returns the wrapped error.
func (e *StorageError) Unwrap() error {
	return e.Err
}

// IsStorageError reports whether err is or wraps a *StorageError.
func IsStorageError(err error) bool {
	var storageErr *StorageError
	return errors.As(err, &storageErr)
}
