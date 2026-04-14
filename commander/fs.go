package commander

import (
	"os"
	"os/exec"
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

// execCommand creates an exec.Cmd
func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
