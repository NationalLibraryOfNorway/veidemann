package upload

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func uploadCandidates(dir string) ([]string, error) {
	return uploadCandidatesWithOpen(dir, true)
}

func uploadCandidatesClosed(dir string) ([]string, error) {
	return uploadCandidatesWithOpen(dir, false)
}

func uploadCandidatesWithOpen(dir string, includeOpen bool) ([]string, error) {
	patterns := []string{
		filepath.Join(dir, "*.warc"),
		filepath.Join(dir, "*.warc.gz"),
	}
	if includeOpen {
		patterns = append(patterns, filepath.Join(dir, "*.open"))
	}

	seen := map[string]struct{}{}
	paths := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			paths = append(paths, match)
		}
	}

	return paths, nil
}

func backlogStats(dir string) (int, int64, error) {
	paths, err := uploadCandidates(dir)
	if err != nil {
		return 0, 0, err
	}

	var (
		count int
		total int64
	)
	for _, path := range paths {
		st, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return 0, 0, err
		}
		count++
		total += st.Size()
	}

	return count, total, nil
}

func isWithinDir(path string, dir string) bool {
	if dir == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && rel != "")
}

func moveFileToDir(src string, dstDir string) (string, error) {
	src = filepath.Clean(src)
	dstDir = filepath.Clean(dstDir)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return "", err
	}

	dst := filepath.Join(dstDir, filepath.Base(src))
	if src == dst {
		return dst, nil
	}

	exists, err := reconcileExistingDestination(src, dst)
	if err != nil {
		return "", err
	}
	if exists {
		return dst, nil
	}

	if err := os.Rename(src, dst); err == nil {
		if err := syncDir(dstDir); err != nil {
			return "", err
		}
		if filepath.Dir(src) != dstDir {
			if err := syncDir(filepath.Dir(src)); err != nil {
				return "", err
			}
		}
		return dst, nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return "", err
	}

	return copyAcrossDevices(src, dst)
}

func reconcileExistingDestination(src string, dst string) (bool, error) {
	if _, err := os.Stat(dst); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	same, err := sameFileContents(src, dst)
	if err != nil {
		return false, err
	}
	if !same {
		return false, fmt.Errorf("fallback destination already exists with different contents: %s", dst)
	}

	if err := os.Remove(src); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := syncDir(filepath.Dir(src)); err != nil {
		return false, err
	}

	return true, nil
}

func sameFileContents(a string, b string) (bool, error) {
	aInfo, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	bInfo, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	if aInfo.Size() != bInfo.Size() {
		return false, nil
	}

	aMD5, err := calculateMD5(a)
	if err != nil {
		return false, err
	}
	bMD5, err := calculateMD5(b)
	if err != nil {
		return false, err
	}

	return aMD5 == bMD5, nil
}

func copyAcrossDevices(src string, dst string) (string, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return "", err
	}

	dstDir := filepath.Dir(dst)
	tmpFile, err := os.CreateTemp(dstDir, "."+filepath.Base(dst)+".tmp-*")
	if err != nil {
		return "", err
	}
	tmpName := tmpFile.Name()
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := io.Copy(tmpFile, srcFile); err != nil {
		_ = tmpFile.Close()
		return "", err
	}
	if err := tmpFile.Chmod(srcInfo.Mode().Perm()); err != nil {
		_ = tmpFile.Close()
		return "", err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	if err := os.Rename(tmpName, dst); err != nil {
		return "", err
	}
	cleanupTmp = false

	if err := syncDir(dstDir); err != nil {
		return "", err
	}
	if err := os.Remove(src); err != nil {
		return "", err
	}
	if err := syncDir(filepath.Dir(src)); err != nil {
		return "", err
	}

	return dst, nil
}

func syncDir(dir string) error {
	f, err := os.Open(filepath.Clean(dir))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := f.Sync(); err != nil {
		if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP) {
			return nil
		}
		return err
	}
	return nil
}
