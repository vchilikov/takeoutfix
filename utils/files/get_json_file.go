package files

import (
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/vchilikov/takeout-fix/internal/mediaext"
)

var numberSuffixRe = regexp.MustCompile(`\(\d+\)`)
var randomSuffixRe = regexp.MustCompile(`-[a-z0-9]{5}$`)

const supplementalFull = ".supplemental-metadata"

var trailingNumberSuffixRe = regexp.MustCompile(`\(\d+\)$`)
var duplicateIndexSuffixRe = regexp.MustCompile(`\((\d+)\)$`)

func getJsonFile(mediaFile string, jsonFiles map[string]struct{}, mediaFiles map[string]struct{}) (string, error) {
	if jsonFile, ok := findJSONByStemForMedia(mediaFile, mediaFile, jsonFiles); ok {
		return jsonFile, nil
	}

	baseMediaFile := strings.TrimSuffix(mediaFile, filepath.Ext(mediaFile))
	if jsonFile, ok := findJSONByStemForMedia(mediaFile, baseMediaFile, jsonFiles); ok {
		return jsonFile, nil
	}

	if strings.Contains(mediaFile, "-edited") {
		return getJsonFile(strings.Replace(mediaFile, "-edited", "", 1), jsonFiles, mediaFiles)
	}

	if numberSuffixRe.MatchString(mediaFile) {
		match := numberSuffixRe.FindString(mediaFile)
		jsonStem := strings.Replace(mediaFile, match, "", 1) + match
		jsonFile, ok := findJSONByStemForMedia(mediaFile, jsonStem, jsonFiles)
		if ok {
			return jsonFile, nil
		}
	}

	if len(mediaFile) > 46 && numberSuffixRe.MatchString(mediaFile) {
		match := numberSuffixRe.FindString(mediaFile)
		jsonStem := mediaFile[:46] + match
		jsonFile, ok := findJSONByStemForMedia(mediaFile, jsonStem, jsonFiles)
		if ok {
			return jsonFile, nil
		}
	}

	if len(mediaFile) > 46 {
		jsonFile, ok := findJSONByStemForMedia(mediaFile, mediaFile[:46], jsonFiles)
		if ok {
			return jsonFile, nil
		}
	}

	mediaExt := filepath.Ext(mediaFile)
	if strings.EqualFold(mediaExt, ".mp4") {
		baseMediaFile := strings.TrimSuffix(mediaFile, mediaExt)
		extensions := [...]string{".jpg", ".jpeg", ".heic"}
		for _, ext := range extensions {
			jsonStem := baseMediaFile + ext
			jsonFile, ok := findJSONByStemForMedia(mediaFile, jsonStem, jsonFiles)
			if ok {
				return jsonFile, nil
			}
			jsonFileUpper, ok := findJSONByStemForMedia(mediaFile, baseMediaFile+strings.ToUpper(ext), jsonFiles)
			if ok {
				return jsonFileUpper, nil
			}
		}
	}

	if jsonFile, ok := findJSONByBasenameForMedia(mediaFile, jsonFiles, false); ok {
		return jsonFile, nil
	}

	if shouldUseRandomSuffixFallback(mediaFile, mediaFiles) {
		if jsonFile, ok := findJSONByBasenameForMedia(mediaFile, jsonFiles, true); ok {
			return jsonFile, nil
		}
	}

	return "", fmt.Errorf("json file not found for %s", mediaFile)
}

func findJSONByStemForMedia(mediaFile string, stem string, jsonFiles map[string]struct{}) (string, bool) {
	if jsonFile, ok := findJSONCaseInsensitive(stem+".json", jsonFiles); ok {
		return jsonFile, true
	}

	candidates := findSupplementalJSONByStem(stem, jsonFiles)
	candidates = filterCandidatesByDuplicateIndex(mediaFile, candidates)
	if len(candidates) != 1 {
		return "", false
	}
	return candidates[0], true
}

func findSupplementalJSONByStem(stem string, jsonFiles map[string]struct{}) []string {
	lowerStem := strings.ToLower(stem)
	unique := make(map[string]struct{})
	for jsonFile := range jsonFiles {
		lower := strings.ToLower(jsonFile)
		if !strings.HasSuffix(lower, ".json") {
			continue
		}
		base := strings.TrimSuffix(lower, ".json")
		base = trailingNumberSuffixRe.ReplaceAllString(base, "")
		if !strings.HasPrefix(base, lowerStem+".") {
			continue
		}
		suffix := base[len(lowerStem):]
		if isSupplementalPrefix(suffix) {
			unique[jsonFile] = struct{}{}
		}
	}

	return slices.Sorted(maps.Keys(unique))
}

func findJSONCaseInsensitive(name string, jsonFiles map[string]struct{}) (string, bool) {
	for jsonFile := range jsonFiles {
		if strings.EqualFold(jsonFile, name) {
			return jsonFile, true
		}
	}
	return "", false
}

func findJSONByBasenameForMedia(mediaFile string, jsonFiles map[string]struct{}, stripRandomSuffix bool) (string, bool) {
	matches := findJSONCandidatesByBasename(mediaFile, jsonFiles, stripRandomSuffix)
	matches = filterCandidatesByDuplicateIndex(mediaFile, matches)
	if len(matches) != 1 {
		return "", false
	}
	return matches[0], true
}

