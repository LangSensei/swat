package commander

import (
	"fmt"
	"os"
	"path/filepath"
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
	if t.PID > 0 && processAlive(t.PID) {
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
		// Read OPERATION.md for summary
		if op, err := c.LoadOperation(t.Squad, opID); err == nil {
			t.PID = 0
			c.Tracker.Save()
			_ = op // summary is in OPERATION.md, available via MergeOperation
		}
	} else {
		reason := "process_exited_without_completion"
		c.Tracker.SetFailed(opID, reason)
		c.RecentFailures++
	}
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
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(nil) == nil
}

// GetUnnotified returns completed/failed operations not yet notified
func (c *Commander) GetUnnotified() ([]*FullOperation, error) {
	ops, err := c.ListOperations()
	if err != nil {
		return nil, err
	}
	var result []*FullOperation
	for _, op := range ops {
		full := c.MergeOperation(op)
		if (full.Status == "completed" || full.Status == "failed") && !full.Notified {
			result = append(result, full)
		}
	}
	return result, nil
}

// MarkNotified sets notified=true for an operation in the tracker
func (c *Commander) MarkNotified(opID string) error {
	return c.Tracker.SetNotified(opID)
}

// Status returns a summary of all operations
func (c *Commander) Status() map[string]interface{} {
	ops, _ := c.ListOperations()
	counts := map[string]int{"queued": 0, "active": 0, "completed": 0, "failed": 0}
	for _, op := range ops {
		full := c.MergeOperation(op)
		counts[full.Status]++
	}
	unnotified, _ := c.GetUnnotified()

	return map[string]interface{}{
		"counts":     counts,
		"unnotified": unnotified,
		"iteration":  c.Iteration,
		"message":    fmt.Sprintf("%d queued, %d active, %d completed, %d failed", counts["queued"], counts["active"], counts["completed"], counts["failed"]),
	}
}
