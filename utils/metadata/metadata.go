package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vchilikov/takeout-fix/internal/timezonemapper"
)

func Apply(mediaPath string, jsonPath string) error {
	var outMediaPath string
	if hasSupportedExtension(mediaPath) {
		outMediaPath = mediaPath
	} else {
		fmt.Printf("🚗 writing metadata in xmp sidecar for %s\n", mediaPath)
		outMediaPath = mediaPath + ".xmp"
	}

	offset, err := computeTimezoneOffsetFromJSON(jsonPath)
	if err != nil {
		fmt.Printf("⚠️ timezone offset for %s could not be computed: %s\n", mediaPath, err)
	}

	cmd := exec.Command("exiftool", buildExiftoolArgs(jsonPath, outMediaPath, offset)...)

	data, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not fix metadata for %s\nerror: %w\noutput: %s", mediaPath, err, string(data))
	}

	return nil
}

type captureContext struct {
	timestamp int64
	latitude  float64
	longitude float64
}

type takeoutMetadata struct {
	PhotoTakenTime struct {
		Timestamp string `json:"timestamp"`
	} `json:"photoTakenTime"`
	GeoData     geoData `json:"geoData"`
	GeoDataExif geoData `json:"geoDataExif"`
}

type geoData struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

func (g geoData) hasLatLon() bool {
	return g.Latitude != nil && g.Longitude != nil
}

func parseCaptureContext(jsonPath string) (captureContext, error) {
	var metadata takeoutMetadata
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return captureContext{}, fmt.Errorf("could not read json metadata file %s: %w", jsonPath, err)
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return captureContext{}, fmt.Errorf("could not parse json metadata file %s: %w", jsonPath, err)
	}

	timestamp, err := parsePhotoTakenTimestamp(metadata.PhotoTakenTime.Timestamp)
	if err != nil {
		return captureContext{}, err
	}

	latitude, longitude, err := getCoordinates(metadata)
	if err != nil {
		return captureContext{}, err
	}

	return captureContext{
		timestamp: timestamp,
		latitude:  latitude,
		longitude: longitude,
	}, nil
}

func parsePhotoTakenTimestamp(timestamp string) (int64, error) {
	if timestamp == "" {
		return 0, errors.New("photoTakenTime.timestamp not found in json metadata")
	}

	parsed, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid photoTakenTime.timestamp value %q: %w", timestamp, err)
	}

	return parsed, nil
}

func getCoordinates(metadata takeoutMetadata) (float64, float64, error) {
	if metadata.GeoData.hasLatLon() {
		return *metadata.GeoData.Latitude, *metadata.GeoData.Longitude, nil
	}
	if metadata.GeoDataExif.hasLatLon() {
		return *metadata.GeoDataExif.Latitude, *metadata.GeoDataExif.Longitude, nil
	}

	return 0, 0, errors.New("geoData/geoDataExif coordinates not found in json metadata")
}

func computeTimezoneOffsetFromJSON(jsonPath string) (string, error) {
	context, err := parseCaptureContext(jsonPath)
	if err != nil {
		return "", err
	}
	return computeTimezoneOffset(context.timestamp, context.latitude, context.longitude)
}

func computeTimezoneOffset(timestamp int64, latitude float64, longitude float64) (string, error) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return "", fmt.Errorf("invalid coordinates latitude=%f longitude=%f", latitude, longitude)
	}

	timezone := timezonemapper.LatLngToTimezoneString(latitude, longitude)
	if timezone == "" || strings.EqualFold(timezone, "unknown") {
		return "", fmt.Errorf("timezone not found for coordinates latitude=%f longitude=%f", latitude, longitude)
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return "", fmt.Errorf("could not load timezone location %s: %w", timezone, err)
	}

	return time.Unix(timestamp, 0).In(location).Format("-07:00"), nil
}

func buildExiftoolArgs(jsonPath string, outMediaPath string, offset string) []string {
	args := []string{
		"-d", "%s",
		"-m",
		"-TagsFromFile", safePathArg(jsonPath),
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
		"-overwrite_original",
	}

	if offset != "" {
		args = append(args,
			"-OffsetTime="+offset,
			"-OffsetTimeOriginal="+offset,
			"-OffsetTimeDigitized="+offset,
		)
	}

	args = append(args, "--", safePathArg(outMediaPath))
	return args
}

func safePathArg(path string) string {
	if strings.HasPrefix(path, "-") {
		return "." + string(filepath.Separator) + path
	}
	return path
}

func hasSupportedExtension(path string) bool {
	ext := filepath.Ext(path)

	var exifSupportExtensions = []string{
		".3gp",
		".dng",
		".gif",
		".heic",
		".jpeg",
		".jpg",
		".m4v",
		".mov",
		".mp4",
		".png",
		".tif",
	}

	for _, supportedExt := range exifSupportExtensions {
		if strings.EqualFold(ext, supportedExt) {
			return true
		}
	}

	return false
}
