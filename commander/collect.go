package commander

import (
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// BackgroundLoop runs the Commander's periodic scan
func (c *Commander) BackgroundLoop(interval time.Duration) {
	// Run once on startup
	c.Scan()
	c.CheckDue()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.Iteration++
		c.Scan()
		c.CheckDue()
		if c.ShouldReview() {
			c.Review()
		}
	}
}

// Scan checks all operations for state transitions
func (c *Commander) Scan() {
	ops, err := c.ListOperations()
	if err != nil {
		log.Printf("[scan] error: %v", err)
		return
	}
	c.RecentFailures = 0
	for _, op := range ops {
		if op.Status == "active" {
			c.handleActive(op)
		}
	}
}

func (c *Commander) handleActive(op *Operation) {
	// Process still running — nothing to do
	if op.PID > 0 && processAlive(op.PID) {
		// Track long-running operations for review
		if op.DispatchedAt != nil && time.Since(*op.DispatchedAt) > 30*time.Minute {
			c.RecentFailures++
		}
		return
	}

	// Process exited — check if Captain completed the operation
	now := time.Now().UTC()
	opDir := c.OperationDir(op.Squad, op.OperationID)
	reportExists := fileExists(filepath.Join(opDir, "report.html"))
	opCompleted := fileContains(c.OperationMDPath(op.Squad, op.OperationID), "status: completed")

	if reportExists && opCompleted {
		op.Status = "completed"
		op.CompletedAt = &now
		op.PID = 0
	} else {
		reason := "process_exited_without_completion"
		op.Status = "failed"
		op.FailedAt = &now
		op.FailureReason = &reason
		op.PID = 0
		c.RecentFailures++
	}
	c.SaveOperation(op)
}

// ShouldReview determines if LLM review is needed
func (c *Commander) ShouldReview() bool {
	if c.Iteration%10 == 0 {
		return true
	}
	if c.RecentFailures > 0 {
		return true
	}
	ops, _ := c.ListOperations()
	for _, op := range ops {
		if op.Status == "active" && op.DispatchedAt != nil {
			if time.Since(*op.DispatchedAt) > 30*time.Minute {
				return true
			}
		}
	}
	return false
}

func processAlive(pid int) bool {
	if pid == 0 {
		return false
	}
	// Signal(nil) broken in Go 1.24: pidfd path returns "unsupported signal type".
	// Signal(syscall.Signal(0)) correctly maps to kill(pid, 0) via pidfd_send_signal.
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}


