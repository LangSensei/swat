package commander

import (
	"crypto/rand"
	"fmt"
	"log"
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
		log.Printf("[scan] list error: %v", err)
		return
	}

	rt, err := runtime.New(c.RuntimeName)
	if err != nil {
		log.Printf("[scan] init runtime: %v", err)
		return
	}

	for _, op := range ops {
		switch op.Status {
		case "queued":
			c.withLock(op, "queued", func(reloaded *operation.Operation) {
				if err := pipeline.SpawnClassify(rt, reloaded); err != nil {
					c.failOperation(reloaded, err.Error())
				}
			})
		case "classifying":
			if !platform.ProcessAlive(op.PID) {
				c.withLock(op, "classifying", func(reloaded *operation.Operation) {
					if err := pipeline.Advance(rt, reloaded, c.RuntimeName, c.NotifyName); err != nil {
						c.failOperation(reloaded, err.Error())
					}
				})
			}
		case "active":
			if op.PID > 0 && platform.ProcessAlive(op.PID) {
				continue
			}
			c.withLock(op, "active", func(reloaded *operation.Operation) {
				if err := pipeline.Collect(reloaded); err != nil {
					log.Printf("[scan] %s: collect save error: %v", reloaded.OperationID, err)
					return
				}
				// Collect sets status — if it's "failed", Collect already saved it,
				// but we still need to notify. Use Notifier directly since Save already done.
				if reloaded.Status == "failed" && c.Notifier != nil {
					reason := "process_exited_without_completion"
					msg := fmt.Sprintf("Operation %s failed: %s", reloaded.OperationID, reason)
					if err := c.Notifier.Notify(msg); err != nil {
						log.Printf("[scan] notify error: %v", err)
					}
				}
			})
		}
	}
}

// withLock acquires a per-operation flock, double-checks status, and calls fn.
// Recovers from panics to prevent BackgroundLoop from crashing.
func (c *Commander) withLock(op *operation.Operation, expectedStatus string, fn func(reloaded *operation.Operation)) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[scan] PANIC in %s: %v", op.OperationID, r)
			c.failOperation(op, fmt.Sprintf("panic: %v", r))
		}
	}()

	lockPath := layout.LockPath(op.OperationID)
	fl := flock.New(lockPath)
	locked, err := fl.TryLock()
	if !locked || err != nil {
		return
	}
	defer fl.Unlock()

	var reloaded *operation.Operation
	if expectedStatus == "queued" || expectedStatus == "classifying" {
		reloaded, err = operation.Load("_unclassified", op.OperationID)
	} else {
		reloaded, err = operation.Load(op.Squad, op.OperationID)
	}
	if err != nil || reloaded.Status != expectedStatus {
		return
	}

	fn(reloaded)
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


