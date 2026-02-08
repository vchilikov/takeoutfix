package preflight

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateAll_MixedBatch(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	badPath := filepath.Join(dir, "bad.zip")

	if err := writeZip(validPath, map[string]string{"a.txt": "hello"}); err != nil {
		t.Fatalf("write valid zip: %v", err)
	}
	if err := writeZip(badPath, map[string]string{"b.txt": "world"}); err != nil {
		t.Fatalf("write bad zip source: %v", err)
	}

	data, err := os.ReadFile(badPath)
	if err != nil {
		t.Fatalf("read bad zip: %v", err)
	}
	if len(data) < 8 {
		t.Fatalf("zip too small")
	}
	if err := os.WriteFile(badPath, data[:len(data)-8], 0o600); err != nil {
		t.Fatalf("truncate bad zip: %v", err)
	}

	zips, err := DiscoverTopLevelZips(dir)
	if err != nil {
		t.Fatalf("DiscoverTopLevelZips: %v", err)
	}
	if len(zips) != 2 {
		t.Fatalf("expected 2 zips, got %d", len(zips))
	}

	summary := ValidateAll(zips)
	if len(summary.Checked) != 2 {
		t.Fatalf("expected 2 checked zips, got %d", len(summary.Checked))
	}
	if len(summary.Corrupt) != 1 {
		t.Fatalf("expected 1 corrupt zip, got %d", len(summary.Corrupt))
	}
	if summary.TotalZipBytes == 0 {
		t.Fatalf("expected zip bytes > 0")
	}
	if summary.TotalUncompressed == 0 {
		t.Fatalf("expected uncompressed bytes > 0")
	}
}

func TestValidateAll_EmptyBatch(t *testing.T) {
	summary := ValidateAll(nil)
	if len(summary.Checked) != 0 {
		t.Fatalf("expected no checked zips, got %d", len(summary.Checked))
	}
	if len(summary.Corrupt) != 0 {
		t.Fatalf("expected no corrupt zips, got %d", len(summary.Corrupt))
	}
	if summary.TotalZipBytes != 0 {
		t.Fatalf("expected total zip bytes 0, got %d", summary.TotalZipBytes)
	}
	if summary.TotalUncompressed != 0 {
		t.Fatalf("expected total uncompressed 0, got %d", summary.TotalUncompressed)
	}
}

func TestValidateAll_SingleArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.zip")
	if err := writeZip(path, map[string]string{"a.txt": "hello"}); err != nil {
		t.Fatalf("write zip: %v", err)
	}

	zips, err := DiscoverTopLevelZips(dir)
	if err != nil {
		t.Fatalf("DiscoverTopLevelZips: %v", err)
	}
	if len(zips) != 1 {
		t.Fatalf("expected 1 zip, got %d", len(zips))
	}

	summary := ValidateAll(zips)
	if len(summary.Checked) != 1 {
		t.Fatalf("expected 1 checked zip, got %d", len(summary.Checked))
	}
	if len(summary.Corrupt) != 0 {
		t.Fatalf("expected no corrupt zips, got %d", len(summary.Corrupt))
	}
	if summary.Checked[0].Archive.Name != zips[0].Name {
		t.Fatalf("expected checked archive %q, got %q", zips[0].Name, summary.Checked[0].Archive.Name)
	}
	if summary.TotalZipBytes != zips[0].SizeBytes {
		t.Fatalf("expected total zip bytes %d, got %d", zips[0].SizeBytes, summary.TotalZipBytes)
	}
	if summary.TotalUncompressed == 0 {
		t.Fatalf("expected total uncompressed > 0")
	}
}

func TestValidateAll_MultipleArchivesKeepsInputOrder(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, fmt.Sprintf("archive-%d.zip", i))
		if err := writeZip(name, map[string]string{"f.txt": "v"}); err != nil {
			t.Fatalf("write zip %s: %v", name, err)
		}
	}

	zips, err := DiscoverTopLevelZips(dir)
	if err != nil {
		t.Fatalf("DiscoverTopLevelZips: %v", err)
	}
	if len(zips) != 5 {
		t.Fatalf("expected 5 zips, got %d", len(zips))
	}

	summary := ValidateAll(zips)
	if len(summary.Checked) != 5 {
		t.Fatalf("expected 5 checked zips, got %d", len(summary.Checked))
	}
	if len(summary.Corrupt) != 0 {
		t.Fatalf("expected no corrupt zips, got %d", len(summary.Corrupt))
	}

	for i := range zips {
		if summary.Checked[i].Archive.Name != zips[i].Name {
			t.Fatalf("order mismatch at %d: want %q, got %q", i, zips[i].Name, summary.Checked[i].Archive.Name)
		}
	}
}

func TestFingerprintChangesWithInput(t *testing.T) {
	fp1 := Fingerprint(100, nowForTest(1))
	fp2 := Fingerprint(100, nowForTest(2))
	if fp1 == fp2 {
		t.Fatalf("expected different fingerprints")
	}
}

func nowForTest(sec int64) time.Time {
	return time.Unix(sec, 0).UTC()
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
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return err
		}
	}
	return zw.Close()
}
