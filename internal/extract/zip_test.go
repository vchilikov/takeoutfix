package extract

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vchilikov/takeout-fix/internal/preflight"
)

func TestExtractArchivesSkipAndDelete(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "a.zip")
	if err := writeZip(zipPath, map[string]string{"one.txt": "1", "two.txt": "2"}); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	info, _ := os.Stat(zipPath)
	archive := preflight.ZipArchive{
		Name:        "a.zip",
		Path:        zipPath,
		SizeBytes:   uint64(info.Size()),
		ModTime:     info.ModTime(),
		Fingerprint: preflight.Fingerprint(uint64(info.Size()), info.ModTime()),
	}

	dest := filepath.Join(dir, "out")
	stats, err := ExtractArchives([]preflight.ZipArchive{archive}, dest, nil, true)
	if err != nil {
		t.Fatalf("ExtractArchives error: %v", err)
	}
	if stats.ArchivesExtracted != 1 || stats.FilesExtracted != 2 || stats.DeletedZips != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if _, err := os.Stat(zipPath); !os.IsNotExist(err) {
		t.Fatalf("expected zip to be deleted")
	}

	stats, err = ExtractArchives([]preflight.ZipArchive{archive}, dest, func(preflight.ZipArchive) bool { return true }, false)
	if err != nil {
		t.Fatalf("ExtractArchives skip error: %v", err)
	}
	if stats.ArchivesSkipped != 1 {
		t.Fatalf("expected one skipped archive, got %+v", stats)
	}
}

func TestSafeJoinBlocksTraversal(t *testing.T) {
	if _, err := safeJoin("/tmp/base", "../escape"); err == nil {
		t.Fatalf("expected traversal error")
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
