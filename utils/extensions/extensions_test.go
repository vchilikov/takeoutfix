package extensions

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAreExtensionsCompatible(t *testing.T) {
	tests := []struct {
		ext1 string
		ext2 string
		want bool
	}{
		{".jpg", ".jpeg", true},
		{".mov", ".mp4", true},
		{".heic", ".heif", true},
		{".PNG", ".png", true},
		{".jpg", ".png", false},
	}

	for _, tt := range tests {
		got := areExtensionsCompatible(tt.ext1, tt.ext2)
		if got != tt.want {
			t.Fatalf("compatibility mismatch for %q/%q: want %v, got %v", tt.ext1, tt.ext2, tt.want, got)
		}
	}
}

func TestGenerateRandomSuffix(t *testing.T) {
	got, err := generateRandomSuffix()
	if err != nil {
		t.Fatalf("generateRandomSuffix error: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("expected suffix length 5, got %d", len(got))
	}
	for _, r := range got {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz0123456789", r) {
			t.Fatalf("unexpected rune in suffix: %q", r)
		}
	}
}

func TestGenerateRandomSuffixFromReaderSkipsBiasedBytes(t *testing.T) {
	source := bytes.NewReader([]byte{252, 253, 254, 255, 0, 1, 2, 3, 4})
	got, err := generateRandomSuffixFromReader(source)
	if err != nil {
		t.Fatalf("generateRandomSuffixFromReader error: %v", err)
	}
	if got != "abcde" {
		t.Fatalf("expected deterministic suffix abcde, got %q", got)
	}
}

func TestGenerateRandomSuffixFromReaderPropagatesError(t *testing.T) {
	_, err := generateRandomSuffixFromReader(bytes.NewReader([]byte{1, 2, 3}))
	if err == nil {
		t.Fatalf("expected reader error")
	}
}

func TestParseFileTypeExtension(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "plain extension",
			output: ".jpg\n",
			want:   ".jpg",
		},
		{
			name:   "without dot",
			output: "jpg\n",
			want:   ".jpg",
		},
		{
			name:   "ignores warnings",
			output: "Warning: duplicate tags\njpg\n",
			want:   ".jpg",
		},
		{
			name:   "only errors",
			output: "Error: bad file\n",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseFileTypeExtension(tt.output); got != tt.want {
				t.Fatalf("parseFileTypeExtension mismatch: want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestGetNewExtension_DoesNotUseDoubleDashSeparator(t *testing.T) {
	var gotArgs []string
	run := func(args []string) (string, error) {
		gotArgs = append([]string(nil), args...)
		return ".jpg\n", nil
	}

	ext, err := getNewExtension("photo.jpg", run)
	if err != nil {
		t.Fatalf("getNewExtension returned error: %v", err)
	}
	if ext != ".jpg" {
		t.Fatalf("extension mismatch: want .jpg, got %s", ext)
	}
	if slices.Contains(gotArgs, "--") {
		t.Fatalf("did not expect -- separator in exiftool args: %v", gotArgs)
	}
}

func TestGetNewExtension_RequiresRunner(t *testing.T) {
	if _, err := getNewExtension("photo.jpg", nil); err == nil {
		t.Fatalf("expected error for nil runner")
	}
}

func TestGetNewExtension_PropagatesRunnerError(t *testing.T) {
	run := func([]string) (string, error) {
		return "", errors.New("boom")
	}
	if _, err := getNewExtension("photo.jpg", run); err == nil {
		t.Fatalf("expected runner error")
	}
}

func TestFixDetailedWithRunner_NoRename(t *testing.T) {
	run := func([]string) (string, error) {
		return ".jpg\n", nil
	}

	result, err := FixDetailedWithRunner("/tmp/photo.jpg", run)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Renamed {
		t.Fatalf("expected Renamed=false when extension matches")
	}
	if result.Path != "/tmp/photo.jpg" {
		t.Fatalf("expected original path, got %q", result.Path)
	}
}

func TestFixDetailedWithRunner_Rename(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := filepath.Join(tmpDir, "photo.jpeg")
	if err := os.WriteFile(origPath, []byte("fake"), 0644); err != nil {
		t.Fatalf("create temp file: %v", err)
	}

	run := func([]string) (string, error) {
		return ".png\n", nil
	}

	result, err := FixDetailedWithRunner(origPath, run)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Renamed {
		t.Fatalf("expected Renamed=true")
	}
	expectedPath := filepath.Join(tmpDir, "photo.png")
	if result.Path != expectedPath {
		t.Fatalf("expected %q, got %q", expectedPath, result.Path)
	}
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("renamed file should exist: %v", err)
	}
}

func TestFixDetailedWithRunner_RunnerError(t *testing.T) {
	run := func([]string) (string, error) {
		return "", errors.New("exiftool failed")
	}

	result, err := FixDetailedWithRunner("/tmp/photo.jpg", run)
	if err == nil {
		t.Fatalf("expected error from runner")
	}
	if result.Path != "/tmp/photo.jpg" {
		t.Fatalf("expected original path on error, got %q", result.Path)
	}
}

func TestGetNewFileName_NoCollision(t *testing.T) {
	tmpDir := t.TempDir()
	base := filepath.Join(tmpDir, "photo")

	got, err := getNewFileName(base, ".jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != base+".jpg" {
		t.Fatalf("expected %q, got %q", base+".jpg", got)
	}
}

func TestGetNewFileName_WithCollision(t *testing.T) {
	tmpDir := t.TempDir()
	base := filepath.Join(tmpDir, "photo")
	collisionPath := base + ".jpg"
	if err := os.WriteFile(collisionPath, []byte("existing"), 0644); err != nil {
		t.Fatalf("create collision file: %v", err)
	}

	got, err := getNewFileName(base, ".jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == collisionPath {
		t.Fatalf("expected a different name due to collision, got same: %q", got)
	}
	if !strings.HasSuffix(got, ".jpg") {
		t.Fatalf("expected .jpg suffix, got %q", got)
	}
	if !strings.HasPrefix(got, base+"-") {
		t.Fatalf("expected base-suffix pattern, got %q", got)
	}
}

func TestDoesFileExist_NonExistent(t *testing.T) {
	if doesFileExist("/nonexistent/path/file.txt") {
		t.Fatalf("expected false for non-existent file")
	}
}

func TestDoesFileExist_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("create temp file: %v", err)
	}

	if !doesFileExist(path) {
		t.Fatalf("expected true for existing file")
	}
}
