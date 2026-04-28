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
	"github.com/LangSensei/swat/commander/pipeline"
	"github.com/LangSensei/swat/commander/platform"
	"github.com/LangSensei/swat/commander/runtime"
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
		switch op.Status {
		case "queued":
			c.spawnClassify(op)
		case "classifying":
			if !platform.ProcessAlive(op.PID) {
				c.finishClassify(op)
			}
		case "active":
			c.handleActive(op)
		}
	}
}

// spawnClassify starts the classifier subprocess for a queued operation.
func (c *Commander) spawnClassify(op *operation.Operation) {
	rt, err := runtime.New(c.RuntimeName)
	if err != nil {
		c.failOperation(op, fmt.Sprintf("init runtime: %v", err))
		return
	}
	if err := pipeline.SpawnClassify(rt, op); err != nil {
		c.failOperation(op, fmt.Sprintf("classify spawn: %v", err))
	}
}

// finishClassify completes the classify phase and runs provision + launch.
func (c *Commander) finishClassify(op *operation.Operation) {
	rt, err := runtime.New(c.RuntimeName)
	if err != nil {
		c.failOperation(op, fmt.Sprintf("init runtime: %v", err))
		return
	}

	reloaded, destDir, err := pipeline.FinishClassify(op, c.Notifier)
	if err != nil {
		// Use original op (still in _unclassified) for failure path
		c.failOperation(op, err.Error())
		return
	}

	if err := pipeline.Provision(rt, reloaded, destDir, c.RuntimeName, c.NotifyName); err != nil {
		c.failOperation(reloaded, fmt.Sprintf("provision: %v", err))
		return
	}

	if err := pipeline.LaunchAgent(rt, reloaded, destDir); err != nil {
		c.failOperation(reloaded, fmt.Sprintf("launch: %v", err))
		return
	}
	log.Printf("[scan] %s: launched successfully (squad=%s)", reloaded.OperationID, reloaded.Squad)
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
// It creates a new operation with status "queued". The scan loop picks it up.
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
	return op.OperationID, nil
}

// processExistingOperation is the callback used by intake.ProcessDue for immediate tasks.
// The operation already exists with status "queued" — the scan loop will pick it up.
func (c *Commander) processExistingOperation(operationID string) error {
	return nil
}
