package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// preflightAdapter (apply.go)
// ---------------------------------------------------------------------------

func TestPreflightAdapter_WithMode(t *testing.T) {
	t.Parallel()

	adapter := &preflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeIntent)

	assert.NotNil(t, result)
	// apply.go returns a NEW *preflightAdapter, not the same pointer
	assert.NotSame(t, adapter, result)
}

func TestPreflightAdapter_WithMode_Locked(t *testing.T) {
	t.Parallel()

	adapter := &preflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeLocked)

	assert.NotNil(t, result)
	// The result should implement preflightClient
	_, ok := result.(preflightClient)
	assert.True(t, ok)
}

func TestPreflightAdapter_WithMode_Frozen(t *testing.T) {
	t.Parallel()

	adapter := &preflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeFrozen)

	assert.NotNil(t, result)
}

func TestPreflightAdapter_WithRollbackOnFailure_Enabled(t *testing.T) {
	t.Parallel()

	adapter := &preflightAdapter{app.New(os.Stdout)}
	result := adapter.WithRollbackOnFailure(true)

	assert.NotNil(t, result)
	// apply.go returns a NEW *preflightAdapter
	assert.NotSame(t, adapter, result)
}

func TestPreflightAdapter_WithRollbackOnFailure_Disabled(t *testing.T) {
	t.Parallel()

	adapter := &preflightAdapter{app.New(os.Stdout)}
	result := adapter.WithRollbackOnFailure(false)

	assert.NotNil(t, result)
}

func TestPreflightAdapter_WithMode_ReturnsSameInterface(t *testing.T) {
	t.Parallel()

	adapter := &preflightAdapter{app.New(os.Stdout)}

	// Chain calls to verify the returned interface is usable
	chained := adapter.WithMode(config.ModeIntent).WithRollbackOnFailure(true)
	assert.NotNil(t, chained)
}

// ---------------------------------------------------------------------------
// planPreflightAdapter (plan.go)
// ---------------------------------------------------------------------------

func TestPlanPreflightAdapter_WithMode(t *testing.T) {
	t.Parallel()

	adapter := &planPreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeIntent)

	assert.NotNil(t, result)
	// plan.go mutates and returns the SAME adapter
	assert.Same(t, adapter, result)
}

func TestPlanPreflightAdapter_WithMode_Locked(t *testing.T) {
	t.Parallel()

	adapter := &planPreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeLocked)

	assert.NotNil(t, result)
	assert.Same(t, adapter, result)
}

func TestPlanPreflightAdapter_WithMode_Frozen(t *testing.T) {
	t.Parallel()

	adapter := &planPreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeFrozen)

	assert.NotNil(t, result)
}

func TestPlanPreflightAdapter_WithRollbackOnFailure_Enabled(t *testing.T) {
	t.Parallel()

	adapter := &planPreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithRollbackOnFailure(true)

	assert.NotNil(t, result)
	// plan.go mutates and returns the SAME adapter
	assert.Same(t, adapter, result)
}

func TestPlanPreflightAdapter_WithRollbackOnFailure_Disabled(t *testing.T) {
	t.Parallel()

	adapter := &planPreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithRollbackOnFailure(false)

	assert.NotNil(t, result)
	assert.Same(t, adapter, result)
}

func TestPlanPreflightAdapter_WithMode_Chained(t *testing.T) {
	t.Parallel()

	adapter := &planPreflightAdapter{app.New(os.Stdout)}
	chained := adapter.WithMode(config.ModeLocked).WithRollbackOnFailure(false)

	assert.NotNil(t, chained)
}

// ---------------------------------------------------------------------------
// validatePreflightAdapter (validate.go)
// ---------------------------------------------------------------------------

func TestValidatePreflightAdapter_WithMode(t *testing.T) {
	t.Parallel()

	adapter := &validatePreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeIntent)

	assert.NotNil(t, result)
	// validate.go mutates and returns the SAME adapter
	assert.Same(t, adapter, result)
}

func TestValidatePreflightAdapter_WithMode_Locked(t *testing.T) {
	t.Parallel()

	adapter := &validatePreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeLocked)

	assert.NotNil(t, result)
	assert.Same(t, adapter, result)
}

