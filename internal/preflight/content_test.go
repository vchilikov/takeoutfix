package preflight

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vchilikov/takeout-fix/utils/files"
)

func TestDetectProcessableTakeoutRoot_TrueForNestedTakeoutWithPairs(t *testing.T) {
	restore := stubContentDeps()
	defer restore()

	cwd := t.TempDir()
	takeoutRoot := filepath.Join(cwd, "Takeout")
	if err := os.MkdirAll(takeoutRoot, 0o755); err != nil {
		t.Fatalf("mkdir takeout: %v", err)
	}

	scanCalls := 0
	scanTakeoutContent = func(path string) (files.MediaScanResult, error) {
		scanCalls++
		if path != takeoutRoot {
			t.Fatalf("unexpected scan path: got %q, want %q", path, takeoutRoot)
		}
		return files.MediaScanResult{
			Pairs: map[string]string{"Takeout/photo.jpg": "Takeout/photo.jpg.json"},
		}, nil
	}

	root, ok, err := DetectProcessableTakeoutRoot(cwd)
	if err != nil {
		t.Fatalf("DetectProcessableTakeoutRoot error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if root != takeoutRoot {
		t.Fatalf("unexpected root: got %q, want %q", root, takeoutRoot)
	}
	if scanCalls != 1 {
		t.Fatalf("expected one scan call, got %d", scanCalls)
	}
}

func TestDetectProcessableTakeoutRoot_TrueWhenPathIsTakeoutDir(t *testing.T) {
	restore := stubContentDeps()
	defer restore()

	parent := t.TempDir()
	takeoutRoot := filepath.Join(parent, "Takeout")
	if err := os.MkdirAll(takeoutRoot, 0o755); err != nil {
		t.Fatalf("mkdir takeout: %v", err)
	}

	scanTakeoutContent = func(path string) (files.MediaScanResult, error) {
		if path != takeoutRoot {
			t.Fatalf("unexpected scan path: got %q, want %q", path, takeoutRoot)
		}
		return files.MediaScanResult{
			Pairs: map[string]string{"photo.jpg": "photo.jpg.json"},
		}, nil
	}

	root, ok, err := DetectProcessableTakeoutRoot(takeoutRoot)
	if err != nil {
		t.Fatalf("DetectProcessableTakeoutRoot error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if root != takeoutRoot {
		t.Fatalf("unexpected root: got %q, want %q", root, takeoutRoot)
	}
}

func TestDetectProcessableTakeoutRoot_FalseForArbitraryMediaFolder(t *testing.T) {
	restore := stubContentDeps()
	defer restore()

	root := t.TempDir()
	writeFile(t, root, "photo.jpg")
	writeFile(t, root, "photo.jpg.json")

	scanCalled := false
	scanTakeoutContent = func(string) (files.MediaScanResult, error) {
		scanCalled = true
		return files.MediaScanResult{}, nil
	}

	detected, ok, err := DetectProcessableTakeoutRoot(root)
	if err != nil {
		t.Fatalf("DetectProcessableTakeoutRoot error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false")
	}
	if detected != "" {
		t.Fatalf("expected empty detected root, got %q", detected)
	}
	if scanCalled {
		t.Fatalf("did not expect scan call when Takeout marker is absent")
	}
}

func TestDetectProcessableTakeoutRoot_FalseWhenNoPairs(t *testing.T) {
	restore := stubContentDeps()
	defer restore()

	cwd := t.TempDir()
	takeoutRoot := filepath.Join(cwd, "Takeout")
	if err := os.MkdirAll(takeoutRoot, 0o755); err != nil {
		t.Fatalf("mkdir takeout: %v", err)
	}

	scanTakeoutContent = func(path string) (files.MediaScanResult, error) {
		return files.MediaScanResult{
			Pairs: make(map[string]string),
		}, nil
	}

	detected, ok, err := DetectProcessableTakeoutRoot(cwd)
	if err != nil {
		t.Fatalf("DetectProcessableTakeoutRoot error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false")
	}
	if detected != "" {
		t.Fatalf("expected empty detected root, got %q", detected)
	}
}

func TestDetectProcessableTakeoutRoot_PropagatesScanError(t *testing.T) {
	restore := stubContentDeps()
	defer restore()

	cwd := t.TempDir()
	takeoutRoot := filepath.Join(cwd, "Takeout")
	if err := os.MkdirAll(takeoutRoot, 0o755); err != nil {
		t.Fatalf("mkdir takeout: %v", err)
	}

	scanTakeoutContent = func(string) (files.MediaScanResult, error) {
		return files.MediaScanResult{}, errors.New("scan failed")
	}

	detected, ok, err := DetectProcessableTakeoutRoot(cwd)
	if err == nil {
		t.Fatalf("expected error")
	}
	if ok {
		t.Fatalf("expected ok=false on error")
	}
	if detected != "" {
		t.Fatalf("expected empty detected root on error, got %q", detected)
	}
}

func TestDetectProcessableTakeoutRoot_PropagatesStatError(t *testing.T) {
	restore := stubContentDeps()
	defer restore()

	scanTakeoutContent = func(string) (files.MediaScanResult, error) {
		t.Fatalf("did not expect scan call on stat error")
		return files.MediaScanResult{}, nil
	}
	statPathForContent = func(string) (os.FileInfo, error) {
		return nil, errors.New("stat failed")
	}

	_, _, err := DetectProcessableTakeoutRoot(t.TempDir())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "stat failed") {
		t.Fatalf("expected stat error, got %v", err)
	}
}

func TestHasProcessableTakeout_UsesDetectedRoot(t *testing.T) {
	restore := stubContentDeps()
	defer restore()

	cwd := t.TempDir()
	takeoutRoot := filepath.Join(cwd, "Takeout")
	if err := os.MkdirAll(takeoutRoot, 0o755); err != nil {
		t.Fatalf("mkdir takeout: %v", err)
	}
	scanTakeoutContent = func(string) (files.MediaScanResult, error) {
		return files.MediaScanResult{
			Pairs: map[string]string{"Takeout/photo.jpg": "Takeout/photo.jpg.json"},
		}, nil
	}

	ok, err := HasProcessableTakeout(cwd)
	if err != nil {
		t.Fatalf("HasProcessableTakeout error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
}

func stubContentDeps() func() {
	origScanTakeoutContent := scanTakeoutContent
	origStatPathForContent := statPathForContent
	return func() {
		scanTakeoutContent = origScanTakeoutContent
		statPathForContent = origStatPathForContent
	}
}

func writeFile(t *testing.T, root string, rel string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
