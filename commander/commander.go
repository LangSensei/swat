package commander

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Operation represents a task parsed from OPERATION.md frontmatter
type Operation struct {
	OperationID   string     `json:"operation_id"`
	Squad         string     `json:"squad,omitempty"`
	Status        string     `json:"status"`
	PID           int        `json:"pid,omitempty"`
	Source        string     `json:"source"`
	CreatedAt     time.Time  `json:"created_at"`
	DispatchedAt  *time.Time `json:"dispatched_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	FailedAt      *time.Time `json:"failed_at,omitempty"`
	FailureReason *string    `json:"failure_reason,omitempty"`
	Summary       string     `json:"summary,omitempty"`
	References    []Reference `json:"references,omitempty"`
	// Brief and Details are stored in the markdown body, not frontmatter
	Brief   string `json:"brief,omitempty"`
	Details string `json:"details,omitempty"`
}

// Reference is a typed reference attached to an operation
type Reference struct {
	Type  string `json:"type"`  // operation, url, email-address, stock-code, etc.
	Value string `json:"value"`
}

// Schedule represents a recurring task definition
type Schedule struct {
	ID        string
	Brief     string
	Details   string
	Squad     string
	Cron      string
	NextRun   time.Time
	Enabled   bool
	CreatedAt time.Time
}

// Commander is the core orchestrator
type Commander struct {
	SwatRoot       string
	Iteration      int
	RecentFailures int
	RetryCount     map[string]int
}

// New creates a new Commander instance
func New(swatRoot string) *Commander {
	if len(swatRoot) >= 2 && swatRoot[:2] == "~/" {
		home, _ := os.UserHomeDir()
		swatRoot = filepath.Join(home, swatRoot[2:])
	}
	return &Commander{
		SwatRoot:   swatRoot,
		RetryCount: make(map[string]int),
	}
}

// GenerateOpID creates a unique operation ID (date-8hex)
func GenerateOpID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s-%08x", time.Now().UTC().Format("20060102"), b)
}

// StagingDir returns the path to the staging area
func (c *Commander) StagingDir() string {
	return filepath.Join(c.SwatRoot, "_staging")
}

// StagingOperationDir returns the path to an operation in staging
func (c *Commander) StagingOperationDir(opID string) string {
	return filepath.Join(c.StagingDir(), "operations", opID)
}

// SquadDir returns the path to a squad's runtime directory
func (c *Commander) SquadDir(squad string) string {
	return filepath.Join(c.SwatRoot, "squads", squad)
}

// OperationDir returns the path to a specific operation within its squad
func (c *Commander) OperationDir(squad, opID string) string {
	return filepath.Join(c.SquadDir(squad), "operations", opID)
}

// OperationMDPath returns the path to OPERATION.md for an operation
func (c *Commander) OperationMDPath(squad, opID string) string {
	return filepath.Join(c.OperationDir(squad, opID), "OPERATION.md")
}

// StagingOperationMDPath returns the path to OPERATION.md in staging
func (c *Commander) StagingOperationMDPath(opID string) string {
	return filepath.Join(c.StagingOperationDir(opID), "OPERATION.md")
}

// CreateOperation writes a new OPERATION.md with full template
func (c *Commander) CreateOperation(op *Operation) error {
	var dir, mdPath string
	if op.Squad == "" {
		dir = c.StagingOperationDir(op.OperationID)
		mdPath = c.StagingOperationMDPath(op.OperationID)
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
		mdPath = c.StagingOperationMDPath(op.OperationID)
	} else {
		mdPath = c.OperationMDPath(op.Squad, op.OperationID)
	}

	// If file doesn't exist, create it
	if !fileExists(mdPath) {
		return c.CreateOperation(op)
	}

	// Patch individual fields
	patches := map[string]string{
		"operation_id":   op.OperationID,
		"squad":          op.Squad,
		"source":         op.Source,
		"pid":            fmt.Sprintf("%d", op.PID),
		"status":         op.Status,
		"created_at":     op.CreatedAt.Format(time.RFC3339),
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

// LoadStagingOperation reads and parses OPERATION.md from staging
func (c *Commander) LoadStagingOperation(opID string) (*Operation, error) {
	path := c.StagingOperationMDPath(opID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseOperationMD(string(data))
}

// ListOperations returns all operations across all squads (and staging)
func (c *Commander) ListOperations() ([]*Operation, error) {
	var ops []*Operation

	// Scan staging
	stagingOpsDir := filepath.Join(c.StagingDir(), "operations")
	if entries, err := os.ReadDir(stagingOpsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if op, err := c.LoadStagingOperation(e.Name()); err == nil {
				ops = append(ops, op)
			}
		}
	}

	// Scan squads
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

// --- OPERATION.md serialization ---

func buildOperationFile(op *Operation) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("# Commander fields (written at dispatch, do not modify)\n")
	sb.WriteString("# format: YYYYMMDD-8hex (e.g., 20260212-a1b2c3d4)\n")
	sb.WriteString(fmt.Sprintf("operation_id: %s\n", op.OperationID))
	sb.WriteString("# filled by classify (Copilot)\n")
	sb.WriteString(fmt.Sprintf("squad: %s\n", op.Squad))
	sb.WriteString("# who initiated this operation (user | schedule | system)\n")
	sb.WriteString(fmt.Sprintf("source: %s\n", op.Source))
	sb.WriteString("# written by Commander after launch\n")
	if op.PID > 0 {
		sb.WriteString(fmt.Sprintf("pid: %d\n", op.PID))
	} else {
		sb.WriteString("pid:\n")
	}
	sb.WriteString("# UTC timestamp when operation was created\n")
	sb.WriteString(fmt.Sprintf("created_at: %s\n", op.CreatedAt.Format(time.RFC3339)))
	sb.WriteString("# UTC timestamp when Copilot CLI was launched\n")
	writeOptionalTime(&sb, "dispatched_at", op.DispatchedAt)
	sb.WriteString("# UTC timestamp when operation failed\n")
	writeOptionalTime(&sb, "failed_at", op.FailedAt)
	sb.WriteString("# filled if status is failed\n")
	writeOptionalStr(&sb, "failure_reason", op.FailureReason)
	sb.WriteString("# filled by classify (Copilot)\n")
	sb.WriteString("# e.g., [{type: \"operation\", value: \"../20260309-xxxx/\"}, {type: \"url\", value: \"https://...\"}]\n")
	if len(op.References) > 0 {
		sb.WriteString("references:\n")
		for _, ref := range op.References {
			sb.WriteString(fmt.Sprintf("  - {type: \"%s\", value: \"%s\"}\n", ref.Type, ref.Value))
		}
	} else {
		sb.WriteString("references: []\n")
	}

	sb.WriteString("\n# Captain output fields (filled during/after execution)\n")
	sb.WriteString("# queued → active → completed / failed\n")
	sb.WriteString(fmt.Sprintf("status: %s\n", op.Status))
	sb.WriteString("# 2-3 sentence summary of outcome\n")
	writeOptionalStr(&sb, "summary", nilIfEmpty(op.Summary))
	sb.WriteString("# UTC timestamp when operation completed successfully\n")
	writeOptionalTime(&sb, "completed_at", op.CompletedAt)
	sb.WriteString("---\n\n")

	// Body
	briefTitle := op.Brief
	if len(briefTitle) > 80 {
		briefTitle = briefTitle[:80] + "..."
	}
	if idx := strings.Index(briefTitle, "\n"); idx > 0 {
		briefTitle = briefTitle[:idx]
	}
	sb.WriteString(fmt.Sprintf("# %s\n", briefTitle))
	sb.WriteString("<!-- Commander: extracted from brief — do not modify -->\n\n")

	sb.WriteString("## Assignment\n")
	sb.WriteString("<!-- Commander: full operation description — do not modify -->\n")
	sb.WriteString(op.Brief + "\n")
	if op.Details != "" {
		sb.WriteString("\n" + op.Details + "\n")
	}

	sb.WriteString("\n## Summary\n")
	sb.WriteString("<!-- Captain: write a rich summary of findings and outcome -->\n\n")

	sb.WriteString("## Findings\n")
	sb.WriteString("<!-- Captain: key discoveries, root cause, impact, affected environments -->\n\n")

	sb.WriteString("## Action Items\n")
	sb.WriteString("<!-- Captain: concrete recommendations and next steps -->\n")

	return sb.String()
}

func writeOptionalTime(sb *strings.Builder, key string, t *time.Time) {
	if t != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, t.Format(time.RFC3339)))
	} else {
		sb.WriteString(fmt.Sprintf("%s:\n", key))
	}
}

func writeOptionalStr(sb *strings.Builder, key string, s *string) {
	if s != nil && *s != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, *s))
	} else {
		sb.WriteString(fmt.Sprintf("%s:\n", key))
	}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseOperationMD(content string) (*Operation, error) {
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("missing frontmatter")
	}
	end := strings.Index(content[3:], "\n---")
	if end < 0 {
		return nil, fmt.Errorf("unterminated frontmatter")
	}
	fm := content[4 : end+3]
	body := content[end+7:] // after closing ---

	op := &Operation{}
	scanner := bufio.NewScanner(strings.NewReader(fm))
	for scanner.Scan() {
		line := scanner.Text()
		// Skip YAML comments
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		switch key {
		case "operation_id":
			op.OperationID = val
		case "squad":
			op.Squad = val
		case "status":
			op.Status = val
		case "source":
			op.Source = val
		case "pid":
			fmt.Sscanf(val, "%d", &op.PID)
		case "created_at":
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				op.CreatedAt = t
			}
		case "dispatched_at":
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				op.DispatchedAt = &t
			}
		case "completed_at":
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				op.CompletedAt = &t
			}
		case "failed_at":
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				op.FailedAt = &t
			}
		case "failure_reason":
			if val != "" {
				op.FailureReason = &val
			}
		case "summary":
			op.Summary = val
		}
	}

	// Parse brief from body: first # heading's content under ## Assignment
	op.Brief = extractBodySection(body, "Assignment")

	if op.OperationID == "" {
		return nil, fmt.Errorf("missing operation_id in frontmatter")
	}
	return op, nil
}

// extractBodySection extracts the content under a ## heading
func extractBodySection(body, heading string) string {
	marker := "## " + heading
	idx := strings.Index(body, marker)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(marker):]
	// Skip the heading line
	if nl := strings.Index(rest, "\n"); nl >= 0 {
		rest = rest[nl+1:]
	}
	// Skip HTML comments
	for strings.HasPrefix(strings.TrimSpace(rest), "<!--") {
		if end := strings.Index(rest, "-->"); end >= 0 {
			rest = rest[end+3:]
			rest = strings.TrimLeft(rest, "\n")
		} else {
			break
		}
	}
	// Read until next ## or end
	if next := strings.Index(rest, "\n## "); next >= 0 {
		rest = rest[:next]
	}
	return strings.TrimSpace(rest)
}

// findOperation locates an operation by ID across all squads and staging
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

// ListSquads returns all installed squad blueprints
func (c *Commander) ListSquads() ([]map[string]string, error) {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	entries, err := os.ReadDir(filepath.Join(bpDir, "squads"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var squads []map[string]string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "_framework" {
			continue
		}
		info := map[string]string{"name": entry.Name()}
		manifestPath := filepath.Join(bpDir, "squads", entry.Name(), "MANIFEST.md")
		if data, err := os.ReadFile(manifestPath); err == nil {
			if desc := extractFrontmatterField(string(data), "description"); desc != "" {
				info["description"] = desc
			}
		}
		squads = append(squads, info)
	}
	return squads, nil
}
