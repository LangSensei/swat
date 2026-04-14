//go:build windows

package platform

import "golang.org/x/sys/windows"

// ProcessAlive checks if a process with the given PID is still running.
func ProcessAlive(pid int) bool {
	if pid == 0 {
		return false
	}
	const STILL_ACTIVE = 259
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid),
	)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == STILL_ACTIVE
}
