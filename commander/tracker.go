package commander

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TrackedOp holds Commander-internal state for a dispatched operation
type TrackedOp struct {
	Squad         string     `json:"squad"`
	PID           int        `json:"pid"`
	Notified      bool       `json:"notified"`
	RetryCount    int        `json:"retry_count"`
	DispatchedAt  *time.Time `json:"dispatched_at,omitempty"`
	FailedAt      *time.Time `json:"failed_at,omitempty"`
	FailureReason *string    `json:"failure_reason,omitempty"`
}

// Tracker manages dispatched.json — Commander's internal state file
type Tracker struct {
	mu   sync.Mutex
	path string
	Ops  map[string]*TrackedOp `json:"ops"`
}

// NewTracker creates a tracker backed by dispatched.json in swatRoot
func NewTracker(swatRoot string) *Tracker {
	return &Tracker{
		path: filepath.Join(swatRoot, "dispatched.json"),
		Ops:  make(map[string]*TrackedOp),
	}
}

// Load reads dispatched.json from disk
func (t *Tracker) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := os.ReadFile(t.path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Ops = make(map[string]*TrackedOp)
			return nil
		}
		return err
	}
	wrapper := struct {
		Ops map[string]*TrackedOp `json:"ops"`
	}{}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	t.Ops = wrapper.Ops
	if t.Ops == nil {
		t.Ops = make(map[string]*TrackedOp)
	}
	return nil
}

// Save writes dispatched.json to disk
func (t *Tracker) Save() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	wrapper := struct {
		Ops map[string]*TrackedOp `json:"ops"`
	}{Ops: t.Ops}
	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.path, data, 0644)
}

// Track adds or updates a tracked operation
func (t *Tracker) Track(opID string, tracked *TrackedOp) error {
	t.mu.Lock()
	t.Ops[opID] = tracked
	t.mu.Unlock()
	return t.Save()
}

// Get returns the tracked state for an operation
func (t *Tracker) Get(opID string) *TrackedOp {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Ops[opID]
}

// Remove deletes a tracked operation
func (t *Tracker) Remove(opID string) error {
	t.mu.Lock()
	delete(t.Ops, opID)
	t.mu.Unlock()
	return t.Save()
}

// SetNotified marks a tracked operation as notified
func (t *Tracker) SetNotified(opID string) error {
	t.mu.Lock()
	if op, ok := t.Ops[opID]; ok {
		op.Notified = true
	}
	t.mu.Unlock()
	return t.Save()
}

// SetFailed marks a tracked operation as failed
func (t *Tracker) SetFailed(opID string, reason string) error {
	t.mu.Lock()
	if op, ok := t.Ops[opID]; ok {
		now := time.Now().UTC()
		op.FailedAt = &now
		op.FailureReason = &reason
		op.PID = 0
	}
	t.mu.Unlock()
	return t.Save()
}
