//go:build !windows

package platform

import (
	"os"
	"syscall"
)

// ProcessAlive checks if a process with the given PID is still running.
func ProcessAlive(pid int) bool {
	if pid == 0 {
		return false
	}
	// Signal(syscall.Signal(0)) maps to kill(pid, 0) via pidfd_send_signal.
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}
