package commander

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/LangSensei/swat/commander/intake"
	"github.com/LangSensei/swat/commander/operation"
)

// Dispatch creates a new operation and queues it for processing via the intake queue.
func (c *Commander) Dispatch(brief, details string) (*operation.Operation, error) {
	now := time.Now().UTC()
	op := &operation.Operation{
		OperationID: GenerateOpID(),
		Brief:       brief,
		Details:     details,
		Status:      "queued",
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

func (c *Commander) failOperation(op *operation.Operation, reason string) error {
	now := time.Now().UTC()
	op.Status = "failed"
	op.FailedAt = &now
	op.FailureReason = &reason
	op.PID = 0
	if err := operation.Save(op); err != nil {
		log.Printf("[dispatch] %s: failed to save failure state: %v", op.OperationID, err)
		return fmt.Errorf("save failure state: %w", err)
	}

	if c.Notifier != nil {
		msg := fmt.Sprintf("Operation %s failed: %s", op.OperationID, reason)
		if err := c.Notifier.Notify(op.OperationID, msg); err != nil {
			log.Printf("[dispatch] %s: notify error: %v", op.OperationID, err)
		}
	}

	return nil
}

// Cancel marks an operation as failed and kills the process if active or classifying.
func (c *Commander) Cancel(opID string) error {
	op, err := operation.Find(opID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	reason := "cancelled_by_user"

	if (op.Status == "active" || op.Status == "classifying") && op.PID > 0 {
		if p, err := os.FindProcess(op.PID); err == nil {
			p.Signal(os.Kill)
		}
	}

	op.Status = "failed"
	op.FailedAt = &now
	op.FailureReason = &reason
	op.PID = 0
	return operation.Save(op)
}