func findJSONCandidatesByBasename(mediaFile string, jsonFiles map[string]struct{}, stripRandomSuffix bool) []string {
	mediaKey := normalizeMediaLookupKeyWithOptions(mediaFile, stripRandomSuffix)
	var matches []string

	for jsonFile := range jsonFiles {
		jsonKey := normalizeJSONKeyWithOptions(jsonFile, stripRandomSuffix)
		if jsonKey == "" {
			continue
		}
		if jsonKey == mediaKey {
			matches = append(matches, jsonFile)
		}
	}
	slices.Sort(matches)
	return matches
}

func filterCandidatesByDuplicateIndex(mediaFile string, candidates []string) []string {
	if len(candidates) == 0 {
		return candidates
	}

	mediaIndex, mediaHasExplicitIndex := extractMediaDuplicateIndexInfo(mediaFile)
	filtered := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		jsonIndex, jsonHasExplicitIndex := extractJSONDuplicateIndexInfo(candidate)
		switch {
		case mediaHasExplicitIndex:
			// Duplicate media (including "(0)") must match the same explicit
			// duplicate index in sidecar naming.
			if jsonHasExplicitIndex && jsonIndex == mediaIndex {
				filtered = append(filtered, candidate)
			}
		default:
			// Base media (without "(n)") prefers base sidecar naming.
			if !jsonHasExplicitIndex {
				filtered = append(filtered, candidate)
			}
		}
	}

	if len(filtered) == 0 {
		if mediaHasExplicitIndex {
			return nil
		}
		return candidates
	}
	slices.Sort(filtered)
	return filtered
}

func extractMediaDuplicateIndexInfo(mediaFile string) (int, bool) {
	name := filepath.Base(mediaFile)
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	return extractTrailingDuplicateIndexInfo(name)
}

func extractJSONDuplicateIndexInfo(jsonFile string) (int, bool) {
	name := strings.ToLower(filepath.Base(jsonFile))
	name = strings.TrimSuffix(name, ".json")

	// Keep legacy sidecar style priority: ...supplemental-metadata(n).json
	if index, ok := extractTrailingDuplicateIndexInfo(name); ok {
		return index, true
	}

	// Support sidecars that encode duplicate index in media stem:
	// name(n).ext.supplemental-metadata.json
	name = stripSupplementalSuffix(name)
	name = stripKnownMediaExtension(name)
	return extractTrailingDuplicateIndexInfo(name)
}

func extractTrailingDuplicateIndexInfo(name string) (int, bool) {
	match := duplicateIndexSuffixRe.FindStringSubmatch(name)
	if len(match) < 2 {
		return 0, false
	}
	index, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	return index, true
}

func removeRandomSuffix(mediaFile string) string {
	ext := filepath.Ext(mediaFile)
	base := strings.TrimSuffix(mediaFile, ext)
	base = randomSuffixRe.ReplaceAllString(base, "")
	return base + ext
}

func normalizeJSONKey(jsonFile string) string {
	return normalizeJSONKeyWithOptions(jsonFile, true)
}

func normalizeJSONKeyWithOptions(jsonFile string, stripRandomSuffix bool) string {
	name := strings.ToLower(jsonFile)
	if !strings.HasSuffix(name, ".json") {
		return ""
	}
	name = strings.TrimSuffix(name, ".json")
	name = trailingNumberSuffixRe.ReplaceAllString(name, "")
	name = stripSupplementalSuffix(name)
	return normalizeNameKeyWithOptions(name, stripRandomSuffix)
}

func isSupplementalPrefix(s string) bool {
	lower := strings.ToLower(s)
	if len(lower) < 2 || len(lower) > len(supplementalFull) {
		return false
	}
	return strings.HasPrefix(supplementalFull, lower)
}

func stripSupplementalSuffix(name string) string {
	for i := len(supplementalFull); i >= 2; i-- {
		prefix := supplementalFull[:i]
		if strings.HasSuffix(name, prefix) {
			return name[:len(name)-i]
		}
	}
	return name
}

func normalizeMediaLookupKey(mediaFile string) string {
	return normalizeMediaLookupKeyWithOptions(mediaFile, true)
}

func normalizeMediaLookupKeyWithOptions(mediaFile string, stripRandomSuffix bool) string {
	name := strings.ToLower(mediaFile)
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	return normalizeNameKeyWithOptions(name, stripRandomSuffix)
}

func normalizeNameKeyWithOptions(name string, stripRandomSuffix bool) string {
	name = strings.Replace(name, "-edited", "", 1)
	if stripRandomSuffix {
		name = randomSuffixRe.ReplaceAllString(name, "")
	}
	name = trailingNumberSuffixRe.ReplaceAllString(name, "")
	name = stripKnownMediaExtension(name)
	name = trailingNumberSuffixRe.ReplaceAllString(name, "")
	return name
}

func shouldUseRandomSuffixFallback(mediaFile string, mediaFiles map[string]struct{}) bool {
	if len(mediaFiles) == 0 {
		return false
	}

	base := strings.TrimSuffix(mediaFile, filepath.Ext(mediaFile))
	if !randomSuffixRe.MatchString(base) {
		return false
	}

	sibling := removeRandomSuffix(mediaFile)
	if sibling == mediaFile {
		return false
	}

	_, ok := mediaFiles[sibling]
	return ok
}

func stripKnownMediaExtension(name string) string {
	for range 2 {
		found := false
		for _, ext := range mediaext.Supported {
			if before, ok := strings.CutSuffix(name, ext); ok {
				name = before
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return name
}
