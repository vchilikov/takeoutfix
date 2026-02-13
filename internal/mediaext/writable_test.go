package mediaext

import (
	"errors"
	"strings"
	"testing"
)

func TestParseWritableExtensionSet(t *testing.T) {
	t.Parallel()

	output := "Writable file types:\n  3GP AVI HEIC JPG MP4 MOV WEBP\n"
	got := parseWritableExtensionSet(output)

	for _, ext := range []string{".3gp", ".avi", ".heic", ".jpg", ".mp4", ".mov", ".webp"} {
		if _, ok := got[ext]; !ok {
			t.Fatalf("expected %s in writable set, got %v", ext, got)
		}
	}
	if _, ok := got[".writable"]; ok {
		t.Fatalf("unexpected non-extension token in set: %v", got)
	}
}

func TestWritableResolver_UsesCachedList(t *testing.T) {
	t.Parallel()

	calls := 0
	resolver := newWritableResolver(func() (string, error) {
		calls++
		return "Writable file types:\n  JPG MP4 HEIC\n", nil
	})

	ok, err := resolver.IsWritableExtension(".jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected .jpg to be writable")
	}

	ok, err = resolver.IsWritableExtension("AVI")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected AVI to be non-writable")
	}

	if calls != 1 {
		t.Fatalf("expected cached list to be loaded once, got %d", calls)
	}
}

func TestWritableResolver_LoadError(t *testing.T) {
	t.Parallel()

	resolver := newWritableResolver(func() (string, error) {
		return "", errors.New("boom")
	})

	_, err := resolver.IsWritableExtension(".jpg")
	if err == nil {
		t.Fatalf("expected error from loader")
	}
}

func TestIsWritableToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		token string
		want  bool
	}{
		{token: "JPG", want: true},
		{token: "3GP", want: true},
		{token: "Writable", want: false},
		{token: "file", want: false},
		{token: "XMP:", want: false},
		{token: strings.ToLower("MP4"), want: false},
	}

	for _, tt := range tests {
		if got := isWritableToken(tt.token); got != tt.want {
			t.Fatalf("isWritableToken(%q) = %v, want %v", tt.token, got, tt.want)
		}
	}
}

func TestNormalizeExtension(t *testing.T) {
	t.Parallel()

	if got := normalizeExtension(""); got != "" {
		t.Fatalf("normalize empty extension: got %q", got)
	}
	if got := normalizeExtension(" JPG "); got != ".jpg" {
		t.Fatalf("normalize JPG: got %q", got)
	}
	if got := normalizeExtension(".HeIc"); got != ".heic" {
		t.Fatalf("normalize .HeIc: got %q", got)
	}
}
