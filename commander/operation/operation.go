package operation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LangSensei/swat/commander/platform"
)

// Store provides CRUD operations for Operation and related files.
type Store struct {
	SwatRoot string
}

// NewStore creates a new operation Store.
func NewStore(swatRoot string) *Store {
	return &Store{SwatRoot: swatRoot}
}

// UnclassifiedOperationDir returns the directory for a specific unclassified operation.
func (s *Store) UnclassifiedOperationDir(opID string) string {
	return filepath.Join(s.SwatRoot, "squads", "_unclassified", "operations", opID)
}

// UnclassifiedOperationMDPath returns the OPERATION.md path for an unclassified operation.
func (s *Store) UnclassifiedOperationMDPath(opID string) string {
	return filepath.Join(s.UnclassifiedOperationDir(opID), "OPERATION.md")
}

// SquadDir returns the runtime directory for a squad.
func (s *Store) SquadDir(squad string) string {
	return filepath.Join(s.SwatRoot, "squads", squad)
}

// OperationDir returns the directory for a classified operation.
func (s *Store) OperationDir(squad, opID string) string {
	return filepath.Join(s.SwatRoot, "squads", squad, "operations", opID)
}

// OperationMDPath returns the OPERATION.md path for a classified operation.
func (s *Store) OperationMDPath(squad, opID string) string {
	return filepath.Join(s.OperationDir(squad, opID), "OPERATION.md")
}

// Create writes a new OPERATION.md with full template.
func (s *Store) Create(op *Operation) error {
	var dir, mdPath string
	if op.Squad == "" {
		dir = s.UnclassifiedOperationDir(op.OperationID)
		mdPath = s.UnclassifiedOperationMDPath(op.OperationID)
	} else {
		dir = s.OperationDir(op.Squad, op.OperationID)
		mdPath = s.OperationMDPath(op.Squad, op.OperationID)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create operation dir: %w", err)
	}
	md, err := s.buildOperationFile(op)
	if err != nil {
		return fmt.Errorf("build operation file: %w", err)
	}
	return os.WriteFile(mdPath, []byte(md), 0644)
}

// Save updates OPERATION.md using field-level patching (preserves comments and body).
func (s *Store) Save(op *Operation) error {
	var mdPath string
	if op.Squad == "" {
		mdPath = s.UnclassifiedOperationMDPath(op.OperationID)
	} else {
		mdPath = s.OperationMDPath(op.Squad, op.OperationID)
	}

	// If file doesn't exist, create it
	if !platform.FileExists(mdPath) {
		return s.Create(op)
	}

	// Patch individual fields
	patches := map[string]string{
		"operation_id": op.OperationID,
		"squad":        op.Squad,
		"source":       op.Source,
	}
	if op.PID > 0 {
		patches["pid"] = fmt.Sprintf("%d", op.PID)
	}
	patches["status"] = op.Status
	patches["created_at"] = op.CreatedAt.Format(time.RFC3339)
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
func (s *Store) Load(squad, opID string) (*Operation, error) {
	path := s.OperationMDPath(squad, opID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseOperationMD(string(data))
}

// LoadUnclassified reads and parses OPERATION.md from unclassified.
func (s *Store) LoadUnclassified(opID string) (*Operation, error) {
	path := s.UnclassifiedOperationMDPath(opID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseOperationMD(string(data))
}

// List returns all operations across all squads (including _unclassified).
func (s *Store) List() ([]*Operation, error) {
	var ops []*Operation

	squadsDir := filepath.Join(s.SwatRoot, "squads")
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
			op, err := s.Load(se.Name(), oe.Name())
			if err != nil {
				continue
			}
			ops = append(ops, op)
		}
	}
	return ops, nil
}

// Find locates an operation by ID across all squads.
func (s *Store) Find(opID string) (*Operation, error) {
	ops, err := s.List()
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
