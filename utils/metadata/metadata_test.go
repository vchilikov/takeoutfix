package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestHasSupportedExtension(t *testing.T) {
	if !hasSupportedExtension("photo.JPEG") {
		t.Fatalf("expected JPEG extension to be supported")
	}
	if !hasSupportedExtension("photo.heIf") {
		t.Fatalf("expected HEIF extension to be supported")
	}
	if !hasSupportedExtension("photo.TIFF") {
		t.Fatalf("expected TIFF extension to be supported")
	}
	if !hasSupportedExtension("photo.webp") {
		t.Fatalf("expected WEBP extension to be supported")
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
	if !slices.Contains(args, "-Keywords<Tags") {
		t.Fatalf("expected Keywords mapping in args")
	}
	if !slices.Contains(args, "-Subject<Tags") {
		t.Fatalf("expected Subject mapping in args")
	}
}

func TestBuildExiftoolArgs_ExcludeCreateDateWhenDisabled(t *testing.T) {
	args := buildExiftoolArgs("meta.json", "photo.jpg", false)
	if slices.Contains(args, "-FileCreateDate<PhotoTakenTimeTimestamp") {
		t.Fatalf("did not expect FileCreateDate mapping when disabled, got: %v", args)
	}
}

func TestBuildExiftoolArgs_HEICIncludesQuickTimeAndKeysDates(t *testing.T) {
	args := buildExiftoolArgs("meta.json", "photo.HEIC", true)

	want := []string{
		"-QuickTime:CreateDate<PhotoTakenTimeTimestamp",
		"-QuickTime:ModifyDate<PhotoTakenTimeTimestamp",
		"-QuickTime:TrackCreateDate<PhotoTakenTimeTimestamp",
		"-QuickTime:TrackModifyDate<PhotoTakenTimeTimestamp",
		"-QuickTime:MediaCreateDate<PhotoTakenTimeTimestamp",
		"-QuickTime:MediaModifyDate<PhotoTakenTimeTimestamp",
		"-Keys:CreationDate<PhotoTakenTimeTimestamp",
	}

	for _, tag := range want {
		if !slices.Contains(args, tag) {
			t.Fatalf("expected %q in args, got: %v", tag, args)
		}
	}
}

func TestBuildExiftoolArgs_JPEGDoesNotIncludeQuickTimeAndKeysDates(t *testing.T) {
	args := buildExiftoolArgs("meta.json", "photo.jpg", true)
	for _, tag := range []string{
		"-QuickTime:CreateDate<PhotoTakenTimeTimestamp",
		"-Keys:CreationDate<PhotoTakenTimeTimestamp",
	} {
		if slices.Contains(args, tag) {
			t.Fatalf("did not expect %q for jpeg, got: %v", tag, args)
		}
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

func TestBuildExiftoolArgsWithOptions_ExcludesDateTagsWhenDisabled(t *testing.T) {
	args := buildExiftoolArgsWithOptions("meta.json", "photo.jpg", true, false)
	for _, arg := range args {
		if strings.Contains(arg, "PhotoTakenTimeTimestamp") {
			t.Fatalf("did not expect date mapping when disabled, got: %v", args)
		}
	}
}

func TestApplyDetailedWithRunner_RetriesOnBadFormat(t *testing.T) {
	var calls [][]string
	runner := func(args []string) (string, error) {
		cp := make([]string, len(args))
		copy(cp, args)
		calls = append(calls, cp)
		switch len(calls) {
		case 1:
			return "Error: Bad format (0) for ExifIFD entry 25 - photo.jpg\n", fmt.Errorf("exiftool failed")
		case 2: // strip
			return "1 image files updated\n", nil
		case 3: // retry apply
			return "1 image files updated\n", nil
		default:
			return "", fmt.Errorf("unexpected call")
		}
	}

	result, err := ApplyDetailedWithRunner("photo.jpg", "meta.json", runner)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.CreateDateWarned {
		t.Fatalf("CreateDateWarned should be false")
	}
	if len(calls) != 3 {
		t.Fatalf("expected 3 runner calls, got %d", len(calls))
	}
	// Second call should be the strip command.
	if !slices.Contains(calls[1], "-all=") {
		t.Fatalf("expected strip call with -all=, got: %v", calls[1])
	}
}

func TestApplyDetailedWithRunner_ReturnsErrorWhenStripFails(t *testing.T) {
	callCount := 0
	runner := func(args []string) (string, error) {
		callCount++
		switch callCount {
		case 1:
			return "Error: Bad format (0) for ExifIFD entry 25\n", fmt.Errorf("exiftool failed")
		case 2: // strip fails too
			return "strip error\n", fmt.Errorf("strip failed")
		default:
			return "", fmt.Errorf("unexpected call")
		}
	}

	_, err := ApplyDetailedWithRunner("photo.jpg", "meta.json", runner)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "could not fix metadata for photo.jpg") {
		t.Fatalf("expected original error message, got: %v", err)
	}
	if strings.Contains(err.Error(), "stripping corrupt EXIF") {
		t.Fatalf("should not mention stripping when strip itself failed")
	}
	if callCount != 2 {
		t.Fatalf("expected 2 runner calls, got %d", callCount)
	}
}

func TestApplyDetailedWithRunner_ReturnsErrorWhenRetryAfterStripFails(t *testing.T) {
	callCount := 0
	runner := func(args []string) (string, error) {
		callCount++
		switch callCount {
		case 1:
			return "Error: Bad format (0) for ExifIFD entry 25\n", fmt.Errorf("exiftool failed")
		case 2: // strip succeeds
			return "1 image files updated\n", nil
		case 3: // retry fails
			return "some other error\n", fmt.Errorf("retry failed")
		default:
			return "", fmt.Errorf("unexpected call")
		}
	}

	_, err := ApplyDetailedWithRunner("photo.jpg", "meta.json", runner)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "after stripping corrupt EXIF") {
		t.Fatalf("expected error to mention stripping, got: %v", err)
	}
	if callCount != 3 {
		t.Fatalf("expected 3 runner calls, got %d", callCount)
	}
}

func TestApplyDetailedWithRunner_RetriesOnErrorReading(t *testing.T) {
	var calls [][]string
	runner := func(args []string) (string, error) {
		cp := make([]string, len(args))
		copy(cp, args)
		calls = append(calls, cp)
		switch len(calls) {
		case 1:
			return "Error: Error reading OtherImageStart data in IFD0 - photo.jpg\n", fmt.Errorf("exiftool failed")
		case 2: // strip
			return "1 image files updated\n", nil
		case 3: // retry apply
			return "1 image files updated\n", nil
		default:
			return "", fmt.Errorf("unexpected call")
		}
	}

	result, err := ApplyDetailedWithRunner("photo.jpg", "meta.json", runner)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.CreateDateWarned {
		t.Fatalf("CreateDateWarned should be false")
	}
	if len(calls) != 3 {
		t.Fatalf("expected 3 runner calls, got %d", len(calls))
	}
	if !slices.Contains(calls[1], "-all=") {
		t.Fatalf("expected strip call with -all=, got: %v", calls[1])
	}
}

func TestApplyDetailedWithRunner_FileCreateDateFallbackAfterStrip(t *testing.T) {
	orig := shouldWriteFileCreateDate
	shouldWriteFileCreateDate = func() bool { return true }
	defer func() { shouldWriteFileCreateDate = orig }()

	callCount := 0
	runner := func(args []string) (string, error) {
		callCount++
		switch callCount {
		case 1: // initial apply fails with corrupt EXIF
			return "Error: Bad format (0) for ExifIFD entry 25\n", fmt.Errorf("exiftool failed")
		case 2: // strip succeeds
			return "1 image files updated\n", nil
		case 3: // retry after strip fails with FileCreateDate error
			return "Warning: Sorry, FileCreateDate is not supported\n", fmt.Errorf("exiftool failed")
		case 4: // fallback without FileCreateDate succeeds
			return "1 image files updated\n", nil
		default:
			return "", fmt.Errorf("unexpected call %d", callCount)
		}
	}

	result, err := ApplyDetailedWithRunner("photo.jpg", "meta.json", runner)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.CreateDateWarned {
		t.Fatalf("expected CreateDateWarned to be true")
	}
	if callCount != 4 {
		t.Fatalf("expected 4 runner calls, got %d", callCount)
	}
}

func TestLooksLikeCorruptExif(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{"bad format lowercase", "Warning: bad format for entry", true},
		{"bad format mixed case", "Error: Bad format (0) for ExifIFD entry 25 - photo.jpg", true},
		{"error reading lowercase", "error reading OtherImageStart data in IFD0", true},
		{"error reading mixed case", "Error: Error reading OtherImageStart data in IFD0 - photo.jpg", true},
		{"clean output", "1 image files updated", false},
		{"empty output", "", false},
		{"unrelated error", "Error: File not found - photo.jpg", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeCorruptExif(tt.output); got != tt.want {
				t.Errorf("looksLikeCorruptExif(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestDetectTimestampStatus(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		jsonPath := writeJSONFixture(t, `{"photoTakenTime":{"timestamp":"1719835200"}}`)
		if got := detectTimestampStatus(jsonPath); got != timestampStatusValid {
			t.Fatalf("expected valid status, got %v", got)
		}
	})

	t.Run("missing photoTakenTime", func(t *testing.T) {
		jsonPath := writeJSONFixture(t, `{"title":"x"}`)
		if got := detectTimestampStatus(jsonPath); got != timestampStatusMissing {
			t.Fatalf("expected missing status, got %v", got)
		}
	})

	t.Run("missing timestamp", func(t *testing.T) {
		jsonPath := writeJSONFixture(t, `{"photoTakenTime":{}}`)
		if got := detectTimestampStatus(jsonPath); got != timestampStatusMissing {
			t.Fatalf("expected missing status, got %v", got)
		}
	})

	t.Run("invalid timestamp", func(t *testing.T) {
		jsonPath := writeJSONFixture(t, `{"photoTakenTime":{"timestamp":"not-a-number"}}`)
		if got := detectTimestampStatus(jsonPath); got != timestampStatusInvalid {
			t.Fatalf("expected invalid status, got %v", got)
		}
	})

	t.Run("unknown malformed json", func(t *testing.T) {
		jsonPath := writeJSONFixture(t, `{"photoTakenTime":`)
		if got := detectTimestampStatus(jsonPath); got != timestampStatusUnknown {
			t.Fatalf("expected unknown status, got %v", got)
		}
	})

	t.Run("unknown invalid shape", func(t *testing.T) {
		jsonPath := writeJSONFixture(t, `{"photoTakenTime":"bad-shape"}`)
		if got := detectTimestampStatus(jsonPath); got != timestampStatusUnknown {
			t.Fatalf("expected unknown status, got %v", got)
		}
	})
}

func TestApplyDetailedWithRunner_ValidTimestampDoesNotUseFilenameFallback(t *testing.T) {
	jsonPath := writeJSONFixture(t, `{"photoTakenTime":{"timestamp":"1719835200"}}`)
	callCount := 0
	runner := func(args []string) (string, error) {
		callCount++
		for _, arg := range args {
			if strings.HasPrefix(arg, "-DateTimeOriginal=") {
				t.Fatalf("did not expect filename date assignment for valid JSON timestamp, args: %v", args)
			}
		}
		return "1 image files updated\n", nil
	}

	result, err := ApplyDetailedWithRunner("2013-06-11 16.19.16.jpg", jsonPath, runner)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.UsedFilenameDate {
		t.Fatalf("expected UsedFilenameDate=false for valid timestamp")
	}
	if callCount != 1 {
		t.Fatalf("expected one exiftool call, got %d", callCount)
	}
}

func TestApplyDetailedWithRunner_MissingTimestampUsesFilenameFallback(t *testing.T) {
	jsonPath := writeJSONFixture(t, `{"title":"x"}`)
	calls := 0
	runner := func(args []string) (string, error) {
		calls++
		switch calls {
		case 1:
			if slices.Contains(args, "-AllDates<PhotoTakenTimeTimestamp") {
				t.Fatalf("did not expect JSON date mapping when timestamp is missing, args: %v", args)
			}
			return "1 image files updated\n", nil
		case 2:
			if !slices.Contains(args, "-DateTimeOriginal=2013:06:11 16:19:16") {
				t.Fatalf("expected filename date assignment, args: %v", args)
			}
			return "1 image files updated\n", nil
		default:
			return "", fmt.Errorf("unexpected call %d", calls)
		}
	}

	result, err := ApplyDetailedWithRunner("2013-06-11 16.19.16.jpg", jsonPath, runner)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.UsedFilenameDate {
		t.Fatalf("expected UsedFilenameDate=true")
	}
	if calls != 2 {
		t.Fatalf("expected two exiftool calls, got %d", calls)
	}
}

func TestApplyDetailedWithRunner_InvalidTimestampUsesFilenameFallback(t *testing.T) {
	jsonPath := writeJSONFixture(t, `{"photoTakenTime":{"timestamp":"not-a-number"}}`)
	calls := 0
	runner := func(args []string) (string, error) {
		calls++
		switch calls {
		case 1:
			if slices.Contains(args, "-AllDates<PhotoTakenTimeTimestamp") {
				t.Fatalf("did not expect JSON date mapping when timestamp is invalid, args: %v", args)
			}
			return "1 image files updated\n", nil
		case 2:
			if !slices.Contains(args, "-DateTimeOriginal=2013:06:11 16:19:16") {
				t.Fatalf("expected filename date assignment, args: %v", args)
			}
			return "1 image files updated\n", nil
		default:
			return "", fmt.Errorf("unexpected call %d", calls)
		}
	}

	result, err := ApplyDetailedWithRunner("2013-06-11 16.19.16.jpg", jsonPath, runner)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.UsedFilenameDate {
		t.Fatalf("expected UsedFilenameDate=true")
	}
	if calls != 2 {
		t.Fatalf("expected two exiftool calls, got %d", calls)
	}
}

func TestApplyDetailedWithRunner_MissingTimestampUnparseableFilenameSkipsFallback(t *testing.T) {
	jsonPath := writeJSONFixture(t, `{"title":"x"}`)
	calls := 0
	runner := func(args []string) (string, error) {
		calls++
		if calls > 1 {
			t.Fatalf("did not expect fallback call for unparseable filename, args: %v", args)
		}
		return "1 image files updated\n", nil
	}

	result, err := ApplyDetailedWithRunner("IMG_1234.jpg", jsonPath, runner)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.UsedFilenameDate {
		t.Fatalf("expected UsedFilenameDate=false")
	}
	if calls != 1 {
		t.Fatalf("expected one exiftool call, got %d", calls)
	}
}

func writeJSONFixture(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "meta.json")
	if err := os.WriteFile(jsonPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write json fixture: %v", err)
	}
	return jsonPath
}
