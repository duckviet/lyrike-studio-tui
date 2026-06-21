package storage

import (
	"os"
	"path/filepath"
)

// DraftDir returns the platform-specific XDG state directory for drafts.
// It respects XDG_STATE_HOME and falls back to ~/.local/state.
func DraftDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "lyrike-studio-tui", "drafts")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "state", "lyrike-studio-tui", "drafts")
}

// NewDefaultStore creates a FileStore rooted at the XDG draft directory.
func NewDefaultStore() *FileStore {
	return NewFileStore(DraftDir())
}
