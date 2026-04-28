package operation

import (
	"fmt"
	"os"
	"time"

	"github.com/LangSensei/swat/commander/deps"
	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/platform"
)

// Operation represents a task parsed from OPERATION.md frontmatter
type Operation struct {
	OperationID   string      `json:"operation_id" yaml:"operation_id"`
	Squad         string      `json:"squad,omitempty" yaml:"squad"`
	Status        string      `json:"status" yaml:"status"`
	PID           int         `json:"pid,omitempty" yaml:"pid"`
	CreatedAt     time.Time   `json:"created_at" yaml:"created_at"`
	DispatchedAt  *time.Time  `json:"dispatched_at,omitempty" yaml:"dispatched_at"`
	CompletedAt   *time.Time  `json:"completed_at,omitempty" yaml:"completed_at"`
	FailedAt      *time.Time  `json:"failed_at,omitempty" yaml:"failed_at"`
	FailureReason *string     `json:"failure_reason,omitempty" yaml:"failure_reason"`
	Summary       string      `json:"summary,omitempty" yaml:"summary"`
	References    []Reference `json:"references,omitempty" yaml:"references"`
	Brief         string      `json:"brief,omitempty" yaml:"-"`
	Details       string      `json:"details,omitempty" yaml:"-"`
}

// Reference is a typed reference attached to an operation
type Reference struct {
	Type  string `json:"type" yaml:"type"`
	Value string `json:"value" yaml:"value"`
}

// Create writes a new OPERATION.md from template.
func Create(op *Operation) error {
	var dir, mdPath string
	if op.Squad == "" {
		dir = layout.UnclassifiedOperationDir(op.OperationID)
		mdPath = layout.UnclassifiedOperationMDPath(op.OperationID)
	} else {
		dir = layout.OperationDir(op.Squad, op.OperationID)
		mdPath = layout.OperationMDPath(op.Squad, op.OperationID)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create operation dir: %w", err)
	}
	md, err := buildOperationFile(op)
	if err != nil {
		return fmt.Errorf("build operation file: %w", err)
	}
	return os.WriteFile(mdPath, []byte(md), 0644)
}

// Save updates OPERATION.md using field-level patching.
func Save(op *Operation) error {
	var mdPath string
	if op.Squad == "" {
		mdPath = layout.UnclassifiedOperationMDPath(op.OperationID)
	} else {
		mdPath = layout.OperationMDPath(op.Squad, op.OperationID)
	}

	if !platform.FileExists(mdPath) {
		return Create(op)
	}

	patches := map[string]string{
		"operation_id": op.OperationID,
		"squad":        op.Squad,
		"status":       op.Status,
		"pid":          fmt.Sprintf("%d", op.PID),
		"created_at":   op.CreatedAt.Format(time.RFC3339),
	}
	if op.DispatchedAt != nil {
		patches["dispatched_at"] = op.DispatchedAt.Format(time.RFC3339)
	}
	if op.CompletedAt != nil {
		patches["completed_at"] = op.CompletedAt.Format(time.RFC3339)
	}
	if op.FailedAt != nil {
		patches["failed_at"] = op.FailedAt.Format(time.RFC3339)
	}
	if op.FailureReason != nil {
		patches["failure_reason"] = *op.FailureReason
	}
	if op.Summary != "" {
		patches["summary"] = op.Summary
	}

	return deps.PatchFrontmatterFields(mdPath, patches)
}

// Load reads and parses OPERATION.md for a classified operation.
func Load(squad, opID string) (*Operation, error) {
	data, err := os.ReadFile(layout.OperationMDPath(squad, opID))
	if err != nil {
		return nil, err
	}
	return parseOperationMD(string(data))
}

// List returns all operations across all squads (including _unclassified).
func List() ([]*Operation, error) {
	var ops []*Operation

	squadEntries, err := os.ReadDir(layout.SquadsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return ops, nil
		}
		return ops, err
	}
	for _, se := range squadEntries {
		if !se.IsDir() {
			continue
		}
		opsDir := layout.OperationsDir(se.Name())
		opEntries, err := os.ReadDir(opsDir)
		if err != nil {
			continue
		}
		for _, oe := range opEntries {
			if !oe.IsDir() {
				continue
			}
			op, err := Load(se.Name(), oe.Name())
			if err != nil {
				continue
			}
			ops = append(ops, op)
		}
	}
	return ops, nil
}

// Find locates an operation by ID across all squads.
func Find(opID string) (*Operation, error) {
	ops, err := List()
	if err != nil {
		return nil, err
	}
	for _, op := range ops {
		if op.OperationID == opID {
			return op, nil
		}
	}
	return nil, fmt.Errorf("operation %s not found", opID)
}
