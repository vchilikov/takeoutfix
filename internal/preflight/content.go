package preflight

import "github.com/vchilikov/takeout-fix/utils/files"

var scanTakeoutContent = files.ScanTakeout

// HasProcessableTakeout returns true when a folder looks like extracted
// Takeout content by finding at least one media evidence item.
func HasProcessableTakeout(path string) (bool, error) {
	result, err := scanTakeoutContent(path)
	if err != nil {
		return false, err
	}

	evidenceCount := len(result.Pairs) + len(result.MissingJSON) + len(result.AmbiguousJSON)
	return evidenceCount > 0, nil
}
