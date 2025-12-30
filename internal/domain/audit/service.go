package audit

import (
	"context"
	"os/user"
	"time"
)

// Service provides high-level audit logging operations.
type Service struct {
	logger Logger
}

// NewService creates a new audit service with the given logger.
func NewService(logger Logger) *Service {
	return &Service{
		logger: logger,
	}
}

// getCurrentUser returns the current system user.
func getCurrentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}

// LogCatalogInstalled logs a catalog installation event.
func (s *Service) LogCatalogInstalled(ctx context.Context, catalog, source, integrity string, signatureVerified bool, signer string, trustLevel string) error {
	event := NewEvent(EventCatalogInstalled).
		WithUser(getCurrentUser()).
		WithCatalog(catalog).
		WithSource(source).
		WithIntegrity(integrity).
		WithSignature(signatureVerified, signer).
		WithTrustLevel(trustLevel).
		Build()

	return s.logger.Log(ctx, event)
}

// LogCatalogRemoved logs a catalog removal event.
func (s *Service) LogCatalogRemoved(ctx context.Context, catalog string) error {
	event := NewEvent(EventCatalogRemoved).
		WithUser(getCurrentUser()).
		WithCatalog(catalog).
		Build()

	return s.logger.Log(ctx, event)
}

// LogCatalogVerified logs a catalog verification event.
func (s *Service) LogCatalogVerified(ctx context.Context, catalog string, success bool, err error) error {
	builder := NewEvent(EventCatalogVerified).
		WithUser(getCurrentUser()).
		WithCatalog(catalog).
		WithSuccess(success)

	if err != nil {
		builder.WithError(err).WithSeverity(SeverityWarning)
	}

	return s.logger.Log(ctx, builder.Build())
}

// LogPluginInstalled logs a plugin installation event.
func (s *Service) LogPluginInstalled(ctx context.Context, plugin, source, version string, capabilities []string) error {
	event := NewEvent(EventPluginInstalled).
		WithUser(getCurrentUser()).
		WithPlugin(plugin).
		WithSource(source).
		WithCapabilitiesGranted(capabilities).
		AddDetail("version", version).
		Build()

	return s.logger.Log(ctx, event)
}

// LogPluginUninstalled logs a plugin uninstallation event.
func (s *Service) LogPluginUninstalled(ctx context.Context, plugin string) error {
	event := NewEvent(EventPluginUninstalled).
		WithUser(getCurrentUser()).
		WithPlugin(plugin).
		Build()

	return s.logger.Log(ctx, event)
}

// LogPluginExecuted logs a plugin execution event.
func (s *Service) LogPluginExecuted(ctx context.Context, plugin, sandboxMode string, duration time.Duration, success bool, err error) error {
	builder := NewEvent(EventPluginExecuted).
		WithUser(getCurrentUser()).
		WithPlugin(plugin).
		WithSandboxMode(sandboxMode).
		WithDuration(duration).
		WithSuccess(success)

	if err != nil {
		builder.WithError(err)
	}

	return s.logger.Log(ctx, builder.Build())
}

// LogTrustAdded logs a trust addition event.
func (s *Service) LogTrustAdded(ctx context.Context, keyID, fingerprint, name string) error {
	event := NewEvent(EventTrustAdded).
		WithUser(getCurrentUser()).
		AddDetail("key_id", keyID).
		AddDetail("fingerprint", fingerprint).
		AddDetail("name", name).
		Build()

	return s.logger.Log(ctx, event)
}

// LogTrustRemoved logs a trust removal event.
func (s *Service) LogTrustRemoved(ctx context.Context, keyID string) error {
	event := NewEvent(EventTrustRemoved).
		WithUser(getCurrentUser()).
		AddDetail("key_id", keyID).
		Build()

	return s.logger.Log(ctx, event)
}

