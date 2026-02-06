package patharg

import (
	"path/filepath"
	"testing"
)

func TestSafe(t *testing.T) {
	want := "." + string(filepath.Separator) + "-input.jpg"
	if got := Safe("-input.jpg"); got != want {
		t.Fatalf("expected sanitized path, got %q", got)
	}
	if got := Safe("/tmp/-input.jpg"); got != "/tmp/-input.jpg" {
		t.Fatalf("absolute path should not change, got %q", got)
	}
}
