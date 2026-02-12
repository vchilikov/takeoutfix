package files

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vchilikov/takeout-fix/internal/mediaext"
)

type MediaScanResult struct {
	Pairs         map[string]string
	MissingJSON   []string
	UnusedJSON    []string
	AmbiguousJSON map[string][]string
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

		if isMediaCandidate(base) {
			mediaByDir[dir] = append(mediaByDir[dir], base)
		}
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
		dirMediaSet := make(map[string]struct{}, len(mediaByDir[dir]))
		localCandidatesByMedia := make(map[string]string, len(mediaByDir[dir]))
		localCandidateClaims := make(map[string][]string)
		for _, mediaFile := range mediaByDir[dir] {
			dirMediaSet[mediaFile] = struct{}{}
		}
		for _, mediaFile := range mediaByDir[dir] {
			mediaRel := joinRelPath(dir, mediaFile)
			jsonFile, err := getJsonFile(mediaFile, dirJSON, dirMediaSet)
			if err != nil {
				unresolvedMedia = append(unresolvedMedia, mediaRel)
				continue
			}

			jsonRel := joinRelPath(dir, jsonFile)
			localCandidatesByMedia[mediaRel] = jsonRel
			localCandidateClaims[jsonRel] = append(localCandidateClaims[jsonRel], mediaRel)
		}

		for _, mediaFile := range mediaByDir[dir] {
			mediaRel := joinRelPath(dir, mediaFile)
			jsonRel, ok := localCandidatesByMedia[mediaRel]
			if !ok {
				continue
			}
			if len(localCandidateClaims[jsonRel]) > 1 {
				unresolvedMedia = append(unresolvedMedia, mediaRel)
				continue
			}

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
	globalCandidatesByMedia := make(map[string][]string, len(unresolvedMedia))
	globalCandidateUsage := make(map[string]int)
	globalCandidateClaims := make(map[string][]string)

	for _, mediaRel := range unresolvedMedia {
		keys := mediaLookupKeys(filepath.Base(mediaRel))
		candidates := collectGlobalCandidates(keys, globalIndex, usedJSON)
		candidates = applyGlobalCandidateRules(mediaRel, candidates)
		globalCandidatesByMedia[mediaRel] = candidates

		if len(candidates) == 1 {
			candidate := candidates[0]
			globalCandidateUsage[candidate]++
			globalCandidateClaims[candidate] = append(globalCandidateClaims[candidate], mediaRel)
		}
	}

	globalCandidateWinner := make(map[string]string)
	for candidate, claims := range globalCandidateClaims {
		if len(claims) <= 1 {
			continue
		}
		if winner, ok := uniqueSameDirClaimant(candidate, claims); ok {
			globalCandidateWinner[candidate] = winner
		}
	}

	for _, mediaRel := range unresolvedMedia {
		candidates := globalCandidatesByMedia[mediaRel]
		switch len(candidates) {
		case 0:
			result.MissingJSON = append(result.MissingJSON, mediaRel)
		case 1:
			candidate := candidates[0]
			if globalCandidateUsage[candidate] > 1 {
				winner, ok := globalCandidateWinner[candidate]
				if !ok || winner != mediaRel {
					result.AmbiguousJSON[mediaRel] = candidates
					continue
				}
			}
			if _, alreadyUsed := usedJSON[candidate]; alreadyUsed {
				result.AmbiguousJSON[mediaRel] = candidates
				continue
			}

			result.Pairs[mediaRel] = candidate
			usedJSON[candidate] = struct{}{}
		default:
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

func applyGlobalCandidateRules(mediaRel string, candidates []string) []string {
	if len(candidates) <= 1 {
		return candidates
	}

	filteredByIndex := filterCandidatesByDuplicateIndex(filepath.Base(mediaRel), candidates)
	if len(filteredByIndex) == 1 {
		return filteredByIndex
	}
	if len(filteredByIndex) > 1 {
		candidates = filteredByIndex
	}

	sameDir := filterCandidatesBySameDir(mediaRel, candidates)
	if len(sameDir) == 1 {
		return sameDir
	}

	return candidates
}

func filterCandidatesBySameDir(mediaRel string, candidates []string) []string {
	if len(candidates) <= 1 {
		return candidates
	}

	mediaDir := filepath.Dir(mediaRel)
	filtered := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if filepath.Dir(candidate) == mediaDir {
			filtered = append(filtered, candidate)
		}
	}
	sort.Strings(filtered)
	return filtered
}

func uniqueSameDirClaimant(candidate string, claims []string) (string, bool) {
	candidateDir := filepath.Dir(candidate)
	sameDirClaims := make([]string, 0, len(claims))
	for _, mediaRel := range claims {
		if filepath.Dir(mediaRel) == candidateDir {
			sameDirClaims = append(sameDirClaims, mediaRel)
		}
	}
	if len(sameDirClaims) != 1 {
		return "", false
	}
	return sameDirClaims[0], true
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

func isMediaCandidate(name string) bool {
	ext := filepath.Ext(name)
	for _, supportedExt := range mediaext.Supported {
		if strings.EqualFold(ext, supportedExt) {
			return true
		}
	}
	return false
}
