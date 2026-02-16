package files

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScanTakeoutFlatDirReportsMissingAndUnusedJSON(t *testing.T) {
	root := t.TempDir()

	mustWrite := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	mustWrite("a.jpg")
	mustWrite("a.jpg.json")
	mustWrite("b.jpg")
	mustWrite("orphan.json")

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
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

func TestScanTakeoutFlatDirRandomSuffixWithoutMatchStaysMissing(t *testing.T) {
	root := t.TempDir()

	mustWrite := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	mustWrite("IMG_0001-abcde.png")
	mustWrite("IMG_9999.jpg.json")

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if len(result.Pairs) != 0 {
		t.Fatalf("expected no pairs, got %v", result.Pairs)
	}
	if !reflect.DeepEqual(result.MissingJSON, []string{"IMG_0001-abcde.png"}) {
		t.Fatalf("missing json mismatch: got %v", result.MissingJSON)
	}
	if !reflect.DeepEqual(result.UnusedJSON, []string{"IMG_9999.jpg.json"}) {
		t.Fatalf("unused json mismatch: got %v", result.UnusedJSON)
	}
}

func TestScanTakeoutFlatDirRandomSuffixWithSiblingAllowsFallback(t *testing.T) {
	root := t.TempDir()

	mustWrite := func(name string, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	mustWrite("IMG_0001-abcde.png", "a")
	mustWrite("IMG_0001.png", "b")
	mustWrite("IMG_0001.jpg.json", "{}")

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if len(result.Pairs) != 0 {
		t.Fatalf("expected no pairs in strict mode, got %v", result.Pairs)
	}
	wantAmbiguous := map[string][]string{
		"IMG_0001-abcde.png": []string{"IMG_0001.jpg.json"},
		"IMG_0001.png":       []string{"IMG_0001.jpg.json"},
	}
	if !reflect.DeepEqual(result.AmbiguousJSON, wantAmbiguous) {
		t.Fatalf("ambiguous mismatch: want %v, got %v", wantAmbiguous, result.AmbiguousJSON)
	}
}

func TestScanTakeoutFlatDirRandomSuffixWithSiblingSharesJSONForExactDuplicates(t *testing.T) {
	root := t.TempDir()

	mustWrite := func(name string, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	mustWrite("IMG_0001-abcde.png", "same")
	mustWrite("IMG_0001.png", "same")
	mustWrite("IMG_0001.jpg.json", "{}")

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs["IMG_0001-abcde.png"]; got != "IMG_0001.jpg.json" {
		t.Fatalf("pair mismatch for random suffix media: want %q, got %q", "IMG_0001.jpg.json", got)
	}
	if got := result.Pairs["IMG_0001.png"]; got != "IMG_0001.jpg.json" {
		t.Fatalf("pair mismatch for base media: want %q, got %q", "IMG_0001.jpg.json", got)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	if len(result.AmbiguousJSON) != 0 {
		t.Fatalf("expected no ambiguous json, got %v", result.AmbiguousJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("expected no unused json, got %v", result.UnusedJSON)
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

func TestScanTakeout_UsesSupportedMediaWhitelist(t *testing.T) {
	root := t.TempDir()
	webpMediaRel := "photo.webp"
	webpJSONRel := "photo.webp.json"
	aviMediaRel := "clip.AVI"
	aviJSONRel := "clip.AVI.supplemental-metadata.json"
	nonMediaRel := "notes.txt"
	nonMediaJSONRel := "notes.txt.json"

	if err := os.WriteFile(filepath.Join(root, webpMediaRel), []byte("media"), 0o600); err != nil {
		t.Fatalf("write webp media: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, webpJSONRel), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write webp json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, aviMediaRel), []byte("media"), 0o600); err != nil {
		t.Fatalf("write avi media: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, aviJSONRel), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write avi json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, nonMediaRel), []byte("notes"), 0o600); err != nil {
		t.Fatalf("write txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, nonMediaJSONRel), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write txt json: %v", err)
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got, ok := result.Pairs[webpMediaRel]; !ok || got != webpJSONRel {
		t.Fatalf("expected pair %q -> %q, got %v", webpMediaRel, webpJSONRel, result.Pairs)
	}
	if got, ok := result.Pairs[aviMediaRel]; !ok || got != aviJSONRel {
		t.Fatalf("expected pair %q -> %q, got %v", aviMediaRel, aviJSONRel, result.Pairs)
	}
	if _, ok := result.Pairs[nonMediaRel]; ok {
		t.Fatalf("did not expect non-media txt file in pairs: %v", result.Pairs)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	if !reflect.DeepEqual(result.UnusedJSON, []string{nonMediaJSONRel}) {
		t.Fatalf("unused mismatch: want %v, got %v", []string{nonMediaJSONRel}, result.UnusedJSON)
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

func TestScanTakeout_DuplicateIndexPairsResolvedLocally(t *testing.T) {
	root := t.TempDir()

	mediaBase := "PIC_0003.JPG"
	mediaDup := "PIC_0003(1).JPG"
	jsonBase := "PIC_0003.JPG.supplemental-metadata.json"
	jsonDup := "PIC_0003.JPG.supplemental-metadata(1).json"

	for _, name := range []string{mediaBase, mediaDup, jsonBase, jsonDup} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs[mediaBase]; got != jsonBase {
		t.Fatalf("base pair mismatch: want %q, got %q", jsonBase, got)
	}
	if got := result.Pairs[mediaDup]; got != jsonDup {
		t.Fatalf("duplicate pair mismatch: want %q, got %q", jsonDup, got)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	if len(result.AmbiguousJSON) != 0 {
		t.Fatalf("expected no ambiguous json, got %v", result.AmbiguousJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("expected no unused json, got %v", result.UnusedJSON)
	}
}

func TestScanTakeout_DuplicateIndexMediaStemSupplementalPairsResolvedLocally(t *testing.T) {
	root := t.TempDir()

	mediaBase := "20170530_170805.jpg"
	mediaDup := "20170530_170805(0).jpg"
	jsonBase := "20170530_170805.jpg.supplemental-metadata.json"
	jsonDup := "20170530_170805(0).jpg.supplemental-metadata.json"

	for _, name := range []string{mediaBase, mediaDup, jsonBase, jsonDup} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs[mediaBase]; got != jsonBase {
		t.Fatalf("base pair mismatch: want %q, got %q", jsonBase, got)
	}
	if got := result.Pairs[mediaDup]; got != jsonDup {
		t.Fatalf("duplicate pair mismatch: want %q, got %q", jsonDup, got)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	if len(result.AmbiguousJSON) != 0 {
		t.Fatalf("expected no ambiguous json, got %v", result.AmbiguousJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("expected no unused json, got %v", result.UnusedJSON)
	}
}

func TestScanTakeout_GlobalSameDirTieBreakPrefersLocalCandidate(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"A", "B"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	mediaRel := filepath.Join("A", "IMG_0001-abcde.png")
	jsonSameDir := filepath.Join("A", "IMG_0001.jpg.supplemental-metadata.json")
	jsonOtherDir := filepath.Join("B", "IMG_0001.jpg.supplemental-metadata.json")

	for _, rel := range []string{mediaRel, jsonSameDir, jsonOtherDir} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs[mediaRel]; got != jsonSameDir {
		t.Fatalf("pair mismatch: want %q, got %q", jsonSameDir, got)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	if len(result.AmbiguousJSON) != 0 {
		t.Fatalf("expected no ambiguous json, got %v", result.AmbiguousJSON)
	}
	if !reflect.DeepEqual(result.UnusedJSON, []string{jsonOtherDir}) {
		t.Fatalf("unused mismatch: want %v, got %v", []string{jsonOtherDir}, result.UnusedJSON)
	}
}

func TestApplyGlobalCandidateRules_SameDirFallbackKeepsCandidates(t *testing.T) {
	mediaRel := filepath.Join("A", "IMG_0001.jpg")
	candidates := []string{
		filepath.Join("B", "IMG_0001.jpg.supplemental-metadata.json"),
		filepath.Join("C", "IMG_0001.jpg.supplemental-metadata.json"),
	}

	got := applyGlobalCandidateRules(mediaRel, candidates)
	if !reflect.DeepEqual(got, candidates) {
		t.Fatalf("same-dir fallback mismatch: want %v, got %v", candidates, got)
	}
}

func TestApplyGlobalCandidateRules_SingleExplicitDuplicateIndexCandidateFilteredForBaseMedia(t *testing.T) {
	mediaRel := filepath.Join("A", "IMG_0001.jpg")
	candidates := []string{
		filepath.Join("B", "IMG_0001(0).jpg.supplemental-metadata.json"),
	}

	got := applyGlobalCandidateRules(mediaRel, candidates)
	if len(got) != 0 {
		t.Fatalf("expected no candidates after duplicate-index filtering, got %v", got)
	}
}

func TestScanTakeout_IgnoresXMPAsInputMedia(t *testing.T) {
	root := t.TempDir()
	jsonRel := "IMG_0001.webp.json"

	if err := os.WriteFile(filepath.Join(root, "IMG_0001.webp.xmp"), []byte("xmp"), 0o600); err != nil {
		t.Fatalf("write xmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, jsonRel), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if len(result.Pairs) != 0 {
		t.Fatalf("expected no pairs, got %v", result.Pairs)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	if len(result.AmbiguousJSON) != 0 {
		t.Fatalf("expected no ambiguous json, got %v", result.AmbiguousJSON)
	}
	if !reflect.DeepEqual(result.UnusedJSON, []string{jsonRel}) {
		t.Fatalf("unused mismatch: want %v, got %v", []string{jsonRel}, result.UnusedJSON)
	}
}

func TestScanTakeout_SharedSingleCandidateUsesUniqueSameDirClaimant(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"A", "B"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	mediaA := filepath.Join("A", "IMG_0001-abcde.png")
	mediaB := filepath.Join("B", "IMG_0001-fghij.png")
	jsonRel := filepath.Join("A", "IMG_0001.jpg.supplemental-metadata.json")

	if err := os.WriteFile(filepath.Join(root, mediaA), []byte("a"), 0o600); err != nil {
		t.Fatalf("write %s: %v", mediaA, err)
	}
	if err := os.WriteFile(filepath.Join(root, mediaB), []byte("b"), 0o600); err != nil {
		t.Fatalf("write %s: %v", mediaB, err)
	}
	if err := os.WriteFile(filepath.Join(root, jsonRel), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write %s: %v", jsonRel, err)
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs[mediaA]; got != jsonRel {
		t.Fatalf("expected same-dir claimant pair %q -> %q, got %v", mediaA, jsonRel, result.Pairs)
	}
	if _, ok := result.Pairs[mediaB]; ok {
		t.Fatalf("did not expect pair for non-winning claimant %q", mediaB)
	}
	wantAmbiguous := map[string][]string{
		mediaB: []string{jsonRel},
	}
	if !reflect.DeepEqual(result.AmbiguousJSON, wantAmbiguous) {
		t.Fatalf("ambiguous mismatch: want %v, got %v", wantAmbiguous, result.AmbiguousJSON)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("expected no unused json, got %v", result.UnusedJSON)
	}
}

func TestScanTakeout_LocalSharedSingleJSONPrefersJSONTargetJPG(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "A")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir A: %v", err)
	}

	mediaJPG := filepath.Join("A", "IMG_0001.jpg")
	mediaMP4 := filepath.Join("A", "IMG_0001.mp4")
	jsonRel := filepath.Join("A", "IMG_0001.jpg.supplemental-metadata.json")

	if err := os.WriteFile(filepath.Join(root, mediaJPG), []byte("jpg-data"), 0o600); err != nil {
		t.Fatalf("write %s: %v", mediaJPG, err)
	}
	if err := os.WriteFile(filepath.Join(root, mediaMP4), []byte("mp4-data"), 0o600); err != nil {
		t.Fatalf("write %s: %v", mediaMP4, err)
	}
	if err := os.WriteFile(filepath.Join(root, jsonRel), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write %s: %v", jsonRel, err)
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs[mediaJPG]; got != jsonRel {
		t.Fatalf("pair mismatch: want %q, got %q", jsonRel, got)
	}
	if _, ok := result.Pairs[mediaMP4]; ok {
		t.Fatalf("did not expect pair for %q, got %v", mediaMP4, result.Pairs)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	wantAmbiguous := map[string][]string{
		mediaMP4: []string{jsonRel},
	}
	if !reflect.DeepEqual(result.AmbiguousJSON, wantAmbiguous) {
		t.Fatalf("ambiguous mismatch: want %v, got %v", wantAmbiguous, result.AmbiguousJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("expected no unused json, got %v", result.UnusedJSON)
	}
}

func TestScanTakeout_LocalSharedSingleJSONPrefersJSONTargetMP4(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "A")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir A: %v", err)
	}

	mediaJPG := filepath.Join("A", "IMG_0001.jpg")
	mediaMP4 := filepath.Join("A", "IMG_0001.mp4")
	jsonRel := filepath.Join("A", "IMG_0001.mp4.supplemental-metadata.json")

	if err := os.WriteFile(filepath.Join(root, mediaJPG), []byte("jpg-data"), 0o600); err != nil {
		t.Fatalf("write %s: %v", mediaJPG, err)
	}
	if err := os.WriteFile(filepath.Join(root, mediaMP4), []byte("mp4-data"), 0o600); err != nil {
		t.Fatalf("write %s: %v", mediaMP4, err)
	}
	if err := os.WriteFile(filepath.Join(root, jsonRel), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write %s: %v", jsonRel, err)
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs[mediaMP4]; got != jsonRel {
		t.Fatalf("pair mismatch: want %q, got %q", jsonRel, got)
	}
	if _, ok := result.Pairs[mediaJPG]; ok {
		t.Fatalf("did not expect pair for %q, got %v", mediaJPG, result.Pairs)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	wantAmbiguous := map[string][]string{
		mediaJPG: []string{jsonRel},
	}
	if !reflect.DeepEqual(result.AmbiguousJSON, wantAmbiguous) {
		t.Fatalf("ambiguous mismatch: want %v, got %v", wantAmbiguous, result.AmbiguousJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("expected no unused json, got %v", result.UnusedJSON)
	}
}

func TestScanTakeout_LocalSharedSingleJSONPrefersJSONTargetWithDuplicateIndex(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "A")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir A: %v", err)
	}

	mediaJPG := filepath.Join("A", "IMG_0001(1).jpg")
	mediaMP4 := filepath.Join("A", "IMG_0001(1).mp4")
	jsonRel := filepath.Join("A", "IMG_0001.jpg.supplemental-metadata(1).json")

	if err := os.WriteFile(filepath.Join(root, mediaJPG), []byte("jpg-data"), 0o600); err != nil {
		t.Fatalf("write %s: %v", mediaJPG, err)
	}
	if err := os.WriteFile(filepath.Join(root, mediaMP4), []byte("mp4-data"), 0o600); err != nil {
		t.Fatalf("write %s: %v", mediaMP4, err)
	}
	if err := os.WriteFile(filepath.Join(root, jsonRel), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write %s: %v", jsonRel, err)
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs[mediaJPG]; got != jsonRel {
		t.Fatalf("pair mismatch: want %q, got %q", jsonRel, got)
	}
	if _, ok := result.Pairs[mediaMP4]; ok {
		t.Fatalf("did not expect pair for %q, got %v", mediaMP4, result.Pairs)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	wantAmbiguous := map[string][]string{
		mediaMP4: []string{jsonRel},
	}
	if !reflect.DeepEqual(result.AmbiguousJSON, wantAmbiguous) {
		t.Fatalf("ambiguous mismatch: want %v, got %v", wantAmbiguous, result.AmbiguousJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("expected no unused json, got %v", result.UnusedJSON)
	}
}

func TestScanTakeout_ExplicitZeroDuplicateIndexDoesNotStealBaseJSON(t *testing.T) {
	root := t.TempDir()
	mediaBase := "20180905_180723.jpg"
	mediaDupZero := "20180905_180723(0).jpg"
	jsonBase := "20180905_180723.jpg.supplemental-metadata.json"

	for _, name := range []string{mediaBase, mediaDupZero, jsonBase} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs[mediaBase]; got != jsonBase {
		t.Fatalf("base pair mismatch: want %q, got %q", jsonBase, got)
	}
	if _, ok := result.Pairs[mediaDupZero]; ok {
		t.Fatalf("did not expect pair for explicit zero duplicate media %q", mediaDupZero)
	}
	if !reflect.DeepEqual(result.MissingJSON, []string{mediaDupZero}) {
		t.Fatalf("missing mismatch: want %v, got %v", []string{mediaDupZero}, result.MissingJSON)
	}
	if len(result.AmbiguousJSON) != 0 {
		t.Fatalf("expected no ambiguous json, got %v", result.AmbiguousJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("expected no unused json, got %v", result.UnusedJSON)
	}
}

func TestScanTakeout_BaseAndDuplicateWithOnlyExplicitDuplicateSidecar(t *testing.T) {
	root := t.TempDir()
	mediaBase := "20160321_192953.jpg"
	mediaDup := "20160321_192953(0).jpg"
	jsonDup := "20160321_192953(0).jpg.supplemental-metadata.json"

	for _, name := range []string{mediaBase, mediaDup, jsonDup} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs[mediaDup]; got != jsonDup {
		t.Fatalf("duplicate pair mismatch: want %q, got %q", jsonDup, got)
	}
	if _, ok := result.Pairs[mediaBase]; ok {
		t.Fatalf("did not expect pair for base media %q", mediaBase)
	}
	if !reflect.DeepEqual(result.MissingJSON, []string{mediaBase}) {
		t.Fatalf("missing mismatch: want %v, got %v", []string{mediaBase}, result.MissingJSON)
	}
	if len(result.AmbiguousJSON) != 0 {
		t.Fatalf("expected no ambiguous json, got %v", result.AmbiguousJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("expected no unused json, got %v", result.UnusedJSON)
	}
}

func TestScanTakeout_BaseAndDuplicateWithOnlyExplicitDuplicateSidecarGlobal(t *testing.T) {
	root := t.TempDir()
	mediaBase := filepath.Join("Photos", "20160321_192953.jpg")
	mediaDup := filepath.Join("Photos", "20160321_192953(0).jpg")
	jsonDup := filepath.Join("Sidecars", "20160321_192953(0).jpg.supplemental-metadata.json")

	for _, dir := range []string{"Photos", "Sidecars"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	for _, rel := range []string{mediaBase, mediaDup, jsonDup} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs[mediaDup]; got != jsonDup {
		t.Fatalf("duplicate pair mismatch: want %q, got %q", jsonDup, got)
	}
	if _, ok := result.Pairs[mediaBase]; ok {
		t.Fatalf("did not expect pair for base media %q", mediaBase)
	}
	if !reflect.DeepEqual(result.MissingJSON, []string{mediaBase}) {
		t.Fatalf("missing mismatch: want %v, got %v", []string{mediaBase}, result.MissingJSON)
	}
	if len(result.AmbiguousJSON) != 0 {
		t.Fatalf("expected no ambiguous json, got %v", result.AmbiguousJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("expected no unused json, got %v", result.UnusedJSON)
	}
}

func TestScanTakeout_DuplicateMediaSingleGlobalJSONSharedWhenBinaryIdentical(t *testing.T) {
	root := t.TempDir()
	mediaA := filepath.Join("A", "IMG_0001.jpg")
	mediaB := filepath.Join("B", "IMG_0001.jpg")
	jsonRel := filepath.Join("JSON", "IMG_0001.jpg.supplemental-metadata.json")

	for _, dir := range []string{"A", "B", "JSON"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, mediaA), []byte("same"), 0o600); err != nil {
		t.Fatalf("write media A: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, mediaB), []byte("same"), 0o600); err != nil {
		t.Fatalf("write media B: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, jsonRel), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if got := result.Pairs[mediaA]; got != jsonRel {
		t.Fatalf("pair mismatch for media A: want %q, got %q", jsonRel, got)
	}
	if got := result.Pairs[mediaB]; got != jsonRel {
		t.Fatalf("pair mismatch for media B: want %q, got %q", jsonRel, got)
	}
	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	if len(result.AmbiguousJSON) != 0 {
		t.Fatalf("expected no ambiguous json, got %v", result.AmbiguousJSON)
	}
	if len(result.UnusedJSON) != 0 {
		t.Fatalf("expected no unused json, got %v", result.UnusedJSON)
	}
}

func TestScanTakeout_DuplicateMediaSingleGlobalJSONIsAmbiguous(t *testing.T) {
	root := t.TempDir()
	mediaA := filepath.Join("A", "IMG_0001.jpg")
	mediaB := filepath.Join("B", "IMG_0001.jpg")
	jsonRel := filepath.Join("JSON", "IMG_0001.jpg.supplemental-metadata.json")

	for _, dir := range []string{"A", "B", "JSON"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, mediaA), []byte("media-a"), 0o600); err != nil {
		t.Fatalf("write media A: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, mediaB), []byte("media-b"), 0o600); err != nil {
		t.Fatalf("write media B: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, jsonRel), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}

	result, err := ScanTakeout(root)
	if err != nil {
		t.Fatalf("ScanTakeout error: %v", err)
	}

	if len(result.Pairs) != 0 {
		t.Fatalf("expected no pairs, got %v", result.Pairs)
	}

	wantAmbiguous := map[string][]string{
		mediaA: []string{jsonRel},
		mediaB: []string{jsonRel},
	}
	if !reflect.DeepEqual(result.AmbiguousJSON, wantAmbiguous) {
		t.Fatalf("ambiguous mismatch: want %v, got %v", wantAmbiguous, result.AmbiguousJSON)
	}

	if len(result.MissingJSON) != 0 {
		t.Fatalf("expected no missing json, got %v", result.MissingJSON)
	}
	if !reflect.DeepEqual(result.UnusedJSON, []string{jsonRel}) {
		t.Fatalf("unused mismatch: want %v, got %v", []string{jsonRel}, result.UnusedJSON)
	}
}
