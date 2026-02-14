package wizard

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/vchilikov/takeout-fix/internal/preflight"
	"github.com/vchilikov/takeout-fix/internal/processor"
	"github.com/vchilikov/takeout-fix/internal/state"
)

func TestRunStopsOnCorruptZip(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}}, nil
	}
	validateAll = func([]preflight.ZipArchive) preflight.IntegritySummary {
		corrupt := preflight.ArchiveIntegrity{Archive: preflight.ZipArchive{Name: "a.zip"}, Err: io.EOF}
		return preflight.IntegritySummary{Checked: []preflight.ArchiveIntegrity{corrupt}, Corrupt: []preflight.ArchiveIntegrity{corrupt}}
	}
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		return preflight.SpaceCheck{Enough: true, EnoughWithDelete: true}, nil
	}

	extractCalled := false
	extractArchiveFile = func(string, string) (int, error) {
		extractCalled = true
		return 0, nil
	}
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		t.Fatalf("process should not be called for corrupt zips")
		return processor.Report{}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail, got %d\n%s", code, out.String())
	}
	if extractCalled {
		t.Fatalf("did not expect extraction to run")
	}
}

func TestRunFailsWhenDependenciesAreMissing(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency {
		return []preflight.Dependency{{Name: "exiftool"}}
	}
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		t.Fatalf("zip scan should not start when dependencies are missing")
		return nil, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail, got %d\n%s", code, out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Please install: exiftool")) {
		t.Fatalf("expected missing dependency message, got:\n%s", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte(installerURLMacLinux)) {
		t.Fatalf("expected installer URL for macOS/Linux, got:\n%s", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte(installerURLWindows)) {
		t.Fatalf("expected installer URL for windows, got:\n%s", out.String())
	}
	if bytes.Contains(out.Bytes(), []byte("Install missing dependencies now? [y/N]: ")) {
		t.Fatalf("did not expect interactive dependency install prompt, got:\n%s", out.String())
	}
}

func TestRunRerunAfterArchiveReplace(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	archive := preflight.ZipArchive{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{archive}, nil
	}
	diskCheckCalls := 0
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		diskCheckCalls++
		return preflight.SpaceCheck{Enough: true, EnoughWithDelete: true, AvailableBytes: 10, RequiredBytes: 1}, nil
	}

	integrityCorrupt := true
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		if integrityCorrupt {
			corrupt := preflight.ArchiveIntegrity{Archive: zips[0], Err: io.EOF}
			return preflight.IntegritySummary{Checked: []preflight.ArchiveIntegrity{corrupt}, Corrupt: []preflight.ArchiveIntegrity{corrupt}}
		}
		ok := preflight.ArchiveIntegrity{Archive: zips[0], FileCount: 1, UncompressedBytes: 100}
		return preflight.IntegritySummary{Checked: []preflight.ArchiveIntegrity{ok}, TotalUncompressed: 100, TotalZipBytes: 20}
	}

	stored := state.New()
	loadState = func(string) (state.RunState, error) { return stored, nil }
	saveState = func(_ string, st state.RunState) error {
		stored = st
		return nil
	}
	removeFile = func(string) error { return nil }

	extractCalls := 0
	extractArchiveFile = func(string, string) (int, error) {
		extractCalls++
		return 2, nil
	}
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{}, nil
	}

	var out1 bytes.Buffer
	if code := Run(t.TempDir(), &out1); code != ExitPreflightFail {
		t.Fatalf("first run expected preflight fail, got %d", code)
	}
	if extractCalls != 0 {
		t.Fatalf("expected 0 extractions on corrupt preflight, got %d", extractCalls)
	}

	integrityCorrupt = false
	var out2 bytes.Buffer
	if code := Run(t.TempDir(), &out2); code != ExitSuccess {
		t.Fatalf("second run expected success, got %d\n%s", code, out2.String())
	}
	if extractCalls != 1 {
		t.Fatalf("expected extraction once after fix, got %d", extractCalls)
	}

	var out3 bytes.Buffer
	if code := Run(t.TempDir(), &out3); code != ExitSuccess {
		t.Fatalf("third run expected success, got %d", code)
	}
	if extractCalls != 1 {
		t.Fatalf("expected no additional extraction on rerun, got %d", extractCalls)
	}
	if diskCheckCalls != 1 {
		t.Fatalf("expected disk check once (run 2 only), got %d", diskCheckCalls)
	}
}

