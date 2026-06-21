package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create draft dir: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temp draft: %w", err)
	}
	tempPath := temp.Name()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("write temp draft: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close temp draft: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace draft: %w", err)
	}
	if err := syncDir(dir); err != nil {
		return err
	}
	return nil
}

func syncDir(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open draft dir: %w", err)
	}
	defer handle.Close()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("sync draft dir: %w", err)
	}
	return nil
}
