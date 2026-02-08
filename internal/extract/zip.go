package extract

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vchilikov/takeout-fix/internal/preflight"
)

type Stats struct {
	ArchivesExtracted int
	ArchivesSkipped   int
	FilesExtracted    int
	DeletedZips       int
	DeleteErrors      []string
}

func ExtractArchives(
	zips []preflight.ZipArchive,
	dest string,
	shouldSkip func(preflight.ZipArchive) bool,
	deleteAfter bool,
) (Stats, error) {
	stats := Stats{}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return stats, fmt.Errorf("mkdir dest: %w", err)
	}

	for _, archive := range zips {
		if shouldSkip != nil && shouldSkip(archive) {
			stats.ArchivesSkipped++
			continue
		}

		files, err := extractOne(archive.Path, dest)
		if err != nil {
			return stats, fmt.Errorf("extract %s: %w", archive.Name, err)
		}
		stats.ArchivesExtracted++
		stats.FilesExtracted += files

		if deleteAfter {
			if err := os.Remove(archive.Path); err != nil {
				stats.DeleteErrors = append(stats.DeleteErrors, fmt.Sprintf("%s: %v", archive.Name, err))
			} else {
				stats.DeletedZips++
			}
		}
	}

	return stats, nil
}

func ExtractArchive(zipPath string, dest string) (int, error) {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir dest: %w", err)
	}
	return extractOne(zipPath, dest)
}

func extractOne(zipPath string, dest string) (files int, err error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, err
	}
	defer func() {
		if closeErr := r.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	for _, f := range r.File {
		target, err := safeJoin(dest, f.Name)
		if err != nil {
			return files, err
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return files, err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return files, err
		}

		rc, err := f.Open()
		if err != nil {
			return files, err
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			_ = rc.Close()
			return files, err
		}

		_, copyErr := io.Copy(out, rc)
		closeOutErr := out.Close()
		closeInErr := rc.Close()
		if copyErr != nil {
			return files, copyErr
		}
		if closeOutErr != nil {
			return files, closeOutErr
		}
		if closeInErr != nil {
			return files, closeInErr
		}

		files++
	}

	return files, nil
}

func safeJoin(base string, name string) (string, error) {
	cleanBase := filepath.Clean(base)
	cleanName := filepath.Clean(name)
	target := filepath.Join(cleanBase, cleanName)

	prefix := cleanBase + string(os.PathSeparator)
	if target != cleanBase && !strings.HasPrefix(target, prefix) {
		return "", fmt.Errorf("invalid zip entry path: %s", name)
	}

	return target, nil
}