func TestRunAlwaysUsesEnglishWithoutLanguagePrompt(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) { return nil, nil }

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail due to no archives, got %d", code)
	}

	text := out.String()
	if !bytes.Contains(out.Bytes(), []byte("TakeoutFix")) {
		t.Fatalf("expected english title in output, got: %s", text)
	}
	if bytes.Contains(out.Bytes(), []byte("Language:")) || bytes.Contains(out.Bytes(), []byte("Язык:")) {
		t.Fatalf("did not expect language prompt in output: %s", text)
	}
}

func TestRunContinuesWhenOnlyAutoDeleteFits(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}}, nil
	}
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked:           []preflight.ArchiveIntegrity{{Archive: zips[0], FileCount: 1, UncompressedBytes: 100}},
			TotalUncompressed: 100,
			TotalZipBytes:     20,
		}
	}
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		return preflight.SpaceCheck{
			AvailableBytes:          90,
			RequiredBytes:           120,
			RequiredWithDeleteBytes: 80,
			Enough:                  false,
			EnoughWithDelete:        true,
		}, nil
	}
	loadState = func(string) (state.RunState, error) { return state.New(), nil }
	saveState = func(string, state.RunState) error { return nil }
	extractArchiveFile = func(string, string) (int, error) { return 1, nil }
	removeFile = func(string) error { return nil }
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Low-space mode: ZIP files will be deleted right after extraction.")) {
		t.Fatalf("expected low-space mode warning, got:\n%s", out.String())
	}
	if bytes.Contains(out.Bytes(), []byte("Enable delete-mode")) {
		t.Fatalf("did not expect interactive delete-mode prompt, got:\n%s", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Run result: Completed")) {
		t.Fatalf("expected success summary, got:\n%s", out.String())
	}
}

func TestRunFailsWhenAutoDeleteIsInsufficient(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}}, nil
	}
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked:           []preflight.ArchiveIntegrity{{Archive: zips[0], FileCount: 1, UncompressedBytes: 100}},
			TotalUncompressed: 100,
			TotalZipBytes:     20,
		}
	}
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		return preflight.SpaceCheck{
			AvailableBytes:          70,
			RequiredBytes:           120,
			RequiredWithDeleteBytes: 90,
			Enough:                  false,
			EnoughWithDelete:        false,
		}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail, got %d\n%s", code, out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Not enough free disk space to continue.")) {
		t.Fatalf("expected explicit auto-delete failure message, got:\n%s", out.String())
	}
}

func TestRunAutoLanguageEnglishWithoutPrompt(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) { return nil, nil }

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail due to no archives, got %d", code)
	}

	text := out.String()
	if !bytes.Contains(out.Bytes(), []byte("TakeoutFix")) {
		t.Fatalf("expected english title in output, got: %s", text)
	}
	if bytes.Contains(out.Bytes(), []byte("Language:")) || bytes.Contains(out.Bytes(), []byte("Язык:")) {
		t.Fatalf("did not expect language prompt in output: %s", text)
	}
}

func TestRunShowsExtractionProgressForSkippedAndExtracted(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	zips := []preflight.ZipArchive{
		{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"},
		{Name: "b.zip", Path: "/tmp/b.zip", Fingerprint: "f2"},
	}
	discoverZips = func(string) ([]preflight.ZipArchive, error) { return zips, nil }
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked: []preflight.ArchiveIntegrity{
				{Archive: zips[0], FileCount: 1, UncompressedBytes: 100},
				{Archive: zips[1], FileCount: 1, UncompressedBytes: 200},
			},
			TotalUncompressed: 300,
			TotalZipBytes:     60,
		}
	}
	var diskCheckArchives []preflight.ArchiveIntegrity
	checkDiskSpace = func(_ string, archives []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		diskCheckArchives = append([]preflight.ArchiveIntegrity(nil), archives...)
		return preflight.SpaceCheck{Enough: true, EnoughWithDelete: true, AvailableBytes: 1024, RequiredBytes: 128}, nil
	}

	shouldSkip = func(_ state.RunState, name string, _ string) bool { return name == "a.zip" }
	loadState = func(string) (state.RunState, error) { return state.New(), nil }
	saveState = func(string, state.RunState) error { return nil }

	extractArchiveFile = func(path string, _ string) (int, error) {
		if path != "/tmp/b.zip" {
			t.Fatalf("unexpected archive extracted: %s", path)
		}
		return 3, nil
	}
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}

	output := out.Bytes()
	if !bytes.Contains(output, []byte("Preparing files from ZIP archives... done")) {
		t.Fatalf("expected archive preparation completion line, got:\n%s", out.String())
	}

	if len(diskCheckArchives) != 1 {
		t.Fatalf("expected disk check for 1 pending archive, got %d", len(diskCheckArchives))
	}
	if diskCheckArchives[0].Archive.Name != "b.zip" {
		t.Fatalf("expected disk check for b.zip, got %s", diskCheckArchives[0].Archive.Name)
	}
}

