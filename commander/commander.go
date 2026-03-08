package commander

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Operation represents a task's metadata (operation.json)
type Operation struct {
	OperationID   string     `json:"operation_id"`
	Brief         string     `json:"brief"`
	Details       string     `json:"details,omitempty"`
	Squad         string     `json:"squad,omitempty"`
	SquadVersion  string     `json:"squad_version,omitempty"`
	Status        string     `json:"status"`
	PID           int        `json:"pid,omitempty"`
	Source        string     `json:"source"`
	SchedulerID   *string    `json:"scheduler_id,omitempty"`
	RetryCount    int        `json:"retry_count"`
	Notified      bool       `json:"notified"`
	CreatedAt     time.Time  `json:"created_at"`
	DispatchedAt  *time.Time `json:"dispatched_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	FailedAt      *time.Time `json:"failed_at,omitempty"`
	FailureReason *string    `json:"failure_reason,omitempty"`
}

// Schedule represents a recurring task definition
type Schedule struct {
	ID        string    `json:"id"`
	Brief     string    `json:"brief"`
	Details   string    `json:"details,omitempty"`
	Squad     string    `json:"squad,omitempty"`
	Cron      string    `json:"cron"`
	NextRun   time.Time `json:"next_run"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
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
	// Expand ~ to home dir
	if swatRoot[:2] == "~/" {
		home, _ := os.UserHomeDir()
		swatRoot = filepath.Join(home, swatRoot[2:])
	}
	return &Commander{
		SwatRoot:   swatRoot,
		RetryCount: make(map[string]int),
	}
}

// GenerateOpID creates a unique operation ID
func GenerateOpID() string {
	return fmt.Sprintf("%s-%04x", time.Now().UTC().Format("20060102"), time.Now().UnixNano()%0xFFFF)
}

// OperationsDir returns the path to operations/
func (c *Commander) OperationsDir() string {
	return filepath.Join(c.SwatRoot, "operations")
}

// OperationDir returns the path to a specific operation
func (c *Commander) OperationDir(opID string) string {
	return filepath.Join(c.OperationsDir(), opID)
}

// SaveOperation writes operation.json
func (c *Commander) SaveOperation(op *Operation) error {
	dir := c.OperationDir(op.OperationID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create operation dir: %w", err)
	}
	data, err := json.MarshalIndent(op, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal operation: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "operation.json"), data, 0644)
}

// LoadOperation reads operation.json
func (c *Commander) LoadOperation(opID string) (*Operation, error) {
	path := filepath.Join(c.OperationDir(opID), "operation.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var op Operation
	if err := json.Unmarshal(data, &op); err != nil {
		return nil, err
	}
	return &op, nil
}

// ListOperations returns all operations
func (c *Commander) ListOperations() ([]*Operation, error) {
	dir := c.OperationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ops []*Operation
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		op, err := c.LoadOperation(entry.Name())
		if err != nil {
			continue
		}
		ops = append(ops, op)
	}
	return ops, nil
}
