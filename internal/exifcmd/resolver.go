package exifcmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

var (
	currentGOOS = runtime.GOOS
	lookPathFn  = exec.LookPath
)

func Candidates(goos string) []string {
	if goos == "windows" {
		return []string{"exiftool", "exiftool.exe", "exiftool(-k).exe"}
	}
	return []string{"exiftool"}
}

func Resolve() (string, error) {
	candidates := Candidates(currentGOOS)
	for _, candidate := range candidates {
		if resolved, err := lookPathFn(candidate); err == nil {
			return resolved, nil
		}
	}

	return "", fmt.Errorf("exiftool executable not found in PATH (tried: %s)", strings.Join(candidates, ", "))
}