func TestRunShowsThrottledProcessingProgress(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}}, nil
	}
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked:           []preflight.ArchiveIntegrity{{Archive: zips[0], FileCount: 1, UncompressedBytes: 100}},
			TotalUncompressed: 100,
			TotalZipBytes:     20,
		}
	}
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		return preflight.SpaceCheck{Enough: true, EnoughWithDelete: true, AvailableBytes: 10, RequiredBytes: 1}, nil
	}
	loadState = func(string) (state.RunState, error) { return state.New(), nil }
	saveState = func(string, state.RunState) error { return nil }
	extractArchiveFile = func(string, string) (int, error) { return 2, nil }
	processTakeout = func(_ string, onProgress func(processor.ProgressEvent)) (processor.Report, error) {
		onProgress(processor.ProgressEvent{Processed: 1, Total: 500, Media: "A.jpg"})
		onProgress(processor.ProgressEvent{Processed: 2, Total: 500, Media: "B.jpg"})
		onProgress(processor.ProgressEvent{Processed: 5, Total: 500, Media: "C.jpg"})
		onProgress(processor.ProgressEvent{Processed: 500, Total: 500, Media: "Z.jpg"})
		return processor.Report{}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}

	output := out.Bytes()
	if !bytes.Contains(output, []byte("Progress: 0%")) {
		t.Fatalf("expected initial progress line, got:\n%s", out.String())
	}
	if bytes.Contains(output, []byte("A.jpg")) || bytes.Contains(output, []byte("B.jpg")) || bytes.Contains(output, []byte("C.jpg")) {
		t.Fatalf("did not expect per-file progress output, got:\n%s", out.String())
	}
	if !bytes.Contains(output, []byte("Progress: 100%")) {
		t.Fatalf("expected final progress line, got:\n%s", out.String())
	}
}

func TestRunShowsEmptyProcessingProgress(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}}, nil
	}
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked:           []preflight.ArchiveIntegrity{{Archive: zips[0], FileCount: 1, UncompressedBytes: 100}},
			TotalUncompressed: 100,
			TotalZipBytes:     20,
		}
	}
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		return preflight.SpaceCheck{Enough: true, EnoughWithDelete: true, AvailableBytes: 10, RequiredBytes: 1}, nil
	}
	loadState = func(string) (state.RunState, error) { return state.New(), nil }
	saveState = func(string, state.RunState) error { return nil }
	extractArchiveFile = func(string, string) (int, error) { return 1, nil }
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Progress: 0%")) {
		t.Fatalf("expected empty processing progress line, got:\n%s", out.String())
	}
}

func TestRunPassesValidatedArchivesToDiskCheck(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}}, nil
	}
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked: []preflight.ArchiveIntegrity{
				{Archive: zips[0], FileCount: 2, UncompressedBytes: 300},
			},
			TotalUncompressed: 300,
			TotalZipBytes:     50,
		}
	}

	var gotArchives []preflight.ArchiveIntegrity
	checkDiskSpace = func(_ string, archives []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		gotArchives = append([]preflight.ArchiveIntegrity(nil), archives...)
		return preflight.SpaceCheck{Enough: true, EnoughWithDelete: true, AvailableBytes: 1024, RequiredBytes: 256}, nil
	}

	loadState = func(string) (state.RunState, error) { return state.New(), nil }
	saveState = func(string, state.RunState) error { return nil }
	extractArchiveFile = func(string, string) (int, error) { return 1, nil }
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{}, nil
	}

	code := Run(t.TempDir(), &bytes.Buffer{})
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d", code)
	}

	if len(gotArchives) != 1 {
		t.Fatalf("expected disk check to receive 1 archive, got %d", len(gotArchives))
	}
	if gotArchives[0].UncompressedBytes != 300 {
		t.Fatalf("unexpected uncompressed bytes: want 300, got %d", gotArchives[0].UncompressedBytes)
	}
	if gotArchives[0].FileCount != 2 {
		t.Fatalf("unexpected file count: want 2, got %d", gotArchives[0].FileCount)
	}
}

