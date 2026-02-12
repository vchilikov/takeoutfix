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
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg.xmp"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write sidecar: %v", err)
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
	if _, ok := mediaFiles["photo.jpg.xmp"]; ok {
		t.Fatalf("photo.jpg.xmp must not be classified as media")
	}
}

func TestClassifyFiles_UsesSupportedMediaWhitelist(t *testing.T) {
	dir := t.TempDir()

	for name := range map[string]string{
		"note.txt":   "text",
		"index.html": "<html/>",
		"data.csv":   "a,b",
		"photo.webp": "x",
		"meta.json":  "{}",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}

	_, mediaFiles := classifyFiles(entries)

	if _, ok := mediaFiles["photo.webp"]; !ok {
		t.Fatalf("expected photo.webp to be classified as media")
	}
	for _, name := range []string{"note.txt", "index.html", "data.csv", "meta.json"} {
		if _, ok := mediaFiles[name]; ok {
			t.Fatalf("%s must not be classified as media", name)
		}
	}
}
