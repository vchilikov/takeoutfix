package processor

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"

	"github.com/vchilikov/takeout-fix/internal/exiftool"
	"github.com/vchilikov/takeout-fix/utils/extensions"
	"github.com/vchilikov/takeout-fix/utils/files"
	"github.com/vchilikov/takeout-fix/utils/metadata"
)

const maxProblemSamples = 5

var (
	scanTakeout                  = files.ScanTakeout
	fixMediaExtension            = extensions.FixDetailed
	fixMediaExtensionWithRunner  = extensions.FixDetailedWithRunner
	applyMediaMetadata           = metadata.ApplyDetailed
	applyMediaMetadataWithRunner = metadata.ApplyDetailedWithRunner
	openExiftoolSession          = func() (exiftoolSession, error) { return exiftool.Start() }
	removeJSONFile               = os.Remove
)

type exiftoolSession interface {
	Run(args []string) (string, error)
	Close() error
}

type Summary struct {
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
}

type Report struct {
	Summary        Summary
	ProblemCounts  map[string]int
	ProblemSamples map[string][]string
}

type ProgressEvent struct {
	Processed int
	Total     int
	Media     string
}

func Run(rootPath string) (Report, error) {
	return RunWithProgress(rootPath, nil)
}

func RunWithProgress(rootPath string, onProgress func(ProgressEvent)) (Report, error) {
	report := Report{
		ProblemCounts:  make(map[string]int),
		ProblemSamples: make(map[string][]string),
	}

	scanResult, err := scanTakeout(rootPath)
	if err != nil {
		return report, fmt.Errorf("scan takeout: %w", err)
	}

	report.Summary.MissingJSON = len(scanResult.MissingJSON)
	report.Summary.AmbiguousMedia = len(scanResult.AmbiguousJSON)
	report.Summary.UnusedJSON = len(scanResult.UnusedJSON)
	report.Summary.MediaFound = len(scanResult.Pairs) + len(scanResult.MissingJSON) + len(scanResult.AmbiguousJSON)

	jsonPairCount := make(map[string]int, len(scanResult.Pairs))
	for _, jsonFile := range scanResult.Pairs {
		jsonPairCount[jsonFile]++
	}
	jsonSuccessCount := make(map[string]int, len(jsonPairCount))

	mediaFiles := slices.Sorted(maps.Keys(scanResult.Pairs))
	total := len(mediaFiles)

	if total > 0 {
		type mediaJob struct {
			mediaFile string
			jsonFile  string
		}
		type mediaResult struct {
			mediaFile string
			mediaPath string
			jsonFile  string
			fixResult extensions.FixResult
			meta      metadata.ApplyResult
			fixErr    error
			metaErr   error
		}

		workers := max(runtime.NumCPU(), 1)
		if workers > total {
			workers = total
		}

		jobs := make(chan mediaJob, total)
		results := make(chan mediaResult, total)
		var wg sync.WaitGroup

		for range workers {
			wg.Go(func() {
				session, err := openExiftoolSession()
				if err != nil {
					session = nil
				}
				defer closeSession(session)

				for job := range jobs {
					mediaPath := filepath.Join(rootPath, job.mediaFile)
					jsonPath := filepath.Join(rootPath, job.jsonFile)

					fixResult, fixErr := runFixWithFallback(mediaPath, &session)
					if fixErr != nil {
						results <- mediaResult{
							mediaFile: job.mediaFile,
							mediaPath: mediaPath,
							jsonFile:  job.jsonFile,
							fixErr:    fixErr,
						}
						continue
					}

					metaResult, metaErr := runMetadataWithFallback(fixResult.Path, jsonPath, &session)
					results <- mediaResult{
						mediaFile: job.mediaFile,
						mediaPath: mediaPath,
						jsonFile:  job.jsonFile,
						fixResult: fixResult,
						meta:      metaResult,
						metaErr:   metaErr,
					}
				}
			})
		}

		for _, mediaFile := range mediaFiles {
			jobs <- mediaJob{
				mediaFile: mediaFile,
				jsonFile:  scanResult.Pairs[mediaFile],
			}
		}
		close(jobs)

		go func() {
			wg.Wait()
			close(results)
		}()

		processed := 0
		for res := range results {
			processed++
			if res.fixErr != nil {
				report.addProblem("extension errors", res.mediaPath)
				notifyProgress(onProgress, processed, total, res.mediaFile)
				continue
			}
			if res.fixResult.Renamed {
				report.Summary.RenamedExtensions++
			}

			if res.metaErr != nil {
				report.addProblem("metadata errors", res.fixResult.Path)
				notifyProgress(onProgress, processed, total, res.mediaFile)
				continue
			}

			jsonSuccessCount[res.jsonFile]++
			report.Summary.MetadataApplied++
			if res.meta.UsedFilenameDate {
				report.Summary.FilenameDateApplied++
			}
			if res.meta.UsedXMPSidecar {
				report.Summary.XMPSidecars++
			}
			if res.meta.CreateDateWarned {
				report.Summary.CreateDateWarnings++
				report.addProblem("create date warnings", res.fixResult.Path)
			}
			if res.meta.FilenameDateWarned {
				report.addProblem("filename date warnings", res.fixResult.Path)
			}
			if res.meta.MediaFileDateWarned {
				report.addProblem("media file date warnings", res.fixResult.Path)
			}

			notifyProgress(onProgress, processed, total, res.mediaFile)
		}
	}

	jsonToRemove := make([]string, 0, len(jsonPairCount))
	for jsonFile, pairCount := range jsonPairCount {
		if jsonSuccessCount[jsonFile] == pairCount {
			jsonToRemove = append(jsonToRemove, jsonFile)
		} else {
			report.Summary.JSONKeptDueToErrors++
		}
	}
	slices.Sort(jsonToRemove)

	for _, jsonFile := range jsonToRemove {
		if err := removeJSONFile(filepath.Join(rootPath, jsonFile)); err != nil {
			report.addProblem("json remove errors", filepath.Join(rootPath, jsonFile))
			continue
		}
		report.Summary.JSONRemoved++
	}

	return report, nil
}

func runFixWithFallback(mediaPath string, session *exiftoolSession) (extensions.FixResult, error) {
	if session != nil && *session != nil {
		result, err := fixMediaExtensionWithRunner(mediaPath, (*session).Run)
		if err == nil {
			return result, nil
		}
		closeAndResetSession(session)
	}
	return fixMediaExtension(mediaPath)
}

func runMetadataWithFallback(mediaPath string, jsonPath string, session *exiftoolSession) (metadata.ApplyResult, error) {
	if session != nil && *session != nil {
		result, err := applyMediaMetadataWithRunner(mediaPath, jsonPath, (*session).Run)
		if err == nil {
			return result, nil
		}
		closeAndResetSession(session)
	}
	return applyMediaMetadata(mediaPath, jsonPath)
}

func closeAndResetSession(session *exiftoolSession) {
	if session == nil || *session == nil {
		return
	}
	_ = (*session).Close()
	*session = nil
}

func closeSession(session exiftoolSession) {
	if session != nil {
		_ = session.Close()
	}
}

func notifyProgress(onProgress func(ProgressEvent), processed int, total int, media string) {
	if onProgress == nil || total == 0 {
		return
	}
	onProgress(ProgressEvent{
		Processed: processed,
		Total:     total,
		Media:     media,
	})
}

func (r *Report) addProblem(category string, value string) {
	r.ProblemCounts[category]++
	if len(r.ProblemSamples[category]) < maxProblemSamples {
		r.ProblemSamples[category] = append(r.ProblemSamples[category], value)
	}
}
