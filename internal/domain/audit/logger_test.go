package audit_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryLogger_Log(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	ctx := context.Background()

	event := audit.NewEvent(audit.EventCatalogInstalled).
		WithCatalog("test-catalog").
		Build()

	err := logger.Log(ctx, event)
	require.NoError(t, err)

	events := logger.Events()
	assert.Len(t, events, 1)
	assert.Equal(t, "test-catalog", events[0].Catalog)
}

func TestMemoryLogger_Query(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	ctx := context.Background()

	// Log several events
	_ = logger.Log(ctx, audit.NewEvent(audit.EventCatalogInstalled).WithCatalog("cat1").Build())
	_ = logger.Log(ctx, audit.NewEvent(audit.EventPluginInstalled).WithPlugin("plugin1").Build())
	_ = logger.Log(ctx, audit.NewEvent(audit.EventCatalogInstalled).WithCatalog("cat2").Build())

	// Query all catalog events
	filter := audit.NewQuery().
		WithEventTypes(audit.EventCatalogInstalled).
		Build()

	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestMemoryLogger_Query_WithLimit(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	ctx := context.Background()

	// Log 5 events
	for i := 0; i < 5; i++ {
		_ = logger.Log(ctx, audit.NewEvent(audit.EventCatalogInstalled).Build())
	}

	filter := audit.NewQuery().Limit(3).Build()
	events, err := logger.Query(ctx, filter)

	require.NoError(t, err)
	assert.Len(t, events, 3)
}

func TestMemoryLogger_Clear(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	ctx := context.Background()

	_ = logger.Log(ctx, audit.NewEvent(audit.EventCatalogInstalled).Build())
	assert.Len(t, logger.Events(), 1)

	logger.Clear()
	assert.Empty(t, logger.Events())
}

func TestMemoryLogger_Close(t *testing.T) {
	t.Parallel()

	logger := audit.NewMemoryLogger()
	err := logger.Close()
	assert.NoError(t, err)
}

func TestNullLogger_Log(t *testing.T) {
	t.Parallel()

	logger := audit.NewNullLogger()
	ctx := context.Background()

	event := audit.NewEvent(audit.EventCatalogInstalled).Build()
	err := logger.Log(ctx, event)

	assert.NoError(t, err)
}

func TestNullLogger_Query(t *testing.T) {
	t.Parallel()

	logger := audit.NewNullLogger()
	ctx := context.Background()

	events, err := logger.Query(ctx, audit.QueryFilter{})

	assert.NoError(t, err)
	assert.Nil(t, events)
}

func TestNullLogger_Close(t *testing.T) {
	t.Parallel()

	logger := audit.NewNullLogger()
	err := logger.Close()
	assert.NoError(t, err)
}

func TestFileLogger_Log(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	config := audit.FileLoggerConfig{
		Dir:          dir,
		MaxSize:      1024 * 1024,
		MaxAge:       24 * time.Hour,
		MaxRotations: 3,
	}

	logger, err := audit.NewFileLogger(config)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	ctx := context.Background()
	event := audit.NewEvent(audit.EventCatalogInstalled).
		WithCatalog("test-catalog").
		Build()

	err = logger.Log(ctx, event)
	require.NoError(t, err)

	// Verify file exists
	logPath := filepath.Join(dir, "audit.jsonl")
	_, err = os.Stat(logPath)
	assert.NoError(t, err)
}

func TestFileLogger_Query(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	config := audit.FileLoggerConfig{
		Dir:          dir,
		MaxSize:      1024 * 1024,
		MaxAge:       24 * time.Hour,
		MaxRotations: 3,
	}

	logger, err := audit.NewFileLogger(config)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	ctx := context.Background()

	// Log events
	_ = logger.Log(ctx, audit.NewEvent(audit.EventCatalogInstalled).WithCatalog("cat1").Build())
	_ = logger.Log(ctx, audit.NewEvent(audit.EventPluginInstalled).WithPlugin("plugin1").Build())
	_ = logger.Log(ctx, audit.NewEvent(audit.EventCatalogInstalled).WithCatalog("cat2").Build())

	// Query
	filter := audit.NewQuery().
		WithEventTypes(audit.EventCatalogInstalled).
		Build()

	events, err := logger.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestFileLogger_Rotation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	config := audit.FileLoggerConfig{
		Dir:          dir,
		MaxSize:      100, // Very small to trigger rotation
		MaxAge:       24 * time.Hour,
		MaxRotations: 3,
	}

	logger, err := audit.NewFileLogger(config)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	ctx := context.Background()

	// Log enough events to trigger rotation
	for i := 0; i < 10; i++ {
		event := audit.NewEvent(audit.EventCatalogInstalled).
			WithCatalog("catalog-" + string(rune('a'+i))).
			WithSource("https://example.com/catalog.yaml").
			Build()
		_ = logger.Log(ctx, event)
	}

	// Check that rotated files exist
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var logFiles int
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".jsonl" {
			logFiles++
		}
	}
	assert.GreaterOrEqual(t, logFiles, 1)
}

func TestFileLogger_Cleanup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	config := audit.FileLoggerConfig{
		Dir:          dir,
		MaxSize:      1024 * 1024,
		MaxAge:       0, // Immediate expiry
		MaxRotations: 3,
	}

	logger, err := audit.NewFileLogger(config)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	ctx := context.Background()

	// Log an event
	_ = logger.Log(ctx, audit.NewEvent(audit.EventCatalogInstalled).Build())

	// Cleanup
	err = logger.Cleanup()
	assert.NoError(t, err)
}

func TestFileLogger_Close(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	config := audit.FileLoggerConfig{
		Dir:          dir,
		MaxSize:      1024 * 1024,
		MaxAge:       24 * time.Hour,
		MaxRotations: 3,
	}

	logger, err := audit.NewFileLogger(config)
	require.NoError(t, err)

	err = logger.Close()
	assert.NoError(t, err)
}

func TestFileLogger_InvalidDir(t *testing.T) {
	t.Parallel()

	config := audit.FileLoggerConfig{
		Dir:          "/nonexistent/path/that/cannot/be/created\x00invalid",
		MaxSize:      1024 * 1024,
		MaxAge:       24 * time.Hour,
		MaxRotations: 3,
	}

	_, err := audit.NewFileLogger(config)
	assert.Error(t, err)
}

func TestDefaultFileLoggerConfig(t *testing.T) {
	t.Parallel()

	config := audit.DefaultFileLoggerConfig()

	assert.NotEmpty(t, config.Dir)
	assert.Greater(t, config.MaxSize, int64(0))
	assert.Greater(t, config.MaxAge, time.Duration(0))
	assert.Greater(t, config.MaxRotations, 0)
}
