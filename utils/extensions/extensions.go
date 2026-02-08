package extensions

import (
	crand "crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vchilikov/takeout-fix/internal/exifcmd"
	"github.com/vchilikov/takeout-fix/internal/patharg"
)

type FixResult struct {
	Path    string
	Renamed bool
}

func Fix(mediaPath string) (string, error) {
	result, err := FixDetailed(mediaPath)
	return result.Path, err
}

func FixDetailed(mediaPath string) (FixResult, error) {
	return FixDetailedWithRunner(mediaPath, runExiftool)
}

func FixDetailedWithRunner(mediaPath string, run func(args []string) (string, error)) (FixResult, error) {
	currentExt := filepath.Ext(mediaPath)
	newExt, err := getNewExtension(mediaPath, run)
	if err != nil {
		return FixResult{Path: mediaPath}, fmt.Errorf("could not get the proper extensions for %s: %w", mediaPath, err)
	}

	if areExtensionsCompatible(currentExt, newExt) {
		return FixResult{Path: mediaPath}, nil
	}

	baseFileName := strings.TrimSuffix(mediaPath, currentExt)
	newMediaPath, err := getNewFileName(baseFileName, newExt)
	if err != nil {
		return FixResult{Path: mediaPath}, fmt.Errorf("could not generate a new file name for %s with %s extensions: %w", mediaPath, newExt, err)
	}

	err = os.Rename(mediaPath, newMediaPath)
	if err != nil {
		return FixResult{Path: mediaPath}, err
	}

	return FixResult{Path: newMediaPath, Renamed: true}, nil
}

func runExiftool(args []string) (string, error) {
	bin, err := exifcmd.Resolve()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(bin, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func getNewExtension(mediaPath string, run func(args []string) (string, error)) (string, error) {
	if run == nil {
		return "", fmt.Errorf("nil exiftool runner")
	}

	out, err := run([]string{"-p", ".$FileTypeExtension", patharg.Safe(mediaPath)})
	if err != nil {
		return "", err
	}

	ext := parseFileTypeExtension(out)
	if ext == "" {
		return "", fmt.Errorf("empty file type extension for %s", mediaPath)
	}
	return ext, nil
}

func parseFileTypeExtension(output string) string {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "warning:") || strings.HasPrefix(lower, "error:") {
			continue
		}
		if !strings.HasPrefix(line, ".") {
			line = "." + line
		}
		return line
	}
	return ""
}

func areExtensionsCompatible(ext1 string, ext2 string) bool {
	if strings.EqualFold(ext1, ext2) {
		return true
	}

	// takeoutfix aims to keep the file names as they are if possible, so if exiftool outputs
	// $FileTypeExtension for a .jpeg file to be .jpg, takeoutfix will not rename it
	compatibleExtensions := map[string]string{
		".jpg": ".jpeg",
		".tif": ".tiff",
		".m4v": ".mp4",
		".mov": ".mp4",
	}

	for k, v := range compatibleExtensions {
		if (strings.EqualFold(ext1, k) && strings.EqualFold(ext2, v)) ||
			(strings.EqualFold(ext1, v) && strings.EqualFold(ext2, k)) {
			return true
		}
	}

	return false
}

func getNewFileName(baseFileName string, newExtension string) (string, error) {
	if !doesFileExist(baseFileName + newExtension) {
		return baseFileName + newExtension, nil
	}

	const maxAttempts = 10
	for i := 0; i < maxAttempts; i++ {
		suffix, err := generateRandomSuffix()
		if err != nil {
			return "", fmt.Errorf("could not generate random suffix: %w", err)
		}
		newFileName := baseFileName + "-" + suffix + newExtension
		if !doesFileExist(newFileName) {
			return newFileName, nil
		}
	}

	return "", fmt.Errorf("could not generate a unique file name after %d attempts", maxAttempts)
}

func generateRandomSuffix() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 5)
	randBytes := make([]byte, len(b))
	if _, err := crand.Read(randBytes); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(randBytes[i])%len(charset)]
	}
	return string(b), nil
}

func doesFileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}
