package state

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".takeoutfix", "state.json")

	st := New()
	st.Archives["a.zip"] = ArchiveState{Fingerprint: "fp1", Extracted: true}

	if err := Save(path, st); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if !ShouldSkipExtraction(got, "a.zip", "fp1") {
		t.Fatalf("expected a.zip to be skippable")
	}
	if ShouldSkipExtraction(got, "a.zip", "fp2") {
		t.Fatalf("did not expect skip for changed fingerprint")
	}
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(got.Archives) != 0 {
		t.Fatalf("expected empty state")
	}
}
