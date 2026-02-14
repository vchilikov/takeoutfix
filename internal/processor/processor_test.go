package processor

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vchilikov/takeout-fix/utils/extensions"
	"github.com/vchilikov/takeout-fix/utils/files"
	"github.com/vchilikov/takeout-fix/utils/metadata"
)

func TestRunWithProgress_AggregatesResultsAndContinuesOnFileErrors(t *testing.T) {
	restore := stubProcessorDeps()
	defer restore()

	root := t.TempDir()

	scanTakeout = func(string) (files.MediaScanResult, error) {
		return files.MediaScanResult{
			Pairs: map[string]string{
				"a.jpg": "a.json",
				"b.jpg": "b.json",
				"c.jpg": "c.json",
				"d.jpg": "d.json",
			},
			MissingJSON: []string{"missing.jpg"},
			AmbiguousJSON: map[string][]string{
				"ambiguous.jpg": {"ambiguous.json"},
			},
			UnusedJSON: []string{"unused-1.json", "unused-2.json"},
		}, nil
	}

	fixMediaExtension = func(mediaPath string) (extensions.FixResult, error) {
		switch filepath.Base(mediaPath) {
		case "a.jpg":
			return extensions.FixResult{Path: mediaPath}, nil
		case "b.jpg":
			return extensions.FixResult{Path: mediaPath}, errors.New("fix failed")
		case "c.jpg":
			return extensions.FixResult{Path: filepath.Join(filepath.Dir(mediaPath), "c-renamed.jpg"), Renamed: true}, nil
		case "d.jpg":
			return extensions.FixResult{Path: mediaPath}, nil
		default:
			t.Fatalf("unexpected media in fix: %s", mediaPath)
			return extensions.FixResult{}, nil
		}
	}

	applyMediaMetadata = func(mediaPath string, jsonPath string) (metadata.ApplyResult, error) {
		switch filepath.Base(mediaPath) {
		case "a.jpg":
			return metadata.ApplyResult{UsedXMPSidecar: true}, nil
		case "c-renamed.jpg":
			return metadata.ApplyResult{}, errors.New("metadata failed")
		case "d.jpg":
			return metadata.ApplyResult{CreateDateWarned: true, UsedFilenameDate: true}, nil
		default:
			t.Fatalf("unexpected media in metadata apply: %s (%s)", mediaPath, jsonPath)
			return metadata.ApplyResult{}, nil
		}
	}

	removed := make(map[string]int)
	removeJSONFile = func(path string) error {
		removed[filepath.Base(path)]++
		if filepath.Base(path) == "d.json" {
			return errors.New("remove failed")
		}
		return nil
	}

	var progress []ProgressEvent
	report, err := RunWithProgress(root, func(event ProgressEvent) {
		progress = append(progress, event)
	})
	if err != nil {
		t.Fatalf("RunWithProgress returned error: %v", err)
	}

	if report.Summary.MediaFound != 6 {
		t.Fatalf("MediaFound: want 6, got %d", report.Summary.MediaFound)
	}
	if report.Summary.MetadataApplied != 2 {
		t.Fatalf("MetadataApplied: want 2, got %d", report.Summary.MetadataApplied)
	}
	if report.Summary.FilenameDateApplied != 1 {
		t.Fatalf("FilenameDateApplied: want 1, got %d", report.Summary.FilenameDateApplied)
	}
	if report.Summary.RenamedExtensions != 1 {
		t.Fatalf("RenamedExtensions: want 1, got %d", report.Summary.RenamedExtensions)
	}
	if report.Summary.XMPSidecars != 1 {
		t.Fatalf("XMPSidecars: want 1, got %d", report.Summary.XMPSidecars)
	}
	if report.Summary.CreateDateWarnings != 1 {
		t.Fatalf("CreateDateWarnings: want 1, got %d", report.Summary.CreateDateWarnings)
	}
	if report.Summary.MissingJSON != 1 {
		t.Fatalf("MissingJSON: want 1, got %d", report.Summary.MissingJSON)
	}
	if report.Summary.AmbiguousMedia != 1 {
		t.Fatalf("AmbiguousMedia: want 1, got %d", report.Summary.AmbiguousMedia)
	}
	if report.Summary.UnusedJSON != 2 {
		t.Fatalf("UnusedJSON: want 2, got %d", report.Summary.UnusedJSON)
	}
	if report.Summary.JSONRemoved != 1 {
		t.Fatalf("JSONRemoved: want 1, got %d", report.Summary.JSONRemoved)
	}
	if report.Summary.JSONKeptDueToErrors != 2 {
		t.Fatalf("JSONKeptDueToErrors: want 2, got %d", report.Summary.JSONKeptDueToErrors)
	}

	if got := report.ProblemCounts["extension errors"]; got != 1 {
		t.Fatalf("extension errors: want 1, got %d", got)
	}
	if got := report.ProblemCounts["metadata errors"]; got != 1 {
		t.Fatalf("metadata errors: want 1, got %d", got)
	}
	if got := report.ProblemCounts["create date warnings"]; got != 1 {
		t.Fatalf("create date warnings: want 1, got %d", got)
	}
	if got := report.ProblemCounts["json remove errors"]; got != 1 {
		t.Fatalf("json remove errors: want 1, got %d", got)
	}

	if len(progress) != 4 {
		t.Fatalf("expected 4 progress events, got %d", len(progress))
	}

	gotProcessed := make([]int, 0, len(progress))
	for _, event := range progress {
		if event.Total != 4 {
			t.Fatalf("progress total: want 4, got %d", event.Total)
		}
		gotProcessed = append(gotProcessed, event.Processed)
	}
	slices.Sort(gotProcessed)
	for i, got := range gotProcessed {
		if got != i+1 {
			t.Fatalf("progress processed values mismatch: got %v", gotProcessed)
		}
	}

	if removed["a.json"] != 1 || removed["d.json"] != 1 {
		t.Fatalf("unexpected json remove calls: %v", removed)
	}
	if removed["b.json"] != 0 || removed["c.json"] != 0 {
		t.Fatalf("json with processing errors must be kept, got remove calls: %v", removed)
	}
}

