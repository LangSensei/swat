package operation

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/LangSensei/swat/commander/platform"
)

// PathResolver provides directory/path lookups. Implemented by commander.Layout.
type PathResolver interface {
	OperationDir(squad, opID string) string
	OperationMDPath(squad, opID string) string
	UnclassifiedOperationDir(opID string) string
	UnclassifiedOperationMDPath(opID string) string
	SquadDir(squad string) string
}

// Create writes a new OPERATION.md from template.
func Create(paths PathResolver, blueprintsRoot string, op *Operation) error {
	var dir, mdPath string
	if op.Squad == "" {
		dir = paths.UnclassifiedOperationDir(op.OperationID)
		mdPath = paths.UnclassifiedOperationMDPath(op.OperationID)
	} else {
		dir = paths.OperationDir(op.Squad, op.OperationID)
		mdPath = paths.OperationMDPath(op.Squad, op.OperationID)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create operation dir: %w", err)
	}
	md, err := buildOperationFile(blueprintsRoot, op)
	if err != nil {
		return fmt.Errorf("build operation file: %w", err)
	}
	return os.WriteFile(mdPath, []byte(md), 0644)
}

// Save updates OPERATION.md using field-level patching (preserves comments and body).
func Save(paths PathResolver, blueprintsRoot string, op *Operation) error {
	var mdPath string
	if op.Squad == "" {
		mdPath = paths.UnclassifiedOperationMDPath(op.OperationID)
	} else {
		mdPath = paths.OperationMDPath(op.Squad, op.OperationID)
	}

	if !platform.FileExists(mdPath) {
		return Create(paths, blueprintsRoot, op)
	}

	patches := map[string]string{
		"operation_id": op.OperationID,
		"squad":        op.Squad,
		"source":       op.Source,
		"status":       op.Status,
		"created_at":   op.CreatedAt.Format(time.RFC3339),
	}
	if op.PID > 0 {
		patches["pid"] = fmt.Sprintf("%d", op.PID)
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

// Load reads and parses OPERATION.md for a classified operation.
func Load(paths PathResolver, squad, opID string) (*Operation, error) {
	path := paths.OperationMDPath(squad, opID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseOperationMD(string(data))
}

// LoadUnclassified reads and parses OPERATION.md from unclassified.
func LoadUnclassified(paths PathResolver, opID string) (*Operation, error) {
	path := paths.UnclassifiedOperationMDPath(opID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseOperationMD(string(data))
}

// List returns all operations across all squads (including _unclassified).
func List(paths PathResolver, squadsDir string) ([]*Operation, error) {
	var ops []*Operation

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
		opsDir := paths.OperationDir(se.Name(), "")
		// OperationDir returns squads/{squad}/operations/{opID}, we need the parent
		opsDir = paths.SquadDir(se.Name()) + "/operations"
		opEntries, err := os.ReadDir(opsDir)
		if err != nil {
			continue
		}
		for _, oe := range opEntries {
			if !oe.IsDir() {
				continue
			}
			op, err := Load(paths, se.Name(), oe.Name())
			if err != nil {
				continue
			}
			ops = append(ops, op)
		}
	}
	return ops, nil
}

// Find locates an operation by ID across all squads.
func Find(paths PathResolver, squadsDir, opID string) (*Operation, error) {
	ops, err := List(paths, squadsDir)
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

// patchFrontmatterFields updates specific key-value pairs in YAML frontmatter.
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
