package files

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type MediaScanResult struct {
	Pairs         map[string]string
	MissingJSON   []string
	UnusedJSON    []string
	AmbiguousJSON map[string][]string
}

// GetAlbums returns a list of directories in the provided path.
// It is used to get the list of albums in the Google Photos export directory.
func GetAlbums(path string) ([]string, error) {
	var albums []string
	files, err := os.ReadDir(path)
	if err != nil {
		return albums, err
	}

	for _, file := range files {
		if file.IsDir() {
			albums = append(albums, file.Name())
		}
	}

	return albums, nil
}

// GetMedia returns media files with their matching JSON metadata and
// reports files that could not be matched.
func GetMedia(albumPath string) (MediaScanResult, error) {
	result := MediaScanResult{
		Pairs:         make(map[string]string),
		AmbiguousJSON: make(map[string][]string),
	}

	entries, err := os.ReadDir(albumPath)
	if err != nil {
		return result, err
	}

	jsonFiles, mediaFiles := classifyFiles(entries)
	usedJsonFiles := make(map[string]struct{})
	mediaNames := make([]string, 0, len(mediaFiles))
	for mediaFile := range mediaFiles {
		mediaNames = append(mediaNames, mediaFile)
	}
	sort.Strings(mediaNames)

	for _, mediaFile := range mediaNames {
		jsonFile, err := getJsonFile(mediaFile, jsonFiles)
		if err != nil {
			result.MissingJSON = append(result.MissingJSON, mediaFile)
			continue
		}

		usedJsonFiles[jsonFile] = struct{}{}
		result.Pairs[mediaFile] = jsonFile
	}

	result.UnusedJSON = checkUnusedJson(jsonFiles, usedJsonFiles)

	return result, nil
}

// ScanTakeout recursively scans a Takeout root and matches media files with
// their metadata json files across all nested folders.
func ScanTakeout(rootPath string) (MediaScanResult, error) {
	result := MediaScanResult{
		Pairs:         make(map[string]string),
		AmbiguousJSON: make(map[string][]string),
	}

	jsonByDir := make(map[string]map[string]struct{})
	mediaByDir := make(map[string][]string)
	var allJSON []string

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}
		dir := filepath.Dir(rel)
		base := filepath.Base(rel)
		if dir == "" {
			dir = "."
		}

		if isJSONFile(base) {
			if _, ok := jsonByDir[dir]; !ok {
				jsonByDir[dir] = make(map[string]struct{})
			}
			jsonByDir[dir][base] = struct{}{}
			allJSON = append(allJSON, rel)
			return nil
		}

		mediaByDir[dir] = append(mediaByDir[dir], base)
		return nil
	})
	if err != nil {
		return result, err
	}

	for dir := range mediaByDir {
		sort.Strings(mediaByDir[dir])
	}
	sort.Strings(allJSON)

	usedJSON := make(map[string]struct{})
	var unresolvedMedia []string

	dirs := sortedDirs(mediaByDir)
	for _, dir := range dirs {
		dirJSON := jsonByDir[dir]
		for _, mediaFile := range mediaByDir[dir] {
			mediaRel := joinRelPath(dir, mediaFile)
			jsonFile, err := getJsonFile(mediaFile, dirJSON)
			if err != nil {
				unresolvedMedia = append(unresolvedMedia, mediaRel)
				continue
			}

			jsonRel := joinRelPath(dir, jsonFile)
			result.Pairs[mediaRel] = jsonRel
			usedJSON[jsonRel] = struct{}{}
		}
	}

	globalIndex := make(map[string][]string)
	for _, jsonRel := range allJSON {
		if _, ok := usedJSON[jsonRel]; ok {
			continue
		}

		key := normalizeJSONKey(filepath.Base(jsonRel))
		if key == "" {
			continue
		}
		globalIndex[key] = append(globalIndex[key], jsonRel)
	}

	sort.Strings(unresolvedMedia)
	for _, mediaRel := range unresolvedMedia {
		keys := mediaLookupKeys(filepath.Base(mediaRel))
		candidates := collectGlobalCandidates(keys, globalIndex, usedJSON)

		switch len(candidates) {
		case 0:
			result.MissingJSON = append(result.MissingJSON, mediaRel)
		case 1:
			result.Pairs[mediaRel] = candidates[0]
			usedJSON[candidates[0]] = struct{}{}
		default:
			sort.Strings(candidates)
			result.AmbiguousJSON[mediaRel] = candidates
		}
	}

	for _, jsonRel := range allJSON {
		if _, ok := usedJSON[jsonRel]; !ok {
			result.UnusedJSON = append(result.UnusedJSON, jsonRel)
		}
	}

	sort.Strings(result.MissingJSON)
	sort.Strings(result.UnusedJSON)

	return result, nil
}

func collectGlobalCandidates(keys []string, globalIndex map[string][]string, usedJSON map[string]struct{}) []string {
	unique := make(map[string]struct{})

	for _, key := range keys {
		for _, jsonRel := range globalIndex[key] {
			if _, used := usedJSON[jsonRel]; used {
				continue
			}
			unique[jsonRel] = struct{}{}
		}
	}

	candidates := make([]string, 0, len(unique))
	for jsonRel := range unique {
		candidates = append(candidates, jsonRel)
	}
	sort.Strings(candidates)
	return candidates
}

func mediaLookupKeys(mediaFile string) []string {
	keys := make(map[string]struct{})
	add := func(name string) {
		key := normalizeMediaLookupKey(name)
		if key != "" {
			keys[key] = struct{}{}
		}
	}

	add(mediaFile)
	add(removeRandomSuffix(mediaFile))

	if strings.Contains(strings.ToLower(mediaFile), "-edited") {
		add(strings.Replace(mediaFile, "-edited", "", 1))
	}

	if numberSuffixRe.MatchString(mediaFile) {
		match := numberSuffixRe.FindString(mediaFile)
		add(strings.Replace(mediaFile, match, "", 1) + match)
	}

	if len(mediaFile) > 46 {
		add(mediaFile[:46])
		if numberSuffixRe.MatchString(mediaFile) {
			match := numberSuffixRe.FindString(mediaFile)
			add(mediaFile[:46] + match)
		}
	}

	mediaExt := filepath.Ext(mediaFile)
	if strings.EqualFold(mediaExt, ".mp4") {
		baseMediaFile := strings.TrimSuffix(mediaFile, mediaExt)
		for _, ext := range [...]string{".jpg", ".jpeg", ".heic"} {
			add(baseMediaFile + ext)
		}
	}

	out := make([]string, 0, len(keys))
	for k := range keys {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func joinRelPath(dir string, base string) string {
	if dir == "." {
		return base
	}
	return filepath.Join(dir, base)
}

func sortedDirs(m map[string][]string) []string {
	dirs := make([]string, 0, len(m))
	for dir := range m {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs
}

func isJSONFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".json")
}
