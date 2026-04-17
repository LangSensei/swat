package commander

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/LangSensei/swat/commander/intake"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/pipeline"
	"github.com/LangSensei/swat/commander/runtime"
)

// Dispatch creates a new operation and queues it for processing via the intake queue.
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

	if err := intake.CreateImmediate(brief, details, op.OperationID); err != nil {
		return nil, fmt.Errorf("create intake entry: %w", err)
	}

	return op, nil
}

// processOperation runs the classify → provision → launch pipeline for an operation.
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

	if err := pipeline.Provision(rt, reloaded, destDir, c.RuntimeName, c.NotifyName); err != nil {
		log.Printf("[dispatch] %s: provision failed: %v", op.OperationID, err)
		c.failOperation(reloaded, fmt.Sprintf("provision: %v", err))
		return
	}

	if err := pipeline.LaunchAgent(rt, reloaded, destDir); err != nil {
		log.Printf("[dispatch] %s: launch failed: %v", op.OperationID, err)
		c.failOperation(reloaded, fmt.Sprintf("launch: %v", err))
		return
	}
	log.Printf("[dispatch] %s: launched successfully (squad=%s)", op.OperationID, reloaded.Squad)
}

func (c *Commander) failOperation(op *operation.Operation, reason string) error {
	now := time.Now().UTC()
	op.Status = "failed"
	op.FailedAt = &now
	op.FailureReason = &reason
	if err := operation.Save(op); err != nil {
		log.Printf("[dispatch] %s: failed to save failure state: %v", op.OperationID, err)
		return fmt.Errorf("save failure state: %w", err)
	}
	return nil
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
