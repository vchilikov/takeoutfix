package metadata

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestHasSupportedExtension(t *testing.T) {
	if !hasSupportedExtension("photo.JPEG") {
		t.Fatalf("expected JPEG extension to be supported")
	}
	if hasSupportedExtension("photo.webp") {
		t.Fatalf("expected WEBP extension to be unsupported")
	}
}

func TestSafePathArg(t *testing.T) {
	want := "." + string(filepath.Separator) + "-meta.json"
	if got := safePathArg("-meta.json"); got != want {
		t.Fatalf("expected sanitized path, got %q", got)
	}
	if got := safePathArg("/tmp/-meta.json"); got != "/tmp/-meta.json" {
		t.Fatalf("absolute path should not change, got %q", got)
	}
}

func TestParseCaptureContext(t *testing.T) {
	tests := []struct {
		name          string
		testDataFile  string
		wantTimestamp int64
		wantLatitude  float64
		wantLongitude float64
		wantErrSubstr string
	}{
		{
			name:          "geoData is used when present",
			testDataFile:  "with_geo_data.json",
			wantTimestamp: 1719835200,
			wantLatitude:  40.7128,
			wantLongitude: -74.0060,
		},
		{
			name:          "geoDataExif fallback is used",
			testDataFile:  "with_geo_data_exif.json",
			wantTimestamp: 1704110400,
			wantLatitude:  34.0522,
			wantLongitude: -118.2437,
		},
		{
			name:          "missing gps returns error",
			testDataFile:  "without_gps.json",
			wantErrSubstr: "geoData/geoDataExif coordinates not found",
		},
		{
			name:          "invalid timestamp returns error",
			testDataFile:  "invalid_timestamp.json",
			wantErrSubstr: "invalid photoTakenTime.timestamp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context, err := parseCaptureContext(filepath.Join("testdata", tt.testDataFile))
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErrSubstr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if context.timestamp != tt.wantTimestamp {
				t.Fatalf("timestamp mismatch: want %d, got %d", tt.wantTimestamp, context.timestamp)
			}
			if context.latitude != tt.wantLatitude {
				t.Fatalf("latitude mismatch: want %f, got %f", tt.wantLatitude, context.latitude)
			}
			if context.longitude != tt.wantLongitude {
				t.Fatalf("longitude mismatch: want %f, got %f", tt.wantLongitude, context.longitude)
			}
		})
	}
}

func TestComputeTimezoneOffset(t *testing.T) {
	newYorkLat := 40.7128
	newYorkLon := -74.0060

	summer := time.Date(2024, time.July, 1, 12, 0, 0, 0, time.UTC).Unix()
	winter := time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC).Unix()

	summerOffset, err := computeTimezoneOffset(summer, newYorkLat, newYorkLon)
	if err != nil {
		t.Fatalf("unexpected summer error: %v", err)
	}
	if summerOffset != "-04:00" {
		t.Fatalf("summer offset mismatch: want -04:00, got %s", summerOffset)
	}

	winterOffset, err := computeTimezoneOffset(winter, newYorkLat, newYorkLon)
	if err != nil {
		t.Fatalf("unexpected winter error: %v", err)
	}
	if winterOffset != "-05:00" {
		t.Fatalf("winter offset mismatch: want -05:00, got %s", winterOffset)
	}
}

func TestComputeTimezoneOffset_InvalidCoordinates(t *testing.T) {
	if _, err := computeTimezoneOffset(1704110400, 999, 10); err == nil {
		t.Fatalf("expected error for invalid latitude")
	}
	if _, err := computeTimezoneOffset(1704110400, 10, -999); err == nil {
		t.Fatalf("expected error for invalid longitude")
	}
}

func TestBuildExiftoolArgs(t *testing.T) {
	withOffset := buildExiftoolArgs("meta.json", "photo.jpg", "+03:00")
	withoutOffset := buildExiftoolArgs("meta.json", "photo.jpg", "")

	expectedWithOffset := []string{
		"-OffsetTime=+03:00",
		"-OffsetTimeOriginal=+03:00",
		"-OffsetTimeDigitized=+03:00",
	}
	for _, arg := range expectedWithOffset {
		if !slices.Contains(withOffset, arg) {
			t.Fatalf("expected argument %q in args with offset: %v", arg, withOffset)
		}
	}
	for _, arg := range expectedWithOffset {
		if slices.Contains(withoutOffset, arg) {
			t.Fatalf("did not expect argument %q in args without offset: %v", arg, withoutOffset)
		}
	}
	if !slices.Contains(withOffset, "--") {
		t.Fatalf("expected -- separator in args with offset")
	}
	if !slices.Contains(withoutOffset, "--") {
		t.Fatalf("expected -- separator in args without offset")
	}
}
