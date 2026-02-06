package files

import "sort"

func checkUnusedJson(jsonFiles map[string]struct{}, usedJsonFiles map[string]struct{}) []string {
	var unused []string

	for jsonFile := range jsonFiles {
		if _, ok := usedJsonFiles[jsonFile]; !ok {
			unused = append(unused, jsonFile)
		}
	}

	sort.Strings(unused)
	return unused
}
