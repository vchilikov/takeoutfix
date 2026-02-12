package extract

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

	for _, f := range r.File {
		target, err := safeJoin(dest, f.Name)
		if err != nil {
			return files, err
		}
		if err := ensureNoSymlinkComponents(dest, target); err != nil {
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

func ensureNoSymlinkComponents(base string, target string) error {
	cleanBase := filepath.Clean(base)
	cleanTarget := filepath.Clean(target)

	if err := ensureExistingPathNotSymlink(cleanBase); err != nil {
		return err
	}

	rel, err := filepath.Rel(cleanBase, cleanTarget)
	if err != nil {
		return fmt.Errorf("invalid zip entry path (symlink component): %w", err)
	}
	if rel == "." {
		return nil
	}

	current := cleanBase
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		current = filepath.Join(current, part)
		if err := ensureExistingPathNotSymlink(current); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
	}

	return nil
}

func ensureExistingPathNotSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("invalid zip entry path (symlink component): %s", path)
	}
	return nil
}
