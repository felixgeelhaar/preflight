package audit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_LogCatalogInstalled(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogCatalogInstalled(
		ctx,
		"company-tools",
		"https://company.com/catalog.yaml",
		"sha256:abc123",
		true,
		"devops@company.com",
		"verified",
	)
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, audit.EventCatalogInstalled, event.Type)
	assert.Equal(t, "company-tools", event.Catalog)
	assert.Equal(t, "https://company.com/catalog.yaml", event.Source)
	assert.Equal(t, "sha256:abc123", event.Integrity)
	assert.True(t, event.SignatureVerified)
	assert.Equal(t, "devops@company.com", event.Signer)
	assert.Equal(t, "verified", event.TrustLevel)
}

func TestService_LogCatalogRemoved(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogCatalogRemoved(ctx, "old-catalog")
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventCatalogRemoved, events[0].Type)
	assert.Equal(t, "old-catalog", events[0].Catalog)
}

func TestService_LogCatalogVerified_Success(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogCatalogVerified(ctx, "my-catalog", true, nil)
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventCatalogVerified, events[0].Type)
	assert.True(t, events[0].Success)
}

func TestService_LogCatalogVerified_Failure(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	verifyErr := errors.New("hash mismatch")
	err := service.LogCatalogVerified(ctx, "my-catalog", false, verifyErr)
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventCatalogVerified, events[0].Type)
	assert.False(t, events[0].Success)
	assert.Equal(t, audit.SeverityWarning, events[0].Severity)
	assert.Contains(t, events[0].Error, "hash mismatch")
}

func TestService_LogPluginInstalled(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogPluginInstalled(
		ctx,
		"my-plugin",
		"https://example.com/plugin.wasm",
		"1.0.0",
		[]string{"files:read", "shell:execute"},
	)
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventPluginInstalled, events[0].Type)
	assert.Equal(t, "my-plugin", events[0].Plugin)
	assert.Equal(t, []string{"files:read", "shell:execute"}, events[0].CapabilitiesGranted)
	assert.Equal(t, "1.0.0", events[0].Details["version"])
}

func TestService_LogPluginUninstalled(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogPluginUninstalled(ctx, "old-plugin")
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventPluginUninstalled, events[0].Type)
	assert.Equal(t, "old-plugin", events[0].Plugin)
}

func TestService_LogPluginExecuted_Success(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogPluginExecuted(ctx, "my-plugin", "restricted", 500*time.Millisecond, true, nil)
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventPluginExecuted, events[0].Type)
	assert.Equal(t, "my-plugin", events[0].Plugin)
	assert.Equal(t, "restricted", events[0].SandboxMode)
	assert.Equal(t, 500*time.Millisecond, events[0].Duration)
	assert.True(t, events[0].Success)
}

func TestService_LogPluginExecuted_Failure(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	execErr := errors.New("timeout")
	err := service.LogPluginExecuted(ctx, "my-plugin", "full", time.Second, false, execErr)
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.False(t, events[0].Success)
	assert.Contains(t, events[0].Error, "timeout")
}

func TestService_LogTrustAdded(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogTrustAdded(ctx, "ABCD1234", "0x123456789", "Developer Key")
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventTrustAdded, events[0].Type)
	assert.Equal(t, "ABCD1234", events[0].Details["key_id"])
	assert.Equal(t, "0x123456789", events[0].Details["fingerprint"])
	assert.Equal(t, "Developer Key", events[0].Details["name"])
}

func TestService_LogTrustRemoved(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogTrustRemoved(ctx, "ABCD1234")
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventTrustRemoved, events[0].Type)
	assert.Equal(t, "ABCD1234", events[0].Details["key_id"])
}

func TestService_LogSignatureVerified(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogSignatureVerified(ctx, "my-catalog", "devops@company.com")
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventSignatureVerified, events[0].Type)
	assert.Equal(t, "my-catalog", events[0].Catalog)
	assert.True(t, events[0].SignatureVerified)
	assert.Equal(t, "devops@company.com", events[0].Signer)
}

func TestService_LogSignatureFailed(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	sigErr := errors.New("invalid signature")
	err := service.LogSignatureFailed(ctx, "bad-catalog", sigErr)
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventSignatureFailed, events[0].Type)
	assert.Equal(t, "bad-catalog", events[0].Catalog)
	assert.False(t, events[0].SignatureVerified)
	assert.Equal(t, audit.SeverityWarning, events[0].Severity)
}

