package commander

import (
	"time"
)

// Dispatch creates a new operation
func (c *Commander) Dispatch(brief, details, squad string) (*Operation, error) {
	now := time.Now().UTC()
	op := &Operation{
		OperationID: GenerateOpID(),
		Brief:       brief,
		Details:     details,
		Squad:       squad,
		Status:      "queued",
		Source:      "user",
		CreatedAt:   now,
	}
	if err := c.SaveOperation(op); err != nil {
		return nil, err
	}
	return op, nil
}

// Cancel marks an operation as failed
func (c *Commander) Cancel(opID string) error {
	op, err := c.LoadOperation(opID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	reason := "cancelled_by_user"
	op.Status = "failed"
	op.FailedAt = &now
	op.FailureReason = &reason
	// TODO: kill PID if active
	return c.SaveOperation(op)
}
