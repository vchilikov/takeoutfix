package wizard

import (
	"bufio"
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
)

var (
	checkDependencies   = preflight.CheckDependencies
	installDependencies = preflight.InstallDependencies
	discoverZips        = preflight.DiscoverTopLevelZips
	validateAll         = preflight.ValidateAll
	checkDiskSpace      = preflight.CheckDiskSpace
	loadState           = state.Load
	saveState           = state.Save
	shouldSkip          = state.ShouldSkipExtraction
	extractArchiveFile  = extract.ExtractArchive
	processTakeout      = processor.RunWithProgress
	removeFile          = os.Remove
)

func Run(cwd string, in io.Reader, out io.Writer) int {
	runStartedAt := time.Now()

	reader := bufio.NewReader(in)
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
		if !canAutoInstall(missing) {
			writeLine(out, "Automatic install is supported on macOS (Homebrew), Linux (apt/dnf/pacman), and Windows (winget). Please install manually and rerun.")
			return finish(ExitPreflightFail)
		}

		if !askYesNo(reader, out, "Install missing dependencies now? [y/N]: ") {
			return finish(ExitPreflightFail)
		}

		for _, dep := range missing {
			writef(out, "Running: %s\n", strings.Join(dep.InstallCmd, " "))
		}
		if err := installDependencies(missing, reader, out); err != nil {
			writef(out, "Dependency install failed: %v\n", err)
			report.addProblem("dependency install errors", 1, err.Error())
			return finish(ExitPreflightFail)
		}

		if stillMissing := checkDependencies(); len(stillMissing) > 0 {
			writef(out, "Missing dependencies: %s\n", dependencyNames(stillMissing))
			return finish(ExitPreflightFail)
		}
	}
	writeLine(out, "Dependencies are OK.")

	dest := filepath.Join(cwd, "takeoutfix-extracted")

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
				writeLine(out, "No ZIP archives found and no extracted data.")
				return finish(ExitPreflightFail)
			}
			report.addProblem("extracted data access error", 1, err.Error())
			return finish(ExitRuntimeFail)
		}
		if !info.IsDir() {
			writeLine(out, "No ZIP archives found and extracted data path is not a directory.")
			return finish(ExitPreflightFail)
		}
		writeLine(out, "No ZIP archives found. Re-processing previously extracted data...")
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

		statePath := filepath.Join(cwd, ".takeoutfix", "state.json")
		st, err := loadState(statePath)
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

		deleteMode := false
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
				"Disk: available=%s, required=%s, required with delete-mode=%s\n",
				preflight.FormatBytes(space.AvailableBytes),
				preflight.FormatBytes(space.RequiredBytes),
				preflight.FormatBytes(space.RequiredWithDeleteBytes),
			)

			if !space.Enough {
				writeLine(out, "Not enough disk space for normal mode.")
				if !askYesNo(reader, out, "Enable delete-mode (remove ZIP right after successful extraction)? [y/N]: ") {
					return finish(ExitPreflightFail)
				}
				if !space.EnoughWithDelete {
					writeLine(out, "Still not enough space even with delete-mode.")
					return finish(ExitPreflightFail)
				}
				deleteMode = true
			}
		}
		report.DeleteMode = deleteMode

		writef(out, "Extracting archives into: %s\n", dest)
		extractStartedAt := time.Now()
		totalArchives := len(zips)
		doneArchives := 0
		for _, archive := range zips {
			if shouldSkip(st, archive.Name, archive.Fingerprint) {
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
			if deleteMode {
				if err := removeFile(archive.Path); err != nil {
					report.DeleteErrors = append(report.DeleteErrors, archive.Name)
				} else {
					report.DeletedZips++
					entry.Deleted = true
				}
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

	report.Status = "SUCCESS"
	return finish(ExitSuccess)
}

func askYesNo(reader *bufio.Reader, out io.Writer, prompt string) bool {
	writef(out, "%s", prompt)
	line, _ := reader.ReadString('\n')
	value := strings.TrimSpace(strings.ToLower(line))
	switch value {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func canAutoInstall(missing []preflight.Dependency) bool {
	for _, dep := range missing {
		if len(dep.InstallCmd) == 0 {
			return false
		}
	}
	return true
}

func dependencyNames(missing []preflight.Dependency) string {
	names := make([]string, 0, len(missing))
	for _, dep := range missing {
		names = append(names, dep.Name)
	}
	return strings.Join(names, ", ")
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
