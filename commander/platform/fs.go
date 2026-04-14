package platform

import (
	"os"
	"os/exec"
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

// ExecCommand creates an exec.Cmd
func ExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
