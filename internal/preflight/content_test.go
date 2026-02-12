package preflight

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vchilikov/takeout-fix/utils/files"
)

func TestHasProcessableTakeout_TrueWithMatchedPairs(t *testing.T) {
	restore := stubContentScanner()
	defer restore()

	root := t.TempDir()
	writeFile(t, root, "photo.jpg")
	writeFile(t, root, "photo.jpg.json")

	ok, err := HasProcessableTakeout(root)
	if err != nil {
		t.Fatalf("HasProcessableTakeout error: %v", err)
	}
	if !ok {
		t.Fatalf("expected true when matched media/json exists")
	}
}

func TestHasProcessableTakeout_TrueWithMissingJSON(t *testing.T) {
	restore := stubContentScanner()
	defer restore()

	root := t.TempDir()
	writeFile(t, root, "photo.jpg")

	ok, err := HasProcessableTakeout(root)
	if err != nil {
		t.Fatalf("HasProcessableTakeout error: %v", err)
	}
	if !ok {
		t.Fatalf("expected true when media exists even without json")
	}
}

func TestHasProcessableTakeout_FalseWhenOnlyUnusedJSON(t *testing.T) {
	restore := stubContentScanner()
	defer restore()

	root := t.TempDir()
	writeFile(t, root, "orphan.json")

	ok, err := HasProcessableTakeout(root)
	if err != nil {
		t.Fatalf("HasProcessableTakeout error: %v", err)
	}
	if ok {
		t.Fatalf("expected false for folder with only unused json")
	}
}

func TestHasProcessableTakeout_TrueWithAmbiguousMedia(t *testing.T) {
	restore := stubContentScanner()
	defer restore()

	scanTakeoutContent = func(string) (files.MediaScanResult, error) {
		return files.MediaScanResult{
			AmbiguousJSON: map[string][]string{
				"IMG_0001.jpg": {"a.json", "b.json"},
			},
		}, nil
	}

	ok, err := HasProcessableTakeout(t.TempDir())
	if err != nil {
		t.Fatalf("HasProcessableTakeout error: %v", err)
	}
	if !ok {
		t.Fatalf("expected true when ambiguous media exists")
	}
}

func TestHasProcessableTakeout_PropagatesScanError(t *testing.T) {
	restore := stubContentScanner()
	defer restore()

	scanTakeoutContent = func(string) (files.MediaScanResult, error) {
		return files.MediaScanResult{}, errors.New("scan failed")
	}

	ok, err := HasProcessableTakeout(t.TempDir())
	if err == nil {
		t.Fatalf("expected error")
	}
	if ok {
		t.Fatalf("expected false on error")
	}
}

func stubContentScanner() func() {
	orig := scanTakeoutContent
	return func() {
		scanTakeoutContent = orig
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
