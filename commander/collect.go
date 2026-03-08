package commander

import (
	"fmt"
	"log"
	"os"
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

// Scan checks all operations for state transitions
func (c *Commander) Scan() {
	ops, err := c.ListOperations()
	if err != nil {
		log.Printf("[scan] error: %v", err)
		return
	}
	c.RecentFailures = 0
	for _, op := range ops {
		switch op.Status {
		case "active":
			c.handleActive(op)
		}
	}
}

func (c *Commander) handleActive(op *Operation) {
	// Check if process is still alive
	if op.PID > 0 && !processAlive(op.PID) {
		now := time.Now().UTC()
		reason := "process_crashed"
		op.Status = "failed"
		op.FailedAt = &now
		op.FailureReason = &reason
		c.RecentFailures++
		c.SaveOperation(op)
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

// processAlive checks if a PID is running
func processAlive(pid int) bool {
	if pid == 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(nil)
	return err == nil
}

// GetUnnotified returns completed/failed operations not yet notified
func (c *Commander) GetUnnotified() ([]*Operation, error) {
	ops, err := c.ListOperations()
	if err != nil {
		return nil, err
	}
	var result []*Operation
	for _, op := range ops {
		if (op.Status == "completed" || op.Status == "failed") && !op.Notified {
			result = append(result, op)
		}
	}
	return result, nil
}

// MarkNotified sets notified=true for an operation
func (c *Commander) MarkNotified(opID string) error {
	op, err := c.LoadOperation(opID)
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
