package extensions

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAreExtensionsCompatible(t *testing.T) {
	tests := []struct {
		ext1 string
		ext2 string
		want bool
	}{
		{".jpg", ".jpeg", true},
		{".mov", ".mp4", true},
		{".PNG", ".png", true},
		{".jpg", ".png", false},
	}

	for _, tt := range tests {
		got := areExtensionsCompatible(tt.ext1, tt.ext2)
		if got != tt.want {
			t.Fatalf("compatibility mismatch for %q/%q: want %v, got %v", tt.ext1, tt.ext2, tt.want, got)
		}
	}
}

func TestGenerateRandomSuffix(t *testing.T) {
	got, err := generateRandomSuffix()
	if err != nil {
		t.Fatalf("generateRandomSuffix error: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("expected suffix length 5, got %d", len(got))
	}
	for _, r := range got {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz0123456789", r) {
			t.Fatalf("unexpected rune in suffix: %q", r)
		}
	}
}

func TestSafePathArg(t *testing.T) {
	want := "." + string(filepath.Separator) + "-input.jpg"
	if got := safePathArg("-input.jpg"); got != want {
		t.Fatalf("expected sanitized path, got %q", got)
	}
	if got := safePathArg("/tmp/-input.jpg"); got != "/tmp/-input.jpg" {
		t.Fatalf("absolute path should not change, got %q", got)
	}
}
