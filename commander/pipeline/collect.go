package pipeline

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/platform"
)

// Collect checks if a finished operation completed successfully or failed,
// and updates its status accordingly.
func Collect(op *operation.Operation) error {
	opDir := layout.OperationDir(op.Squad, op.OperationID)
	reportExists := platform.FileExists(filepath.Join(opDir, "report.html"))
	opCompleted := platform.FileContains(layout.OperationMDPath(op.Squad, op.OperationID), "status: completed")

	now := time.Now().UTC()
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
		return fmt.Errorf("save collected state: %w", err)
	}
	return nil
}
