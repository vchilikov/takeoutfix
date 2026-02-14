package extract

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

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

	root, err := os.OpenRoot(dest)
	if err != nil {
		return 0, fmt.Errorf("open root %q: %w", dest, err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	for _, f := range r.File {
		name := f.Name
		if f.FileInfo().IsDir() {
			if err := root.MkdirAll(name, 0o755); err != nil {
				return files, err
			}
			continue
		}

		dir := filepath.Dir(name)
		if dir != "." {
			if err := root.MkdirAll(dir, 0o755); err != nil {
				return files, err
			}
		}

		rc, err := f.Open()
		if err != nil {
			return files, err
		}

		out, err := root.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
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
