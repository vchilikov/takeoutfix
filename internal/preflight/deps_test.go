package preflight

import (
	"errors"
	"testing"
)

func TestCheckDependencies_UsesResolver(t *testing.T) {
	origResolver := resolveExiftool
	defer func() {
		resolveExiftool = origResolver
	}()

	resolveExiftool = func() (string, error) {
		return "/usr/bin/exiftool", nil
	}

	got := CheckDependencies()
	if len(got) != 0 {
		t.Fatalf("expected no missing dependencies, got %#v", got)
	}
}

func TestCheckDependencies_MissingExiftool(t *testing.T) {
	origResolver := resolveExiftool
	defer func() {
		resolveExiftool = origResolver
	}()

	resolveExiftool = func() (string, error) {
		return "", errors.New("not found")
	}

	got := CheckDependencies()
	if len(got) != 1 {
		t.Fatalf("expected one missing dependency, got %#v", got)
	}
	if got[0].Name != "exiftool" {
		t.Fatalf("unexpected dependency name: %q", got[0].Name)
	}
}
