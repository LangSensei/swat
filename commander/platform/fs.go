package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// FileExists checks if a path exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirExists checks if a path exists and is a directory
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// FileContains checks if a file contains a substring
func FileContains(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// CopyDir recursively copies a directory tree.
func CopyDir(src, dst string) error {
	return CopyDirExclude(src, dst)
}

// CopyDirExclude recursively copies a directory tree, skipping directories
// whose names match any of the provided exclude list.
func CopyDirExclude(src, dst string, exclude ...string) error {
	excludeSet := make(map[string]bool, len(exclude))
	for _, e := range exclude {
		excludeSet[e] = true
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if info.IsDir() && rel != "." && excludeSet[info.Name()] {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