func TestRunReprocessesWhenNoZipsButExtractedDirExists(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) { return nil, nil }

	processCalled := false
	processTakeout = func(dir string, _ func(processor.ProgressEvent)) (processor.Report, error) {
		processCalled = true
		if !bytes.Contains([]byte(dir), []byte("takeoutfix-extracted")) {
			t.Fatalf("expected process dir to contain takeoutfix-extracted, got %s", dir)
		}
		return processor.Report{}, nil
	}

	cwd := t.TempDir()
	if err := os.MkdirAll(cwd+"/takeoutfix-extracted/subdir", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cwd+"/takeoutfix-extracted/subdir/photo.jpg", []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	code := Run(cwd, &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}
	if !processCalled {
		t.Fatalf("expected processTakeout to be called")
	}
	if !bytes.Contains(out.Bytes(), []byte("Using previously extracted Takeout data from:")) {
		t.Fatalf("expected re-processing message, got:\n%s", out.String())
	}
}

func TestRunFailsWhenNoZipsAndNoExtractedDir(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) { return nil, nil }
	detectTakeoutRoot = func(string) (string, bool, error) { return "", false, nil }

	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		t.Fatalf("process should not be called when no zips and no extracted dir")
		return processor.Report{}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail, got %d\n%s", code, out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("No ZIP files or extracted Takeout data found in this folder.")) {
		t.Fatalf("expected no-data message, got:\n%s", out.String())
	}
}

func TestResolveNoZipProcessRoot_StatErrorIncludesPath(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	extractedRoot := "\x00takeoutfix-extracted"
	_, _, preflightFail, err := resolveNoZipProcessRoot(t.TempDir(), extractedRoot)
	if err == nil {
		t.Fatalf("expected stat error")
	}
	if preflightFail {
		t.Fatalf("expected runtime path, got preflight fail")
	}
	if !strings.Contains(err.Error(), `access extracted data path "\x00takeoutfix-extracted":`) {
		t.Fatalf("expected error to include quoted path, got: %v", err)
	}
}

func TestRunProcessesDetectedTakeoutRootWhenNoZips(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) { return nil, nil }
	cwd := t.TempDir()
	detectedRoot := filepath.Join(cwd, "Takeout")
	detectTakeoutRoot = func(path string) (string, bool, error) {
		if path != cwd {
			t.Fatalf("unexpected detect path: %q", path)
		}
		return detectedRoot, true, nil
	}
	extractArchiveFile = func(string, string) (int, error) {
		t.Fatalf("did not expect extraction when processing existing content")
		return 0, nil
	}

	var processedDir string
	processTakeout = func(dir string, _ func(processor.ProgressEvent)) (processor.Report, error) {
		processedDir = dir
		return processor.Report{}, nil
	}

	var out bytes.Buffer
	code := Run(cwd, &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}
	if processedDir != detectedRoot {
		t.Fatalf("expected process dir to be detected root, got %q", processedDir)
	}
	if !bytes.Contains(out.Bytes(), []byte("Using existing Takeout content from: "+detectedRoot)) {
		t.Fatalf("expected processing-existing-content message, got:\n%s", out.String())
	}
}

func TestRunFailsWhenExtractedPathIsFile(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) { return nil, nil }

	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		t.Fatalf("process should not be called when extracted path is a file")
		return processor.Report{}, nil
	}

	cwd := t.TempDir()
	if err := os.WriteFile(cwd+"/takeoutfix-extracted", []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	code := Run(cwd, &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail, got %d\n%s", code, out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("but it is not a folder")) {
		t.Fatalf("expected not-a-directory message, got:\n%s", out.String())
	}
}

