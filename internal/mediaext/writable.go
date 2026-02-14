package mediaext

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/vchilikov/takeout-fix/internal/exifcmd"
)

var runListWritableTypes = func() (string, error) {
	bin, err := exifcmd.Resolve()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(bin, "-listwf")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("run exiftool -listwf: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

type writableResolver struct {
	loadOnce func() (map[string]struct{}, error)
}

func newWritableResolver(loader func() (string, error)) *writableResolver {
	return &writableResolver{
		loadOnce: sync.OnceValues(func() (map[string]struct{}, error) {
			output, err := loader()
			if err != nil {
				return nil, err
			}
			return parseWritableExtensionSet(output), nil
		}),
	}
}

var defaultWritableResolver = newWritableResolver(runListWritableTypes)

// IsWritableExtension reports whether exiftool can write metadata into files
// with the provided extension (for example ".jpg" or ".avi").
func IsWritableExtension(ext string) (bool, error) {
	return defaultWritableResolver.IsWritableExtension(ext)
}

func (r *writableResolver) IsWritableExtension(ext string) (bool, error) {
	normalized := normalizeExtension(ext)
	if normalized == "" {
		return false, nil
	}

	extSet, err := r.loadOnce()
	if err != nil {
		return false, err
	}

	_, ok := extSet[normalized]
	return ok, nil
}

func normalizeExtension(ext string) string {
	normalized := strings.ToLower(strings.TrimSpace(ext))
	if normalized == "" {
		return ""
	}
	if !strings.HasPrefix(normalized, ".") {
		normalized = "." + normalized
	}
	return normalized
}

func parseWritableExtensionSet(output string) map[string]struct{} {
	set := make(map[string]struct{})
	for token := range strings.FieldsSeq(output) {
		cleaned := strings.Trim(token, " \t\r\n,;:()[]{}")
		if cleaned == "" {
			continue
		}
		if !isWritableToken(cleaned) {
			continue
		}
		set["."+strings.ToLower(cleaned)] = struct{}{}
	}
	return set
}

func isWritableToken(token string) bool {
	if token == "" || token != strings.ToUpper(token) {
		return false
	}

	hasLetter := false
	for _, r := range token {
		switch {
		case r >= 'A' && r <= 'Z':
			hasLetter = true
		case r >= '0' && r <= '9':
			// allowed
		default:
			return false
		}
	}

	return hasLetter
}
