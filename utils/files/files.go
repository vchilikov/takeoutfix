package files

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
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
		slices.Sort(mediaByDir[dir])
	}
	slices.Sort(allJSON)

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

		localCandidateWinner := make(map[string]string)
		for jsonRel, claims := range localCandidateClaims {
			if len(claims) <= 1 {
				continue
			}
			if winner, ok := uniqueClaimantByJSONTargetExtension(jsonRel, claims); ok {
				localCandidateWinner[jsonRel] = winner
			}
		}

		for _, mediaFile := range mediaByDir[dir] {
			mediaRel := joinRelPath(dir, mediaFile)
			jsonRel, ok := localCandidatesByMedia[mediaRel]
			if !ok {
				continue
			}
			claims := localCandidateClaims[jsonRel]
			if len(claims) > 1 {
				winner, ok := localCandidateWinner[jsonRel]
				if !ok || winner != mediaRel {
					unresolvedMedia = append(unresolvedMedia, mediaRel)
					continue
				}
				if _, alreadyUsed := usedJSON[jsonRel]; alreadyUsed {
					unresolvedMedia = append(unresolvedMedia, mediaRel)
					continue
				}

				result.Pairs[mediaRel] = jsonRel
				usedJSON[jsonRel] = struct{}{}
				continue
			}
			if _, alreadyUsed := usedJSON[jsonRel]; alreadyUsed {
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

	slices.Sort(unresolvedMedia)
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
		if winner, ok := uniqueClaimantByJSONTargetExtension(candidate, claims); ok {
			globalCandidateWinner[candidate] = winner
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
			// Defensive guard: candidates are precomputed before assignment, so keep
			// this check to avoid double-claiming if future rule changes reintroduce overlaps.
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

	slices.Sort(result.MissingJSON)
	slices.Sort(result.UnusedJSON)

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

	return slices.Sorted(maps.Keys(unique))
}

func applyGlobalCandidateRules(mediaRel string, candidates []string) []string {
	if len(candidates) <= 1 {
		return candidates
	}

	filteredByIndex := filterCandidatesByDuplicateIndex(mediaRel, candidates)
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

	// If same-dir narrowing finds none or several matches, keep the current
	// candidate set unchanged and let ambiguity handling decide later.
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

func uniqueClaimantByJSONTargetExtension(jsonRel string, claims []string) (string, bool) {
	targetExt := jsonTargetExtension(jsonRel)
	if targetExt == "" {
		return "", false
	}

	var winner string
	matchCount := 0
	for _, mediaRel := range claims {
		if strings.EqualFold(filepath.Ext(mediaRel), targetExt) {
			winner = mediaRel
			matchCount++
		}
	}
	if matchCount != 1 {
		return "", false
	}
	return winner, true
}

func jsonTargetExtension(jsonRel string) string {
	name := strings.ToLower(filepath.Base(jsonRel))
	if !strings.HasSuffix(name, ".json") {
		return ""
	}

	name = strings.TrimSuffix(name, ".json")
	name = trailingNumberSuffixRe.ReplaceAllString(name, "")
	name = stripSupplementalSuffix(name)
	ext := filepath.Ext(name)
	if !isSupportedMediaExtension(ext) {
		return ""
	}
	return ext
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

	return slices.Sorted(maps.Keys(keys))
}

func joinRelPath(dir string, base string) string {
	if dir == "." {
		return base
	}
	return filepath.Join(dir, base)
}

func sortedDirs(m map[string][]string) []string {
	return slices.Sorted(maps.Keys(m))
}

func isJSONFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".json")
}

func isMediaCandidate(name string) bool {
	return isSupportedMediaExtension(filepath.Ext(name))
}

func isSupportedMediaExtension(ext string) bool {
	return ext != "" && slices.ContainsFunc(mediaext.Supported, func(s string) bool {
		return strings.EqualFold(ext, s)
	})
}