func TestRunSkipsDiskCheckWhenAllArchivesExtracted(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{
			{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"},
			{Name: "b.zip", Path: "/tmp/b.zip", Fingerprint: "f2"},
		}, nil
	}
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked: []preflight.ArchiveIntegrity{
				{Archive: zips[0], FileCount: 1, UncompressedBytes: 100},
				{Archive: zips[1], FileCount: 1, UncompressedBytes: 200},
			},
			TotalUncompressed: 300,
			TotalZipBytes:     60,
		}
	}

	diskCheckCalled := false
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		diskCheckCalled = true
		return preflight.SpaceCheck{Enough: true, EnoughWithDelete: true}, nil
	}

	shouldSkip = func(_ state.RunState, _ string, _ string) bool { return true }
	stored := state.New()
	loadState = func(string) (state.RunState, error) { return stored, nil }
	saveState = func(_ string, st state.RunState) error {
		stored = st
		return nil
	}
	removed := make([]string, 0, 2)
	removeFile = func(path string) error {
		removed = append(removed, path)
		return nil
	}
	extractArchiveFile = func(string, string) (int, error) {
		t.Fatalf("extract should not be called when all archives are skipped")
		return 0, nil
	}
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}
	if diskCheckCalled {
		t.Fatalf("expected disk check to be skipped when all archives are already extracted")
	}
	if bytes.Contains(out.Bytes(), []byte("Checking free disk space...")) {
		t.Fatalf("expected no disk check in output, got:\n%s", out.String())
	}
	if len(removed) != 2 {
		t.Fatalf("expected 2 skipped archives to be deleted, got %d", len(removed))
	}
	for _, name := range []string{"a.zip", "b.zip"} {
		entry, ok := stored.Archives[name]
		if !ok {
			t.Fatalf("expected state entry for %s", name)
		}
		if !entry.Extracted || !entry.Deleted {
			t.Fatalf("expected state entry for %s to be extracted+deleted, got %+v", name, entry)
		}
	}
}

func TestRunSkipAlreadyDeletedArchiveDoesNotDeleteAgain(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}}, nil
	}
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked:           []preflight.ArchiveIntegrity{{Archive: zips[0], FileCount: 1, UncompressedBytes: 100}},
			TotalUncompressed: 100,
			TotalZipBytes:     20,
		}
	}
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		t.Fatalf("disk check should not run when all archives are skipped")
		return preflight.SpaceCheck{}, nil
	}

	stored := state.New()
	stored.Archives["a.zip"] = state.ArchiveState{
		Fingerprint: "f1",
		Extracted:   true,
		Deleted:     true,
	}
	loadState = func(string) (state.RunState, error) { return stored, nil }
	saveState = func(_ string, st state.RunState) error {
		stored = st
		return nil
	}

	removeCalls := 0
	removeFile = func(string) error {
		removeCalls++
		return nil
	}
	extractArchiveFile = func(string, string) (int, error) {
		t.Fatalf("extract should not be called when archive is skipped")
		return 0, nil
	}
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}
	if removeCalls != 0 {
		t.Fatalf("expected no delete attempts for already deleted archive, got %d", removeCalls)
	}
	if bytes.Contains(out.Bytes(), []byte("zip delete errors")) {
		t.Fatalf("did not expect zip delete errors, got:\n%s", out.String())
	}
	if entry := stored.Archives["a.zip"]; !entry.Deleted {
		t.Fatalf("expected archive to remain marked deleted, got %+v", entry)
	}
}

func TestRunMissingZipDuringDeleteDoesNotReportError(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}}, nil
	}
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked:           []preflight.ArchiveIntegrity{{Archive: zips[0], FileCount: 1, UncompressedBytes: 100}},
			TotalUncompressed: 100,
			TotalZipBytes:     20,
		}
	}
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		return preflight.SpaceCheck{
			AvailableBytes:          90,
			RequiredBytes:           120,
			RequiredWithDeleteBytes: 80,
			Enough:                  false,
			EnoughWithDelete:        true,
		}, nil
	}

	stored := state.New()
	loadState = func(string) (state.RunState, error) { return stored, nil }
	saveState = func(_ string, st state.RunState) error {
		stored = st
		return nil
	}

	removeFile = func(string) error { return os.ErrNotExist }
	extractArchiveFile = func(string, string) (int, error) { return 1, nil }
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}
	if bytes.Contains(out.Bytes(), []byte("zip delete errors")) {
		t.Fatalf("did not expect zip delete errors for missing files, got:\n%s", out.String())
	}
	entry, ok := stored.Archives["a.zip"]
	if !ok {
		t.Fatalf("expected state entry for a.zip")
	}
	if !entry.Deleted {
		t.Fatalf("expected archive to be marked deleted when zip file is missing, got %+v", entry)
	}
}