func TestRunWithProgress_FilenameDateWarningIsNonFatal(t *testing.T) {
	restore := stubProcessorDeps()
	defer restore()

	root := t.TempDir()

	scanTakeout = func(string) (files.MediaScanResult, error) {
		return files.MediaScanResult{
			Pairs: map[string]string{
				"a.jpg": "a.json",
			},
		}, nil
	}

	fixMediaExtension = func(mediaPath string) (extensions.FixResult, error) {
		return extensions.FixResult{Path: mediaPath}, nil
	}

	applyMediaMetadata = func(mediaPath string, jsonPath string) (metadata.ApplyResult, error) {
		if filepath.Base(mediaPath) != "a.jpg" || filepath.Base(jsonPath) != "a.json" {
			t.Fatalf("unexpected metadata input: media=%s json=%s", mediaPath, jsonPath)
		}
		return metadata.ApplyResult{FilenameDateWarned: true}, nil
	}

	removed := make(map[string]int)
	removeJSONFile = func(path string) error {
		removed[filepath.Base(path)]++
		return nil
	}

	report, err := RunWithProgress(root, nil)
	if err != nil {
		t.Fatalf("RunWithProgress returned error: %v", err)
	}

	if report.Summary.MetadataApplied != 1 {
		t.Fatalf("MetadataApplied: want 1, got %d", report.Summary.MetadataApplied)
	}
	if report.Summary.JSONRemoved != 1 {
		t.Fatalf("JSONRemoved: want 1, got %d", report.Summary.JSONRemoved)
	}
	if report.Summary.JSONKeptDueToErrors != 0 {
		t.Fatalf("JSONKeptDueToErrors: want 0, got %d", report.Summary.JSONKeptDueToErrors)
	}
	if got := report.ProblemCounts["filename date warnings"]; got != 1 {
		t.Fatalf("filename date warnings: want 1, got %d", got)
	}
	if got := report.ProblemCounts["metadata errors"]; got != 0 {
		t.Fatalf("metadata errors: want 0, got %d", got)
	}
	if removed["a.json"] != 1 {
		t.Fatalf("expected a.json remove call once, got remove calls: %v", removed)
	}
}

func TestRunWithProgress_MediaFileDateWarningIsNonFatal(t *testing.T) {
	restore := stubProcessorDeps()
	defer restore()

	root := t.TempDir()

	scanTakeout = func(string) (files.MediaScanResult, error) {
		return files.MediaScanResult{
			Pairs: map[string]string{
				"a.avi": "a.json",
			},
		}, nil
	}

	fixMediaExtension = func(mediaPath string) (extensions.FixResult, error) {
		return extensions.FixResult{Path: mediaPath}, nil
	}

	applyMediaMetadata = func(mediaPath string, jsonPath string) (metadata.ApplyResult, error) {
		if filepath.Base(mediaPath) != "a.avi" || filepath.Base(jsonPath) != "a.json" {
			t.Fatalf("unexpected metadata input: media=%s json=%s", mediaPath, jsonPath)
		}
		return metadata.ApplyResult{
			UsedXMPSidecar:      true,
			MediaFileDateWarned: true,
		}, nil
	}

	removed := make(map[string]int)
	removeJSONFile = func(path string) error {
		removed[filepath.Base(path)]++
		return nil
	}

	report, err := RunWithProgress(root, nil)
	if err != nil {
		t.Fatalf("RunWithProgress returned error: %v", err)
	}

	if report.Summary.MetadataApplied != 1 {
		t.Fatalf("MetadataApplied: want 1, got %d", report.Summary.MetadataApplied)
	}
	if report.Summary.XMPSidecars != 1 {
		t.Fatalf("XMPSidecars: want 1, got %d", report.Summary.XMPSidecars)
	}
	if report.Summary.JSONRemoved != 1 {
		t.Fatalf("JSONRemoved: want 1, got %d", report.Summary.JSONRemoved)
	}
	if got := report.ProblemCounts["media file date warnings"]; got != 1 {
		t.Fatalf("media file date warnings: want 1, got %d", got)
	}
	if got := report.ProblemCounts["metadata errors"]; got != 0 {
		t.Fatalf("metadata errors: want 0, got %d", got)
	}
	if removed["a.json"] != 1 {
		t.Fatalf("expected a.json remove call once, got remove calls: %v", removed)
	}
}

