package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWorkDir_DefaultsToCWD(t *testing.T) {
	target := t.TempDir()

	got, err := resolveWorkDir(nil, func() (string, error) {
		return target, nil
	}, os.Stat)
	if err != nil {
		t.Fatalf("resolveWorkDir error: %v", err)
	}
	if got != target {
		t.Fatalf("expected %q, got %q", target, got)
	}
}

func TestResolveWorkDir_FlagValue(t *testing.T) {
	target := t.TempDir()

	got, err := resolveWorkDir([]string{"--workdir", target}, func() (string, error) {
		return "/should/not/be/used", nil
	}, os.Stat)
	if err != nil {
		t.Fatalf("resolveWorkDir error: %v", err)
	}
	if got != target {
		t.Fatalf("expected %q, got %q", target, got)
	}
}

func TestResolveWorkDir_NotExists(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	_, err := resolveWorkDir([]string{"--workdir", missing}, os.Getwd, os.Stat)
	if err == nil {
		t.Fatalf("expected error for missing path")
	}
}

func TestResolveWorkDir_NotDirectory(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := resolveWorkDir([]string{"--workdir", filePath}, os.Getwd, os.Stat)
	if err == nil {
		t.Fatalf("expected error for file path")
	}
}