func TestRunPartialSuccessOnHardProcessingErrorsKeepsZipsInSafeMode(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}}, nil
	}
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked:           []preflight.ArchiveIntegrity{{Archive: zips[0], FileCount: 1, UncompressedBytes: 100}},
			TotalUncompressed: 100,
			TotalZipBytes:     20,
		}
	}
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		return preflight.SpaceCheck{
			AvailableBytes:          200,
			RequiredBytes:           120,
			RequiredWithDeleteBytes: 80,
			Enough:                  true,
			EnoughWithDelete:        true,
		}, nil
	}

	stored := state.New()
	loadState = func(string) (state.RunState, error) { return stored, nil }
	saveState = func(_ string, st state.RunState) error {
		stored = st
		return nil
	}

	removeCalled := false
	removeFile = func(string) error {
		removeCalled = true
		return nil
	}
	extractArchiveFile = func(string, string) (int, error) { return 1, nil }
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{
			ProblemCounts: map[string]int{
				"metadata errors": 1,
			},
			ProblemSamples: map[string][]string{
				"metadata errors": []string{"Takeout/photo.jpg"},
			},
		}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitRuntimeFail {
		t.Fatalf("expected runtime fail on hard processing errors, got %d\n%s", code, out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Run result: Completed with issues")) {
		t.Fatalf("expected PARTIAL_SUCCESS status, got:\n%s", out.String())
	}
	if removeCalled {
		t.Fatalf("did not expect zip deletion in safe mode on hard errors")
	}
	entry, ok := stored.Archives["a.zip"]
	if !ok {
		t.Fatalf("expected state entry for a.zip")
	}
	if entry.Deleted {
		t.Fatalf("expected a.zip to remain undeleted on hard processing errors")
	}
}

func TestRunWarningOnlyProblemsStillSucceed(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}}, nil
	}
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked:           []preflight.ArchiveIntegrity{{Archive: zips[0], FileCount: 1, UncompressedBytes: 100}},
			TotalUncompressed: 100,
			TotalZipBytes:     20,
		}
	}
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		return preflight.SpaceCheck{
			AvailableBytes:          200,
			RequiredBytes:           120,
			RequiredWithDeleteBytes: 80,
			Enough:                  true,
			EnoughWithDelete:        true,
		}, nil
	}
	loadState = func(string) (state.RunState, error) { return state.New(), nil }
	saveState = func(string, state.RunState) error { return nil }

	removeCalls := 0
	removeFile = func(string) error {
		removeCalls++
		return nil
	}
	extractArchiveFile = func(string, string) (int, error) { return 1, nil }
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{
			Summary: processor.Summary{
				CreateDateWarnings: 1,
			},
			ProblemCounts: map[string]int{
				"create date warnings": 1,
				"json remove errors":   1,
			},
			ProblemSamples: map[string][]string{
				"create date warnings": []string{"Takeout/photo.jpg"},
				"json remove errors":   []string{"Takeout/photo.jpg.json"},
			},
		}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success for warning-only problems, got %d\n%s", code, out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Run result: Completed")) {
		t.Fatalf("expected SUCCESS status, got:\n%s", out.String())
	}
	if removeCalls != 1 {
		t.Fatalf("expected deferred zip deletion on success, got %d calls", removeCalls)
	}
}

