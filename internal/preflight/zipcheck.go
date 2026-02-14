package preflight

import (
	"archive/zip"
	"cmp"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

type ZipArchive struct {
	Name        string
	Path        string
	SizeBytes   uint64
	ModTime     time.Time
	Fingerprint string
}

type ArchiveIntegrity struct {
	Archive           ZipArchive
	FileCount         int
	UncompressedBytes uint64
	Err               error
}

type IntegritySummary struct {
	Checked           []ArchiveIntegrity
	Corrupt           []ArchiveIntegrity
	TotalUncompressed uint64
	TotalZipBytes     uint64
}

func DiscoverTopLevelZips(cwd string) ([]ZipArchive, error) {
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return nil, err
	}

	var zips []ZipArchive
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".zip") {
			continue
		}

		fullPath := filepath.Join(cwd, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", fullPath, err)
		}
		size := uint64(info.Size())
		zips = append(zips, ZipArchive{
			Name:        entry.Name(),
			Path:        fullPath,
			SizeBytes:   size,
			ModTime:     info.ModTime().UTC(),
			Fingerprint: Fingerprint(size, info.ModTime().UTC()),
		})
	}

	slices.SortFunc(zips, func(a, b ZipArchive) int {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return zips, nil
}

func Fingerprint(size uint64, modTime time.Time) string {
	return fmt.Sprintf("%d:%d", size, modTime.UnixNano())
}

func ValidateAll(zips []ZipArchive) IntegritySummary {
	summary := IntegritySummary{}
	if len(zips) == 0 {
		return summary
	}

	type job struct {
		index   int
		archive ZipArchive
	}
	type result struct {
		index int
		check ArchiveIntegrity
	}

	workers := max(runtime.NumCPU(), 1)
	if workers > len(zips) {
		workers = len(zips)
	}

	jobs := make(chan job, len(zips))
	results := make(chan result, len(zips))

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for j := range jobs {
				results <- result{
					index: j.index,
					check: ValidateZip(j.archive),
				}
			}
		})
	}

	for i, z := range zips {
		jobs <- job{
			index:   i,
			archive: z,
		}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	checked := make([]ArchiveIntegrity, len(zips))
	for res := range results {
		checked[res.index] = res.check
	}

	summary.Checked = checked
	for i, res := range checked {
		summary.TotalZipBytes += zips[i].SizeBytes
		if res.Err != nil {
			summary.Corrupt = append(summary.Corrupt, res)
			continue
		}
		summary.TotalUncompressed += res.UncompressedBytes
	}
	return summary
}

func ValidateZip(z ZipArchive) (res ArchiveIntegrity) {
	res = ArchiveIntegrity{Archive: z}

	r, err := zip.OpenReader(z.Path)
	if err != nil {
		res.Err = fmt.Errorf("open zip: %w", err)
		return res
	}
	defer func() {
		if closeErr := r.Close(); closeErr != nil && res.Err == nil {
			res.Err = fmt.Errorf("close zip: %w", closeErr)
		}
	}()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		res.FileCount++
		res.UncompressedBytes += f.UncompressedSize64

		rc, err := f.Open()
		if err != nil {
			res.Err = fmt.Errorf("open entry %s: %w", f.Name, err)
			return res
		}
		_, copyErr := io.Copy(io.Discard, rc)
		closeErr := rc.Close()
		if copyErr != nil {
			res.Err = fmt.Errorf("read entry %s: %w", f.Name, copyErr)
			return res
		}
		if closeErr != nil {
			res.Err = fmt.Errorf("close entry %s: %w", f.Name, closeErr)
			return res
		}
	}

	return res
}
