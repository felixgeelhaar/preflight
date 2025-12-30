package agent

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Schedule represents when reconciliation should run.
type Schedule struct {
	interval time.Duration
	cron     string
}

// NewIntervalSchedule creates a schedule from a duration.
func NewIntervalSchedule(d time.Duration) Schedule {
	return Schedule{interval: d}
}

// NewCronSchedule creates a schedule from a cron expression.
// Note: Cron scheduling requires additional implementation for next run calculation.
func NewCronSchedule(expr string) Schedule {
	return Schedule{cron: expr}
}

// ParseSchedule parses a schedule string.
// Supports formats:
//   - Duration: "30m", "1h", "2h30m"
//   - Cron: "0 */30 * * *" (must start with number or *)
func ParseSchedule(s string) (Schedule, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Schedule{}, fmt.Errorf("empty schedule")
	}

	// Try to parse as cron (starts with number or *)
	if isCronExpression(s) {
		if err := validateCronExpression(s); err != nil {
			return Schedule{}, err
		}
		return NewCronSchedule(s), nil
	}

	// Try to parse as duration
	d, err := parseDuration(s)
	if err != nil {
		return Schedule{}, fmt.Errorf("invalid schedule format: %w", err)
	}

	return NewIntervalSchedule(d), nil
}

// isCronExpression checks if the string looks like a cron expression.
func isCronExpression(s string) bool {
	// Cron expressions typically start with a number or *
	// and have multiple space-separated fields
	parts := strings.Fields(s)
	if len(parts) < 5 {
		return false
	}
	// First character should be digit or *
	if len(s) > 0 && (s[0] >= '0' && s[0] <= '9' || s[0] == '*') {
		return true
	}
	return false
}

// validateCronExpression performs basic validation of a cron expression.
func validateCronExpression(s string) error {
	parts := strings.Fields(s)
	if len(parts) != 5 {
		return fmt.Errorf("cron expression must have 5 fields (minute hour day month weekday)")
	}
	return nil
}

// parseDuration parses a human-friendly duration string.
// Supports: "30s", "5m", "1h", "2h30m", "1d", etc.
func parseDuration(s string) (time.Duration, error) {
	// Try standard Go duration first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Try custom formats (e.g., "1d" for days)
	s = strings.ToLower(strings.TrimSpace(s))

	// Match pattern like "1d", "2d12h", etc.
	dayPattern := regexp.MustCompile(`^(\d+)d(.*)$`)
	if matches := dayPattern.FindStringSubmatch(s); matches != nil {
		days, _ := strconv.Atoi(matches[1])
		d := time.Duration(days) * 24 * time.Hour
		if matches[2] != "" {
			rest, err := time.ParseDuration(matches[2])
			if err != nil {
				return 0, err
			}
			d += rest
		}
		return d, nil
	}

	return 0, fmt.Errorf("invalid duration: %s", s)
}

// Interval returns the interval duration.
// For cron schedules, this returns a reasonable default.
func (s Schedule) Interval() time.Duration {
	if s.interval > 0 {
		return s.interval
	}
	// Default for cron: 1 hour (actual timing is determined by cron expression)
	return time.Hour
}

// IsCron returns true if this is a cron-based schedule.
func (s Schedule) IsCron() bool {
	return s.cron != ""
}

// Cron returns the cron expression if this is a cron schedule.
func (s Schedule) Cron() string {
	return s.cron
}

// String returns a human-readable representation.
func (s Schedule) String() string {
	if s.cron != "" {
		return fmt.Sprintf("cron(%s)", s.cron)
	}
	return s.interval.String()
}

// MarshalText implements encoding.TextMarshaler.
func (s Schedule) MarshalText() ([]byte, error) {
	if s.cron != "" {
		return []byte(s.cron), nil
	}
	return []byte(s.interval.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *Schedule) UnmarshalText(text []byte) error {
	parsed, err := ParseSchedule(string(text))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// MarshalYAML implements yaml.Marshaler.
func (s Schedule) MarshalYAML() (interface{}, error) {
	if s.cron != "" {
		return s.cron, nil
	}
	return s.interval.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (s *Schedule) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	parsed, err := ParseSchedule(str)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}
