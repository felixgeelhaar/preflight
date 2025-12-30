package lockfile

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
	"github.com/felixgeelhaar/preflight/internal/domain/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncAdapter_ToLockfileState(t *testing.T) {
	t.Parallel()

	adapter := NewSyncAdapter()

	t.Run("nil lockfile returns empty state", func(t *testing.T) {
		t.Parallel()
		state := adapter.ToLockfileState(nil)
		require.NotNil(t, state)
		assert.True(t, state.IsEmpty())
	})

	t.Run("converts lockfile with packages", func(t *testing.T) {
		t.Parallel()

		machineInfo := createTestMachineInfo(t)
		lockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)

		// Add a package
		pkg := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")
		require.NoError(t, lockfile.AddPackage(pkg))

		state := adapter.ToLockfileState(lockfile)
		require.NotNil(t, state)
		assert.Len(t, state.Packages, 1)

		info, ok := state.Packages["brew:ripgrep"]
		require.True(t, ok)
		assert.Equal(t, "14.1.0", info.Version())
	})

	t.Run("preserves sync metadata", func(t *testing.T) {
		t.Parallel()

		machineID := mustParseMachineID(t, "abc123")
		vector := sync.NewVersionVector().Increment(machineID)
		syncMeta := sync.NewSyncMetadata(vector)

		machineInfo := createTestMachineInfo(t)
		lockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		lockfile = lockfile.WithSyncMetadata(syncMeta)

		state := adapter.ToLockfileState(lockfile)
		require.NotNil(t, state)
		assert.Equal(t, uint64(1), state.Metadata.Vector().Get(machineID.String()))
	})
}

func TestSyncAdapter_PackageLockToInfo(t *testing.T) {
	t.Parallel()

	adapter := NewSyncAdapter()

	pkg := createTestPackageLock(t, "brew", "git", "2.43.0")
	info := adapter.PackageLockToInfo(pkg)

	assert.Equal(t, "2.43.0", info.Version())
	assert.Equal(t, pkg.InstalledAt(), info.ModifiedAt())
	assert.True(t, info.Provenance().IsZero()) // No provenance in lock.PackageLock yet
}

func TestSyncAdapter_CompareStates(t *testing.T) {
	t.Parallel()

	adapter := NewSyncAdapter()
	machineInfo := createTestMachineInfo(t)
	machineID := mustParseMachineID(t, "machine-a")

	t.Run("nil lockfiles are concurrent", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, sync.Concurrent, adapter.CompareStates(nil, nil))
	})

	t.Run("one nil is concurrent", func(t *testing.T) {
		t.Parallel()
		local := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		assert.Equal(t, sync.Concurrent, adapter.CompareStates(local, nil))
		assert.Equal(t, sync.Concurrent, adapter.CompareStates(nil, local))
	})

	t.Run("equal states", func(t *testing.T) {
		t.Parallel()

		vector := sync.NewVersionVector().Increment(machineID)
		syncMeta := sync.NewSyncMetadata(vector)

		local := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		local = local.WithSyncMetadata(syncMeta)

		remote := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		remote = remote.WithSyncMetadata(syncMeta)

		assert.Equal(t, sync.Equal, adapter.CompareStates(local, remote))
	})

	t.Run("local ahead of remote", func(t *testing.T) {
		t.Parallel()

		remoteVector := sync.NewVersionVector().Increment(machineID)
		localVector := remoteVector.Increment(machineID)

		local := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		local = local.WithSyncMetadata(sync.NewSyncMetadata(localVector))

		remote := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		remote = remote.WithSyncMetadata(sync.NewSyncMetadata(remoteVector))

		assert.Equal(t, sync.After, adapter.CompareStates(local, remote))
	})

	t.Run("local behind remote", func(t *testing.T) {
		t.Parallel()

		localVector := sync.NewVersionVector().Increment(machineID)
		remoteVector := localVector.Increment(machineID)

		local := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		local = local.WithSyncMetadata(sync.NewSyncMetadata(localVector))

		remote := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		remote = remote.WithSyncMetadata(sync.NewSyncMetadata(remoteVector))

		assert.Equal(t, sync.Before, adapter.CompareStates(local, remote))
	})

	t.Run("concurrent changes", func(t *testing.T) {
		t.Parallel()

		machineA := mustParseMachineID(t, "machine-a")
		machineB := mustParseMachineID(t, "machine-b")

		// Local modified by A, remote modified by B
		localVector := sync.NewVersionVector().Increment(machineA)
		remoteVector := sync.NewVersionVector().Increment(machineB)

		local := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		local = local.WithSyncMetadata(sync.NewSyncMetadata(localVector))

		remote := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		remote = remote.WithSyncMetadata(sync.NewSyncMetadata(remoteVector))

		assert.Equal(t, sync.Concurrent, adapter.CompareStates(local, remote))
	})
}

