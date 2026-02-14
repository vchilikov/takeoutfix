package wizard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type jsonProblem struct {
	Category string   `json:"category"`
	Count    int      `json:"count"`
	Samples  []string `json:"samples,omitempty"`
}

type jsonReport struct {
	Status          string          `json:"status"`
	ExitCode        int             `json:"exit_code"`
	Workdir         string          `json:"workdir"`
	StartedAtLocal  string          `json:"started_at_local"`
	FinishedAtLocal string          `json:"finished_at_local"`
	DurationMS      int64           `json:"duration_ms"`
	Archives        jsonArchives    `json:"archives"`
	Disk            jsonDisk        `json:"disk"`
	Extraction      jsonExtraction  `json:"extraction"`
	Metadata        jsonMetadata    `json:"metadata"`
	JSONCleanup     jsonJSONCleanup `json:"json_cleanup"`
	TimingsMS       jsonTimingsMS   `json:"timings_ms"`
	Problems        []jsonProblem   `json:"problems,omitempty"`
}

type jsonArchives struct {
	Found        int      `json:"found"`
	Valid        int      `json:"valid"`
	Corrupt      int      `json:"corrupt"`
	CorruptNames []string `json:"corrupt_names,omitempty"`
}

type jsonDisk struct {
	AvailableBytes          uint64 `json:"available_bytes"`
	RequiredBytes           uint64 `json:"required_bytes"`
	RequiredWithDeleteBytes uint64 `json:"required_with_delete_bytes"`
	Enough                  bool   `json:"enough"`
	EnoughWithDelete        bool   `json:"enough_with_delete"`
	AutoDelete              bool   `json:"auto_delete"`
}

type jsonExtraction struct {
	ExtractedArchives int      `json:"extracted_archives"`
	SkippedArchives   int      `json:"skipped_archives"`
	ExtractedFiles    int      `json:"extracted_files"`
	DeletedZips       int      `json:"deleted_zips"`
	DeleteErrors      []string `json:"delete_errors,omitempty"`
}

type jsonMetadata struct {
	MediaFound          int `json:"media_found"`
	MetadataApplied     int `json:"metadata_applied"`
	FilenameDateApplied int `json:"filename_date_applied"`
	RenamedExtensions   int `json:"renamed_extensions"`
	XMPSidecars         int `json:"xmp_sidecars"`
	MissingJSON         int `json:"missing_json"`
	AmbiguousMedia      int `json:"ambiguous_media"`
}

type jsonJSONCleanup struct {
	Removed         int `json:"removed"`
	KeptDueToErrors int `json:"kept_due_to_errors"`
	KeptUnused      int `json:"kept_unused"`
}

type jsonTimingsMS struct {
	ZipScan     int64 `json:"zip_scan"`
	ZipValidate int64 `json:"zip_validate"`
	DiskCheck   int64 `json:"disk_check"`
	Extract     int64 `json:"extract"`
	Process     int64 `json:"process"`
	Total       int64 `json:"total"`
}

func writeReportJSONImpl(report Report) (string, error) {
	reportDir := filepath.Join(report.Workdir, ".takeoutfix", "reports")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return "", fmt.Errorf("create report directory: %w", err)
	}

	reportTime := report.FinishedAtLocal
	if reportTime.IsZero() {
		reportTime = time.Now()
	}
	fileName := fmt.Sprintf("report-%s.json", reportTime.Format("20060102-150405"))
	reportPath := filepath.Join(reportDir, fileName)
	absReportPath, absErr := filepath.Abs(reportPath)
	if absErr == nil {
		reportPath = absReportPath
	}

	payload := buildJSONReport(report)
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return reportPath, fmt.Errorf("marshal report json: %w", err)
	}
	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		return reportPath, fmt.Errorf("write report json: %w", err)
	}

	return reportPath, nil
}

func buildJSONReport(report Report) jsonReport {
	problems := make([]jsonProblem, 0, len(report.ProblemCounts))
	if len(report.ProblemCounts) > 0 {
		categories := make([]string, 0, len(report.ProblemCounts))
		for category := range report.ProblemCounts {
			categories = append(categories, category)
		}
		slices.Sort(categories)
		for _, category := range categories {
			problems = append(problems, jsonProblem{
				Category: category,
				Count:    report.ProblemCounts[category],
				Samples:  append([]string(nil), report.ProblemSample[category]...),
			})
		}
	}

	return jsonReport{
		Status:          report.Status,
		ExitCode:        report.ExitCode,
		Workdir:         report.Workdir,
		StartedAtLocal:  report.StartedAtLocal.Format(time.RFC3339Nano),
		FinishedAtLocal: report.FinishedAtLocal.Format(time.RFC3339Nano),
		DurationMS:      report.TotalDuration.Milliseconds(),
		Archives: jsonArchives{
			Found:        report.ArchiveFound,
			Valid:        report.ArchiveValid,
			Corrupt:      report.ArchiveCorrupt,
			CorruptNames: append([]string(nil), report.CorruptNames...),
		},
		Disk: jsonDisk{
			AvailableBytes:          report.Disk.AvailableBytes,
			RequiredBytes:           report.Disk.RequiredBytes,
			RequiredWithDeleteBytes: report.Disk.RequiredWithDeleteBytes,
			Enough:                  report.Disk.Enough,
			EnoughWithDelete:        report.Disk.EnoughWithDelete,
			AutoDelete:              report.AutoDelete,
		},
		Extraction: jsonExtraction{
			ExtractedArchives: report.ExtractedArchives,
			SkippedArchives:   report.SkippedArchives,
			ExtractedFiles:    report.ExtractedFiles,
			DeletedZips:       report.DeletedZips,
			DeleteErrors:      append([]string(nil), report.DeleteErrors...),
		},
		Metadata: jsonMetadata{
			MediaFound:          report.MediaFound,
			MetadataApplied:     report.MetadataApplied,
			FilenameDateApplied: report.FilenameDateApplied,
			RenamedExtensions:   report.RenamedExtensions,
			XMPSidecars:         report.XMPSidecars,
			MissingJSON:         report.MissingJSON,
			AmbiguousMedia:      report.AmbiguousMedia,
		},
		JSONCleanup: jsonJSONCleanup{
			Removed:         report.JSONRemoved,
			KeptDueToErrors: report.JSONKeptDueToErrors,
			KeptUnused:      report.UnusedJSON,
		},
		TimingsMS: jsonTimingsMS{
			ZipScan:     report.ZipScanDuration.Milliseconds(),
			ZipValidate: report.ZipValidateDuration.Milliseconds(),
			DiskCheck:   report.DiskCheckDuration.Milliseconds(),
			Extract:     report.ExtractDuration.Milliseconds(),
			Process:     report.ProcessDuration.Milliseconds(),
			Total:       report.TotalDuration.Milliseconds(),
		},
		Problems: problems,
	}
}
