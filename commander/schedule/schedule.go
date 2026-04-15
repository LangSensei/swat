package schedule

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
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/platform"
)

// Schedule represents a recurring task definition.
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

// DispatchFunc is a callback for dispatching a scheduled task.
// Returns the source tag (e.g. operation ID) on success.
type DispatchFunc func(brief, details string) (sourceTag string, err error)

// Create creates a new schedule and persists it.
func Create(brief, details, cronExpr, tz string, immediate bool) (*Schedule, error) {
	if brief == "" {
		return nil, fmt.Errorf("brief is required")
	}
	if cronExpr == "" {
		return nil, fmt.Errorf("cron expression is required")
	}
	if err := Validate(cronExpr); err != nil {
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
		next = NextTime(cronExpr, now, loc)
	}

	sched := &Schedule{
		ID:        id,
		Brief:     brief,
		Details:   details,
		Cron:      cronExpr,
		Timezone:  tz,
		Enabled:   true,
		CreatedAt: now,
		NextRun:   next,
	}

	if err := os.MkdirAll(layout.SchedulesDir(), 0755); err != nil {
		return nil, err
	}
	return sched, save(sched)
}

// List returns all schedules sorted by next_run.
func List() ([]*Schedule, error) {
	dir := layout.SchedulesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var schedules []*Schedule
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		s, err := load(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		schedules = append(schedules, s)
	}

	sort.Slice(schedules, func(i, j int) bool {
		if schedules[i].NextRun == nil {
			return false
		}
		if schedules[j].NextRun == nil {
			return true
		}
		return schedules[i].NextRun.Before(*schedules[j].NextRun)
	})
	return schedules, nil
}

// Delete removes a schedule by ID.
func Delete(id string) error {
	path := filePath(id)
	if !platform.FileExists(path) {
		return fmt.Errorf("schedule %q not found", id)
	}
	return os.Remove(path)
}

// CheckDue finds due schedules and dispatches them via the provided callback.
func CheckDue(dispatch DispatchFunc) {
	schedules, err := List()
	if err != nil || len(schedules) == 0 {
		return
	}

	now := time.Now().UTC()

	// Build set of schedule IDs with in-flight operations
	inFlight := make(map[string]bool)
	ops, err := operation.List()
	if err != nil {
		log.Printf("[schedule] failed to list operations for duplicate detection: %v", err)
	}
	for _, op := range ops {
		if op.Status == "queued" || op.Status == "active" {
			if strings.HasPrefix(op.Source, "schedule/") {
				schedID := strings.TrimPrefix(op.Source, "schedule/")
				inFlight[schedID] = true
			}
		}
	}

	for _, s := range schedules {
		if !s.Enabled || s.NextRun == nil {
			continue
		}
		if s.NextRun.After(now) {
			continue
		}
		if inFlight[s.ID] {
			continue
		}

		sourceTag, err := dispatch(s.Brief, s.Details)
		if err != nil {
			continue
		}

		// Update the dispatched operation's source
		if op, err := operation.Find(sourceTag); err == nil {
			op.Source = "schedule/" + s.ID
			if err := operation.Save(op); err != nil {
				log.Printf("[schedule] %s: failed to save source update: %v", op.OperationID, err)
			}
		}

		loc, _ := time.LoadLocation(s.Timezone)
		if loc == nil {
			loc = time.UTC
		}
		s.LastRun = &now
		s.NextRun = NextTime(s.Cron, now, loc)
		_ = save(s)
	}
}

func filePath(id string) string {
	return filepath.Join(layout.SchedulesDir(), id+".json")
}

func save(s *Schedule) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath(s.ID), data, 0644)
}

func load(path string) (*Schedule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Schedule
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