func TestSyncAdapter_NeedsMerge(t *testing.T) {
	t.Parallel()

	adapter := NewSyncAdapter()
	machineInfo := createTestMachineInfo(t)

	machineA := mustParseMachineID(t, "machine-a")
	machineB := mustParseMachineID(t, "machine-b")

	// Concurrent changes need merge
	localVector := sync.NewVersionVector().Increment(machineA)
	remoteVector := sync.NewVersionVector().Increment(machineB)

	local := lock.NewLockfileV2(config.ModeIntent, machineInfo)
	local = local.WithSyncMetadata(sync.NewSyncMetadata(localVector))

	remote := lock.NewLockfileV2(config.ModeIntent, machineInfo)
	remote = remote.WithSyncMetadata(sync.NewSyncMetadata(remoteVector))

	assert.True(t, adapter.NeedsMerge(local, remote))

	// Same state doesn't need merge
	same := lock.NewLockfileV2(config.ModeIntent, machineInfo)
	same = same.WithSyncMetadata(sync.NewSyncMetadata(localVector))

	assert.False(t, adapter.NeedsMerge(local, same))
}

func TestSyncAdapter_IsAhead(t *testing.T) {
	t.Parallel()

	adapter := NewSyncAdapter()
	machineInfo := createTestMachineInfo(t)
	machineID := mustParseMachineID(t, "machine-a")

	remoteVector := sync.NewVersionVector().Increment(machineID)
	localVector := remoteVector.Increment(machineID)

	local := lock.NewLockfileV2(config.ModeIntent, machineInfo)
	local = local.WithSyncMetadata(sync.NewSyncMetadata(localVector))

	remote := lock.NewLockfileV2(config.ModeIntent, machineInfo)
	remote = remote.WithSyncMetadata(sync.NewSyncMetadata(remoteVector))

	assert.True(t, adapter.IsAhead(local, remote))
	assert.False(t, adapter.IsBehind(local, remote))
}

func TestSyncAdapter_IsBehind(t *testing.T) {
	t.Parallel()

	adapter := NewSyncAdapter()
	machineInfo := createTestMachineInfo(t)
	machineID := mustParseMachineID(t, "machine-a")

	localVector := sync.NewVersionVector().Increment(machineID)
	remoteVector := localVector.Increment(machineID)

	local := lock.NewLockfileV2(config.ModeIntent, machineInfo)
	local = local.WithSyncMetadata(sync.NewSyncMetadata(localVector))

	remote := lock.NewLockfileV2(config.ModeIntent, machineInfo)
	remote = remote.WithSyncMetadata(sync.NewSyncMetadata(remoteVector))

	assert.True(t, adapter.IsBehind(local, remote))
	assert.False(t, adapter.IsAhead(local, remote))
}

func TestSyncAdapter_IsInSync(t *testing.T) {
	t.Parallel()

	adapter := NewSyncAdapter()
	machineInfo := createTestMachineInfo(t)
	machineID := mustParseMachineID(t, "machine-a")

	vector := sync.NewVersionVector().Increment(machineID)
	syncMeta := sync.NewSyncMetadata(vector)

	local := lock.NewLockfileV2(config.ModeIntent, machineInfo)
	local = local.WithSyncMetadata(syncMeta)

	remote := lock.NewLockfileV2(config.ModeIntent, machineInfo)
	remote = remote.WithSyncMetadata(syncMeta)

	assert.True(t, adapter.IsInSync(local, remote))
}

// Helper functions

func createTestMachineInfo(t *testing.T) lock.MachineInfo {
	t.Helper()
	info, err := lock.NewMachineInfo("darwin", "arm64", "test-host", time.Now())
	require.NoError(t, err)
	return info
}

//nolint:unparam // provider is parameterized for flexibility
func createTestPackageLock(t *testing.T, provider, name, version string) lock.PackageLock {
	t.Helper()
	// SHA256 hash must be 64 hex characters
	integrity, err := lock.NewIntegrity(lock.AlgorithmSHA256, "abc123def456abc123def456abc123def456abc123def456abc123def456abcd")
	require.NoError(t, err)

	pkg, err := lock.NewPackageLock(provider, name, version, integrity, time.Now())
	require.NoError(t, err)
	return pkg
}