func TestRunWithProgress_ReturnsScanError(t *testing.T) {
	restore := stubProcessorDeps()
	defer restore()

	scanTakeout = func(string) (files.MediaScanResult, error) {
		return files.MediaScanResult{}, errors.New("scan failed")
	}

	_, err := RunWithProgress(t.TempDir(), nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunFixWithFallback_ClosesBrokenSessionAndFallsBack(t *testing.T) {
	restore := stubProcessorDeps()
	defer restore()

	fakeSession := &fakeExiftoolSession{}
	session := exiftoolSession(fakeSession)

	fixMediaExtensionWithRunner = func(string, func([]string) (string, error)) (extensions.FixResult, error) {
		return extensions.FixResult{}, errors.New("session path failed")
	}

	oneshotCalls := 0
	fixMediaExtension = func(mediaPath string) (extensions.FixResult, error) {
		oneshotCalls++
		return extensions.FixResult{Path: mediaPath}, nil
	}

	result, err := runFixWithFallback("/tmp/a.jpg", &session)
	if err != nil {
		t.Fatalf("runFixWithFallback returned error: %v", err)
	}
	if result.Path != "/tmp/a.jpg" {
		t.Fatalf("unexpected result path: %s", result.Path)
	}
	if oneshotCalls != 1 {
		t.Fatalf("expected one fallback oneshot call, got %d", oneshotCalls)
	}
	if session != nil {
		t.Fatalf("expected session to be reset after runner failure")
	}
	if fakeSession.closeCalls != 1 {
		t.Fatalf("expected session close once, got %d", fakeSession.closeCalls)
	}
}

func TestRunMetadataWithFallback_ClosesBrokenSessionAndFallsBack(t *testing.T) {
	restore := stubProcessorDeps()
	defer restore()

	fakeSession := &fakeExiftoolSession{}
	session := exiftoolSession(fakeSession)

	applyMediaMetadataWithRunner = func(string, string, func([]string) (string, error)) (metadata.ApplyResult, error) {
		return metadata.ApplyResult{}, errors.New("session path failed")
	}

	oneshotCalls := 0
	applyMediaMetadata = func(string, string) (metadata.ApplyResult, error) {
		oneshotCalls++
		return metadata.ApplyResult{UsedXMPSidecar: true}, nil
	}

	result, err := runMetadataWithFallback("/tmp/a.jpg", "/tmp/a.json", &session)
	if err != nil {
		t.Fatalf("runMetadataWithFallback returned error: %v", err)
	}
	if !result.UsedXMPSidecar {
		t.Fatalf("expected fallback oneshot result")
	}
	if oneshotCalls != 1 {
		t.Fatalf("expected one fallback oneshot call, got %d", oneshotCalls)
	}
	if session != nil {
		t.Fatalf("expected session to be reset after runner failure")
	}
	if fakeSession.closeCalls != 1 {
		t.Fatalf("expected session close once, got %d", fakeSession.closeCalls)
	}
}

func stubProcessorDeps() func() {
	origScanTakeout := scanTakeout
	origFixMediaExtension := fixMediaExtension
	origFixMediaExtensionWithRunner := fixMediaExtensionWithRunner
	origApplyMediaMetadata := applyMediaMetadata
	origApplyMediaMetadataWithRunner := applyMediaMetadataWithRunner
	origOpenExiftoolSession := openExiftoolSession
	origRemoveJSONFile := removeJSONFile

	openExiftoolSession = func() (exiftoolSession, error) {
		return nil, errors.New("disabled in tests")
	}

	return func() {
		scanTakeout = origScanTakeout
		fixMediaExtension = origFixMediaExtension
		fixMediaExtensionWithRunner = origFixMediaExtensionWithRunner
		applyMediaMetadata = origApplyMediaMetadata
		applyMediaMetadataWithRunner = origApplyMediaMetadataWithRunner
		openExiftoolSession = origOpenExiftoolSession
		removeJSONFile = origRemoveJSONFile
	}
}

type fakeExiftoolSession struct {
	closeCalls int
}

func (f *fakeExiftoolSession) Run([]string) (string, error) {
	return "", nil
}

func (f *fakeExiftoolSession) Close() error {
	f.closeCalls++
	return nil
}
