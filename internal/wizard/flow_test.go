package wizard

import (
	"bytes"
	"io"
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
		return preflight.SpaceCheck{Enough: true}, nil
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
	code := Run(t.TempDir(), bytes.NewBufferString("\n"), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail, got %d\n%s", code, out.String())
	}
	if extractCalled {
		t.Fatalf("did not expect extraction to run")
	}
}

func TestRunFailsWhenAutoInstallUnavailable(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency {
		return []preflight.Dependency{{Name: "exiftool", InstallCmd: nil}}
	}
	discoverZips = func(string) ([]preflight.ZipArchive, error) {
		t.Fatalf("zip scan should not start when install is unavailable")
		return nil, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), bytes.NewBufferString("\n"), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail, got %d\n%s", code, out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Automatic install is supported on macOS (Homebrew), Linux (apt/dnf/pacman), and Windows (winget). Please install manually and rerun.")) {
		t.Fatalf("expected unsupported auto-install message, got:\n%s", out.String())
	}
}

func TestRunInstallsDependenciesWhenAutoInstallAvailable(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	missing := []preflight.Dependency{
		{Name: "exiftool", InstallCmd: []string{"sudo", "apt-get", "install", "-y", "libimage-exiftool-perl"}},
	}
	checkCalls := 0
	checkDependencies = func() []preflight.Dependency {
		checkCalls++
		if checkCalls == 1 {
			return missing
		}
		return nil
	}

	installCalls := 0
	installDependencies = func(deps []preflight.Dependency, in io.Reader, out io.Writer) error {
		installCalls++
		if in == nil {
			t.Fatalf("expected non-nil input reader for installer")
		}
		if len(deps) != 1 || deps[0].Name != "exiftool" {
			t.Fatalf("unexpected deps passed to installer: %#v", deps)
		}
		return nil
	}

	discoverZips = func(string) ([]preflight.ZipArchive, error) { return nil, nil }

	var out bytes.Buffer
	code := Run(t.TempDir(), bytes.NewBufferString("y\n"), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail due to no archives, got %d\n%s", code, out.String())
	}
	if installCalls != 1 {
		t.Fatalf("expected installer to run once, got %d", installCalls)
	}
	if checkCalls != 2 {
		t.Fatalf("expected dependency check twice (before/after install), got %d", checkCalls)
	}
	if !bytes.Contains(out.Bytes(), []byte("Running: sudo apt-get install -y libimage-exiftool-perl")) {
		t.Fatalf("expected install command to be shown, got:\n%s", out.String())
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
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		return preflight.SpaceCheck{Enough: true, AvailableBytes: 10, RequiredBytes: 1}, nil
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
	if code := Run(t.TempDir(), bytes.NewBufferString("\n"), &out1); code != ExitPreflightFail {
		t.Fatalf("first run expected preflight fail, got %d", code)
	}
	if extractCalls != 0 {
		t.Fatalf("expected 0 extractions on corrupt preflight, got %d", extractCalls)
	}

	integrityCorrupt = false
	var out2 bytes.Buffer
	if code := Run(t.TempDir(), bytes.NewBufferString("\n"), &out2); code != ExitSuccess {
		t.Fatalf("second run expected success, got %d\n%s", code, out2.String())
	}
	if extractCalls != 1 {
		t.Fatalf("expected extraction once after fix, got %d", extractCalls)
	}

	var out3 bytes.Buffer
	if code := Run(t.TempDir(), bytes.NewBufferString("\n"), &out3); code != ExitSuccess {
		t.Fatalf("third run expected success, got %d", code)
	}
	if extractCalls != 1 {
		t.Fatalf("expected no additional extraction on rerun, got %d", extractCalls)
	}
}

func TestRunAlwaysUsesEnglishWithoutLanguagePrompt(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) { return nil, nil }

	var out bytes.Buffer
	code := Run(t.TempDir(), bytes.NewBufferString(""), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail due to no archives, got %d", code)
	}

	text := out.String()
	if !bytes.Contains(out.Bytes(), []byte("TakeoutFix interactive mode")) {
		t.Fatalf("expected english title in output, got: %s", text)
	}
	if bytes.Contains(out.Bytes(), []byte("Language:")) || bytes.Contains(out.Bytes(), []byte("Язык:")) {
		t.Fatalf("did not expect language prompt in output: %s", text)
	}
}

