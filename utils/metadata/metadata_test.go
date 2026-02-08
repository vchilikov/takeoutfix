package metadata

import (
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestHasSupportedExtension(t *testing.T) {
	if !hasSupportedExtension("photo.JPEG") {
		t.Fatalf("expected JPEG extension to be supported")
	}
	if !hasSupportedExtension("photo.TIFF") {
		t.Fatalf("expected TIFF extension to be supported")
	}
	if hasSupportedExtension("photo.webp") {
		t.Fatalf("expected WEBP extension to be unsupported")
	}
}

func TestBuildExiftoolArgs_DoesNotIncludeOffsetTags(t *testing.T) {
	args := buildExiftoolArgs("meta.json", "photo.jpg", true)

	if slices.Contains(args, "--") {
		t.Fatalf("did not expect -- separator in args: %v", args)
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "-OffsetTime") {
			t.Fatalf("did not expect OffsetTime* arguments, got: %v", args)
		}
	}

	if !slices.Contains(args, "-FileModifyDate<PhotoTakenTimeTimestamp") {
		t.Fatalf("expected FileModifyDate mapping in args")
	}
	if !slices.Contains(args, "-FileCreateDate<PhotoTakenTimeTimestamp") {
		t.Fatalf("expected FileCreateDate mapping in args")
	}
	if !slices.Contains(args, "-GPSLatitude<GeoDataExifLatitude") {
		t.Fatalf("expected GeoDataExif latitude fallback mapping in args")
	}
	if !slices.Contains(args, "-GPSLongitude<GeoDataExifLongitude") {
		t.Fatalf("expected GeoDataExif longitude fallback mapping in args")
	}
}

func TestBuildExiftoolArgs_ExcludeCreateDateWhenDisabled(t *testing.T) {
	args := buildExiftoolArgs("meta.json", "photo.jpg", false)
	if slices.Contains(args, "-FileCreateDate<PhotoTakenTimeTimestamp") {
		t.Fatalf("did not expect FileCreateDate mapping when disabled, got: %v", args)
	}
}

func TestShouldWriteFileCreateDate(t *testing.T) {
	want := runtime.GOOS == "darwin"
	if got := shouldWriteFileCreateDate(); got != want {
		t.Fatalf("shouldWriteFileCreateDate mismatch: want %v, got %v", want, got)
	}
}

func TestBuildExiftoolArgs_GeoDataExifComesAfterGeoData(t *testing.T) {
	args := buildExiftoolArgs("meta.json", "photo.jpg", true)

	indexOf := func(prefix string) int {
		for i, arg := range args {
			if strings.HasPrefix(arg, prefix) {
				return i
			}
		}
		return -1
	}

	geoData := indexOf("-GPSLatitude<GeoDataLatitude")
	geoDataExif := indexOf("-GPSLatitude<GeoDataExifLatitude")
	if geoData == -1 || geoDataExif == -1 {
		t.Fatalf("expected both GeoData and GeoDataExif latitude mappings, got %v", args)
	}
	if geoDataExif <= geoData {
		t.Fatalf("expected GeoDataExif mapping to be applied after GeoData mapping, got %v", args)
	}
}
