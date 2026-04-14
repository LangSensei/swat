package operation

import "time"

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

// BrowseResult represents a squad available in the marketplace
type BrowseResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
}

// SkillPrereq represents a skill that has prerequisites needing user setup
type SkillPrereq struct {
	Skill string `json:"skill"`
	Path  string `json:"path"` // relative path within the skill dir, e.g. "references/setup.md"
}
