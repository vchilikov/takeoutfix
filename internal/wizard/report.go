package wizard

import (
	"io"
	"sort"
	"time"

	"github.com/vchilikov/takeout-fix/internal/preflight"
)

type Report struct {
	Status string

	ArchiveFound   int
	ArchiveValid   int
	ArchiveCorrupt int
	CorruptNames   []string

	Disk       preflight.SpaceCheck
	DeleteMode bool

	ExtractedArchives int
	SkippedArchives   int
	ExtractedFiles    int
	DeletedZips       int
	DeleteErrors      []string

	MediaFound          int
	MetadataApplied     int
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

func printReport(out io.Writer, report Report) {
	writeLine(out, "")
	writeLine(out, "Final summary")
	writef(out, "Status: %s\n", report.Status)
	writef(out, "Archives: found=%d, valid=%d, corrupt=%d\n", report.ArchiveFound, report.ArchiveValid, report.ArchiveCorrupt)
	writef(out, "Disk: available=%s, required=%s, delete-mode=%t\n", preflight.FormatBytes(report.Disk.AvailableBytes), preflight.FormatBytes(report.Disk.RequiredBytes), report.DeleteMode)
	writef(out, "Extraction: extracted archives=%d, skipped=%d, extracted files=%d, deleted zips=%d\n", report.ExtractedArchives, report.SkippedArchives, report.ExtractedFiles, report.DeletedZips)
	writef(out, "Metadata: media=%d, applied=%d, renamed=%d, xmp=%d, missing json=%d, ambiguous=%d\n", report.MediaFound, report.MetadataApplied, report.RenamedExtensions, report.XMPSidecars, report.MissingJSON, report.AmbiguousMedia)
	writef(out, "JSON cleanup: removed=%d, kept due to errors=%d, kept unused=%d\n", report.JSONRemoved, report.JSONKeptDueToErrors, report.UnusedJSON)
	writef(
		out,
		"Timing: zip scan=%s, zip validate=%s, disk check=%s, extract=%s, process=%s, total=%s\n",
		formatDuration(report.ZipScanDuration),
		formatDuration(report.ZipValidateDuration),
		formatDuration(report.DiskCheckDuration),
		formatDuration(report.ExtractDuration),
		formatDuration(report.ProcessDuration),
		formatDuration(report.TotalDuration),
	)

	if len(report.CorruptNames) > 0 {
		report.addProblem("corrupt zips", len(report.CorruptNames), report.CorruptNames...)
	}
	if len(report.DeleteErrors) > 0 {
		report.addProblem("zip delete errors", len(report.DeleteErrors), report.DeleteErrors...)
	}
	if len(report.ProblemCounts) > 0 {
		writeLine(out, "Problems:")
		categories := make([]string, 0, len(report.ProblemCounts))
		for category := range report.ProblemCounts {
			categories = append(categories, category)
		}
		sort.Strings(categories)
		for _, category := range categories {
			writef(out, "- %s (%d)\n", category, report.ProblemCounts[category])
			if samples := report.ProblemSample[category]; len(samples) > 0 {
				for _, sample := range samples {
					writef(out, "  - %s\n", sample)
				}
			}
		}
	}
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	return d.Round(time.Millisecond).String()
}
