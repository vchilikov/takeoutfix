package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFixAlbum_HappyPathWithMockExiftool(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mock exiftool script uses POSIX shell")
	}

	root := t.TempDir()
	albumPath := filepath.Join(root, "Album")
	if err := os.MkdirAll(albumPath, 0o755); err != nil {
		t.Fatalf("mkdir album: %v", err)
	}

	mediaPath := filepath.Join(albumPath, "IMG_0001.jpg")
	if err := os.WriteFile(mediaPath, []byte("media"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	jsonData := `{
  "photoTakenTime": {"timestamp": "1719835200"},
  "geoData": {"latitude": 40.7128, "longitude": -74.0060}
}`
	if err := os.WriteFile(filepath.Join(albumPath, "IMG_0001.jpg.json"), []byte(jsonData), 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}

	binDir := t.TempDir()
	mockExiftool := filepath.Join(binDir, "exiftool")
	markerFile := filepath.Join(t.TempDir(), "marker.log")
	script := `#!/bin/sh
set -eu

if [ "$1" = "-p" ]; then
  [ "$2" = '.$FileTypeExtension' ] || exit 10
  [ "$3" = "--" ] || exit 11
  echo ".png"
  exit 0
fi

found_separator=0
has_offset=0
has_offset_original=0
has_offset_digitized=0
for arg in "$@"; do
  if [ "$arg" = "--" ]; then
    found_separator=1
  fi
  case "$arg" in
    -OffsetTime=*) has_offset=1 ;;
    -OffsetTimeOriginal=*) has_offset_original=1 ;;
    -OffsetTimeDigitized=*) has_offset_digitized=1 ;;
  esac
done
[ "$found_separator" -eq 1 ] || exit 12
[ "$has_offset" -eq 1 ] || exit 13
[ "$has_offset_original" -eq 1 ] || exit 14
[ "$has_offset_digitized" -eq 1 ] || exit 15
printf 'metadata\n' >> "$MARKER_FILE"
exit 0
`
	if err := os.WriteFile(mockExiftool, []byte(script), 0o755); err != nil {
		t.Fatalf("write mock exiftool: %v", err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("MARKER_FILE", markerFile)

	fixTakeout(albumPath)
	fixTakeout(albumPath)

	if _, err := os.Stat(filepath.Join(albumPath, "IMG_0001.png")); err != nil {
		t.Fatalf("renamed media file not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(albumPath, "IMG_0001.jpg")); !os.IsNotExist(err) {
		t.Fatalf("old media file should not exist after rename")
	}

	data, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("read marker file: %v", err)
	}
	if count := strings.Count(string(data), "metadata\n"); count != 2 {
		t.Fatalf("expected metadata apply to run twice, got %d", count)
	}
}

func TestCleanAlbumJson_RemovesOnlyMatchedJSON(t *testing.T) {
	albumPath := t.TempDir()

	if err := os.WriteFile(filepath.Join(albumPath, "IMG_0001.jpg"), []byte("media"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	if err := os.WriteFile(filepath.Join(albumPath, "IMG_0001.jpg.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write matched json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(albumPath, "orphan.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write orphan json: %v", err)
	}

	cleanTakeoutJson(albumPath)

	if _, err := os.Stat(filepath.Join(albumPath, "IMG_0001.jpg.json")); !os.IsNotExist(err) {
		t.Fatalf("matched json should be removed")
	}
	if _, err := os.Stat(filepath.Join(albumPath, "orphan.json")); err != nil {
		t.Fatalf("orphan json should be kept: %v", err)
	}
}

func TestFixTakeout_CrossFolderSupplementalJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mock exiftool script uses POSIX shell")
	}

	root := t.TempDir()
	mediaDir := filepath.Join(root, "Photos from 2022")
	jsonDir := filepath.Join(root, "Album X")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatalf("mkdir media dir: %v", err)
	}
	if err := os.MkdirAll(jsonDir, 0o755); err != nil {
		t.Fatalf("mkdir json dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(mediaDir, "IMG_0001.jpg"), []byte("media"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	jsonData := `{
  "photoTakenTime": {"timestamp": "1719835200"},
  "geoData": {"latitude": 40.7128, "longitude": -74.0060}
}`
	if err := os.WriteFile(filepath.Join(jsonDir, "IMG_0001.jpg.supplemental-metadata.json"), []byte(jsonData), 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}

	binDir := t.TempDir()
	markerFile := filepath.Join(t.TempDir(), "cross-folder-marker.log")
	mockExiftool := filepath.Join(binDir, "exiftool")
	script := `#!/bin/sh
set -eu

if [ "$1" = "-p" ]; then
  [ "$2" = '.$FileTypeExtension' ] || exit 10
  [ "$3" = "--" ] || exit 11
  echo ".jpg"
  exit 0
fi
printf 'metadata\n' >> "$MARKER_FILE"
exit 0
`
	if err := os.WriteFile(mockExiftool, []byte(script), 0o755); err != nil {
		t.Fatalf("write mock exiftool: %v", err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("MARKER_FILE", markerFile)

	fixTakeout(root)

	data, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("read marker file: %v", err)
	}
	if count := strings.Count(string(data), "metadata\n"); count != 1 {
		t.Fatalf("expected metadata apply once, got %d", count)
	}
}

func TestFixTakeout_AmbiguousJSONDoesNotApplyMetadata(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mock exiftool script uses POSIX shell")
	}

	root := t.TempDir()
	mediaDir := filepath.Join(root, "Photos")
	jsonADir := filepath.Join(root, "Album A")
	jsonBDir := filepath.Join(root, "Album B")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatalf("mkdir media dir: %v", err)
	}
	if err := os.MkdirAll(jsonADir, 0o755); err != nil {
		t.Fatalf("mkdir json A dir: %v", err)
	}
	if err := os.MkdirAll(jsonBDir, 0o755); err != nil {
		t.Fatalf("mkdir json B dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(mediaDir, "IMG_0001.jpg"), []byte("media"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jsonADir, "IMG_0001.jpg.supplemental-metadata.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write json A: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jsonBDir, "IMG_0001.jpg.supplemental-metada.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write json B: %v", err)
	}

	binDir := t.TempDir()
	markerFile := filepath.Join(t.TempDir(), "ambiguous-marker.log")
	mockExiftool := filepath.Join(binDir, "exiftool")
	script := `#!/bin/sh
set -eu
if [ "$1" = "-p" ]; then
  [ "$2" = '.$FileTypeExtension' ] || exit 10
  [ "$3" = "--" ] || exit 11
  echo ".jpg"
  exit 0
fi
printf 'metadata\n' >> "$MARKER_FILE"
exit 0
`
	if err := os.WriteFile(mockExiftool, []byte(script), 0o755); err != nil {
		t.Fatalf("write mock exiftool: %v", err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("MARKER_FILE", markerFile)

	fixTakeout(root)

	data, err := os.ReadFile(markerFile)
	if err == nil && len(data) > 0 {
		t.Fatalf("did not expect metadata apply for ambiguous json match")
	}
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("unexpected marker read error: %v", err)
	}
}

func TestCleanTakeoutJson_CrossFolderRemovesOnlyMatched(t *testing.T) {
	root := t.TempDir()
	mediaDir := filepath.Join(root, "Photos")
	jsonDir := filepath.Join(root, "Album X")
	ambiguousDir := filepath.Join(root, "Album Y")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatalf("mkdir media dir: %v", err)
	}
	if err := os.MkdirAll(jsonDir, 0o755); err != nil {
		t.Fatalf("mkdir json dir: %v", err)
	}
	if err := os.MkdirAll(ambiguousDir, 0o755); err != nil {
		t.Fatalf("mkdir ambiguous dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(mediaDir, "IMG_0001.jpg"), []byte("media"), 0o600); err != nil {
		t.Fatalf("write media 1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "IMG_0002.jpg"), []byte("media"), 0o600); err != nil {
		t.Fatalf("write media 2: %v", err)
	}
	matched := filepath.Join(jsonDir, "IMG_0001.jpg.supplemental-metadata.json")
	ambiguousA := filepath.Join(jsonDir, "IMG_0002.jpg.supplemental-metadata.json")
	ambiguousB := filepath.Join(ambiguousDir, "IMG_0002.jpg.supplemental-metada.json")
	unmatched := filepath.Join(jsonDir, "orphan.json")

	for _, p := range []string{matched, ambiguousA, ambiguousB, unmatched} {
		if err := os.WriteFile(p, []byte("{}"), 0o600); err != nil {
			t.Fatalf("write json %s: %v", p, err)
		}
	}

	cleanTakeoutJson(root)

	if _, err := os.Stat(matched); !os.IsNotExist(err) {
		t.Fatalf("matched json should be removed")
	}
	if _, err := os.Stat(ambiguousA); err != nil {
		t.Fatalf("ambiguous json A should remain: %v", err)
	}
	if _, err := os.Stat(ambiguousB); err != nil {
		t.Fatalf("ambiguous json B should remain: %v", err)
	}
	if _, err := os.Stat(unmatched); err != nil {
		t.Fatalf("unmatched json should remain: %v", err)
	}
}
