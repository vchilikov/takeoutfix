package files

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGetMedia_ReportsMissingAndUnusedJSON(t *testing.T) {
	album := t.TempDir()

	mustWrite := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(album, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	mustWrite("a.jpg")
	mustWrite("a.jpg.json")
	mustWrite("b.jpg")
	mustWrite("orphan.json")

	result, err := GetMedia(album)
	if err != nil {
		t.Fatalf("GetMedia error: %v", err)
	}

	if got, want := result.Pairs["a.jpg"], "a.jpg.json"; got != want {
		t.Fatalf("pair mismatch: want %q, got %q", want, got)
	}

	if !reflect.DeepEqual(result.MissingJSON, []string{"b.jpg"}) {
		t.Fatalf("missing json mismatch: got %v", result.MissingJSON)
	}

	if !reflect.DeepEqual(result.UnusedJSON, []string{"orphan.json"}) {
		t.Fatalf("unused json mismatch: got %v", result.UnusedJSON)
	}
}

func TestCheckUnusedJson_Sorted(t *testing.T) {
	unused := checkUnusedJson(
		map[string]struct{}{
			"z.json": {},
			"a.json": {},
		},
		map[string]struct{}{},
	)

	if !reflect.DeepEqual(unused, []string{"a.json", "z.json"}) {
		t.Fatalf("expected sorted unused json list, got %v", unused)
	}
}

func TestScanTakeout_CrossFolderSupplementalMatch(t *testing.T) {
	root := t.TempDir()

	mediaDir := filepath.Join(root, "Photos from 2022")
	jsonDir := filepath.Join(root, "Album X")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatalf("mkdir media dir: %v", err)
	}
	if err := os.MkdirAll(jsonDir, 0o755); err != nil {
		t.Fatalf("mkdir json dir: %v", err)
	}

	mediaRel := filepath.Join("Photos from 2022", "IMG_0001.jpg")
	jsonRel := filepath.Join("Album X", "IMG_0001.jpg.supplemental-metadata.json")

	if err := os.WriteFile(filepath.Join(root, mediaRel), []byte("media"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, jsonRel), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got, ok := result.Pairs[mediaRel]; !ok || got != jsonRel {
		t.Fatalf("expected pair %q -> %q, got %v", mediaRel, jsonRel, result.Pairs)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("unexpected missing json: %v", result.MissingJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("unexpected unused json: %v", result.UnusedJSON)
	}
}

func TestScanTakeout_AmbiguousGlobalMatch(t *testing.T) {
	root := t.TempDir()
	photosDir := filepath.Join(root, "Photos")
	aDir := filepath.Join(root, "Album A")
	bDir := filepath.Join(root, "Album B")
	if err := os.MkdirAll(photosDir, 0o755); err != nil {
		t.Fatalf("mkdir photos dir: %v", err)
	}
	if err := os.MkdirAll(aDir, 0o755); err != nil {
		t.Fatalf("mkdir album a dir: %v", err)
	}
	if err := os.MkdirAll(bDir, 0o755); err != nil {
		t.Fatalf("mkdir album b dir: %v", err)
	}

	mediaRel := filepath.Join("Photos", "IMG_0001.jpg")
	jsonA := filepath.Join("Album A", "IMG_0001.jpg.supplemental-metadata.json")
	jsonB := filepath.Join("Album B", "IMG_0001.jpg.supplemental-metada.json")

	if err := os.WriteFile(filepath.Join(root, mediaRel), []byte("media"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, jsonA), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write json A: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, jsonB), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write json B: %v", err)
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if _, ok := result.Pairs[mediaRel]; ok {
		t.Fatalf("did not expect a pair for ambiguous media %q", mediaRel)
	}

	ambiguous, ok := result.AmbiguousJSON[mediaRel]
	if !ok {
		t.Fatalf("expected ambiguous entry for %q, got %v", mediaRel, result.AmbiguousJSON)
	}
	wantAmbiguous := []string{jsonA, jsonB}
	if !reflect.DeepEqual(ambiguous, wantAmbiguous) {
		t.Fatalf("ambiguous mismatch: want %v, got %v", wantAmbiguous, ambiguous)
	}

	wantUnused := []string{jsonA, jsonB}
	if !reflect.DeepEqual(result.UnusedJSON, wantUnused) {
		t.Fatalf("unused mismatch: want %v, got %v", wantUnused, result.UnusedJSON)
	}
}
