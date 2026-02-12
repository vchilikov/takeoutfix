package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vchilikov/takeout-fix/internal/exifcmd"
	"github.com/vchilikov/takeout-fix/internal/mediaext"
	"github.com/vchilikov/takeout-fix/internal/patharg"
)

type ApplyResult struct {
	UsedFilenameDate   bool
	UsedXMPSidecar     bool
	CreateDateWarned   bool
	FilenameDateWarned bool
}

type timestampStatus int

const (
	timestampStatusUnknown timestampStatus = iota
	timestampStatusValid
	timestampStatusMissing
	timestampStatusInvalid
)

var filenameDatePrefixRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}\.\d{2}\.\d{2})`)

type gpsInclusion struct {
	geoData     bool
	geoDataExif bool
}

func detectGPSInclusion(jsonPath string) gpsInclusion {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return gpsInclusion{true, true} // fail-open
	}

	var payload struct {
		GeoData *struct {
			Latitude  *float64 `json:"latitude"`
			Longitude *float64 `json:"longitude"`
		} `json:"geoData"`
		GeoDataExif *struct {
			Latitude  *float64 `json:"latitude"`
			Longitude *float64 `json:"longitude"`
		} `json:"geoDataExif"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return gpsInclusion{true, true} // fail-open
	}

	hasCoords := func(lat, lon *float64) bool {
		if lat == nil || lon == nil {
			return false
		}
		return *lat != 0 || *lon != 0
	}

	gd := payload.GeoData != nil && hasCoords(payload.GeoData.Latitude, payload.GeoData.Longitude)
	gde := payload.GeoDataExif != nil && hasCoords(payload.GeoDataExif.Latitude, payload.GeoDataExif.Longitude)
	return gpsInclusion{gd, gde}
}

func Apply(mediaPath string, jsonPath string) error {
	_, err := ApplyDetailed(mediaPath, jsonPath)
	return err
}

func ApplyDetailed(mediaPath string, jsonPath string) (ApplyResult, error) {
	return ApplyDetailedWithRunner(mediaPath, jsonPath, runExiftool)
}

func ApplyDetailedWithRunner(
	mediaPath string,
	jsonPath string,
	run func(args []string) (string, error),
) (ApplyResult, error) {
	result := ApplyResult{}
	if run == nil {
		return result, fmt.Errorf("nil exiftool runner")
	}

	var outMediaPath string
	if hasSupportedExtension(mediaPath) {
		outMediaPath = mediaPath
	} else {
		outMediaPath = mediaPath + ".xmp"
		result.UsedXMPSidecar = true
	}

	includeCreateDate := shouldWriteFileCreateDate()
	status := detectTimestampStatus(jsonPath)
	includeJSONDate := status == timestampStatusValid || status == timestampStatusUnknown

	createDateWarned, err := applyJSONMetadata(mediaPath, jsonPath, outMediaPath, includeCreateDate, includeJSONDate, run)
	if err != nil {
		return result, err
	}
	result.CreateDateWarned = createDateWarned

	if status == timestampStatusMissing || status == timestampStatusInvalid {
		usedFilenameDate, filenameCreateDateWarned, err := applyFilenameDate(mediaPath, outMediaPath, includeCreateDate, run)
		if err != nil {
			result.FilenameDateWarned = true
		} else {
			result.UsedFilenameDate = usedFilenameDate
			if filenameCreateDateWarned {
				result.CreateDateWarned = true
			}
		}
	}

	return result, nil
}

func runExiftool(args []string) (string, error) {
	bin, err := exifcmd.Resolve()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(bin, args...)
	data, err := cmd.CombinedOutput()
	return string(data), err
}

func buildExiftoolArgs(jsonPath string, outMediaPath string, includeCreateDate bool) []string {
	return buildExiftoolArgsWithOptions(jsonPath, outMediaPath, includeCreateDate, true, gpsInclusion{true, true})
}

