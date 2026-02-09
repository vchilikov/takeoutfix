package metadata

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vchilikov/takeout-fix/internal/exifcmd"
	"github.com/vchilikov/takeout-fix/internal/mediaext"
	"github.com/vchilikov/takeout-fix/internal/patharg"
)

type ApplyResult struct {
	UsedXMPSidecar   bool
	CreateDateWarned bool
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
	args := buildExiftoolArgs(jsonPath, outMediaPath, includeCreateDate)
	output, err := run(args)
	if err != nil {
		if includeCreateDate && strings.Contains(strings.ToLower(output), "filecreatedate") {
			// Some filesystems and formats may not support FileCreateDate writes.
			retryArgs := buildExiftoolArgs(jsonPath, outMediaPath, false)
			retryOutput, retryErr := run(retryArgs)
			if retryErr == nil {
				result.CreateDateWarned = true
				return result, nil
			}
			return result, fmt.Errorf("could not fix metadata for %s\nerror: %w\noutput: %s", mediaPath, retryErr, retryOutput)
		}

		// Corrupt EXIF (e.g. Samsung "Bad format (0) for ExifIFD entry 25",
		// or "Error reading OtherImageStart data in IFD0"):
		// strip all metadata, then re-apply from JSON.
		if looksLikeCorruptExif(output) {
			stripArgs := []string{"-all=", "-overwrite_original", patharg.Safe(outMediaPath)}
			if _, stripErr := run(stripArgs); stripErr == nil {
				retryArgs := buildExiftoolArgs(jsonPath, outMediaPath, includeCreateDate)
				retryOutput, retryErr := run(retryArgs)
				if retryErr == nil {
					return result, nil
				}
				if includeCreateDate && strings.Contains(strings.ToLower(retryOutput), "filecreatedate") {
					fallbackArgs := buildExiftoolArgs(jsonPath, outMediaPath, false)
					fallbackOutput, fallbackErr := run(fallbackArgs)
					if fallbackErr == nil {
						result.CreateDateWarned = true
						return result, nil
					}
					return result, fmt.Errorf("could not fix metadata for %s after stripping corrupt EXIF\nerror: %w\noutput: %s", mediaPath, fallbackErr, fallbackOutput)
				}
				return result, fmt.Errorf("could not fix metadata for %s after stripping corrupt EXIF\nerror: %w\noutput: %s", mediaPath, retryErr, retryOutput)
			}
		}

		return result, fmt.Errorf("could not fix metadata for %s\nerror: %w\noutput: %s", mediaPath, err, output)
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
	args := []string{
		"-d", "%s",
		"-m",
		"-TagsFromFile", patharg.Safe(jsonPath),
		"-Title<Title",
		"-Description<Description",
		"-ImageDescription<Description",
		"-Caption-Abstract<Description",
		"-AllDates<PhotoTakenTimeTimestamp",
		"-GPSAltitude<GeoDataAltitude",
		"-GPSLatitude<GeoDataLatitude",
		"-GPSLatitudeRef<GeoDataLatitude",
		"-GPSLongitude<GeoDataLongitude",
		"-GPSLongitudeRef<GeoDataLongitude",
		"-GPSAltitude<GeoDataExifAltitude",
		"-GPSLatitude<GeoDataExifLatitude",
		"-GPSLatitudeRef<GeoDataExifLatitude",
		"-GPSLongitude<GeoDataExifLongitude",
		"-GPSLongitudeRef<GeoDataExifLongitude",
		"-FileModifyDate<PhotoTakenTimeTimestamp",
	}

	if includeCreateDate {
		args = append(args, "-FileCreateDate<PhotoTakenTimeTimestamp")
	}

	args = append(args,
		"-overwrite_original",
	)

	args = append(args, patharg.Safe(outMediaPath))
	return args
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
