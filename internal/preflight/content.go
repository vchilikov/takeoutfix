package preflight

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vchilikov/takeout-fix/utils/files"
)

var (
	scanTakeoutContent = files.ScanTakeout
	statPathForContent = os.Stat
)

const takeoutDirName = "Takeout"

// DetectProcessableTakeoutRoot returns a strict extracted-Takeout root when one
// can be identified and contains at least one media/json pair.
func DetectProcessableTakeoutRoot(path string) (string, bool, error) {
	candidates, err := takeoutRootCandidates(path)
	if err != nil {
		return "", false, err
	}

	for _, candidate := range candidates {
		scan, scanErr := scanTakeoutContent(candidate)
		if scanErr != nil {
			return "", false, scanErr
		}
		if len(scan.Pairs) > 0 {
			return candidate, true, nil
		}
	}
	return "", false, nil
}

// HasProcessableTakeout reports whether DetectProcessableTakeoutRoot finds a
// strict extracted-Takeout root.
func HasProcessableTakeout(path string) (bool, error) {
	_, ok, err := DetectProcessableTakeoutRoot(path)
	return ok, err
}

func takeoutRootCandidates(path string) ([]string, error) {
	candidates := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	addCandidate := func(root string) {
		clean := filepath.Clean(root)
		if _, ok := seen[clean]; ok {
			return
		}
		seen[clean] = struct{}{}
		candidates = append(candidates, clean)
	}

	if strings.EqualFold(filepath.Base(path), takeoutDirName) {
		addCandidate(path)
	}

	nestedTakeout := filepath.Join(path, takeoutDirName)
	info, err := statPathForContent(nestedTakeout)
	if err != nil {
		if os.IsNotExist(err) {
			return candidates, nil
		}
		return nil, err
	}
	if info.IsDir() {
		addCandidate(nestedTakeout)
	}

	return candidates, nil
}