func TestRunAutoLanguageEnglishWithoutPrompt(t *testing.T) {
	restore := stubWizardDeps()
	defer restore()

	checkDependencies = func() []preflight.Dependency { return nil }
	discoverZips = func(string) ([]preflight.ZipArchive, error) { return nil, nil }

	var out bytes.Buffer
	code := Run(t.TempDir(), bytes.NewBufferString(""), &out)
	if code != ExitPreflightFail {
		t.Fatalf("expected preflight fail due to no archives, got %d", code)
	}

	text := out.String()
	if !bytes.Contains(out.Bytes(), []byte("TakeoutFix interactive mode")) {
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
	checkDiskSpace = func(string, []preflight.ArchiveIntegrity) (preflight.SpaceCheck, error) {
		return preflight.SpaceCheck{Enough: true, AvailableBytes: 1024, RequiredBytes: 128}, nil
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
	code := Run(t.TempDir(), bytes.NewBufferString("\n"), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}

	output := out.Bytes()
	if !bytes.Contains(output, []byte("Extraction progress: 1/2 (50%) skipped: a.zip")) {
		t.Fatalf("expected skipped extraction progress line, got:\n%s", out.String())
	}
	if !bytes.Contains(output, []byte("Extraction progress: 2/2 (100%) extracted: b.zip")) {
		t.Fatalf("expected extracted progress line, got:\n%s", out.String())
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
		return preflight.SpaceCheck{Enough: true, AvailableBytes: 10, RequiredBytes: 1}, nil
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
	code := Run(t.TempDir(), bytes.NewBufferString("\n"), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}

	output := out.Bytes()
	if !bytes.Contains(output, []byte("Processing progress: 1/500 (0%) media: A.jpg")) {
		t.Fatalf("expected first progress line, got:\n%s", out.String())
	}
	if bytes.Contains(output, []byte("Processing progress: 2/500 (0%) media: B.jpg")) {
		t.Fatalf("unexpected duplicate same-percent progress line, got:\n%s", out.String())
	}
	if !bytes.Contains(output, []byte("Processing progress: 5/500 (1%) media: C.jpg")) {
		t.Fatalf("expected percent change progress line, got:\n%s", out.String())
	}
	if !bytes.Contains(output, []byte("Processing progress: 500/500 (100%) media: Z.jpg")) {
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
		return preflight.SpaceCheck{Enough: true, AvailableBytes: 10, RequiredBytes: 1}, nil
	}
	loadState = func(string) (state.RunState, error) { return state.New(), nil }
	saveState = func(string, state.RunState) error { return nil }
	extractArchiveFile = func(string, string) (int, error) { return 1, nil }
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{}, nil
	}

	var out bytes.Buffer
	code := Run(t.TempDir(), bytes.NewBufferString("\n"), &out)
	if code != ExitSuccess {
		t.Fatalf("expected success, got %d\n%s", code, out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("Processing progress: 0/0 (0%)")) {
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
		return preflight.SpaceCheck{Enough: true, AvailableBytes: 1024, RequiredBytes: 256}, nil
	}

	loadState = func(string) (state.RunState, error) { return state.New(), nil }
	saveState = func(string, state.RunState) error { return nil }
	extractArchiveFile = func(string, string) (int, error) { return 1, nil }
	processTakeout = func(string, func(processor.ProgressEvent)) (processor.Report, error) {
		return processor.Report{}, nil
	}

	code := Run(t.TempDir(), bytes.NewBufferString("\n"), &bytes.Buffer{})
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

func stubWizardDeps() func() {
	origCheckDependencies := checkDependencies
	origInstallDependencies := installDependencies
	origDiscoverZips := discoverZips
	origValidateAll := validateAll
	origCheckDiskSpace := checkDiskSpace
	origLoadState := loadState
	origSaveState := saveState
	origShouldSkip := shouldSkip
	origExtractArchiveFile := extractArchiveFile
	origProcessTakeout := processTakeout
	origRemoveFile := removeFile

	return func() {
		checkDependencies = origCheckDependencies
		installDependencies = origInstallDependencies
		discoverZips = origDiscoverZips
		validateAll = origValidateAll
		checkDiskSpace = origCheckDiskSpace
		loadState = origLoadState
		saveState = origSaveState
		shouldSkip = origShouldSkip
		extractArchiveFile = origExtractArchiveFile
		processTakeout = origProcessTakeout
		removeFile = origRemoveFile
	}
}
