//go:build !windows

package commander

import (
	"os"
	"syscall"
)

func processAlive(pid int) bool {
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