// LogSignatureVerified logs a successful signature verification.
func (s *Service) LogSignatureVerified(ctx context.Context, catalog, signer string) error {
	event := NewEvent(EventSignatureVerified).
		WithUser(getCurrentUser()).
		WithCatalog(catalog).
		WithSignature(true, signer).
		Build()

	return s.logger.Log(ctx, event)
}

// LogSignatureFailed logs a failed signature verification.
func (s *Service) LogSignatureFailed(ctx context.Context, catalog string, err error) error {
	event := NewEvent(EventSignatureFailed).
		WithUser(getCurrentUser()).
		WithCatalog(catalog).
		WithSignature(false, "").
		WithError(err).
		WithSeverity(SeverityWarning).
		Build()

	return s.logger.Log(ctx, event)
}

// LogCapabilityGranted logs capabilities being granted to a plugin.
func (s *Service) LogCapabilityGranted(ctx context.Context, plugin string, capabilities []string) error {
	event := NewEvent(EventCapabilityGranted).
		WithUser(getCurrentUser()).
		WithPlugin(plugin).
		WithCapabilitiesGranted(capabilities).
		Build()

	return s.logger.Log(ctx, event)
}

// LogCapabilityDenied logs capabilities being denied to a plugin.
func (s *Service) LogCapabilityDenied(ctx context.Context, plugin string, capabilities []string, reason string) error {
	event := NewEvent(EventCapabilityDenied).
		WithUser(getCurrentUser()).
		WithPlugin(plugin).
		WithCapabilitiesDenied(capabilities).
		WithSeverity(SeverityWarning).
		AddDetail("reason", reason).
		Build()

	return s.logger.Log(ctx, event)
}

// LogSandboxViolation logs a sandbox security violation.
func (s *Service) LogSandboxViolation(ctx context.Context, plugin, violation string) error {
	event := NewEvent(EventSandboxViolation).
		WithUser(getCurrentUser()).
		WithPlugin(plugin).
		WithSeverity(SeverityCritical).
		AddDetail("violation", violation).
		Build()

	return s.logger.Log(ctx, event)
}

// LogSecurityAudit logs the result of a security audit.
func (s *Service) LogSecurityAudit(ctx context.Context, catalog string, passed bool, criticalCount, highCount int) error {
	severity := SeverityInfo
	if criticalCount > 0 {
		severity = SeverityCritical
	} else if highCount > 0 {
		severity = SeverityWarning
	}

	event := NewEvent(EventSecurityAudit).
		WithUser(getCurrentUser()).
		WithCatalog(catalog).
		WithSuccess(passed).
		WithSeverity(severity).
		AddDetail("critical_count", criticalCount).
		AddDetail("high_count", highCount).
		Build()

	return s.logger.Log(ctx, event)
}

// Query retrieves events matching the filter.
func (s *Service) Query(ctx context.Context, filter QueryFilter) ([]Event, error) {
	return s.logger.Query(ctx, filter)
}

// Recent returns the most recent N events.
func (s *Service) Recent(ctx context.Context, limit int) ([]Event, error) {
	filter := NewQuery().Limit(limit).Build()
	return s.logger.Query(ctx, filter)
}

// SecurityEvents returns security-related events from the last N days.
func (s *Service) SecurityEvents(ctx context.Context, days int) ([]Event, error) {
	filter := NewQuery().
		WithEventTypes(
			EventCapabilityDenied,
			EventSandboxViolation,
			EventSignatureFailed,
			EventSecurityAudit,
		).
		LastDays(days).
		Build()

	return s.logger.Query(ctx, filter)
}

// Summary returns a summary of events matching the filter.
func (s *Service) Summary(ctx context.Context, filter QueryFilter) (Summary, error) {
	events, err := s.logger.Query(ctx, filter)
	if err != nil {
		return Summary{}, err
	}
	return Summarize(events), nil
}

// Close releases resources.
func (s *Service) Close() error {
	return s.logger.Close()
}
