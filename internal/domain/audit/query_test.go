package audit_test

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/audit"
	"github.com/stretchr/testify/assert"
)

func TestQueryFilter_Matches_EventTypes(t *testing.T) {
	t.Parallel()

	filter := audit.QueryFilter{
		EventTypes: []audit.EventType{audit.EventCatalogInstalled, audit.EventCatalogRemoved},
	}

	// Should match
	event1 := audit.NewEvent(audit.EventCatalogInstalled).Build()
	assert.True(t, filter.Matches(event1))

	// Should not match
	event2 := audit.NewEvent(audit.EventPluginInstalled).Build()
	assert.False(t, filter.Matches(event2))
}

func TestQueryFilter_Matches_Severities(t *testing.T) {
	t.Parallel()

	filter := audit.QueryFilter{
		Severities: []audit.Severity{audit.SeverityCritical, audit.SeverityWarning},
	}

	// Should match
	event1 := audit.NewEvent(audit.EventSandboxViolation).WithSeverity(audit.SeverityCritical).Build()
	assert.True(t, filter.Matches(event1))

	// Should not match
	event2 := audit.NewEvent(audit.EventCatalogInstalled).WithSeverity(audit.SeverityInfo).Build()
	assert.False(t, filter.Matches(event2))
}

func TestQueryFilter_Matches_Catalog(t *testing.T) {
	t.Parallel()

	filter := audit.QueryFilter{
		Catalog: "company",
	}

	// Should match (contains)
	event1 := audit.NewEvent(audit.EventCatalogInstalled).WithCatalog("company-tools").Build()
	assert.True(t, filter.Matches(event1))

	// Case insensitive
	event2 := audit.NewEvent(audit.EventCatalogInstalled).WithCatalog("COMPANY-DEVOPS").Build()
	assert.True(t, filter.Matches(event2))

	// Should not match
	event3 := audit.NewEvent(audit.EventCatalogInstalled).WithCatalog("personal").Build()
	assert.False(t, filter.Matches(event3))
}

func TestQueryFilter_Matches_Plugin(t *testing.T) {
	t.Parallel()

	filter := audit.QueryFilter{
		Plugin: "my-plugin",
	}

	event1 := audit.NewEvent(audit.EventPluginInstalled).WithPlugin("my-plugin-v2").Build()
	assert.True(t, filter.Matches(event1))

	event2 := audit.NewEvent(audit.EventPluginInstalled).WithPlugin("other").Build()
	assert.False(t, filter.Matches(event2))
}

func TestQueryFilter_Matches_User(t *testing.T) {
	t.Parallel()

	filter := audit.QueryFilter{
		User: "admin",
	}

	event1 := audit.NewEvent(audit.EventCatalogInstalled).WithUser("admin").Build()
	assert.True(t, filter.Matches(event1))

	event2 := audit.NewEvent(audit.EventCatalogInstalled).WithUser("guest").Build()
	assert.False(t, filter.Matches(event2))
}

func TestQueryFilter_Matches_TimeRange(t *testing.T) {
	t.Parallel()

	now := time.Now()
	filter := audit.QueryFilter{
		Since: now.Add(-1 * time.Hour),
		Until: now.Add(1 * time.Hour),
	}

	// Should match (within range)
	event1 := audit.NewEvent(audit.EventCatalogInstalled).Build()
	assert.True(t, filter.Matches(event1))
}

func TestQueryFilter_Matches_Since(t *testing.T) {
	t.Parallel()

	filter := audit.QueryFilter{
		Since: time.Now().Add(1 * time.Hour), // Future
	}

	// Should not match (event is before Since)
	event := audit.NewEvent(audit.EventCatalogInstalled).Build()
	assert.False(t, filter.Matches(event))
}

func TestQueryFilter_Matches_Until(t *testing.T) {
	t.Parallel()

	filter := audit.QueryFilter{
		Until: time.Now().Add(-1 * time.Hour), // Past
	}

	// Should not match (event is after Until)
	event := audit.NewEvent(audit.EventCatalogInstalled).Build()
	assert.False(t, filter.Matches(event))
}

func TestQueryFilter_Matches_SuccessOnly(t *testing.T) {
	t.Parallel()

	filter := audit.QueryFilter{
		SuccessOnly: true,
	}

	event1 := audit.NewEvent(audit.EventCatalogInstalled).WithSuccess(true).Build()
	assert.True(t, filter.Matches(event1))

	event2 := audit.NewEvent(audit.EventCatalogInstalled).WithSuccess(false).Build()
	assert.False(t, filter.Matches(event2))
}

func TestQueryFilter_Matches_FailuresOnly(t *testing.T) {
	t.Parallel()

	filter := audit.QueryFilter{
		FailuresOnly: true,
	}

	event1 := audit.NewEvent(audit.EventCatalogInstalled).WithSuccess(false).Build()
	assert.True(t, filter.Matches(event1))

	event2 := audit.NewEvent(audit.EventCatalogInstalled).WithSuccess(true).Build()
	assert.False(t, filter.Matches(event2))
}