// Test UUIDs for machine IDs - mapping test names to UUIDs
var testMachineUUIDs = map[string]string{
	"abc123":    "550e8400-e29b-41d4-a716-446655440001",
	"machine-a": "550e8400-e29b-41d4-a716-446655440002",
	"machine-b": "550e8400-e29b-41d4-a716-446655440003",
}

func mustParseMachineID(t *testing.T, name string) sync.MachineID {
	t.Helper()
	uuid, ok := testMachineUUIDs[name]
	if !ok {
		t.Fatalf("no test UUID defined for %q", name)
	}
	machineID, err := sync.ParseMachineID(uuid)
	require.NoError(t, err)
	return machineID
}

func TestSyncAdapter_ApplyMergeResult(t *testing.T) {
	t.Parallel()

	adapter := NewSyncAdapter()
	machineInfo := createTestMachineInfo(t)
	machineID := mustParseMachineID(t, "machine-a")

	t.Run("nil result returns unchanged lockfile", func(t *testing.T) {
		t.Parallel()

		lockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		result, err := adapter.ApplyMergeResult(lockfile, nil)
		require.NoError(t, err)
		assert.Equal(t, lockfile, result)
	})

	t.Run("nil state in result returns unchanged lockfile", func(t *testing.T) {
		t.Parallel()

		lockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		mergeResult := &sync.MergeResult{
			State:   nil,
			Changes: nil,
		}
		result, err := adapter.ApplyMergeResult(lockfile, mergeResult)
		require.NoError(t, err)
		assert.Equal(t, lockfile, result)
	})

	t.Run("updates sync metadata", func(t *testing.T) {
		t.Parallel()

		vector := sync.NewVersionVector().Increment(machineID)
		syncMeta := sync.NewSyncMetadata(vector)
		state := sync.NewLockfileStateWithMetadata(syncMeta)

		lockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		mergeResult := &sync.MergeResult{
			State:   state,
			Changes: nil,
		}

		result, err := adapter.ApplyMergeResult(lockfile, mergeResult)
		require.NoError(t, err)
		assert.Equal(t, uint64(1), result.SyncMetadata().Vector().Get(machineID.String()))
	})

	t.Run("removes package on ChangeRemoved", func(t *testing.T) {
		t.Parallel()

		lockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		pkg := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")
		require.NoError(t, lockfile.AddPackage(pkg))

		// Verify package exists
		_, exists := lockfile.GetPackage("brew", "ripgrep")
		require.True(t, exists)

		vector := sync.NewVersionVector().Increment(machineID)
		syncMeta := sync.NewSyncMetadata(vector)
		state := sync.NewLockfileStateWithMetadata(syncMeta)

		mergeResult := &sync.MergeResult{
			State: state,
			Changes: []sync.MergeChange{
				{
					PackageKey: "brew:ripgrep",
					Type:       sync.ChangeRemoved,
				},
			},
		}

		result, err := adapter.ApplyMergeResult(lockfile, mergeResult)
		require.NoError(t, err)

		// Package should be removed
		_, exists = result.GetPackage("brew", "ripgrep")
		assert.False(t, exists)
	})

	t.Run("updates package version on ChangeUpdated", func(t *testing.T) {
		t.Parallel()

		lockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		pkg := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
		require.NoError(t, lockfile.AddPackage(pkg))

		vector := sync.NewVersionVector().Increment(machineID)
		syncMeta := sync.NewSyncMetadata(vector)
		state := sync.NewLockfileStateWithMetadata(syncMeta)

		newInfo := sync.NewPackageLockInfoWithTime("14.1.0", sync.PackageProvenance{}, time.Now())

		mergeResult := &sync.MergeResult{
			State: state,
			Changes: []sync.MergeChange{
				{
					PackageKey: "brew:ripgrep",
					Type:       sync.ChangeUpdated,
					After:      newInfo,
				},
			},
		}

		result, err := adapter.ApplyMergeResult(lockfile, mergeResult)
		require.NoError(t, err)

		// Package should have new version
		updatedPkg, exists := result.GetPackage("brew", "ripgrep")
		require.True(t, exists)
		assert.Equal(t, "14.1.0", updatedPkg.Version())
	})

	t.Run("ChangeKept makes no changes", func(t *testing.T) {
		t.Parallel()

		lockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		pkg := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")
		require.NoError(t, lockfile.AddPackage(pkg))

		vector := sync.NewVersionVector().Increment(machineID)
		syncMeta := sync.NewSyncMetadata(vector)
		state := sync.NewLockfileStateWithMetadata(syncMeta)

		mergeResult := &sync.MergeResult{
			State: state,
			Changes: []sync.MergeChange{
				{
					PackageKey: "brew:ripgrep",
					Type:       sync.ChangeKept,
				},
			},
		}

		result, err := adapter.ApplyMergeResult(lockfile, mergeResult)
		require.NoError(t, err)

		// Package should be unchanged
		keptPkg, exists := result.GetPackage("brew", "ripgrep")
		require.True(t, exists)
		assert.Equal(t, "14.1.0", keptPkg.Version())
	})

	t.Run("returns error for invalid package key", func(t *testing.T) {
		t.Parallel()

		lockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)

		vector := sync.NewVersionVector().Increment(machineID)
		syncMeta := sync.NewSyncMetadata(vector)
		state := sync.NewLockfileStateWithMetadata(syncMeta)

		mergeResult := &sync.MergeResult{
			State: state,
			Changes: []sync.MergeChange{
				{
					PackageKey: "invalid-key-no-colon",
					Type:       sync.ChangeRemoved,
				},
			},
		}

		_, err := adapter.ApplyMergeResult(lockfile, mergeResult)
		assert.Error(t, err)
	})

	t.Run("ChangeAdded copies package from remote lockfile", func(t *testing.T) {
		t.Parallel()

		// Local lockfile has no packages
		localLockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)

		// Remote lockfile has the package we want to add
		remoteLockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		remotePkg := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")
		require.NoError(t, remoteLockfile.AddPackage(remotePkg))

		vector := sync.NewVersionVector().Increment(machineID)
		syncMeta := sync.NewSyncMetadata(vector)
		state := sync.NewLockfileStateWithMetadata(syncMeta)

		newInfo := sync.NewPackageLockInfoWithTime("14.1.0", sync.PackageProvenance{}, time.Now())

		mergeResult := &sync.MergeResult{
			State: state,
			Changes: []sync.MergeChange{
				{
					PackageKey: "brew:ripgrep",
					Type:       sync.ChangeAdded,
					After:      newInfo,
				},
			},
		}

		result, err := adapter.ApplyMergeResultWithRemote(localLockfile, mergeResult, remoteLockfile)
		require.NoError(t, err)

		// Package should be added from remote
		addedPkg, exists := result.GetPackage("brew", "ripgrep")
		require.True(t, exists)
		assert.Equal(t, "14.1.0", addedPkg.Version())
	})

	t.Run("ChangeAdded updates existing package if present locally", func(t *testing.T) {
		t.Parallel()

		// Local lockfile already has the package (older version)
		localLockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)
		localPkg := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
		require.NoError(t, localLockfile.AddPackage(localPkg))

		vector := sync.NewVersionVector().Increment(machineID)
		syncMeta := sync.NewSyncMetadata(vector)
		state := sync.NewLockfileStateWithMetadata(syncMeta)

		newInfo := sync.NewPackageLockInfoWithTime("14.1.0", sync.PackageProvenance{}, time.Now())

		mergeResult := &sync.MergeResult{
			State: state,
			Changes: []sync.MergeChange{
				{
					PackageKey: "brew:ripgrep",
					Type:       sync.ChangeAdded,
					After:      newInfo,
				},
			},
		}

		// Even without remote lockfile, should update existing package
		result, err := adapter.ApplyMergeResultWithRemote(localLockfile, mergeResult, nil)
		require.NoError(t, err)

		// Package should be updated
		updatedPkg, exists := result.GetPackage("brew", "ripgrep")
		require.True(t, exists)
		assert.Equal(t, "14.1.0", updatedPkg.Version())
	})

	t.Run("ChangeAdded without remote skips silently", func(t *testing.T) {
		t.Parallel()

		// Local lockfile has no packages, no remote provided
		localLockfile := lock.NewLockfileV2(config.ModeIntent, machineInfo)

		vector := sync.NewVersionVector().Increment(machineID)
		syncMeta := sync.NewSyncMetadata(vector)
		state := sync.NewLockfileStateWithMetadata(syncMeta)

		newInfo := sync.NewPackageLockInfoWithTime("14.1.0", sync.PackageProvenance{}, time.Now())

		mergeResult := &sync.MergeResult{
			State: state,
			Changes: []sync.MergeChange{
				{
					PackageKey: "brew:ripgrep",
					Type:       sync.ChangeAdded,
					After:      newInfo,
				},
			},
		}

		// Without remote lockfile and no local package, should skip silently
		result, err := adapter.ApplyMergeResultWithRemote(localLockfile, mergeResult, nil)
		require.NoError(t, err)

		// Package should NOT exist (couldn't be added)
		_, exists := result.GetPackage("brew", "ripgrep")
		assert.False(t, exists)
	})
}
