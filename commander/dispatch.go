package commander

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/pipeline"
	"github.com/LangSensei/swat/commander/pipeline/provision"
	"github.com/LangSensei/swat/commander/runtime"
)

// Dispatch creates a new operation and starts async classify+provision+launch.
func (c *Commander) Dispatch(brief, details string) (*operation.Operation, error) {
	now := time.Now().UTC()
	op := &operation.Operation{
		OperationID: GenerateOpID(),
		Brief:       brief,
		Details:     details,
		Status:      "queued",
		Source:      "user",
		CreatedAt:   now,
	}
	if err := operation.Create(op); err != nil {
		return nil, err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[dispatch] PANIC in processOperation %s: %v", op.OperationID, r)
				reason := fmt.Sprintf("panic: %v", r)
				op.Status = "failed"
				now := time.Now().UTC()
				op.FailedAt = &now
				op.FailureReason = &reason
				operation.Save(op)
			}
		}()
		c.processOperation(op)
	}()

	return op, nil
}

func (c *Commander) processOperation(op *operation.Operation) {
	log.Printf("[dispatch] processOperation started: %s", op.OperationID)

	rt, err := runtime.New(c.RuntimeName)
	if err != nil {
		c.failOperation(op, fmt.Sprintf("init runtime: %v", err))
		return
	}

	reloaded, destDir, err := pipeline.Classify(rt, op, c.Notifier)
	if err != nil {
		log.Printf("[dispatch] %s: classify failed: %v", op.OperationID, err)
		c.failOperation(op, err.Error())
		return
	}

	if err := provision.Run(rt, reloaded, destDir, layout.Root(), c.RuntimeName, c.NotifyBackend); err != nil {
		log.Printf("[dispatch] %s: provision failed: %v", op.OperationID, err)
		c.failOperation(reloaded, fmt.Sprintf("provision: %v", err))
		return
	}

	if err := provision.LaunchAgent(rt, reloaded, destDir); err != nil {
		log.Printf("[dispatch] %s: launch failed: %v", op.OperationID, err)
		c.failOperation(reloaded, fmt.Sprintf("launch: %v", err))
		return
	}
	log.Printf("[dispatch] %s: launched successfully (squad=%s)", op.OperationID, reloaded.Squad)
}

func (c *Commander) failOperation(op *operation.Operation, reason string) {
	now := time.Now().UTC()
	op.Status = "failed"
	op.FailedAt = &now
	op.FailureReason = &reason
	operation.Save(op)
}

// Cancel marks an operation as failed and kills the process if active.
func (c *Commander) Cancel(opID string) error {
	op, err := operation.Find(opID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	reason := "cancelled_by_user"

	if op.Status == "active" && op.PID > 0 {
		if p, err := os.FindProcess(op.PID); err == nil {
			p.Signal(os.Kill)
		}
	}

	op.Status = "failed"
	op.FailedAt = &now
	op.FailureReason = &reason
	return operation.Save(op)
}
