package commander

import (
	"crypto/rand"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/LangSensei/swat/commander/intake"
	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/notify"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/platform"
)

// Commander is the core orchestrator
type Commander struct {
	RuntimeName string
	NotifyName  string
	Notifier    notify.Notifier
}

// New creates a new Commander instance
func New(runtimeName, notifyName string) *Commander {
	layout.EnsureDirs()

	n, err := notify.New(notifyName)
	if err != nil {
		log.Printf("[commander] notify init error (falling back to desktop): %v", err)
		n = &notify.DesktopNotifier{}
	}

	return &Commander{
		RuntimeName: runtimeName,
		NotifyName:  notifyName,
		Notifier:    n,
	}
}

// GenerateOpID creates a unique operation identifier.
func GenerateOpID() string {
	now := time.Now().UTC()
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", now.Format("20060102"), b)
}

// BackgroundLoop runs the Commander's periodic scan.
func (c *Commander) BackgroundLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.scan()
		intake.ProcessDue(c.dispatchForIntake, c.processExistingOperation)
	}
}

func (c *Commander) scan() {
	ops, err := operation.List()
	if err != nil {
		log.Printf("[scan] error: %v", err)
		return
	}
	for _, op := range ops {
		if op.Status == "active" {
			c.handleActive(op)
		}
	}
}

func (c *Commander) handleActive(op *operation.Operation) {
	if op.PID > 0 && platform.ProcessAlive(op.PID) {
		return
	}

	now := time.Now().UTC()
	opDir := layout.OperationDir(op.Squad, op.OperationID)
	reportExists := platform.FileExists(filepath.Join(opDir, "report.html"))
	opCompleted := platform.FileContains(layout.OperationMDPath(op.Squad, op.OperationID), "status: completed")

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
	}
	if err := operation.Save(op); err != nil {
		log.Printf("[scan] %s: failed to save collected state: %v", op.OperationID, err)
	}

	if c.Notifier != nil && op.Status != "completed" {
		msg := "Operation " + op.OperationID + " failed"
		if err := c.Notifier.Notify(msg); err != nil {
			log.Printf("[scan] notify error: %v", err)
		}
	}
}

// dispatchForIntake is the callback used by intake.ProcessDue for recurring tasks.
// It creates a new operation and runs the full pipeline: classify → provision → launch.
func (c *Commander) dispatchForIntake(brief, details string) (string, error) {
	now := time.Now().UTC()
	op := &operation.Operation{
		OperationID: GenerateOpID(),
		Brief:       brief,
		Details:     details,
		Status:      "queued",
		CreatedAt:   now,
	}
	if err := operation.Create(op); err != nil {
		return "", err
	}

	c.processOperation(op)
	return op.OperationID, nil
}

// processExistingOperation is the callback used by intake.ProcessDue for immediate tasks.
// It loads an existing operation and runs the pipeline on it.
func (c *Commander) processExistingOperation(operationID string) error {
	op, err := operation.Find(operationID)
	if err != nil {
		return fmt.Errorf("find operation %s: %w", operationID, err)
	}

	c.processOperation(op)
	return nil
}