func TestQueryFilter_Matches_EmptyFilter(t *testing.T) {
	t.Parallel()

	filter := audit.QueryFilter{}

	// Empty filter should match everything
	event := audit.NewEvent(audit.EventCatalogInstalled).Build()
	assert.True(t, filter.Matches(event))
}

func TestQueryBuilder(t *testing.T) {
	t.Parallel()

	filter := audit.NewQuery().
		WithEventTypes(audit.EventCatalogInstalled, audit.EventCatalogRemoved).
		WithSeverities(audit.SeverityCritical).
		WithCatalog("test").
		WithPlugin("plugin").
		WithUser("admin").
		Limit(10).
		Build()

	assert.Equal(t, []audit.EventType{audit.EventCatalogInstalled, audit.EventCatalogRemoved}, filter.EventTypes)
	assert.Equal(t, []audit.Severity{audit.SeverityCritical}, filter.Severities)
	assert.Equal(t, "test", filter.Catalog)
	assert.Equal(t, "plugin", filter.Plugin)
	assert.Equal(t, "admin", filter.User)
	assert.Equal(t, 10, filter.Limit)
}

func TestQueryBuilder_LastDays(t *testing.T) {
	t.Parallel()

	filter := audit.NewQuery().LastDays(7).Build()

	expectedSince := time.Now().AddDate(0, 0, -7)
	assert.WithinDuration(t, expectedSince, filter.Since, time.Second)
}

func TestQueryBuilder_LastHours(t *testing.T) {
	t.Parallel()

	filter := audit.NewQuery().LastHours(24).Build()

	expectedSince := time.Now().Add(-24 * time.Hour)
	assert.WithinDuration(t, expectedSince, filter.Since, time.Second)
}

func TestQueryBuilder_Since(t *testing.T) {
	t.Parallel()

	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	filter := audit.NewQuery().Since(since).Build()

	assert.Equal(t, since, filter.Since)
}

func TestQueryBuilder_Until(t *testing.T) {
	t.Parallel()

	until := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	filter := audit.NewQuery().Until(until).Build()

	assert.Equal(t, until, filter.Until)
}

func TestQueryBuilder_SuccessOnly(t *testing.T) {
	t.Parallel()

	filter := audit.NewQuery().SuccessOnly().Build()

	assert.True(t, filter.SuccessOnly)
	assert.False(t, filter.FailuresOnly)
}

func TestQueryBuilder_FailuresOnly(t *testing.T) {
	t.Parallel()

	filter := audit.NewQuery().FailuresOnly().Build()

	assert.True(t, filter.FailuresOnly)
	assert.False(t, filter.SuccessOnly)
}

func TestSummarize(t *testing.T) {
	t.Parallel()

	events := []audit.Event{
		audit.NewEvent(audit.EventCatalogInstalled).WithCatalog("cat1").WithSeverity(audit.SeverityInfo).WithSuccess(true).Build(),
		audit.NewEvent(audit.EventCatalogInstalled).WithCatalog("cat2").WithSeverity(audit.SeverityWarning).WithSuccess(true).Build(),
		audit.NewEvent(audit.EventPluginInstalled).WithPlugin("plugin1").WithSeverity(audit.SeverityInfo).WithSuccess(false).Build(),
		audit.NewEvent(audit.EventSandboxViolation).WithPlugin("plugin2").WithSeverity(audit.SeverityCritical).WithSuccess(false).Build(),
	}

	summary := audit.Summarize(events)

	assert.Equal(t, 4, summary.TotalEvents)
	assert.Equal(t, 2, summary.SuccessCount)
	assert.Equal(t, 2, summary.FailureCount)
	assert.Equal(t, 1, summary.CriticalCount)
	assert.Equal(t, 1, summary.SecurityEvents) // SandboxViolation

	assert.Equal(t, 2, summary.BySeverity["info"])
	assert.Equal(t, 1, summary.BySeverity["warning"])
	assert.Equal(t, 1, summary.BySeverity["critical"])

	assert.Equal(t, 2, summary.ByType["catalog_installed"])
	assert.Equal(t, 1, summary.ByType["plugin_installed"])
	assert.Equal(t, 1, summary.ByType["sandbox_violation"])

	assert.Equal(t, 1, summary.ByCatalog["cat1"])
	assert.Equal(t, 1, summary.ByCatalog["cat2"])
	assert.Equal(t, 1, summary.ByPlugin["plugin1"])
	assert.Equal(t, 1, summary.ByPlugin["plugin2"])
}

func TestSummarize_Empty(t *testing.T) {
	t.Parallel()

	summary := audit.Summarize(nil)

	assert.Equal(t, 0, summary.TotalEvents)
	assert.Equal(t, 0, summary.SuccessCount)
	assert.Equal(t, 0, summary.FailureCount)
}

func TestSummarize_TimeRange(t *testing.T) {
	t.Parallel()

	now := time.Now()
	events := []audit.Event{
		audit.NewEvent(audit.EventCatalogInstalled).Build(),
	}

	summary := audit.Summarize(events)

	assert.WithinDuration(t, now, summary.FirstEvent, time.Second)
	assert.WithinDuration(t, now, summary.LastEvent, time.Second)
}
