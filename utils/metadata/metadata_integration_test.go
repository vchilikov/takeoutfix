package metadata

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const tinyJPEGB64 = "/9j/4AAQSkZJRgABAQAAAQABAAD/2wCEAAkGBxAQEBUQEA8QDw8QEA8PDw8QDw8QFREWFhURFRUYHSggGBolGxUVITEhJSkrLi4uFx8zODMsNygtLisBCgoKDg0OFQ8QFS0dFR0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLf/AABEIAAEAAQMBIgACEQEDEQH/xAAXAAADAQAAAAAAAAAAAAAAAAAAAQID/8QAFhABAQEAAAAAAAAAAAAAAAAAAAEQ/9oADAMBAAIQAxAAAAHdA//EABgQAAMBAQAAAAAAAAAAAAAAAAABEQIh/9oACAEBAAEFAo0xv//EABYRAAMAAAAAAAAAAAAAAAAAAAABEf/aAAgBAwEBPwGn/8QAFhEBAQEAAAAAAAAAAAAAAAAAABEB/9oACAECAQE/AY//xAAbEAACAgMBAAAAAAAAAAAAAAABEQAhMUFhcf/aAAgBAQAGPwKNmZ5T0//EABwQAQEAAgMBAAAAAAAAAAAAAAERACExQVFhcf/aAAgBAQABPyHyri6E6vbjf4qp4D8Vn//aAAwDAQACAAMAAAAQw//EABYRAQEBAAAAAAAAAAAAAAAAAAARIf/aAAgBAwEBPxBqf//EABcRAAMBAAAAAAAAAAAAAAAAAAABESH/2gAIAQIBAT8Qyqf/xAAcEAEAAQQDAAAAAAAAAAAAAAABABEhMUFRYXH/2gAIAQEAAT8Q2eR0qQh8jFYnwP5QuQnKdm6a3P/Z"

func TestApplyDetailed_UpdatesFileModifyAndCreateDatesOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("FileCreateDate verification in this test is darwin-specific")
	}
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not available")
	}

	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "file.jpg")
	jsonPath := filepath.Join(dir, "meta.json")

	decoded, err := base64.StdEncoding.DecodeString(tinyJPEGB64)
	if err != nil {
		t.Fatalf("decode tiny jpeg: %v", err)
	}
	if err := os.WriteFile(mediaPath, decoded, 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}

	jsonData := `{
  "photoTakenTime": {"timestamp": "1719835200"}
}`
	if err := os.WriteFile(jsonPath, []byte(jsonData), 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}

	if err := exec.Command("touch", "-t", "202001010101.01", mediaPath).Run(); err != nil {
		t.Fatalf("touch old timestamp: %v", err)
	}

	beforeModify, beforeCreate := statSeconds(t, mediaPath)
	if beforeModify == 1719835200 || beforeCreate == 1719835200 {
		t.Fatalf("unexpected precondition: timestamps already set")
	}

	_, err = ApplyDetailed(mediaPath, jsonPath)
	if err != nil {
		t.Fatalf("ApplyDetailed error: %v", err)
	}

	afterModify, afterCreate := statSeconds(t, mediaPath)
	if afterModify != 1719835200 {
		t.Fatalf("modify timestamp mismatch: want %d, got %d", 1719835200, afterModify)
	}
	if afterCreate != 1719835200 {
		t.Fatalf("create timestamp mismatch: want %d, got %d", 1719835200, afterCreate)
	}
}

func TestApplyDetailed_UsesGeoDataExifFallback(t *testing.T) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not available")
	}

	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "file.jpg")
	jsonPath := filepath.Join(dir, "meta.json")

	decoded, err := base64.StdEncoding.DecodeString(tinyJPEGB64)
	if err != nil {
		t.Fatalf("decode tiny jpeg: %v", err)
	}
	if err := os.WriteFile(mediaPath, decoded, 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}

	geoJSON, err := os.ReadFile(filepath.Join("testdata", "with_geo_data_exif.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(jsonPath, geoJSON, 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}

	if _, err := ApplyDetailed(mediaPath, jsonPath); err != nil {
		t.Fatalf("ApplyDetailed error: %v", err)
	}

	latOut, err := exec.Command("exiftool", "-s3", "-n", "-GPSLatitude", "--", mediaPath).Output()
	if err != nil {
		t.Fatalf("read GPSLatitude: %v", err)
	}
	lonOut, err := exec.Command("exiftool", "-s3", "-n", "-GPSLongitude", "--", mediaPath).Output()
	if err != nil {
		t.Fatalf("read GPSLongitude: %v", err)
	}

	lat, err := strconv.ParseFloat(strings.TrimSpace(string(latOut)), 64)
	if err != nil {
		t.Fatalf("parse latitude: %v (%q)", err, string(latOut))
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(string(lonOut)), 64)
	if err != nil {
		t.Fatalf("parse longitude: %v (%q)", err, string(lonOut))
	}

	if lat != 34.0522 {
		t.Fatalf("latitude mismatch: want 34.0522, got %f", lat)
	}
	if lon != -118.2437 {
		t.Fatalf("longitude mismatch: want -118.2437, got %f", lon)
	}
}

func statSeconds(t *testing.T, path string) (int64, int64) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	modify := info.ModTime().Unix()

	cmd := exec.Command("stat", "-f", "%B", path)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("stat birthtime: %v", err)
	}
	birthRaw := strings.TrimSpace(string(out))
	birth, err := strconv.ParseInt(birthRaw, 10, 64)
	if err != nil {
		t.Fatalf("parse birthtime: %v", err)
	}

	return modify, birth
}