func buildExiftoolArgsWithOptions(jsonPath string, outMediaPath string, includeCreateDate bool, includeJSONDateTags bool, gps gpsInclusion) []string {
	args := []string{
		"-d", "%s",
		"-m",
		"-TagsFromFile", patharg.Safe(jsonPath),
		"-Title<Title",
		"-Description<Description",
		"-ImageDescription<Description",
		"-Caption-Abstract<Description",
		// Google Takeout stores Tags as a JSON list. Exiftool maps this list
		// into list-like target tags such as Keywords/Subject.
		"-Keywords<Tags",
		"-Subject<Tags",
	}

	if gps.geoData {
		args = append(args,
			"-GPSAltitude<GeoDataAltitude",
			"-GPSLatitude<GeoDataLatitude",
			"-GPSLatitudeRef<GeoDataLatitude",
			"-GPSLongitude<GeoDataLongitude",
			"-GPSLongitudeRef<GeoDataLongitude",
		)
	}
	if gps.geoDataExif {
		args = append(args,
			"-GPSAltitude<GeoDataExifAltitude",
			"-GPSLatitude<GeoDataExifLatitude",
			"-GPSLatitudeRef<GeoDataExifLatitude",
			"-GPSLongitude<GeoDataExifLongitude",
			"-GPSLongitudeRef<GeoDataExifLongitude",
		)
	}

	if includeJSONDateTags {
		args = append(args, "-AllDates<PhotoTakenTimeTimestamp")
		args = append(args, "-FileModifyDate<PhotoTakenTimeTimestamp")
	}

	if includeJSONDateTags && isHEIFContainer(outMediaPath) {
		// HEIC/HEIF consumers (e.g. Apple Photos) often read container-level tags
		// instead of EXIF AllDates, so write both sets.
		args = append(args,
			"-QuickTime:CreateDate<PhotoTakenTimeTimestamp",
			"-QuickTime:ModifyDate<PhotoTakenTimeTimestamp",
			"-QuickTime:TrackCreateDate<PhotoTakenTimeTimestamp",
			"-QuickTime:TrackModifyDate<PhotoTakenTimeTimestamp",
			"-QuickTime:MediaCreateDate<PhotoTakenTimeTimestamp",
			"-QuickTime:MediaModifyDate<PhotoTakenTimeTimestamp",
			"-Keys:CreationDate<PhotoTakenTimeTimestamp",
		)
	}

	if includeJSONDateTags && includeCreateDate {
		args = append(args, "-FileCreateDate<PhotoTakenTimeTimestamp")
	}

	args = append(args,
		"-overwrite_original",
	)

	args = append(args, patharg.Safe(outMediaPath))
	return args
}

func applyJSONMetadata(
	mediaPath string,
	jsonPath string,
	outMediaPath string,
	includeCreateDate bool,
	includeJSONDateTags bool,
	run func(args []string) (string, error),
) (bool, error) {
	gps := detectGPSInclusion(jsonPath)
	args := buildExiftoolArgsWithOptions(jsonPath, outMediaPath, includeCreateDate, includeJSONDateTags, gps)
	output, err := run(args)
	if err != nil {
		if includeJSONDateTags && includeCreateDate && strings.Contains(strings.ToLower(output), "filecreatedate") {
			// Some filesystems and formats may not support FileCreateDate writes.
			retryArgs := buildExiftoolArgsWithOptions(jsonPath, outMediaPath, false, includeJSONDateTags, gps)
			retryOutput, retryErr := run(retryArgs)
			if retryErr == nil {
				return true, nil
			}
			return false, fmt.Errorf("could not fix metadata for %s\nerror: %w\noutput: %s", mediaPath, retryErr, retryOutput)
		}

		// Corrupt EXIF (e.g. Samsung "Bad format (0) for ExifIFD entry 25",
		// or "Error reading OtherImageStart data in IFD0"):
		// strip all metadata, then re-apply from JSON.
		if looksLikeCorruptExif(output) {
			stripArgs := []string{"-all=", "-overwrite_original", patharg.Safe(outMediaPath)}
			if _, stripErr := run(stripArgs); stripErr == nil {
				retryArgs := buildExiftoolArgsWithOptions(jsonPath, outMediaPath, includeCreateDate, includeJSONDateTags, gps)
				retryOutput, retryErr := run(retryArgs)
				if retryErr == nil {
					return false, nil
				}
				if includeJSONDateTags && includeCreateDate && strings.Contains(strings.ToLower(retryOutput), "filecreatedate") {
					fallbackArgs := buildExiftoolArgsWithOptions(jsonPath, outMediaPath, false, includeJSONDateTags, gps)
					fallbackOutput, fallbackErr := run(fallbackArgs)
					if fallbackErr == nil {
						return true, nil
					}
					return false, fmt.Errorf("could not fix metadata for %s after stripping corrupt EXIF\nerror: %w\noutput: %s", mediaPath, fallbackErr, fallbackOutput)
				}
				return false, fmt.Errorf("could not fix metadata for %s after stripping corrupt EXIF\nerror: %w\noutput: %s", mediaPath, retryErr, retryOutput)
			}
		}

		return false, fmt.Errorf("could not fix metadata for %s\nerror: %w\noutput: %s", mediaPath, err, output)
	}
	return false, nil
}

