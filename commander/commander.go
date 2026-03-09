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
	Squad         string     `json:"squad"`
	Brief         string     `json:"brief"`
	Details       string     `json:"details,omitempty"`
	Status        string     `json:"status"`
	PID           int        `json:"pid,omitempty"`
	RetryCount    int        `json:"retry_count"`
	Source        string     `json:"source"`
	CreatedAt     time.Time  `json:"created_at"`
	DispatchedAt  *time.Time `json:"dispatched_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	FailedAt      *time.Time `json:"failed_at,omitempty"`
	FailureReason *string    `json:"failure_reason,omitempty"`
	Summary       string     `json:"summary,omitempty"`
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

// SaveOperation writes OPERATION.md with frontmatter + body
func (c *Commander) SaveOperation(op *Operation) error {
	dir := c.OperationDir(op.Squad, op.OperationID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create operation dir: %w", err)
	}
	md := buildOperationFile(op)
	return os.WriteFile(c.OperationMDPath(op.Squad, op.OperationID), []byte(md), 0644)
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

// ListOperations returns all operations across all squads
func (c *Commander) ListOperations() ([]*Operation, error) {
	squadsDir := filepath.Join(c.SwatRoot, "squads")
	squadEntries, err := os.ReadDir(squadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ops []*Operation
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
	sb.WriteString(fmt.Sprintf("operation_id: %s\n", op.OperationID))
	sb.WriteString(fmt.Sprintf("squad: %s\n", op.Squad))
	sb.WriteString(fmt.Sprintf("brief: %s\n", op.Brief))
	sb.WriteString(fmt.Sprintf("status: %s\n", op.Status))
	sb.WriteString(fmt.Sprintf("source: %s\n", op.Source))
	sb.WriteString(fmt.Sprintf("pid: %d\n", op.PID))
	sb.WriteString(fmt.Sprintf("retry_count: %d\n", op.RetryCount))
	sb.WriteString(fmt.Sprintf("created_at: %s\n", op.CreatedAt.Format(time.RFC3339)))
	writeOptionalTime(&sb, "dispatched_at", op.DispatchedAt)
	writeOptionalTime(&sb, "completed_at", op.CompletedAt)
	writeOptionalTime(&sb, "failed_at", op.FailedAt)
	writeOptionalStr(&sb, "failure_reason", op.FailureReason)
	writeOptionalStr(&sb, "summary", nilIfEmpty(op.Summary))
	sb.WriteString("references: []\n")
	sb.WriteString("---\n\n")
	sb.WriteString("# OPERATION\n\n")
	sb.WriteString("## Task\n\n")
	sb.WriteString(op.Brief + "\n")
	if op.Details != "" {
		sb.WriteString("\n## Details\n\n")
		sb.WriteString(op.Details + "\n")
	}
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

	op := &Operation{}
	scanner := bufio.NewScanner(strings.NewReader(fm))
	for scanner.Scan() {
		line := scanner.Text()
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
		case "brief":
			op.Brief = val
		case "status":
			op.Status = val
		case "source":
			op.Source = val
		case "pid":
			fmt.Sscanf(val, "%d", &op.PID)
		case "retry_count":
			fmt.Sscanf(val, "%d", &op.RetryCount)
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
	if op.OperationID == "" {
		return nil, fmt.Errorf("missing operation_id in frontmatter")
	}
	return op, nil
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
