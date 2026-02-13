package wizard

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vchilikov/takeout-fix/internal/extract"
	"github.com/vchilikov/takeout-fix/internal/preflight"
	"github.com/vchilikov/takeout-fix/internal/processor"
	"github.com/vchilikov/takeout-fix/internal/state"
)

const (
	ExitSuccess       = 0
	ExitPreflightFail = 2
	ExitRuntimeFail   = 3

	installerURLMacLinux = "https://github.com/vchilikov/takeout-fix/releases/latest/download/install.sh"
	installerURLWindows  = "https://github.com/vchilikov/takeout-fix/releases/latest/download/install.ps1"
)

var (
	checkDependencies  = preflight.CheckDependencies
	discoverZips       = preflight.DiscoverTopLevelZips
	validateAll        = preflight.ValidateAll
	checkDiskSpace     = preflight.CheckDiskSpace
	detectTakeoutRoot  = preflight.DetectProcessableTakeoutRoot
	loadState          = state.Load
	saveState          = state.Save
	shouldSkip         = state.ShouldSkipExtraction
	extractArchiveFile = extract.ExtractArchive
	processTakeout     = processor.RunWithProgress
	removeFile         = os.Remove
	writeReportJSON    = writeReportJSONImpl
)

func Run(cwd string, out io.Writer) int {
	absCwd := cwd
	if resolved, err := filepath.Abs(cwd); err == nil {
		absCwd = resolved
	}

	runStartedAt := time.Now()
	report := Report{
		Status:         "FAILED",
		Workdir:        absCwd,
		StartedAtLocal: runStartedAt,
	}
	finish := func(code int) int {
		finishedAt := time.Now()
		report.ExitCode = code
		report.FinishedAtLocal = finishedAt
		report.TotalDuration = finishedAt.Sub(runStartedAt)
		report.normalizeProblems()

		reportPath, err := writeReportJSON(report)
		if err != nil {
			errText := err.Error()
			if reportPath != "" {
				errText = fmt.Sprintf("%s (target path: %s)", errText, reportPath)
			}
			report.DetailedReportWriteError = errText
			report.addProblem("report write errors", 1, errText)
		} else {
			report.DetailedReportPath = reportPath
		}

		printReport(out, report)
		return code
	}
	writeLine(out, "TakeoutFix")
	writef(out, "Folder: %s\n", report.Workdir)
	writeLine(out, "")

	writef(out, "Step 1/3: Checking dependencies... ")
	missing := checkDependencies()
	if len(missing) > 0 {
		writeLine(out, "missing")
		var names []string
		for _, dep := range missing {
			names = append(names, dep.Name)
		}
		writef(out, "Please install: %s\n", strings.Join(names, ", "))
		writeLine(out, "Quick install (macOS/Linux): curl -fsSL "+installerURLMacLinux+" | sh")
		writeLine(out, "Quick install (Windows PowerShell): iwr -useb "+installerURLWindows+" | iex")
		return finish(ExitPreflightFail)
	}
	writeLine(out, "OK")

	dest := filepath.Join(report.Workdir, "takeoutfix-extracted")
	statePath := filepath.Join(report.Workdir, ".takeoutfix", "state.json")
	st := state.New()
	lowSpaceDelete := false
	deferredDelete := make([]preflight.ZipArchive, 0)

	writef(out, "Step 2/3: Looking for ZIP files... ")
	zipScanStartedAt := time.Now()
	zips, err := discoverZips(report.Workdir)
	report.ZipScanDuration = time.Since(zipScanStartedAt)
	if err != nil {
		writeLine(out, "failed")
		report.addProblem("zip scan errors", 1, err.Error())
		return finish(ExitRuntimeFail)
	}
	report.ArchiveFound = len(zips)
	if len(zips) == 0 {
		writeLine(out, "none found")
		processRoot, message, preflightFail, err := resolveNoZipProcessRoot(report.Workdir, dest)
		if err != nil {
			report.addProblem("takeout content detection errors", 1, err.Error())
			return finish(ExitRuntimeFail)
		}
		writeLine(out, message)
		if preflightFail {
			return finish(ExitPreflightFail)
		}
		dest = processRoot
	} else {
		writef(out, "%d found\n", len(zips))
	}

	if len(zips) > 0 {
		writeLine(out, "Checking ZIP integrity...")
		zipValidateStartedAt := time.Now()
		integrity := validateAll(zips)
		report.ZipValidateDuration = time.Since(zipValidateStartedAt)
		report.ArchiveCorrupt = len(integrity.Corrupt)
		report.ArchiveValid = len(integrity.Checked) - report.ArchiveCorrupt
		for _, corrupt := range integrity.Corrupt {
			report.CorruptNames = append(report.CorruptNames, corrupt.Archive.Name)
		}
		if report.ArchiveCorrupt > 0 {
			writeLine(out, "Some ZIP files are corrupted. Please re-download them and run again.")
			return finish(ExitPreflightFail)
		}
		writeLine(out, "ZIP files look good.")

		st, err = loadState(statePath)
		if err != nil {
			report.addProblem("state load errors", 1, err.Error())
			st = state.New()
		}

		var pending []preflight.ArchiveIntegrity
		for _, ai := range integrity.Checked {
			if !shouldSkip(st, ai.Archive.Name, ai.Archive.Fingerprint) {
				pending = append(pending, ai)
			}
		}

		report.AutoDelete = true
		if len(pending) > 0 {
			writeLine(out, "Checking free disk space...")
			diskCheckStartedAt := time.Now()
			space, err := checkDiskSpace(report.Workdir, pending)
			report.DiskCheckDuration = time.Since(diskCheckStartedAt)
			if err != nil {
				report.addProblem("disk check errors", 1, err.Error())
				return finish(ExitRuntimeFail)
			}
			report.Disk = space

			if !space.EnoughWithDelete {
				writeLine(out, "Not enough free disk space to continue.")
				return finish(ExitPreflightFail)
			}
			if !space.Enough {
				lowSpaceDelete = true
				writeLine(out, "Low-space mode: ZIP files will be deleted right after extraction.")
			}
		}
		writeLine(out, "Preparing files from ZIP archives...")
		extractStartedAt := time.Now()
		for _, archive := range zips {
			if shouldSkip(st, archive.Name, archive.Fingerprint) {
				entry := st.Archives[archive.Name]
				entry.Fingerprint = archive.Fingerprint
				entry.Extracted = true
				if lowSpaceDelete {
					if !entry.Deleted {
						entry.Deleted = deleteArchiveZip(&report, archive)
					}
				} else {
					if !entry.Deleted {
						deferredDelete = append(deferredDelete, archive)
					}
				}
				st.Archives[archive.Name] = entry
				if err := saveState(statePath, st); err != nil {
					report.addProblem("state save errors", 1, err.Error())
				}

				report.SkippedArchives++
				continue
			}

			filesExtracted, err := extractArchiveFile(archive.Path, dest)
			if err != nil {
				report.addProblem("extract errors", 1, archive.Name)
				report.ExtractDuration = time.Since(extractStartedAt)
				return finish(ExitRuntimeFail)
			}

			report.ExtractedArchives++
			report.ExtractedFiles += filesExtracted

			entry := state.ArchiveState{Fingerprint: archive.Fingerprint, Extracted: true}
			if lowSpaceDelete {
				entry.Deleted = deleteArchiveZip(&report, archive)
			} else {
				entry.Deleted = false
				deferredDelete = append(deferredDelete, archive)
			}
			st.Archives[archive.Name] = entry

			if err := saveState(statePath, st); err != nil {
				report.addProblem("state save errors", 1, err.Error())
			}
		}
		report.ExtractDuration = time.Since(extractStartedAt)
		writeLine(out, "Preparing files from ZIP archives... done")
	}

	writeLine(out, "Step 3/3: Applying metadata and cleaning JSON...")
	writeLine(out, "Progress: 0%")
	processStartedAt := time.Now()
	lastProcessBucket := 0
	sawProcessEvent := false
	procReport, err := processTakeout(dest, func(event processor.ProgressEvent) {
		sawProcessEvent = true
		bucket := progressBucket10(event.Processed, event.Total)
		if bucket > lastProcessBucket {
			writef(out, "Progress: %d%%\n", bucket)
			lastProcessBucket = bucket
		}
	})
	if err != nil {
		report.addProblem("processing errors", 1, err.Error())
		report.ProcessDuration = time.Since(processStartedAt)
		return finish(ExitRuntimeFail)
	}
	report.ProcessDuration = time.Since(processStartedAt)
	if sawProcessEvent && lastProcessBucket < 100 {
		writeLine(out, "Progress: 100%")
	}

	report.MediaFound = procReport.Summary.MediaFound
	report.MetadataApplied = procReport.Summary.MetadataApplied
	report.FilenameDateApplied = procReport.Summary.FilenameDateApplied
	report.RenamedExtensions = procReport.Summary.RenamedExtensions
	report.XMPSidecars = procReport.Summary.XMPSidecars
	report.CreateDateWarnings = procReport.Summary.CreateDateWarnings
	report.MissingJSON = procReport.Summary.MissingJSON
	report.AmbiguousMedia = procReport.Summary.AmbiguousMedia
	report.UnusedJSON = procReport.Summary.UnusedJSON
	report.JSONRemoved = procReport.Summary.JSONRemoved
	report.JSONKeptDueToErrors = procReport.Summary.JSONKeptDueToErrors

	for category, count := range procReport.ProblemCounts {
		report.addProblem(category, count, procReport.ProblemSamples[category]...)
	}

	if hasHardProcessingProblems(procReport.ProblemCounts) {
		report.Status = "PARTIAL_SUCCESS"
		if !lowSpaceDelete {
			writeLine(out, "Some files could not be updated. ZIP files were kept for a rerun.")
		}
		return finish(ExitRuntimeFail)
	}

	if !lowSpaceDelete {
		for _, archive := range deferredDelete {
			entry := st.Archives[archive.Name]
			entry.Fingerprint = archive.Fingerprint
			entry.Extracted = true
			if !entry.Deleted {
				entry.Deleted = deleteArchiveZip(&report, archive)
			}
			st.Archives[archive.Name] = entry
			if err := saveState(statePath, st); err != nil {
				report.addProblem("state save errors", 1, err.Error())
			}
		}
	}

	report.Status = "SUCCESS"
	return finish(ExitSuccess)
}

