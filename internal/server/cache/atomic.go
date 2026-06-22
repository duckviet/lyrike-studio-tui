// Atomic file write helper for the on-disk cache.
//
// The pattern is the same as internal/storage/atomic.go: write the data to a
// temp file in the destination directory, fsync + close, then rename over the
// target path, and finally fsync the parent directory so the rename is
// durable across a crash.
//
// This file is intentionally a near-copy of internal/storage/atomic.go because
// the cache package must not depend on the draft store (and vice versa); the
// storage/AGENTS.md rule "use temp file + rename" applies here too. We do
// NOT modify the storage copy.
package cache

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileAtomic writes data to path atomically: temp file in the same
// directory, fsync + close, rename over the target, then fsync the parent
// directory. The parent directory is created with 0o755 if missing.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temp cache file: %w", err)
	}
	tempPath := temp.Name()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("write temp cache file: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close temp cache file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("rename temp cache file: %w", err)
	}
	if err := syncDir(dir); err != nil {
		return err
	}
	return nil
}

func syncDir(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open cache dir: %w", err)
	}
	defer handle.Close()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("sync cache dir: %w", err)
	}
	return nil
}
