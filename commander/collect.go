package commander

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// BackgroundLoop runs the Commander's periodic scan
func (c *Commander) BackgroundLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.Iteration++
		c.Scan()
		if c.ShouldReview() {
			c.Review()
		}
	}
}

// Scan checks all tracked operations for state transitions
func (c *Commander) Scan() {
	c.Tracker.mu.Lock()
	tracked := make(map[string]*TrackedOp, len(c.Tracker.Ops))
	for id, t := range c.Tracker.Ops {
		tracked[id] = t
	}
	c.Tracker.mu.Unlock()

	c.RecentFailures = 0
	for opID, t := range tracked {
		if t.FailedAt != nil || t.PID == 0 {
			// Already resolved or not dispatched
			continue
		}
		c.handleTracked(opID, t)
	}
}

func (c *Commander) handleTracked(opID string, t *TrackedOp) {
	// Process still running — nothing to do
	alive := processAlive(t.PID)
	debugLog(fmt.Sprintf("[scan] op=%s pid=%d alive=%v", opID, t.PID, alive))
	if t.PID > 0 && alive {
		if t.DispatchedAt != nil && time.Since(*t.DispatchedAt) > 30*time.Minute {
			c.RecentFailures++
		}
		return
	}

	// Process exited — check if Captain completed the operation
	opDir := c.OperationDir(t.Squad, opID)
	reportExists := fileExists(filepath.Join(opDir, "report.html"))
	opCompleted := fileContains(c.OperationMDPath(t.Squad, opID), "status: completed")

	if reportExists && opCompleted {
		// Finalize: write tracker info back to OPERATION.md, then clean tracker
		c.finalizeOperation(opID, t, "completed", nil)
	} else {
		reason := "process_exited_without_completion"
		c.finalizeOperation(opID, t, "failed", &reason)
		c.RecentFailures++
	}
}

// finalizeOperation writes tracker state back into OPERATION.md and removes from tracker.
// Safe because the process has exited — Captain is no longer writing.
func (c *Commander) finalizeOperation(opID string, t *TrackedOp, status string, failReason *string) {
	op, err := c.LoadOperation(t.Squad, opID)
	if err != nil {
		return
	}

	// Write Commander-tracked fields into OPERATION.md
	now := time.Now().UTC()
	op.DispatchedAt = t.DispatchedAt
	op.FailedAt = t.FailedAt
	op.FailureReason = t.FailureReason

	if status == "completed" {
		op.CompletedAt = &now
	} else if status == "failed" {
		op.Status = "failed"
		op.FailedAt = &now
		op.FailureReason = failReason
	}

	c.SaveOperation(op)
	c.Tracker.Remove(opID)
}

// ShouldReview determines if LLM review is needed
func (c *Commander) ShouldReview() bool {
	if c.Iteration%10 == 0 {
		return true
	}
	if c.RecentFailures > 0 {
		return true
	}
	c.Tracker.mu.Lock()
	defer c.Tracker.mu.Unlock()
	for _, t := range c.Tracker.Ops {
		if t.PID > 0 && t.DispatchedAt != nil {
			if time.Since(*t.DispatchedAt) > 30*time.Minute {
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
	// Use Signal(0) instead of Signal(nil) — Go 1.24's pidfd path
	// rejects nil as "unsupported signal type", but Signal(0) works correctly.
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

// GetUnnotified returns completed/failed operations not yet notified
func (c *Commander) GetUnnotified() ([]*Operation, error) {
	ops, err := c.ListOperations()
	if err != nil {
		return nil, err
	}
	var result []*Operation
	for _, op := range ops {
		c.EnrichOperation(op)
		if (op.Status == "completed" || op.Status == "failed") && !op.Notified {
			result = append(result, op)
		}
	}
	return result, nil
}

// MarkNotified sets notified=true. Writes to tracker if in-flight, otherwise to OPERATION.md.
func (c *Commander) MarkNotified(opID string) error {
	if tracked := c.Tracker.Get(opID); tracked != nil {
		return c.Tracker.SetNotified(opID)
	}
	// Finalized — update OPERATION.md directly
	op, err := c.findOperation(opID)
	if err != nil {
		return err
	}
	op.Notified = true
	return c.SaveOperation(op)
}

// Status returns a summary of all operations
func (c *Commander) Status() map[string]interface{} {
	ops, _ := c.ListOperations()
	counts := map[string]int{"queued": 0, "active": 0, "completed": 0, "failed": 0}
	for _, op := range ops {
		c.EnrichOperation(op)
		counts[op.Status]++
	}
	unnotified, _ := c.GetUnnotified()

	return map[string]interface{}{
		"counts":     counts,
		"unnotified": unnotified,
		"iteration":  c.Iteration,
		"message":    fmt.Sprintf("%d queued, %d active, %d completed, %d failed", counts["queued"], counts["active"], counts["completed"], counts["failed"]),
	}
}
