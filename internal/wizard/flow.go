package wizard

import (
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
	hasTakeoutContent  = preflight.HasProcessableTakeout
	loadState          = state.Load
	saveState          = state.Save
	shouldSkip         = state.ShouldSkipExtraction
	extractArchiveFile = extract.ExtractArchive
	processTakeout     = processor.RunWithProgress
	removeFile         = os.Remove
)

func Run(cwd string, out io.Writer) int {
	runStartedAt := time.Now()
	report := Report{Status: "FAILED"}
	finish := func(code int) int {
		report.TotalDuration = time.Since(runStartedAt)
		printReport(out, report)
		return code
	}
	writeLine(out, "TakeoutFix interactive mode")
	writef(out, "Working directory: %s\n", cwd)

	writeLine(out, "Checking dependencies...")
	missing := checkDependencies()
	if len(missing) > 0 {
		var names []string
		for _, dep := range missing {
			names = append(names, dep.Name)
		}
		writef(out, "Missing dependencies: %s\n", strings.Join(names, ", "))
		writeLine(out, "Install dependencies and rerun.")
		writeLine(out, "macOS/Linux: curl -fsSL "+installerURLMacLinux+" | sh")
		writeLine(out, "Windows (PowerShell): iwr -useb "+installerURLWindows+" | iex")
		writeLine(out, "Manual fallback: install exiftool and ensure it is available in PATH.")
		return finish(ExitPreflightFail)
	}
	writeLine(out, "Dependencies are OK.")

	dest := filepath.Join(cwd, "takeoutfix-extracted")
	statePath := filepath.Join(cwd, ".takeoutfix", "state.json")
	st := state.New()
	lowSpaceDelete := false
	deferredDelete := make([]preflight.ZipArchive, 0)

	writeLine(out, "Scanning ZIP archives in current folder...")
	zipScanStartedAt := time.Now()
	zips, err := discoverZips(cwd)
	report.ZipScanDuration = time.Since(zipScanStartedAt)
	if err != nil {
		report.addProblem("zip scan errors", 1, err.Error())
		return finish(ExitRuntimeFail)
	}
	report.ArchiveFound = len(zips)
	if len(zips) == 0 {
		info, err := os.Stat(dest)
		if err != nil {
			if os.IsNotExist(err) {
				writeLine(out, "No ZIP archives found. Detecting extracted Takeout content in working directory...")
				processable, detectErr := hasTakeoutContent(cwd)
				if detectErr != nil {
					report.addProblem("takeout content detection errors", 1, detectErr.Error())
					return finish(ExitRuntimeFail)
				}
				if !processable {
					writeLine(out, "No ZIP archives found and no extracted data.")
					return finish(ExitPreflightFail)
				}
				dest = cwd
				writeLine(out, "No ZIP archives found. Processing existing Takeout content from working directory...")
			} else {
				report.addProblem("extracted data access error", 1, err.Error())
				return finish(ExitRuntimeFail)
			}
		}
		if err == nil && !info.IsDir() {
			writeLine(out, "No ZIP archives found and extracted data path is not a directory.")
			return finish(ExitPreflightFail)
		}
		if err == nil {
			writeLine(out, "No ZIP archives found. Re-processing previously extracted data...")
		}
	}

	if len(zips) > 0 {
		writeLine(out, "Validating ZIP integrity (all archives)...")
		zipValidateStartedAt := time.Now()
		integrity := validateAll(zips)
		report.ZipValidateDuration = time.Since(zipValidateStartedAt)
		report.ArchiveCorrupt = len(integrity.Corrupt)
		report.ArchiveValid = len(integrity.Checked) - report.ArchiveCorrupt
		for _, corrupt := range integrity.Corrupt {
			report.CorruptNames = append(report.CorruptNames, corrupt.Archive.Name)
		}
		if report.ArchiveCorrupt > 0 {
			writeLine(out, "Corrupt ZIP files found. Processing stopped.")
			return finish(ExitPreflightFail)
		}
		writeLine(out, "All ZIP archives are valid.")

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
			writeLine(out, "Checking available disk space...")
			diskCheckStartedAt := time.Now()
			space, err := checkDiskSpace(cwd, pending)
			report.DiskCheckDuration = time.Since(diskCheckStartedAt)
			if err != nil {
				report.addProblem("disk check errors", 1, err.Error())
				return finish(ExitRuntimeFail)
			}
			report.Disk = space
			writef(
				out,
				"Disk: available=%s, required=%s, required with auto-delete=%s\n",
				preflight.FormatBytes(space.AvailableBytes),
				preflight.FormatBytes(space.RequiredBytes),
				preflight.FormatBytes(space.RequiredWithDeleteBytes),
			)

			if !space.EnoughWithDelete {
				writeLine(out, "Not enough disk space even with auto-delete enabled.")
				return finish(ExitPreflightFail)
			}
			if !space.Enough {
				lowSpaceDelete = true
				writeLine(out, "Low-space mode enabled: ZIP archives will be deleted immediately after extraction.")
			}
		}
		writef(out, "Extracting archives into: %s\n", dest)
		extractStartedAt := time.Now()
		totalArchives := len(zips)
		doneArchives := 0
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
				doneArchives++
				writef(out, "Extraction progress: %d/%d (%d%%) skipped: %s\n", doneArchives, totalArchives, progressPercent(doneArchives, totalArchives), archive.Name)
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
			doneArchives++
			writef(out, "Extraction progress: %d/%d (%d%%) extracted: %s\n", doneArchives, totalArchives, progressPercent(doneArchives, totalArchives), archive.Name)

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
	}

	writeLine(out, "Applying metadata and cleaning matched JSON...")
	processStartedAt := time.Now()
	lastProcessPercent := -1
	sawProcessEvent := false
	procReport, err := processTakeout(dest, func(event processor.ProgressEvent) {
		sawProcessEvent = true
		percent := progressPercent(event.Processed, event.Total)
		if event.Processed == 1 || event.Processed == event.Total || percent != lastProcessPercent {
			writef(out, "Processing progress: %d/%d (%d%%) media: %s\n", event.Processed, event.Total, percent, event.Media)
			lastProcessPercent = percent
		}
	})
	if err != nil {
		report.addProblem("processing errors", 1, err.Error())
		report.ProcessDuration = time.Since(processStartedAt)
		return finish(ExitRuntimeFail)
	}
	report.ProcessDuration = time.Since(processStartedAt)
	if !sawProcessEvent {
		writeLine(out, "Processing progress: 0/0 (0%)")
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
			writeLine(out, "Hard processing errors detected. Keeping ZIP archives for rerun.")
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
