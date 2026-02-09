package files

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vchilikov/takeout-fix/internal/mediaext"
)

var numberSuffixRe = regexp.MustCompile(`\(\d+\)`)
var randomSuffixRe = regexp.MustCompile(`-[a-z0-9]{5}$`)

const supplementalFull = ".supplemental-metadata"

var trailingNumberSuffixRe = regexp.MustCompile(`\(\d+\)$`)

func getJsonFile(mediaFile string, jsonFiles map[string]struct{}) (string, error) {
	if jsonFile, ok := findJSONByStem(mediaFile, jsonFiles); ok {
		return jsonFile, nil
	}

	baseMediaFile := strings.TrimSuffix(mediaFile, filepath.Ext(mediaFile))
	if jsonFile, ok := findJSONByStem(baseMediaFile, jsonFiles); ok {
		return jsonFile, nil
	}

	if strings.Contains(mediaFile, "-edited") {
		return getJsonFile(strings.Replace(mediaFile, "-edited", "", 1), jsonFiles)
	}

	if numberSuffixRe.MatchString(mediaFile) {
		match := numberSuffixRe.FindString(mediaFile)
		jsonStem := strings.Replace(mediaFile, match, "", 1) + match
		jsonFile, ok := findJSONByStem(jsonStem, jsonFiles)
		if ok {
			return jsonFile, nil
		}
	}

	if len(mediaFile) > 46 && numberSuffixRe.MatchString(mediaFile) {
		match := numberSuffixRe.FindString(mediaFile)
		jsonStem := mediaFile[:46] + match
		jsonFile, ok := findJSONByStem(jsonStem, jsonFiles)
		if ok {
			return jsonFile, nil
		}
	}

	if len(mediaFile) > 46 {
		jsonFile, ok := findJSONByStem(mediaFile[:46], jsonFiles)
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
			jsonFile, ok := findJSONByStem(jsonStem, jsonFiles)
			if ok {
				return jsonFile, nil
			}
			jsonFileUpper, ok := findJSONByStem(baseMediaFile+strings.ToUpper(ext), jsonFiles)
			if ok {
				return jsonFileUpper, nil
			}
		}
	}

	if jsonFile, ok := findJSONByBasename(mediaFile, jsonFiles); ok {
		return jsonFile, nil
	}

	if jsonFile, ok := findJSONByBasename(removeRandomSuffix(mediaFile), jsonFiles); ok {
		return jsonFile, nil
	}

	return "", fmt.Errorf("json file not found for %s", mediaFile)
}

func findJSONByStem(stem string, jsonFiles map[string]struct{}) (string, bool) {
	if jsonFile, ok := findJSONCaseInsensitive(stem+".json", jsonFiles); ok {
		return jsonFile, true
	}

	lowerStem := strings.ToLower(stem)
	var matches []string
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
			matches = append(matches, jsonFile)
		}
	}

	if len(matches) != 1 {
		return "", false
	}
	return matches[0], true
}

func findJSONCaseInsensitive(name string, jsonFiles map[string]struct{}) (string, bool) {
	for jsonFile := range jsonFiles {
		if strings.EqualFold(jsonFile, name) {
			return jsonFile, true
		}
	}
	return "", false
}

func findJSONByBasename(mediaFile string, jsonFiles map[string]struct{}) (string, bool) {
	mediaKey := normalizeMediaLookupKey(mediaFile)
	var matches []string

	for jsonFile := range jsonFiles {
		jsonKey := normalizeJSONKey(jsonFile)
		if jsonKey == "" {
			continue
		}
		if jsonKey == mediaKey {
			matches = append(matches, jsonFile)
		}
	}

	if len(matches) != 1 {
		return "", false
	}

	return matches[0], true
}

func removeRandomSuffix(mediaFile string) string {
	ext := filepath.Ext(mediaFile)
	base := strings.TrimSuffix(mediaFile, ext)
	base = randomSuffixRe.ReplaceAllString(base, "")
	return base + ext
}

func normalizeJSONKey(jsonFile string) string {
	name := strings.ToLower(jsonFile)
	if !strings.HasSuffix(name, ".json") {
		return ""
	}
	name = strings.TrimSuffix(name, ".json")
	name = trailingNumberSuffixRe.ReplaceAllString(name, "")
	name = stripSupplementalSuffix(name)
	return normalizeNameKey(name)
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
	name := strings.ToLower(mediaFile)
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	return normalizeNameKey(name)
}

func normalizeNameKey(name string) string {
	name = strings.Replace(name, "-edited", "", 1)
	name = randomSuffixRe.ReplaceAllString(name, "")
	name = trailingNumberSuffixRe.ReplaceAllString(name, "")
	name = stripKnownMediaExtension(name)
	return name
}

func stripKnownMediaExtension(name string) string {
	for range 2 {
		found := false
		for _, ext := range mediaext.Supported {
			if strings.HasSuffix(name, ext) {
				name = strings.TrimSuffix(name, ext)
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
