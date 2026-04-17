package intake

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/LangSensei/swat/commander/layout"
	"github.com/gofrs/flock"
)

// Immediate represents a one-shot task in the intake queue.
type Immediate struct {
	Type        string    `json:"type"`
	ID          string    `json:"id"`
	Brief       string    `json:"brief"`
	Details     string    `json:"details,omitempty"`
	OperationID string    `json:"operation_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// Recurring represents a recurring scheduled task in the intake queue.
type Recurring struct {
	Type     string     `json:"type"`
	ID       string     `json:"id"`
	Cron     string     `json:"cron"`
	Timezone string     `json:"timezone,omitempty"`
	Brief    string     `json:"brief"`
	Details  string     `json:"details,omitempty"`
	Enabled  bool       `json:"enabled"`
	LastRun  *time.Time `json:"last_run,omitempty"`
	NextRun  *time.Time `json:"next_run,omitempty"`
}

// Entry is a union type returned by List — callers switch on Type().
type Entry struct {
	Immediate *Immediate
	Recurring *Recurring
}

// Type returns "immediate" or "recurring".
func (e *Entry) Type() string {
	if e.Immediate != nil {
		return "immediate"
	}
	return "recurring"
}

// ID returns the entry's unique identifier.
func (e *Entry) ID() string {
	if e.Immediate != nil {
		return e.Immediate.ID
	}
	return e.Recurring.ID
}

// envelope is used for initial JSON deserialization to determine type.
type envelope struct {
	Type string `json:"type"`
}

// DispatchFunc is a callback for dispatching a recurring task.
// Creates a new operation and returns its ID.
type DispatchFunc func(brief, details string) (operationID string, err error)

// ProcessFunc is a callback for processing an existing immediate task.
// Receives the operation ID that was already created by Dispatch().
type ProcessFunc func(operationID string) error

// CreateImmediate writes an immediate intake entry.
func CreateImmediate(brief, details, operationID string) error {
	item := &Immediate{
		Type:        "immediate",
		ID:          operationID,
		Brief:       brief,
		Details:     details,
		OperationID: operationID,
		CreatedAt:   time.Now().UTC(),
	}
	return writeWithLock(item.ID, item)
}

// CreateRecurring creates a new recurring intake entry.
func CreateRecurring(brief, details, cronExpr, tz string, immediate bool) (*Recurring, error) {
	if brief == "" {
		return nil, fmt.Errorf("brief is required")
	}
	if cronExpr == "" {
		return nil, fmt.Errorf("cron expression is required")
	}
	if err := ValidateCron(cronExpr); err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}

	b := make([]byte, 4)
	rand.Read(b)
	id := hex.EncodeToString(b)

	now := time.Now().UTC()
	var next *time.Time
	if immediate {
		next = &now
	} else {
		next = NextCronTime(cronExpr, now, loc)
	}

	item := &Recurring{
		Type:     "recurring",
		ID:       id,
		Cron:     cronExpr,
		Timezone: tz,
		Brief:    brief,
		Details:  details,
		Enabled:  true,
		NextRun:  next,
	}

	if err := writeWithLock(item.ID, item); err != nil {
		return nil, err
	}
	return item, nil
}

// Delete removes an intake entry by ID.
func Delete(id string) error {
	p := jsonPath(id)
	lp := lockPath(id)

	fl := flock.New(lp)
	if err := fl.Lock(); err != nil {
		return fmt.Errorf("lock %s: %w", id, err)
	}
	defer fl.Unlock()

	if _, err := os.Stat(p); os.IsNotExist(err) {
		return fmt.Errorf("intake entry %q not found", id)
	}

	if err := os.Remove(p); err != nil {
		return err
	}
	// Best-effort cleanup of lock file
	os.Remove(lp)
	return nil
}

// List returns all intake entries sorted by type (recurring first) then by next_run.
func List() ([]*Entry, error) {
	dir := layout.IntakeDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []*Entry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		entry, err := loadEntry(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool {
		// Recurring before immediate
		if result[i].Type() != result[j].Type() {
			return result[i].Type() == "recurring"
		}
		// Within recurring, sort by next_run
		if result[i].Recurring != nil && result[j].Recurring != nil {
			ri, rj := result[i].Recurring, result[j].Recurring
			if ri.NextRun == nil {
				return false
			}
			if rj.NextRun == nil {
				return true
			}
			return ri.NextRun.Before(*rj.NextRun)
		}
		return false
	})
	return result, nil
}

// ProcessDue processes all due entries in the intake queue.
// Immediate entries are processed via processFn then deleted.
// Recurring entries are dispatched via dispatchFn if due, then updated with next_run.
func ProcessDue(dispatchFn DispatchFunc, processFn ProcessFunc) {
	dir := layout.IntakeDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	now := time.Now().UTC()

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(e.Name(), ".json")
		lp := lockPath(id)
		fl := flock.New(lp)

		locked, err := fl.TryLock()
		if err != nil || !locked {
			continue // another process holds this entry
		}

		entry, err := loadEntry(filepath.Join(dir, e.Name()))
		if err != nil {
			fl.Unlock()
			continue
		}

		switch {
		case entry.Immediate != nil:
			processImmediate(entry.Immediate, processFn, fl)
		case entry.Recurring != nil:
			processRecurring(entry.Recurring, dispatchFn, now, fl)
		default:
			fl.Unlock()
		}
	}
}

func processImmediate(item *Immediate, processFn ProcessFunc, fl *flock.Flock) {
	defer fl.Unlock()

	if err := processFn(item.OperationID); err != nil {
		log.Printf("[intake] immediate %s: process failed: %v", item.ID, err)
		return
	}

	// Delete the intake file after successful processing
	p := jsonPath(item.ID)
	if err := os.Remove(p); err != nil {
		log.Printf("[intake] immediate %s: failed to remove intake file: %v", item.ID, err)
	}
	// Best-effort cleanup of lock file
	os.Remove(lockPath(item.ID))
}

func processRecurring(item *Recurring, dispatch DispatchFunc, now time.Time, fl *flock.Flock) {
	defer fl.Unlock()

	if !item.Enabled || item.NextRun == nil {
		return
	}
	if item.NextRun.After(now) {
		return
	}

	_, err := dispatch(item.Brief, item.Details)
	if err != nil {
		log.Printf("[intake] recurring %s: dispatch failed: %v", item.ID, err)
		return
	}

	loc, _ := time.LoadLocation(item.Timezone)
	if loc == nil {
		loc = time.UTC
	}
	item.LastRun = &now
	item.NextRun = NextCronTime(item.Cron, now, loc)
	if err := saveEntry(item.ID, item); err != nil {
		log.Printf("[intake] recurring %s: failed to save updated state: %v", item.ID, err)
	}
}

// MigrateFromSchedules migrates existing schedule files to intake format.
// Called once during initialization.
func MigrateFromSchedules() {
	schedDir := layout.SchedulesDir()
	intakeDir := layout.IntakeDir()

	// Only migrate if schedules dir exists and intake dir doesn't
	if _, err := os.Stat(schedDir); os.IsNotExist(err) {
		return
	}

	if err := os.MkdirAll(intakeDir, 0755); err != nil {
		log.Printf("[intake] migration: failed to create intake dir: %v", err)
		return
	}

	entries, err := os.ReadDir(schedDir)
	if err != nil {
		log.Printf("[intake] migration: failed to read schedules dir: %v", err)
		return
	}

	migrated := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		srcPath := filepath.Join(schedDir, e.Name())
		data, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}

		// Parse existing schedule and add type field
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		raw["type"] = "recurring"

		newData, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			continue
		}

		dstPath := filepath.Join(intakeDir, e.Name())
		if err := os.WriteFile(dstPath, newData, 0644); err != nil {
			continue
		}
		migrated++
	}

	if migrated > 0 {
		log.Printf("[intake] migration: migrated %d schedule(s) to intake/", migrated)
		// Remove old schedules directory after successful migration
		os.RemoveAll(schedDir)
	}
}

// --- internal helpers ---

func jsonPath(id string) string {
	return filepath.Join(layout.IntakeDir(), id+".json")
}

func lockPath(id string) string {
	return filepath.Join(layout.IntakeDir(), id+".lock")
}

func writeWithLock(id string, v interface{}) error {
	if err := os.MkdirAll(layout.IntakeDir(), 0755); err != nil {
		return err
	}

	fl := flock.New(lockPath(id))
	if err := fl.Lock(); err != nil {
		return fmt.Errorf("lock %s: %w", id, err)
	}
	defer fl.Unlock()

	return saveEntry(id, v)
}

func saveEntry(id string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(jsonPath(id), data, 0644)
}

func loadEntry(path string) (*Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}

	switch env.Type {
	case "immediate":
		var item Immediate
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &Entry{Immediate: &item}, nil
	case "recurring":
		var item Recurring
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &Entry{Recurring: &item}, nil
	default:
		return nil, fmt.Errorf("unknown intake type: %q", env.Type)
	}
}
