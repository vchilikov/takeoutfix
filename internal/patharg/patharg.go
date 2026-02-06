package patharg

import (
	"path/filepath"
	"strings"
)

// Safe returns a path argument that cannot be interpreted as an option by tools.
func Safe(path string) string {
	if strings.HasPrefix(path, "-") {
		return "." + string(filepath.Separator) + path
	}
	return path
}
