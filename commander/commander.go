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
	"github.com/gofrs/flock"
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
		intake.ProcessDue(c.dispatchForIntake, nil)
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
	opDir := layout.UnclassifiedOperationDir(op.OperationID)
	lockPath := filepath.Join(opDir, ".lock")
	fl := flock.New(lockPath)
	locked, err := fl.TryLock()
	if !locked || err != nil {
		return
	}
	defer fl.Unlock()

	reloaded, err := operation.Load("_unclassified", op.OperationID)
	if err != nil || reloaded.Status != "queued" {
		return
	}

	rt, err := runtime.New(c.RuntimeName)
	if err != nil {
		c.failOperation(reloaded, fmt.Sprintf("init runtime: %v", err))
		return
	}
	if err := pipeline.SpawnClassify(rt, reloaded); err != nil {
		c.failOperation(reloaded, fmt.Sprintf("classify spawn: %v", err))
	}
}

// finishClassify completes the classify phase and runs provision + launch.
func (c *Commander) finishClassify(op *operation.Operation) {
	opDir := layout.UnclassifiedOperationDir(op.OperationID)
	lockPath := filepath.Join(opDir, ".lock")
	fl := flock.New(lockPath)
	locked, err := fl.TryLock()
	if !locked || err != nil {
		return
	}
	defer fl.Unlock()

	reloaded, err := operation.Load("_unclassified", op.OperationID)
	if err != nil || reloaded.Status != "classifying" {
		return
	}

	rt, err := runtime.New(c.RuntimeName)
	if err != nil {
		c.failOperation(reloaded, fmt.Sprintf("init runtime: %v", err))
		return
	}

	classified, destDir, err := pipeline.FinishClassify(reloaded, c.Notifier)
	if err != nil {
		c.failOperation(reloaded, err.Error())
		return
	}

	if err := pipeline.Provision(rt, classified, destDir, c.RuntimeName, c.NotifyName); err != nil {
		c.failOperation(classified, fmt.Sprintf("provision: %v", err))
		return
	}

	if err := pipeline.LaunchAgent(rt, classified, destDir); err != nil {
		c.failOperation(classified, fmt.Sprintf("launch: %v", err))
		return
	}
	log.Printf("[scan] %s: launched successfully (squad=%s)", classified.OperationID, classified.Squad)
}

func (c *Commander) handleActive(op *operation.Operation) {
	if op.PID > 0 && platform.ProcessAlive(op.PID) {
		return
	}

	opDir := layout.OperationDir(op.Squad, op.OperationID)
	lockPath := filepath.Join(opDir, ".lock")
	fl := flock.New(lockPath)
	locked, err := fl.TryLock()
	if !locked || err != nil {
		return
	}
	defer fl.Unlock()

	reloaded, err := operation.Load(op.Squad, op.OperationID)
	if err != nil || reloaded.Status != "active" {
		return
	}

	now := time.Now().UTC()
	reportExists := platform.FileExists(filepath.Join(opDir, "report.html"))
	opCompleted := platform.FileContains(layout.OperationMDPath(reloaded.Squad, reloaded.OperationID), "status: completed")

	if reportExists && opCompleted {
		reloaded.Status = "completed"
		reloaded.CompletedAt = &now
		reloaded.PID = 0
	} else {
		reason := "process_exited_without_completion"
		reloaded.Status = "failed"
		reloaded.FailedAt = &now
		reloaded.FailureReason = &reason
		reloaded.PID = 0
	}
	if err := operation.Save(reloaded); err != nil {
		log.Printf("[scan] %s: failed to save collected state: %v", reloaded.OperationID, err)
	}

	if c.Notifier != nil && reloaded.Status != "completed" {
		msg := "Operation " + reloaded.OperationID + " failed"
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