func TestService_LogCapabilityGranted(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogCapabilityGranted(ctx, "my-plugin", []string{"files:read", "files:write"})
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventCapabilityGranted, events[0].Type)
	assert.Equal(t, "my-plugin", events[0].Plugin)
	assert.Equal(t, []string{"files:read", "files:write"}, events[0].CapabilitiesGranted)
}

func TestService_LogCapabilityDenied(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogCapabilityDenied(ctx, "my-plugin", []string{"network:fetch"}, "blocked by policy")
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventCapabilityDenied, events[0].Type)
	assert.Equal(t, "my-plugin", events[0].Plugin)
	assert.Equal(t, []string{"network:fetch"}, events[0].CapabilitiesDenied)
	assert.Equal(t, audit.SeverityWarning, events[0].Severity)
	assert.Equal(t, "blocked by policy", events[0].Details["reason"])
}

func TestService_LogSandboxViolation(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogSandboxViolation(ctx, "bad-plugin", "attempted to access /etc/passwd")
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventSandboxViolation, events[0].Type)
	assert.Equal(t, "bad-plugin", events[0].Plugin)
	assert.Equal(t, audit.SeverityCritical, events[0].Severity)
	assert.Equal(t, "attempted to access /etc/passwd", events[0].Details["violation"])
}

func TestService_LogSecurityAudit_Passed(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogSecurityAudit(ctx, "my-catalog", true, 0, 0)
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventSecurityAudit, events[0].Type)
	assert.True(t, events[0].Success)
	assert.Equal(t, audit.SeverityInfo, events[0].Severity)
}

func TestService_LogSecurityAudit_Critical(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogSecurityAudit(ctx, "bad-catalog", false, 2, 1)
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.EventSecurityAudit, events[0].Type)
	assert.False(t, events[0].Success)
	assert.Equal(t, audit.SeverityCritical, events[0].Severity)
	assert.Equal(t, 2, events[0].Details["critical_count"])
	assert.Equal(t, 1, events[0].Details["high_count"])
}

func TestService_LogSecurityAudit_High(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	err := service.LogSecurityAudit(ctx, "risky-catalog", false, 0, 3)
	require.NoError(t, err)

	events := logger.Events()
	require.Len(t, events, 1)

	assert.Equal(t, audit.SeverityWarning, events[0].Severity)
}

func TestService_Query(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	_ = service.LogCatalogInstalled(ctx, "cat1", "", "", false, "", "")
	_ = service.LogCatalogInstalled(ctx, "cat2", "", "", false, "", "")
	_ = service.LogPluginInstalled(ctx, "plugin1", "", "", nil)

	filter := audit.NewQuery().WithEventTypes(audit.EventCatalogInstalled).Build()
	events, err := service.Query(ctx, filter)

	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestService_Recent(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_ = service.LogCatalogInstalled(ctx, "catalog", "", "", false, "", "")
	}

	events, err := service.Recent(ctx, 5)

	require.NoError(t, err)
	assert.Len(t, events, 5)
}

func TestService_SecurityEvents(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	_ = service.LogCatalogInstalled(ctx, "catalog", "", "", false, "", "")
	_ = service.LogSandboxViolation(ctx, "bad-plugin", "violation")
	_ = service.LogCapabilityDenied(ctx, "plugin", []string{"net"}, "blocked")

	events, err := service.SecurityEvents(ctx, 1)

	require.NoError(t, err)
	assert.Len(t, events, 2) // SandboxViolation and CapabilityDenied
}

func TestService_Summary(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)
	ctx := context.Background()

	_ = service.LogCatalogInstalled(ctx, "cat1", "", "", false, "", "")
	_ = service.LogCatalogInstalled(ctx, "cat2", "", "", false, "", "")
	_ = service.LogPluginInstalled(ctx, "plugin1", "", "", nil)

	summary, err := service.Summary(ctx, audit.QueryFilter{})

	require.NoError(t, err)
	assert.Equal(t, 3, summary.TotalEvents)
	assert.Equal(t, 3, summary.SuccessCount)
}

func TestService_Close(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	service := audit.NewService(logger)

	err := service.Close()
	assert.NoError(t, err)
}
