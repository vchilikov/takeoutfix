package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ArchiveState struct {
	Fingerprint string `json:"fingerprint"`
	Extracted   bool   `json:"extracted"`
	Deleted     bool   `json:"deleted"`
}

type RunState struct {
	Archives  map[string]ArchiveState `json:"archives"`
	LastRunAt string                  `json:"last_run_at,omitempty"`
}

func New() RunState {
	return RunState{Archives: make(map[string]ArchiveState)}
}

func Load(path string) (RunState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return New(), nil
		}
		return RunState{}, fmt.Errorf("read state: %w", err)
	}

	st := New()
	if err := json.Unmarshal(data, &st); err != nil {
		return RunState{}, fmt.Errorf("parse state: %w", err)
	}
	if st.Archives == nil {
		st.Archives = make(map[string]ArchiveState)
	}

	return st, nil
}

func Save(path string, st RunState) error {
	if st.Archives == nil {
		st.Archives = make(map[string]ArchiveState)
	}
	st.LastRunAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), "state-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp state: %w", err)
	}
	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temp state: %w", err)
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp state: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp state: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp state: %w", err)
	}
	cleanup = false
	return nil
}

func ShouldSkipExtraction(st RunState, archiveName string, fingerprint string) bool {
	entry, ok := st.Archives[archiveName]
	if !ok {
		return false
	}
	return entry.Extracted && entry.Fingerprint == fingerprint
}