func applyFilenameDate(
	mediaPath string,
	outMediaPath string,
	includeCreateDate bool,
	run func(args []string) (string, error),
) (bool, bool, error) {
	parsed, ok := parseFilenameDate(mediaPath)
	if !ok {
		return false, false, nil
	}

	args := buildFilenameDateArgs(outMediaPath, parsed, includeCreateDate)
	output, err := run(args)
	if err == nil {
		return true, false, nil
	}

	if includeCreateDate && strings.Contains(strings.ToLower(output), "filecreatedate") {
		retryArgs := buildFilenameDateArgs(outMediaPath, parsed, false)
		retryOutput, retryErr := run(retryArgs)
		if retryErr == nil {
			return true, true, nil
		}
		return false, false, fmt.Errorf("could not apply filename date for %s\nerror: %w\noutput: %s", mediaPath, retryErr, retryOutput)
	}

	return false, false, fmt.Errorf("could not apply filename date for %s\nerror: %w\noutput: %s", mediaPath, err, output)
}

func buildFilenameDateArgs(outMediaPath string, value time.Time, includeCreateDate bool) []string {
	formatted := value.Format("2006:01:02 15:04:05")

	// Intentionally omit -d "%s": fallback writes fully formatted date-time strings, not epoch values.
	args := []string{
		"-m",
		"-DateTimeOriginal=" + formatted,
		"-CreateDate=" + formatted,
		"-ModifyDate=" + formatted,
		"-FileModifyDate=" + formatted,
	}

	if isHEIFContainer(outMediaPath) {
		args = append(args,
			"-QuickTime:CreateDate="+formatted,
			"-QuickTime:ModifyDate="+formatted,
			"-QuickTime:TrackCreateDate="+formatted,
			"-QuickTime:TrackModifyDate="+formatted,
			"-QuickTime:MediaCreateDate="+formatted,
			"-QuickTime:MediaModifyDate="+formatted,
			"-Keys:CreationDate="+formatted,
		)
	}

	if includeCreateDate {
		args = append(args, "-FileCreateDate="+formatted)
	}

	args = append(args,
		"-overwrite_original",
		patharg.Safe(outMediaPath),
	)
	return args
}

func parseFilenameDate(mediaPath string) (time.Time, bool) {
	base := filepath.Base(mediaPath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	match := filenameDatePrefixRe.FindStringSubmatch(stem)
	if len(match) != 2 {
		return time.Time{}, false
	}
	// Filenames do not encode timezone, so fallback parsing assumes UTC.
	parsed, err := time.ParseInLocation("2006-01-02 15.04.05", match[1], time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func detectTimestampStatus(jsonPath string) timestampStatus {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return timestampStatusUnknown
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return timestampStatusUnknown
	}

	photoTakenTimeValue, ok := payload["photoTakenTime"]
	if !ok {
		return timestampStatusMissing
	}
	photoTakenTime, ok := photoTakenTimeValue.(map[string]any)
	if !ok {
		return timestampStatusUnknown
	}

	timestampValue, ok := photoTakenTime["timestamp"]
	if !ok {
		return timestampStatusMissing
	}

	switch v := timestampValue.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return timestampStatusMissing
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return timestampStatusInvalid
		}
		if parsed <= 0 {
			return timestampStatusInvalid
		}
		return timestampStatusValid
	case float64:
		if v <= 0 {
			return timestampStatusInvalid
		}
		return timestampStatusValid
	default:
		return timestampStatusInvalid
	}
}

var shouldWriteFileCreateDate = func() bool {
	return runtime.GOOS == "darwin"
}

func looksLikeCorruptExif(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "bad format") || strings.Contains(lower, "error reading")
}

func hasSupportedExtension(path string) bool {
	ext := filepath.Ext(path)
	for _, supportedExt := range mediaext.Supported {
		if strings.EqualFold(ext, supportedExt) {
			return true
		}
	}
	return false
}

func isHEIFContainer(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".heic", ".heif":
		return true
	default:
		return false
	}
}
