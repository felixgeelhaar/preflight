// Package audit provides security event logging for plugin operations.
package audit

import (
	"encoding/json"
	"time"
)

// EventType represents the type of audit event.
type EventType string

// Event types for catalog operations.
const (
	EventCatalogInstalled EventType = "catalog_installed"
	EventCatalogRemoved   EventType = "catalog_removed"
	EventCatalogVerified  EventType = "catalog_verified"
	EventCatalogUpdated   EventType = "catalog_updated"
)

// Event types for plugin operations.
const (
	EventPluginInstalled   EventType = "plugin_installed"
	EventPluginUninstalled EventType = "plugin_uninstalled"
	EventPluginDiscovered  EventType = "plugin_discovered"
	EventPluginExecuted    EventType = "plugin_executed"
	EventPluginValidated   EventType = "plugin_validated"
)

// Event types for trust operations.
const (
	EventTrustAdded        EventType = "trust_added"
	EventTrustRemoved      EventType = "trust_removed"
	EventSignatureVerified EventType = "signature_verified"
	EventSignatureFailed   EventType = "signature_failed"
)

// Event types for security operations.
const (
	EventCapabilityGranted EventType = "capability_granted"
	EventCapabilityDenied  EventType = "capability_denied"
	EventSandboxViolation  EventType = "sandbox_violation"
	EventSecurityAudit     EventType = "security_audit"
)

// Severity represents the importance level of an event.
type Severity string

// Severity levels.
const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Event represents a single audit log entry.
type Event struct {
	// ID is the unique event identifier
	ID string `json:"id"`

	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Type of the event
	Type EventType `json:"event"`

	// Severity level
	Severity Severity `json:"severity"`

	// User who triggered the event (if known)
	User string `json:"user,omitempty"`

	// Catalog name (for catalog-related events)
	Catalog string `json:"catalog,omitempty"`

	// Plugin name (for plugin-related events)
	Plugin string `json:"plugin,omitempty"`

	// Source URL or path
	Source string `json:"source,omitempty"`

	// Integrity hash (SHA256)
	Integrity string `json:"integrity,omitempty"`

	// SignatureVerified indicates if signature was verified
	SignatureVerified bool `json:"signature_verified,omitempty"`

	// Signer identity (if signature was verified)
	Signer string `json:"signer,omitempty"`

	// CapabilitiesGranted list of capabilities granted
	CapabilitiesGranted []string `json:"capabilities_granted,omitempty"`

	// CapabilitiesDenied list of capabilities denied
	CapabilitiesDenied []string `json:"capabilities_denied,omitempty"`

	// TrustLevel of the catalog/plugin
	TrustLevel string `json:"trust_level,omitempty"`

	// SandboxMode used for execution
	SandboxMode string `json:"sandbox_mode,omitempty"`

	// Duration of operation (for executions)
	Duration time.Duration `json:"duration,omitempty"`

	// Success indicates if the operation succeeded
	Success bool `json:"success"`

	// Error message if operation failed
	Error string `json:"error,omitempty"`

	// Details contains additional event-specific data
	Details map[string]interface{} `json:"details,omitempty"`
}

// MarshalJSON implements json.Marshaler with duration as milliseconds.
func (e Event) MarshalJSON() ([]byte, error) {
	type Alias Event
	return json.Marshal(&struct {
		Alias
		DurationMs int64 `json:"duration_ms,omitempty"`
	}{
		Alias:      Alias(e),
		DurationMs: e.Duration.Milliseconds(),
	})
}

// EventBuilder provides a fluent API for building events.
type EventBuilder struct {
	event Event
}

// NewEvent creates a new event builder with required fields.
func NewEvent(eventType EventType) *EventBuilder {
	return &EventBuilder{
		event: Event{
			ID:        generateEventID(),
			Timestamp: time.Now().UTC(),
			Type:      eventType,
			Severity:  SeverityInfo,
			Success:   true,
		},
	}
}

// WithSeverity sets the severity level.
func (b *EventBuilder) WithSeverity(severity Severity) *EventBuilder {
	b.event.Severity = severity
	return b
}

// WithUser sets the user who triggered the event.
func (b *EventBuilder) WithUser(user string) *EventBuilder {
	b.event.User = user
	return b
}

// WithCatalog sets the catalog name.
func (b *EventBuilder) WithCatalog(catalog string) *EventBuilder {
	b.event.Catalog = catalog
	return b
}

// WithPlugin sets the plugin name.
func (b *EventBuilder) WithPlugin(plugin string) *EventBuilder {
	b.event.Plugin = plugin
	return b
}

// WithSource sets the source URL or path.
func (b *EventBuilder) WithSource(source string) *EventBuilder {
	b.event.Source = source
	return b
}

// WithIntegrity sets the integrity hash.
func (b *EventBuilder) WithIntegrity(hash string) *EventBuilder {
	b.event.Integrity = hash
	return b
}

// WithSignature sets signature verification details.
func (b *EventBuilder) WithSignature(verified bool, signer string) *EventBuilder {
	b.event.SignatureVerified = verified
	b.event.Signer = signer
	return b
}

// WithCapabilitiesGranted sets the granted capabilities.
func (b *EventBuilder) WithCapabilitiesGranted(caps []string) *EventBuilder {
	b.event.CapabilitiesGranted = caps
	return b
}

// WithCapabilitiesDenied sets the denied capabilities.
func (b *EventBuilder) WithCapabilitiesDenied(caps []string) *EventBuilder {
	b.event.CapabilitiesDenied = caps
	return b
}

// WithTrustLevel sets the trust level.
func (b *EventBuilder) WithTrustLevel(level string) *EventBuilder {
	b.event.TrustLevel = level
	return b
}

// WithSandboxMode sets the sandbox mode.
func (b *EventBuilder) WithSandboxMode(mode string) *EventBuilder {
	b.event.SandboxMode = mode
	return b
}

// WithDuration sets the operation duration.
func (b *EventBuilder) WithDuration(d time.Duration) *EventBuilder {
	b.event.Duration = d
	return b
}

// WithSuccess sets the success status.
func (b *EventBuilder) WithSuccess(success bool) *EventBuilder {
	b.event.Success = success
	return b
}

// WithError sets the error message and marks as failed.
func (b *EventBuilder) WithError(err error) *EventBuilder {
	if err != nil {
		b.event.Success = false
		b.event.Error = err.Error()
	}
	return b
}

// WithDetails sets additional details.
func (b *EventBuilder) WithDetails(details map[string]interface{}) *EventBuilder {
	b.event.Details = details
	return b
}

// AddDetail adds a single detail.
func (b *EventBuilder) AddDetail(key string, value interface{}) *EventBuilder {
	if b.event.Details == nil {
		b.event.Details = make(map[string]interface{})
	}
	b.event.Details[key] = value
	return b
}

// Build creates the final Event.
func (b *EventBuilder) Build() Event {
	return b.event
}

// generateEventID creates a unique event identifier.
func generateEventID() string {
	// Use timestamp + random suffix for uniqueness
	now := time.Now()
	return now.Format("20060102150405") + "-" + randomSuffix(8)
}

// randomSuffix generates a random alphanumeric suffix.
func randomSuffix(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		// Use time-based pseudo-random for simplicity
		// In production, use crypto/rand
		result[i] = chars[(time.Now().UnixNano()+int64(i))%int64(len(chars))]
	}
	return string(result)
}