func resolveNoZipProcessRoot(cwd string, extractedRoot string) (string, string, bool, error) {
	info, err := os.Stat(extractedRoot)
	if err == nil {
		if !info.IsDir() {
			return "", "Found takeoutfix-extracted, but it is not a folder.", true, nil
		}
		return extractedRoot, fmt.Sprintf("Using previously extracted Takeout data from: %s", extractedRoot), false, nil
	}

	if !os.IsNotExist(err) {
		return "", "", false, fmt.Errorf("access extracted data path %q: %w", extractedRoot, err)
	}

	processRoot, processable, detectErr := detectTakeoutRoot(cwd)
	if detectErr != nil {
		return "", "", false, fmt.Errorf("detect extracted takeout content: %w", detectErr)
	}
	if !processable {
		return "", "No ZIP files or extracted Takeout data found in this folder.", true, nil
	}

	return processRoot, fmt.Sprintf("Using existing Takeout content from: %s", processRoot), false, nil
}

func progressPercent(done int, total int) int {
	if total <= 0 {
		return 0
	}
	if done < 0 {
		done = 0
	}
	if done > total {
		done = total
	}
	return done * 100 / total
}

func progressBucket10(done int, total int) int {
	percent := progressPercent(done, total)
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return (percent / 10) * 10
}

func hasHardProcessingProblems(problemCounts map[string]int) bool {
	if len(problemCounts) == 0 {
		return false
	}
	return problemCounts["extension errors"] > 0 || problemCounts["metadata errors"] > 0
}

func deleteArchiveZip(report *Report, archive preflight.ZipArchive) bool {
	if err := removeFile(archive.Path); err != nil {
		if os.IsNotExist(err) {
			return true
		}
		report.DeleteErrors = append(report.DeleteErrors, archive.Name)
		return false
	}
	report.DeletedZips++
	return true
}
