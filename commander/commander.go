package commander

import (
	"crypto/rand"
	"fmt"
	"log"
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
	ID        string     `json:"id"`
	Brief     string     `json:"brief"`
	Details   string     `json:"details,omitempty"`
	Cron      string     `json:"cron"`
	Timezone  string     `json:"timezone,omitempty"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	LastRun   *time.Time `json:"last_run,omitempty"`
	NextRun   *time.Time `json:"next_run,omitempty"`
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

	// Set up commander log file (daily rotation)
	logDir := filepath.Join(swatRoot, "logs")
	os.MkdirAll(logDir, 0755)
	logName := fmt.Sprintf("commander-%s.log", time.Now().UTC().Format("2006-01-02"))
	logPath := filepath.Join(logDir, logName)
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		log.SetOutput(logFile)
	}
	log.Printf("[commander] started, swatRoot=%s", swatRoot)

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

// UnclassifiedDir returns the path to the unclassified operations area
func (c *Commander) UnclassifiedDir() string {
	return filepath.Join(c.SwatRoot, "squads", "_unclassified")
}

// UnclassifiedOperationDir returns the path to an unclassified operation
func (c *Commander) UnclassifiedOperationDir(opID string) string {
	return filepath.Join(c.UnclassifiedDir(), "operations", opID)
}

// UnclassifiedOperationMDPath returns the path to OPERATION.md for an unclassified operation
func (c *Commander) UnclassifiedOperationMDPath(opID string) string {
	return filepath.Join(c.UnclassifiedOperationDir(opID), "OPERATION.md")
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
