package files

import "slices"

func checkUnusedJson(jsonFiles map[string]struct{}, usedJsonFiles map[string]struct{}) []string {
	var unused []string

	for jsonFile := range jsonFiles {
		if _, ok := usedJsonFiles[jsonFile]; !ok {
			unused = append(unused, jsonFile)
		}
	}

	slices.Sort(unused)
	return unused
}
