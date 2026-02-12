package patharg

import (
	"path/filepath"
	"strings"
)

// Safe prevents option-style interpretation for paths starting with '-'.
// It does not escape protocol delimiters such as newline bytes.
func Safe(path string) string {
	if strings.HasPrefix(path, "-") {
		return "." + string(filepath.Separator) + path
	}
	return path
}
