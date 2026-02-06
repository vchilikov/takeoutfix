package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyFiles_CaseInsensitiveJSON(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "meta.JSON"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}

	jsonFiles, mediaFiles := classifyFiles(entries)

	if _, ok := jsonFiles["meta.JSON"]; !ok {
		t.Fatalf("expected meta.JSON in jsonFiles")
	}
	if _, ok := mediaFiles["photo.jpg"]; !ok {
		t.Fatalf("expected photo.jpg in mediaFiles")
	}
	if _, ok := mediaFiles["meta.JSON"]; ok {
		t.Fatalf("meta.JSON must not be classified as media")
	}
}
