package files

import (
	"os"
)

func classifyFiles(entries []os.DirEntry) (map[string]struct{}, map[string]struct{}) {
	jsonFiles := make(map[string]struct{})
	mediaFiles := make(map[string]struct{})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if isJSONFile(entry.Name()) {
			jsonFiles[entry.Name()] = struct{}{}
		} else if isMediaCandidate(entry.Name()) {
			mediaFiles[entry.Name()] = struct{}{}
		}
	}

	return jsonFiles, mediaFiles
}
