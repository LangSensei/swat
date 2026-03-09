package commander

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Operation represents a task parsed from OPERATION.md frontmatter
type Operation struct {
	OperationID   string      `json:"operation_id"`
	Squad         string      `json:"squad,omitempty"`
	Status        string      `json:"status"`
	PID           int         `json:"pid,omitempty"`
	Source        string      `json:"source"`
	CreatedAt     time.Time   `json:"created_at"`
	DispatchedAt  *time.Time  `json:"dispatched_at,omitempty"`
	CompletedAt   *time.Time  `json:"completed_at,omitempty"`
	FailedAt      *time.Time  `json:"failed_at,omitempty"`
	FailureReason *string     `json:"failure_reason,omitempty"`
	Summary       string      `json:"summary,omitempty"`
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

// --- Path helpers ---

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
