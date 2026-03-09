package commander

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CreateOperation writes a new OPERATION.md with full template
func (c *Commander) CreateOperation(op *Operation) error {
	var dir, mdPath string
	if op.Squad == "" {
		dir = c.UnclassifiedOperationDir(op.OperationID)
		mdPath = c.UnclassifiedOperationMDPath(op.OperationID)
	} else {
		dir = c.OperationDir(op.Squad, op.OperationID)
		mdPath = c.OperationMDPath(op.Squad, op.OperationID)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create operation dir: %w", err)
	}
	md := buildOperationFile(op)
	return os.WriteFile(mdPath, []byte(md), 0644)
}

// SaveOperation updates OPERATION.md using field-level patching (preserves comments and body)
func (c *Commander) SaveOperation(op *Operation) error {
	var mdPath string
	if op.Squad == "" {
		mdPath = c.UnclassifiedOperationMDPath(op.OperationID)
	} else {
		mdPath = c.OperationMDPath(op.Squad, op.OperationID)
	}

	// If file doesn't exist, create it
	if !fileExists(mdPath) {
		return c.CreateOperation(op)
	}

	// Patch individual fields
	patches := map[string]string{
		"operation_id": op.OperationID,
		"squad":        op.Squad,
		"source":       op.Source,
		"pid":          fmt.Sprintf("%d", op.PID),
		"status":       op.Status,
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

	return patchFrontmatterFields(mdPath, patches)
}

// patchFrontmatterFields updates specific key-value pairs in YAML frontmatter
func patchFrontmatterFields(path string, patches map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return fmt.Errorf("missing frontmatter")
	}

	end := strings.Index(content[3:], "\n---")
	if end < 0 {
		return fmt.Errorf("unterminated frontmatter")
	}

	fm := content[4 : end+3]
	body := content[end+7:]

	// Patch each field in frontmatter
	lines := strings.Split(fm, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if val, ok := patches[key]; ok {
			lines[i] = key + ": " + val
		}
	}

	result := "---\n" + strings.Join(lines, "\n") + "\n---" + body
	return os.WriteFile(path, []byte(result), 0644)
}

// LoadOperation reads and parses OPERATION.md
func (c *Commander) LoadOperation(squad, opID string) (*Operation, error) {
	path := c.OperationMDPath(squad, opID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseOperationMD(string(data))
}

// LoadUnclassifiedOperation reads and parses OPERATION.md from unclassified
func (c *Commander) LoadUnclassifiedOperation(opID string) (*Operation, error) {
	path := c.UnclassifiedOperationMDPath(opID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseOperationMD(string(data))
}

// ListOperations returns all operations across all squads (including _unclassified)
func (c *Commander) ListOperations() ([]*Operation, error) {
	var ops []*Operation

	squadsDir := filepath.Join(c.SwatRoot, "squads")
	squadEntries, err := os.ReadDir(squadsDir)
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
		opsDir := filepath.Join(squadsDir, se.Name(), "operations")
		opEntries, err := os.ReadDir(opsDir)
		if err != nil {
			continue
		}
		for _, oe := range opEntries {
			if !oe.IsDir() {
				continue
			}
			op, err := c.LoadOperation(se.Name(), oe.Name())
			if err != nil {
				continue
			}
			ops = append(ops, op)
		}
	}
	return ops, nil
}

// findOperation locates an operation by ID across all squads
func (c *Commander) findOperation(opID string) (*Operation, error) {
	ops, err := c.ListOperations()
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
