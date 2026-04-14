package commander

import (
	"crypto/rand"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/notify"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/platform"
)

// Commander is the core orchestrator
type Commander struct {
	RuntimeName string
	Notify      string
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
		Notify:      notifyName,
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
		c.CheckDue()
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
	operation.Save(op)

	if c.Notifier != nil && op.Status != "completed" {
		msg := "Operation " + op.OperationID + " failed"
		if err := c.Notifier.Notify(msg); err != nil {
			log.Printf("[scan] notify error: %v", err)
		}
	}
}