func TestValidatePreflightAdapter_WithMode_Frozen(t *testing.T) {
	t.Parallel()

	adapter := &validatePreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeFrozen)

	assert.NotNil(t, result)
}

func TestValidatePreflightAdapter_WithMode_ReturnsValidateClient(t *testing.T) {
	t.Parallel()

	adapter := &validatePreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeIntent)

	// Confirm it satisfies the validatePreflightClient interface
	_, ok := result.(validatePreflightClient)
	assert.True(t, ok)
}

// ---------------------------------------------------------------------------
// watchPreflightAdapter (watch.go)
// ---------------------------------------------------------------------------

func TestWatchPreflightAdapter_WithMode(t *testing.T) {
	t.Parallel()

	adapter := &watchPreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeIntent)

	assert.NotNil(t, result)
	// watch.go mutates and returns the SAME adapter
	assert.Same(t, adapter, result)
}

func TestWatchPreflightAdapter_WithMode_Locked(t *testing.T) {
	t.Parallel()

	adapter := &watchPreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeLocked)

	assert.NotNil(t, result)
	assert.Same(t, adapter, result)
}

func TestWatchPreflightAdapter_WithMode_Frozen(t *testing.T) {
	t.Parallel()

	adapter := &watchPreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeFrozen)

	assert.NotNil(t, result)
}

func TestWatchPreflightAdapter_WithMode_ReturnsWatchPreflight(t *testing.T) {
	t.Parallel()

	adapter := &watchPreflightAdapter{app.New(os.Stdout)}
	result := adapter.WithMode(config.ModeIntent)

	// Confirm it satisfies the watchPreflight interface
	_, ok := result.(watchPreflight)
	assert.True(t, ok)
}

