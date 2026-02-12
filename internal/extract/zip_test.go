package extract

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExtractArchiveExtractsFiles(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "a.zip")
	if err := writeZip(zipPath, map[string]string{"one.txt": "1", "two.txt": "2"}); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	dest := filepath.Join(dir, "out")
	files, err := ExtractArchive(zipPath, dest)
	if err != nil {
		t.Fatalf("ExtractArchive error: %v", err)
	}
	if files != 2 {
		t.Fatalf("expected 2 extracted files, got %d", files)
	}
	for _, name := range []string{"one.txt", "two.txt"} {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			t.Fatalf("expected extracted file %s: %v", name, err)
		}
	}
}

func TestSafeJoinBlocksTraversal(t *testing.T) {
	if _, err := safeJoin("/tmp/base", "../escape"); err == nil {
		t.Fatalf("expected traversal error")
	}
}

func TestExtractArchiveRejectsSymlinkComponent(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "a.zip")
	if err := writeZip(zipPath, map[string]string{"link/file.txt": "1"}); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}
	outside := filepath.Join(dir, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(dest, "link")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	_, err := ExtractArchive(zipPath, dest)
	if err == nil {
		t.Fatalf("expected symlink component error")
	}
	if !strings.Contains(err.Error(), "symlink component") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractArchiveRejectsSymlinkTargetFile(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "a.zip")
	if err := writeZip(zipPath, map[string]string{"file.txt": "1"}); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}
	outsideFile := filepath.Join(dir, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(dest, "file.txt")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	_, err := ExtractArchive(zipPath, dest)
	if err == nil {
		t.Fatalf("expected symlink component error")
	}
	if !strings.Contains(err.Error(), "symlink component") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeZip(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	zw := zip.NewWriter(f)
	for name, content := range files {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.Modified = time.Now()
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return err
		}
	}
	return zw.Close()
}
