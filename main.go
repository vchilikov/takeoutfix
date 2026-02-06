package main

import (
	"fmt"
	"github.com/vchilikov/takeout-fix/utils/extensions"
	"github.com/vchilikov/takeout-fix/utils/files"
	"github.com/vchilikov/takeout-fix/utils/metadata"
	"log"
	"os"
	"path/filepath"
	"sort"
)

const (
	OperationFix       = "fix"
	OperationCleanJson = "clean-json"
)

func init() {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

func main() {
	operation, path := getArgs()
	switch operation {
	case OperationFix:
		fixTakeout(path)
	case OperationCleanJson:
		cleanTakeoutJson(path)
	}
}

func getArgs() (string, string) {
	help := `Usage: takeoutfix <operation> <path>

Supported operations:
$ takeoutfix fix /path/to/takeout
$ takeoutfix clean-json /path/to/takeout

`

	if len(os.Args) < 3 {
		log.Fatal(help)
	}

	if os.Args[1] != OperationFix && os.Args[1] != OperationCleanJson {
		log.Fatalf("Operation %s is not supported.\n\n%s", os.Args[1], help)
	}

	return os.Args[1], os.Args[2]
}

func fixTakeout(rootPath string) {
	scanResult, err := files.ScanTakeout(rootPath)
	if err != nil {
		fmt.Printf("🚨 could not scan takeout %s: %s\n", rootPath, err)
		return
	}

	printScanWarnings(rootPath, scanResult)

	mediaFiles := make([]string, 0, len(scanResult.Pairs))
	for mediaFile := range scanResult.Pairs {
		mediaFiles = append(mediaFiles, mediaFile)
	}
	sort.Strings(mediaFiles)

	for _, mediaFile := range mediaFiles {
		jsonFile := scanResult.Pairs[mediaFile]
		mediaPath := filepath.Join(rootPath, mediaFile)
		jsonPath := filepath.Join(rootPath, jsonFile)

		newMediaPath, err := extensions.Fix(mediaPath)
		if err != nil {
			fmt.Printf("🚨 extension for file %s could not be fixed: %s\n", mediaPath, err)
			continue
		}

		err = metadata.Apply(newMediaPath, jsonPath)
		if err != nil {
			fmt.Printf("🚨 metadata for file %s could not be applied: %s\n", newMediaPath, err)
			continue
		}
	}
}

func cleanTakeoutJson(rootPath string) {
	scanResult, err := files.ScanTakeout(rootPath)
	if err != nil {
		fmt.Printf("🚨 could not scan takeout %s: %s\n", rootPath, err)
		return
	}

	printScanWarnings(rootPath, scanResult)

	jsonToRemove := make(map[string]struct{})
	for _, jsonFile := range scanResult.Pairs {
		jsonToRemove[jsonFile] = struct{}{}
	}

	for jsonFile := range jsonToRemove {
		err := os.Remove(filepath.Join(rootPath, jsonFile))
		if err != nil {
			fmt.Printf("🚨 could not remove %s: %s\n", filepath.Join(rootPath, jsonFile), err)
		}
	}
}

func printScanWarnings(rootPath string, scanResult files.MediaScanResult) {
	for _, mediaFile := range scanResult.MissingJSON {
		fmt.Printf("⚠️ no matching json for %s\n", filepath.Join(rootPath, mediaFile))
	}
	for _, jsonFile := range scanResult.UnusedJSON {
		fmt.Printf("⚠️ unused json kept: %s\n", filepath.Join(rootPath, jsonFile))
	}
	ambiguousMedia := make([]string, 0, len(scanResult.AmbiguousJSON))
	for mediaFile := range scanResult.AmbiguousJSON {
		ambiguousMedia = append(ambiguousMedia, mediaFile)
	}
	sort.Strings(ambiguousMedia)
	for _, mediaFile := range ambiguousMedia {
		fmt.Printf("⚠️ ambiguous json candidates for %s: %v\n", filepath.Join(rootPath, mediaFile), scanResult.AmbiguousJSON[mediaFile])
	}
}
