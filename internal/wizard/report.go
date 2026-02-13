package wizard

import (
	"io"
	"time"

	"github.com/vchilikov/takeout-fix/internal/preflight"
)

type Report struct {
	Status                   string
	ExitCode                 int
	Workdir                  string
	StartedAtLocal           time.Time
	FinishedAtLocal          time.Time
	DetailedReportPath       string
	DetailedReportWriteError string

	ArchiveFound   int
	ArchiveValid   int
	ArchiveCorrupt int
	CorruptNames   []string

	Disk       preflight.SpaceCheck
	AutoDelete bool

	ExtractedArchives int
	SkippedArchives   int
	ExtractedFiles    int
	DeletedZips       int
	DeleteErrors      []string

	MediaFound          int
	MetadataApplied     int
	FilenameDateApplied int
	RenamedExtensions   int
	XMPSidecars         int
	CreateDateWarnings  int
	MissingJSON         int
	AmbiguousMedia      int
	UnusedJSON          int
	JSONRemoved         int
	JSONKeptDueToErrors int

	ZipScanDuration     time.Duration
	ZipValidateDuration time.Duration
	DiskCheckDuration   time.Duration
	ExtractDuration     time.Duration
	ProcessDuration     time.Duration
	TotalDuration       time.Duration

	ProblemCounts map[string]int
	ProblemSample map[string][]string
}

func (r *Report) addProblem(category string, n int, sample ...string) {
	if r.ProblemCounts == nil {
		r.ProblemCounts = make(map[string]int)
	}
	r.ProblemCounts[category] += n
	if len(sample) == 0 {
		return
	}
	if r.ProblemSample == nil {
		r.ProblemSample = make(map[string][]string)
	}
	current := r.ProblemSample[category]
	limit := 5 - len(current)
	if limit <= 0 {
		return
	}
	if len(sample) < limit {
		limit = len(sample)
	}
	r.ProblemSample[category] = append(current, sample[:limit]...)
}

func (r *Report) normalizeProblems() {
	if len(r.CorruptNames) > 0 && r.ProblemCounts["corrupt zips"] == 0 {
		r.addProblem("corrupt zips", len(r.CorruptNames), r.CorruptNames...)
	}
	if len(r.DeleteErrors) > 0 && r.ProblemCounts["zip delete errors"] == 0 {
		r.addProblem("zip delete errors", len(r.DeleteErrors), r.DeleteErrors...)
	}
}

func printReport(out io.Writer, report Report) {
	writeLine(out, "")
	writef(out, "Run result: %s\n", runResultLabel(report.Status))
	writef(out, "Metadata updated: %d of %d files\n", report.MetadataApplied, report.MediaFound)
	writef(out, "Date restored from filename: %d\n", report.FilenameDateApplied)
	writef(out, "JSON removed: %d\n", report.JSONRemoved)
	writef(out, "Missing metadata JSON: %d\n", report.MissingJSON)

	if report.Status != "SUCCESS" {
		writeLine(out, "Some files need attention. See the detailed report.")
	}

	if report.DetailedReportPath != "" {
		writef(out, "Detailed report: %s\n", report.DetailedReportPath)
	} else {
		writeLine(out, "Detailed report: unavailable")
	}

	if report.DetailedReportWriteError != "" {
		writef(out, "Report save warning: %s\n", report.DetailedReportWriteError)
	}
}

func runResultLabel(status string) string {
	switch status {
	case "SUCCESS":
		return "Completed"
	case "PARTIAL_SUCCESS":
		return "Completed with issues"
	default:
		return "Failed"
	}
}
