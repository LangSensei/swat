package pipeline

import (
	"log"
	"path/filepath"
	"time"

	"github.com/LangSensei/swat/commander/notify"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/platform"
)

// HandleActive checks an active operation and updates its status if the process has exited.
func HandleActive(op *operation.Operation, store *operation.Store, notifier notify.Notifier) {
	// Process still running — nothing to do
	if op.PID > 0 && platform.ProcessAlive(op.PID) {
		return
	}

	// Process exited — check if Captain completed the operation
	now := time.Now().UTC()
	opDir := store.OperationDir(op.Squad, op.OperationID)
	reportExists := platform.FileExists(filepath.Join(opDir, "report.html"))
	opCompleted := platform.FileContains(store.OperationMDPath(op.Squad, op.OperationID), "status: completed")

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
	store.Save(op)

	// Send desktop notification on abnormal exit (crash/timeout/fail).
	// Successful completions rely on the Operator calling swat_notify during debrief.
	if notifier != nil && op.Status != "completed" {
		msg := "Operation " + op.OperationID + " failed"
		if err := notifier.Notify(msg); err != nil {
			log.Printf("[collect] notify error: %v", err)
		}
	}
}
