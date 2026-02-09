package preflight

import (
	"github.com/vchilikov/takeout-fix/internal/exifcmd"
)

type Dependency struct {
	Name string
}

var (
	resolveExiftool = exifcmd.Resolve
)

func CheckDependencies() []Dependency {
	var missing []Dependency

	if _, err := resolveExiftool(); err != nil {
		missing = append(missing, Dependency{Name: "exiftool"})
	}

	return missing
}