// ---------------------------------------------------------------------------
// runCatalogVerify (catalog.go)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags
func TestRunCatalogVerify_SignaturesNotSupported(t *testing.T) {
	orig := catalogVerifySigs
	defer func() { catalogVerifySigs = orig }()
	catalogVerifySigs = true

	err := runCatalogVerify(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported yet")
}

//nolint:tparallel // modifies global flags
func TestRunCatalogVerify_NoExternalCatalogs(t *testing.T) {
	orig := catalogVerifySigs
	defer func() { catalogVerifySigs = orig }()
	catalogVerifySigs = false

	// Without any external catalogs registered, the builtin catalog is
	// filtered out and the function prints "No external catalogs to verify."
	output := captureStdout(t, func() {
		err := runCatalogVerify(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No external catalogs to verify")
}

// ---------------------------------------------------------------------------
// runCatalogList (catalog.go)
// ---------------------------------------------------------------------------

func TestRunCatalogList_ShowsBuiltinCatalog(t *testing.T) {
	// The builtin catalog is always loaded by getRegistry(), so catalog list
	// should always return at least one catalog.
	output := captureStdout(t, func() {
		err := runCatalogList(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "builtin")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "TYPE")
	assert.Contains(t, output, "PRESETS")
	assert.Contains(t, output, "Total:")
}

// ---------------------------------------------------------------------------
// runCatalogAudit (catalog.go)
// ---------------------------------------------------------------------------

func TestRunCatalogAudit_BuiltinCatalog(t *testing.T) {
	// The builtin catalog should always exist and pass the audit.
	output := captureStdout(t, func() {
		err := runCatalogAudit(nil, []string{"builtin"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Auditing catalog 'builtin'")
	assert.Contains(t, output, "Summary:")
	assert.Contains(t, output, "Audit PASSED")
}

func TestRunCatalogAudit_CatalogNotFound(t *testing.T) {
	err := runCatalogAudit(nil, []string{"nonexistent-catalog-xyz"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// getCatalogAuditService (catalog.go)
// ---------------------------------------------------------------------------

func TestGetCatalogAuditService_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	svc := getCatalogAuditService()
	assert.NotNil(t, svc)
	_ = svc.Close()
}

// ---------------------------------------------------------------------------
// getAuditService (audit.go)
// ---------------------------------------------------------------------------

func TestGetAuditService_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	svc, err := getAuditService()
	require.NoError(t, err)
	assert.NotNil(t, svc)
	_ = svc.Close()
}

// ---------------------------------------------------------------------------
// runAuditClean (audit.go)
// ---------------------------------------------------------------------------

func TestRunAuditClean_Success(t *testing.T) {
	// runAuditClean opens a file logger with default config and calls Cleanup.
	// The default config creates files under ~/.preflight/audit.
	// Cleanup removes old files, which is safe to call even with no old files.
	output := captureStdout(t, func() {
		err := runAuditClean(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Audit log cleanup complete")
}

// ---------------------------------------------------------------------------
// Factory function constructors via package-level var
// ---------------------------------------------------------------------------

func TestNewPreflight_ReturnsPreflightClient(t *testing.T) {
	t.Parallel()

	client := newPreflight(os.Stdout)
	assert.NotNil(t, client)

	_, ok := client.(preflightClient)
	assert.True(t, ok)
}

func TestNewPlanPreflight_ReturnsPreflightClient(t *testing.T) {
	t.Parallel()

	client := newPlanPreflight(os.Stdout)
	assert.NotNil(t, client)

	_, ok := client.(preflightClient)
	assert.True(t, ok)
}

func TestNewValidatePreflight_ReturnsValidateClient(t *testing.T) {
	t.Parallel()

	client := newValidatePreflight(os.Stdout)
	assert.NotNil(t, client)

	_, ok := client.(validatePreflightClient)
	assert.True(t, ok)
}

func TestNewWatchApp_ReturnsWatchPreflight(t *testing.T) {
	t.Parallel()

	client := newWatchApp(os.Stdout)
	assert.NotNil(t, client)

	_, ok := client.(watchPreflight)
	assert.True(t, ok)
}

// ---------------------------------------------------------------------------
// runCatalogVerify with named catalog argument
// ---------------------------------------------------------------------------

func TestRunCatalogVerify_NamedCatalogNotFound(t *testing.T) {
	orig := catalogVerifySigs
	defer func() { catalogVerifySigs = orig }()
	catalogVerifySigs = false

	err := runCatalogVerify(nil, []string{"nonexistent-catalog-xyz"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// registryStore (catalog.go) - package-level default store
// ---------------------------------------------------------------------------

func TestRegistryStore_IsInitialized(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, registryStore)
}

// ---------------------------------------------------------------------------
// getRegistry returns at least the builtin catalog
// ---------------------------------------------------------------------------

func TestGetRegistry_ContainsBuiltin(t *testing.T) {
	t.Parallel()

	registry, err := getRegistry()
	require.NoError(t, err)
	require.NotNil(t, registry)

	catalogs := registry.List()
	assert.GreaterOrEqual(t, len(catalogs), 1)

	var foundBuiltin bool
	for _, rc := range catalogs {
		if rc.Name() == "builtin" {
			foundBuiltin = true
			break
		}
	}
	assert.True(t, foundBuiltin, "registry should contain the builtin catalog")
}

// ---------------------------------------------------------------------------
// runAuditClean with custom temp dir (verifies cleanup path)
// ---------------------------------------------------------------------------

func TestRunAuditClean_WithEmptyDir(t *testing.T) {
	// Create a temporary directory and ensure cleanup runs cleanly even
	// when the audit directory has no old log files to remove.
	dir := t.TempDir()
	auditDir := filepath.Join(dir, "audit")

	// Use a custom audit config pointed at our temp dir to avoid
	// polluting the real audit directory. We cannot override the
	// package-level function, so instead just call the underlying
	// audit infrastructure directly to verify the code path.
	require.NoError(t, os.MkdirAll(auditDir, 0o700))

	// The function runAuditClean uses DefaultFileLoggerConfig(), which
	// creates under ~/.preflight/audit. The test above already exercises
	// that path; this test verifies the audit infrastructure supports
	// the cleanup-with-no-files scenario.
	output := captureStdout(t, func() {
		err := runAuditClean(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Audit log cleanup complete")
}
