package intake

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type cronSpec struct {
	minute, hour, dom, month, dow []int
}

// ValidateCron checks if a cron expression is valid (5-field: min hour dom month dow).
func ValidateCron(expr string) error {
	_, err := parseCron(expr)
	return err
}

// NextCronTime finds the next time after `after` that matches the cron spec.
func NextCronTime(expr string, after time.Time, loc *time.Location) *time.Time {
	spec, err := parseCron(expr)
	if err != nil {
		return nil
	}

	t := after.In(loc).Truncate(time.Minute).Add(time.Minute)
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

func parseCron(expr string) (*cronSpec, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("expected 5 fields, got %d", len(fields))
	}

	minute, err := parseField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute: %w", err)
	}
	hour, err := parseField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}
	dom, err := parseField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("day-of-month: %w", err)
	}
	month, err := parseField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	dow, err := parseField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("day-of-week: %w", err)
	}

	return &cronSpec{minute, hour, dom, month, dow}, nil
}

func parseField(field string, min, max int) ([]int, error) {
	if field == "*" {
		var vals []int
		for i := min; i <= max; i++ {
			vals = append(vals, i)
		}
		return vals, nil
	}

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
