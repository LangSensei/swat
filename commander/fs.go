package commander

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// fileExists checks if a path exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// dirExists checks if a path exists and is a directory
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// fileContains checks if a file contains a substring
func fileContains(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// copyDir recursively copies a directory tree
func copyDir(src, dst string) error {
	return copyDirExclude(src, dst, nil)
}

// copyDirExclude recursively copies a directory tree, skipping directories whose
// base name appears in the exclude set
func copyDirExclude(src, dst string, exclude map[string]bool) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if info.IsDir() && exclude[info.Name()] && rel != "." {
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

// execCommand creates an exec.Cmd
func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
