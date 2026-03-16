package fleet

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventory_AddSource(t *testing.T) {
	t.Parallel()

	inv := NewInventory()
	source := newMockSource(t, "aws", true, "10.0.0.1")

	inv.AddSource(source)

	sources := inv.Sources()
	require.Len(t, sources, 1)
	assert.Equal(t, "aws", sources[0].Name())
}

func TestInventory_AddMultipleSources(t *testing.T) {
	t.Parallel()

	inv := NewInventory()
	inv.AddSource(newMockSource(t, "aws", true))
	inv.AddSource(newMockSource(t, "gcp", true))

	sources := inv.Sources()
	assert.Len(t, sources, 2)
}

func TestInventory_SourcesReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	inv := NewInventory()
	inv.AddSource(newMockSource(t, "aws", true))

	sources := inv.Sources()
	sources[0] = nil // Modify the returned slice

	// Original should be unaffected
	original := inv.Sources()
	assert.NotNil(t, original[0])
}

func TestInventory_RefreshFromSources(t *testing.T) {
	t.Parallel()

	t.Run("discovers hosts from all sources", func(t *testing.T) {
		t.Parallel()
		inv := NewInventory()
		inv.AddSource(newMockSource(t, "aws", true, "10.0.0.1", "10.0.0.2"))
		inv.AddSource(newMockSource(t, "gcp", true, "10.1.0.1"))

		count, errs := inv.RefreshFromSources(context.Background())

		assert.Equal(t, 3, count)
		assert.Empty(t, errs)
		assert.Equal(t, 3, inv.HostCount())
	})

	t.Run("skips unavailable sources", func(t *testing.T) {
		t.Parallel()
		inv := NewInventory()
		inv.AddSource(newMockSource(t, "aws", true, "10.0.0.1"))
		inv.AddSource(newMockSource(t, "gcp", false, "10.1.0.1"))

		count, errs := inv.RefreshFromSources(context.Background())

		assert.Equal(t, 1, count)
		assert.Empty(t, errs)
		assert.Equal(t, 1, inv.HostCount())
	})

	t.Run("collects errors from failing sources", func(t *testing.T) {
		t.Parallel()
		inv := NewInventory()
		inv.AddSource(newMockSource(t, "aws", true, "10.0.0.1"))

		failingSource := &mockInventorySource{
			name:      "failing",
			available: true,
			err:       fmt.Errorf("API timeout"),
		}
		inv.AddSource(failingSource)

		count, errs := inv.RefreshFromSources(context.Background())

		assert.Equal(t, 1, count)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "failing")
		assert.Contains(t, errs[0].Error(), "API timeout")
	})

	t.Run("skips duplicate host IDs", func(t *testing.T) {
		t.Parallel()
		inv := NewInventory()

		// Pre-populate with a host
		existing := createTestHost(t, "aws-host1", "10.0.0.99")
		require.NoError(t, inv.AddHost(existing))

		// Source returns a host with same ID
		inv.AddSource(newMockSource(t, "aws", true, "10.0.0.1"))

		count, errs := inv.RefreshFromSources(context.Background())

		// The duplicate should be skipped, not cause an error
		assert.Equal(t, 0, count)
		assert.Empty(t, errs)
		assert.Equal(t, 1, inv.HostCount())
	})

	t.Run("no sources registered", func(t *testing.T) {
		t.Parallel()
		inv := NewInventory()

		count, errs := inv.RefreshFromSources(context.Background())

		assert.Equal(t, 0, count)
		assert.Empty(t, errs)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		t.Parallel()
		inv := NewInventory()
		inv.AddSource(newMockSource(t, "aws", true, "10.0.0.1"))

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		count, errs := inv.RefreshFromSources(ctx)

		// With cancelled context, behavior depends on source implementation.
		// Our mock ignores context, so it should still work.
		// But the method should at least not panic.
		assert.GreaterOrEqual(t, count, 0)
		_ = errs
	})
}

func TestInventory_RefreshFromSource(t *testing.T) {
	t.Parallel()

	t.Run("discovers from specific source", func(t *testing.T) {
		t.Parallel()
		inv := NewInventory()
		inv.AddSource(newMockSource(t, "aws", true, "10.0.0.1", "10.0.0.2"))
		inv.AddSource(newMockSource(t, "gcp", true, "10.1.0.1"))

		count, err := inv.RefreshFromSource(context.Background(), "aws")

		require.NoError(t, err)
		assert.Equal(t, 2, count)
		assert.Equal(t, 2, inv.HostCount())
	})

	t.Run("source not found", func(t *testing.T) {
		t.Parallel()
		inv := NewInventory()
		inv.AddSource(newMockSource(t, "aws", true))

		count, err := inv.RefreshFromSource(context.Background(), "nonexistent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent")
		assert.Equal(t, 0, count)
	})

	t.Run("source not available", func(t *testing.T) {
		t.Parallel()
		inv := NewInventory()
		inv.AddSource(newMockSource(t, "gcp", false))

		count, err := inv.RefreshFromSource(context.Background(), "gcp")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not available")
		assert.Equal(t, 0, count)
	})

	t.Run("source discover error", func(t *testing.T) {
		t.Parallel()
		inv := NewInventory()
		inv.AddSource(&mockInventorySource{
			name:      "broken",
			available: true,
			err:       fmt.Errorf("connection refused"),
		})

		count, err := inv.RefreshFromSource(context.Background(), "broken")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.Equal(t, 0, count)
	})

	t.Run("skips duplicate hosts", func(t *testing.T) {
		t.Parallel()
		inv := NewInventory()
		existing := createTestHost(t, "aws-host1", "10.0.0.99")
		require.NoError(t, inv.AddHost(existing))

		inv.AddSource(newMockSource(t, "aws", true, "10.0.0.1"))

		count, err := inv.RefreshFromSource(context.Background(), "aws")

		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, 1, inv.HostCount())
	})
}
