package metadata

import (
	"slices"
	"strings"
	"testing"
)

func TestHasSupportedExtension(t *testing.T) {
	if !hasSupportedExtension("photo.JPEG") {
		t.Fatalf("expected JPEG extension to be supported")
	}
	if !hasSupportedExtension("photo.TIFF") {
		t.Fatalf("expected TIFF extension to be supported")
	}
	if hasSupportedExtension("photo.webp") {
		t.Fatalf("expected WEBP extension to be unsupported")
	}
}

func TestBuildExiftoolArgs_DoesNotIncludeOffsetTags(t *testing.T) {
	args := buildExiftoolArgs("meta.json", "photo.jpg")

	if !slices.Contains(args, "--") {
		t.Fatalf("expected -- separator in args")
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "-OffsetTime") {
			t.Fatalf("did not expect OffsetTime* arguments, got: %v", args)
		}
	}
}