func TestRunWritesDetailedJSONReport(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		return []preflight.ZipArchive{{Name: "a.zip", Path: "/tmp/a.zip", Fingerprint: "f1"}}, nil
	}
	validateAll = func(zips []preflight.ZipArchive) preflight.IntegritySummary {
		return preflight.IntegritySummary{
			Checked:           []preflight.ArchiveIntegrity{{Archive: zips[0], FileCount: 1, UncompressedBytes: 100}},
			TotalUncompressed: 100,
			TotalZipBytes:     20,
		}
	}
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		return preflight.SpaceCheck{
			AvailableBytes:          300,
			RequiredBytes:           120,
			RequiredWithDeleteBytes: 80,
			Enough:                  true,
			EnoughWithDelete:        true,
		}, nil
	}
	loadState = func(string) (state.RunState, error) { return state.New(), nil }
	saveState = func(string, state.RunState) error { return nil }
	removeFile = func(string) error { return nil }
	extractArchiveFile = func(string, string) (int, error) { return 1, nil }
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{
			Summary: processor.Summary{
				MediaFound:          2,
				MetadataApplied:     1,
				FilenameDateApplied: 1,
				JSONRemoved:         1,
				MissingJSON:         1,
			},
		}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}

	reportPath := detailedReportPathFromOutput(out.String())
	if reportPath == "" {
		t.Fatalf("expected detailed report path in output, got:\n%s", out.String())
	}

	base := filepath.Base(reportPath)
	matched, err := regexp.MatchString(`^report-\d{8}-\d{6}\.json$`, base)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Fatalf("unexpected report file name %q", base)
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected report json at %s: %v", reportPath, err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("expected valid json report, got parse error: %v", err)
	}
	if got, _ := parsed["status"].(string); got != "SUCCESS" {
		t.Fatalf("status mismatch: want SUCCESS, got %v", parsed["status"])
	}
	if got, ok := parsed["exit_code"].(float64); !ok || int(got) != ExitSuccess {
		t.Fatalf("exit_code mismatch: want %d, got %v", ExitSuccess, parsed["exit_code"])
	}
	if _, ok := parsed["timings_ms"].(map[string]any); !ok {
		t.Fatalf("expected timings_ms object in json report, got %T", parsed["timings_ms"])
	}
	if _, ok := parsed["metadata"].(map[string]any); !ok {
		t.Fatalf("expected metadata object in json report, got %T", parsed["metadata"])
	}
}

func TestRunDoesNotPrintDetailedPathWhenReportWriteFails(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency {
		return []preflight.Dependency{{Name: "exiftool"}}
	}
	writeReportJSON = func(Report) (string, error) {
		return "/tmp/report-failed.json", errors.New("write report json: disk full")
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail, got %d\n%s", code, out.String())
	}

	text := out.String()
	if !strings.Contains(text, "Detailed report: unavailable") {
		t.Fatalf("expected unavailable detailed report, got:\n%s", text)
	}
	if strings.Contains(text, "Detailed report: /tmp/report-failed.json") {
		t.Fatalf("did not expect detailed report path on write failure, got:\n%s", text)
	}
	if !strings.Contains(text, "Report save warning: write report json: disk full (path: /tmp/report-failed.json)") {
		t.Fatalf("expected warning with failed report path, got:\n%s", text)
	}
}

func detailedReportPathFromOutput(output string) string {
	const prefix = "Detailed report: "
	for line := range strings.SplitSeq(output, "\n") {
		if after, ok := strings.CutPrefix(line, prefix); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

func stubWizardDeps() func() {
	origCheckDependencies := checkDependencies
	origDiscoverZips := discoverZips
	origValidateAll := validateAll
	origCheckDiskSpace := checkDiskSpace
	origLoadState := loadState
	origSaveState := saveState
	origShouldSkip := shouldSkip
	origExtractArchiveFile := extractArchiveFile
	origProcessTakeout := processTakeout
	origRemoveFile := removeFile
	origDetectTakeoutRoot := detectTakeoutRoot
	origWriteReportJSON := writeReportJSON

	return func() {
		checkDependencies = origCheckDependencies
		discoverZips = origDiscoverZips
		validateAll = origValidateAll
		checkDiskSpace = origCheckDiskSpace
		loadState = origLoadState
		saveState = origSaveState
		shouldSkip = origShouldSkip
		extractArchiveFile = origExtractArchiveFile
		processTakeout = origProcessTakeout
		removeFile = origRemoveFile
		detectTakeoutRoot = origDetectTakeoutRoot
		writeReportJSON = origWriteReportJSON
	}
}
