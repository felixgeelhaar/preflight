package audit

import (
	"strings"
	"time"
)

// QueryFilter defines criteria for filtering audit events.
type QueryFilter struct {
	// EventTypes filters by event type (empty = all types)
	EventTypes []EventType

	// Severities filters by severity level (empty = all levels)
	Severities []Severity

	// Catalog filters by catalog name
	Catalog string

	// Plugin filters by plugin name
	Plugin string

	// User filters by user
	User string

	// Since filters events after this time
	Since time.Time

	// Until filters events before this time
	Until time.Time

	// SuccessOnly includes only successful events
	SuccessOnly bool

	// FailuresOnly includes only failed events
	FailuresOnly bool

	// Limit maximum number of results (0 = no limit)
	Limit int
}

// Matches returns true if the event matches the filter.
func (f QueryFilter) Matches(event Event) bool {
	// Check event types
	if len(f.EventTypes) > 0 {
		found := false
		for _, t := range f.EventTypes {
			if event.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check severities
	if len(f.Severities) > 0 {
		found := false
		for _, s := range f.Severities {
			if event.Severity == s {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check catalog (case-insensitive contains)
	if f.Catalog != "" && !strings.Contains(strings.ToLower(event.Catalog), strings.ToLower(f.Catalog)) {
		return false
	}

	// Check plugin (case-insensitive contains)
	if f.Plugin != "" && !strings.Contains(strings.ToLower(event.Plugin), strings.ToLower(f.Plugin)) {
		return false
	}

	// Check user (case-insensitive contains)
	if f.User != "" && !strings.Contains(strings.ToLower(event.User), strings.ToLower(f.User)) {
		return false
	}

	// Check time range
	if !f.Since.IsZero() && event.Timestamp.Before(f.Since) {
		return false
	}
	if !f.Until.IsZero() && event.Timestamp.After(f.Until) {
		return false
	}

	// Check success/failure
	if f.SuccessOnly && !event.Success {
		return false
	}
	if f.FailuresOnly && event.Success {
		return false
	}

	return true
}

// QueryBuilder provides a fluent API for building queries.
type QueryBuilder struct {
	filter QueryFilter
}

// NewQuery creates a new query builder.
func NewQuery() *QueryBuilder {
	return &QueryBuilder{}
}

// WithEventTypes filters by event types.
func (b *QueryBuilder) WithEventTypes(types ...EventType) *QueryBuilder {
	b.filter.EventTypes = types
	return b
}

// WithSeverities filters by severity levels.
func (b *QueryBuilder) WithSeverities(severities ...Severity) *QueryBuilder {
	b.filter.Severities = severities
	return b
}

// WithCatalog filters by catalog name.
func (b *QueryBuilder) WithCatalog(catalog string) *QueryBuilder {
	b.filter.Catalog = catalog
	return b
}

// WithPlugin filters by plugin name.
func (b *QueryBuilder) WithPlugin(plugin string) *QueryBuilder {
	b.filter.Plugin = plugin
	return b
}

// WithUser filters by user.
func (b *QueryBuilder) WithUser(user string) *QueryBuilder {
	b.filter.User = user
	return b
}

// Since filters events after the given time.
func (b *QueryBuilder) Since(t time.Time) *QueryBuilder {
	b.filter.Since = t
	return b
}

// Until filters events before the given time.
func (b *QueryBuilder) Until(t time.Time) *QueryBuilder {
	b.filter.Until = t
	return b
}

// LastDays filters events from the last N days.
func (b *QueryBuilder) LastDays(days int) *QueryBuilder {
	b.filter.Since = time.Now().AddDate(0, 0, -days)
	return b
}

// LastHours filters events from the last N hours.
func (b *QueryBuilder) LastHours(hours int) *QueryBuilder {
	b.filter.Since = time.Now().Add(-time.Duration(hours) * time.Hour)
	return b
}

// SuccessOnly includes only successful events.
func (b *QueryBuilder) SuccessOnly() *QueryBuilder {
	b.filter.SuccessOnly = true
	b.filter.FailuresOnly = false
	return b
}

// FailuresOnly includes only failed events.
func (b *QueryBuilder) FailuresOnly() *QueryBuilder {
	b.filter.FailuresOnly = true
	b.filter.SuccessOnly = false
	return b
}

// Limit sets maximum number of results.
func (b *QueryBuilder) Limit(n int) *QueryBuilder {
	b.filter.Limit = n
	return b
}

// Build returns the constructed filter.
func (b *QueryBuilder) Build() QueryFilter {
	return b.filter
}

// Summary generates a summary of events.
type Summary struct {
	TotalEvents     int            `json:"total_events"`
	SuccessCount    int            `json:"success_count"`
	FailureCount    int            `json:"failure_count"`
	BySeverity      map[string]int `json:"by_severity"`
	ByType          map[string]int `json:"by_type"`
	ByCatalog       map[string]int `json:"by_catalog"`
	ByPlugin        map[string]int `json:"by_plugin"`
	FirstEvent      time.Time      `json:"first_event,omitempty"`
	LastEvent       time.Time      `json:"last_event,omitempty"`
	CriticalCount   int            `json:"critical_count"`
	SecurityEvents  int            `json:"security_events"`
}

// Summarize generates a summary from events.
func Summarize(events []Event) Summary {
	summary := Summary{
		BySeverity: make(map[string]int),
		ByType:     make(map[string]int),
		ByCatalog:  make(map[string]int),
		ByPlugin:   make(map[string]int),
	}

	for _, event := range events {
		summary.TotalEvents++

		if event.Success {
			summary.SuccessCount++
		} else {
			summary.FailureCount++
		}

		summary.BySeverity[string(event.Severity)]++
		summary.ByType[string(event.Type)]++

		if event.Catalog != "" {
			summary.ByCatalog[event.Catalog]++
		}
		if event.Plugin != "" {
			summary.ByPlugin[event.Plugin]++
		}

		if event.Severity == SeverityCritical {
			summary.CriticalCount++
		}

		// Count security-related events
		if isSecurityEvent(event.Type) {
			summary.SecurityEvents++
		}

		// Track time range
		if summary.FirstEvent.IsZero() || event.Timestamp.Before(summary.FirstEvent) {
			summary.FirstEvent = event.Timestamp
		}
		if summary.LastEvent.IsZero() || event.Timestamp.After(summary.LastEvent) {
			summary.LastEvent = event.Timestamp
		}
	}

	return summary
}

// isSecurityEvent returns true for security-related event types.
func isSecurityEvent(t EventType) bool {
	switch t {
	case EventCapabilityDenied, EventSandboxViolation, EventSignatureFailed, EventSecurityAudit:
		return true
	default:
		return false
	}
}
