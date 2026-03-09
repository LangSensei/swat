package commander

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (c *Commander) SchedulesDir() string {
	return filepath.Join(c.SwatRoot, "schedules")
}

func (c *Commander) scheduleFile(id string) string {
	return filepath.Join(c.SchedulesDir(), id+".json")
}

// CreateSchedule creates a new schedule and persists it
func (c *Commander) CreateSchedule(brief, details, cronExpr, tz, name string) (*Schedule, error) {
	if brief == "" {
		return nil, fmt.Errorf("brief is required")
	}
	if cronExpr == "" {
		return nil, fmt.Errorf("cron expression is required")
	}
	if err := validateCron(cronExpr); err != nil {
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
	next := nextCronTime(cronExpr, now, loc)

	sched := &Schedule{
		ID:        id,
		Name:      name,
		Brief:     brief,
		Details:   details,
		Cron:      cronExpr,
		Timezone:  tz,
		Enabled:   true,
		CreatedAt: now,
		NextRun:   next,
	}

	if err := os.MkdirAll(c.SchedulesDir(), 0755); err != nil {
		return nil, err
	}
	return sched, c.saveSchedule(sched)
}

// ListSchedules returns all schedules sorted by next_run
func (c *Commander) ListSchedules() ([]*Schedule, error) {
	dir := c.SchedulesDir()
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
		s, err := c.loadSchedule(filepath.Join(dir, e.Name()))
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

// DeleteSchedule removes a schedule by ID
func (c *Commander) DeleteSchedule(id string) error {
	path := c.scheduleFile(id)
	if !fileExists(path) {
		return fmt.Errorf("schedule %q not found", id)
	}
	return os.Remove(path)
}

// CheckDue finds due schedules and dispatches them
func (c *Commander) CheckDue() {
	schedules, err := c.ListSchedules()
	if err != nil || len(schedules) == 0 {
		return
	}

	now := time.Now().UTC()
	for _, s := range schedules {
		if !s.Enabled || s.NextRun == nil {
			continue
		}
		if s.NextRun.After(now) {
			continue
		}

		// Dispatch the scheduled task
		op, err := c.Dispatch(s.Brief, s.Details)
		if err != nil {
			continue
		}
		// Update source to "schedule"
		op.Source = "schedule"
		_ = c.SaveOperation(op)

		// Update schedule: last_run, next_run
		loc, _ := time.LoadLocation(s.Timezone)
		if loc == nil {
			loc = time.UTC
		}
		s.LastRun = &now
		s.NextRun = nextCronTime(s.Cron, now, loc)
		_ = c.saveSchedule(s)
	}
}

func (c *Commander) saveSchedule(s *Schedule) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.scheduleFile(s.ID), data, 0644)
}

func (c *Commander) loadSchedule(path string) (*Schedule, error) {
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

// --- Cron parsing (standard 5-field: min hour dom month dow) ---

type cronSpec struct {
	minute, hour, dom, month, dow []int
}

func validateCron(expr string) error {
	_, err := parseCron(expr)
	return err
}

func parseCron(expr string) (*cronSpec, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("expected 5 fields, got %d", len(fields))
	}

	minute, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute: %w", err)
	}
	hour, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}
	dom, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("day-of-month: %w", err)
	}
	month, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	dow, err := parseCronField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("day-of-week: %w", err)
	}

	return &cronSpec{minute, hour, dom, month, dow}, nil
}

func parseCronField(field string, min, max int) ([]int, error) {
	if field == "*" {
		var vals []int
		for i := min; i <= max; i++ {
			vals = append(vals, i)
		}
		return vals, nil
	}

	// Handle */step
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step in %q", field)
		}
		var vals []int
		for i := min; i <= max; i += step {
			vals = append(vals, i)
		}
		return vals, nil
	}

	var result []int
	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		// Handle range: a-b or a-b/step
		if strings.Contains(part, "-") {
			rangeParts := strings.SplitN(part, "/", 2)
			bounds := strings.SplitN(rangeParts[0], "-", 2)
			lo, err1 := strconv.Atoi(bounds[0])
			hi, err2 := strconv.Atoi(bounds[1])
			if err1 != nil || err2 != nil || lo < min || hi > max || lo > hi {
				return nil, fmt.Errorf("invalid range %q", part)
			}
			step := 1
			if len(rangeParts) == 2 {
				step, err1 = strconv.Atoi(rangeParts[1])
				if err1 != nil || step <= 0 {
					return nil, fmt.Errorf("invalid step in range %q", part)
				}
			}
			for i := lo; i <= hi; i += step {
				result = append(result, i)
			}
		} else {
			v, err := strconv.Atoi(part)
			if err != nil || v < min || v > max {
				return nil, fmt.Errorf("invalid value %q (range %d-%d)", part, min, max)
			}
			result = append(result, v)
		}
	}
	return result, nil
}

func contains(vals []int, v int) bool {
	for _, x := range vals {
		if x == v {
			return true
		}
	}
	return false
}

// nextCronTime finds the next time after `after` that matches the cron spec
func nextCronTime(expr string, after time.Time, loc *time.Location) *time.Time {
	spec, err := parseCron(expr)
	if err != nil {
		return nil
	}

	// Start from the next minute
	t := after.In(loc).Truncate(time.Minute).Add(time.Minute)

	// Search up to 366 days ahead
	limit := t.Add(366 * 24 * time.Hour)
	for t.Before(limit) {
		if contains(spec.month, int(t.Month())) &&
			contains(spec.dom, t.Day()) &&
			contains(spec.dow, int(t.Weekday())) &&
			contains(spec.hour, t.Hour()) &&
			contains(spec.minute, t.Minute()) {
			result := t.UTC()
			return &result
		}

		// Advance smartly: skip months/days/hours that don't match
		if !contains(spec.month, int(t.Month())) {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, loc)
			continue
		}
		if !contains(spec.dom, t.Day()) || !contains(spec.dow, int(t.Weekday())) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, loc)
			continue
		}
		if !contains(spec.hour, t.Hour()) {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, loc)
			continue
		}
		t = t.Add(time.Minute)
	}
	return nil
}
