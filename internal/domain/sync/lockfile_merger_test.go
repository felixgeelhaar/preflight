package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLockfileMerger(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	merger := NewLockfileMerger(machineID, "test-host")

	assert.NotNil(t, merger)
	assert.Equal(t, machineID, merger.machineID)
	assert.Equal(t, "test-host", merger.hostname)
}

func TestMergeChangeType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		changeType MergeChangeType
		want       string
	}{
		{ChangeAdded, "added"},
		{ChangeRemoved, "removed"},
		{ChangeUpdated, "updated"},
		{ChangeKept, "kept"},
		{MergeChangeType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.changeType.String())
		})
	}
}

func TestLockfileMerger_Merge(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	merger := NewLockfileMerger(machineID, "test-host")

	t.Run("nil result returns error", func(t *testing.T) {
		t.Parallel()
		_, err := merger.Merge(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sync result is required")
	})

	t.Run("nil merged state returns error", func(t *testing.T) {
		t.Parallel()
		result := &SyncResult{Merged: nil}
		_, err := merger.Merge(result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no merged state")
	})

	t.Run("successful merge with resolutions", func(t *testing.T) {
		t.Parallel()

		// Create a sync result with various resolutions
		merged := NewLockfileState()
		merged.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", PackageProvenance{}))
		merged.AddPackage("brew:fd", NewPackageLockInfo("9.0.0", PackageProvenance{}))

		// Create conflicts for the resolutions
		localOnly := NewLockConflict("brew:ripgrep", LocalOnly,
			NewPackageLockInfo("14.1.0", PackageProvenance{}),
			PackageLockInfo{},
			PackageLockInfo{},
		)
		remoteOnly := NewLockConflict("brew:fd", RemoteOnly,
			PackageLockInfo{},
			NewPackageLockInfo("9.0.0", PackageProvenance{}),
			PackageLockInfo{},
		)

		resolutions := []Resolution{
			NewResolution(localOnly, ChooseLocal, localOnly.Local(), "auto: local addition"),
			NewResolution(remoteOnly, ChooseRemote, remoteOnly.Remote(), "auto: remote addition"),
		}

		result := &SyncResult{
			Merged:      merged,
			Resolutions: resolutions,
		}

		mergeResult, err := merger.Merge(result)
		require.NoError(t, err)
		require.NotNil(t, mergeResult)

		assert.Equal(t, 2, mergeResult.Stats.TotalPackages)
		assert.Len(t, mergeResult.Changes, 2)
	})

	t.Run("merge with skipped resolution", func(t *testing.T) {
		t.Parallel()

		merged := NewLockfileState()
		merged.AddPackage("brew:git", NewPackageLockInfo("2.40.0", PackageProvenance{}))

		conflict := NewLockConflict("brew:git", VersionMismatch,
			NewPackageLockInfo("2.40.0", PackageProvenance{}),
			NewPackageLockInfo("2.41.0", PackageProvenance{}),
			PackageLockInfo{},
		)

		skippedRes := NewResolution(conflict, ChooseSkip, PackageLockInfo{}, "manually skipped")

		result := &SyncResult{
			Merged:      merged,
			Resolutions: []Resolution{skippedRes},
		}

		mergeResult, err := merger.Merge(result)
		require.NoError(t, err)

		assert.Equal(t, 1, mergeResult.Stats.Kept)
		assert.Equal(t, ChangeKept, mergeResult.Changes[0].Type)
	})

	t.Run("merge with delete resolution", func(t *testing.T) {
		t.Parallel()

		merged := NewLockfileState()
		// Package is removed, so not in merged state

		conflict := NewLockConflict("brew:removed", BothModified,
			NewPackageLockInfo("1.0.0", PackageProvenance{}),
			PackageLockInfo{}, // Remote deleted
			NewPackageLockInfo("1.0.0", PackageProvenance{}),
		)

		deleteRes := NewResolution(conflict, ChooseRemote, PackageLockInfo{}, "selected remote (deleted)")

		result := &SyncResult{
			Merged:      merged,
			Resolutions: []Resolution{deleteRes},
		}

		mergeResult, err := merger.Merge(result)
		require.NoError(t, err)

		assert.Equal(t, 1, mergeResult.Stats.Removed)
		assert.Equal(t, ChangeRemoved, mergeResult.Changes[0].Type)
	})
}

func TestLockfileMerger_ApplyResolution(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	merger := NewLockfileMerger(machineID, "test-host")
	engine := NewSyncEngine()

	t.Run("apply manual resolution", func(t *testing.T) {
		t.Parallel()

		// Create a conflict that requires manual resolution by using concurrent versions
		// (version vectors that don't have happened-before relationship)
		machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
		machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440001")

		vectorA := NewVersionVector().Increment(machineA) // {A:1}
		vectorB := NewVersionVector().Increment(machineB) // {B:1} - concurrent with vectorA

		localProv := NewPackageProvenance(machineA, vectorA)
		remoteProv := NewPackageProvenance(machineB, vectorB)

		local := NewLockfileState()
		local.AddPackage("brew:git", NewPackageLockInfo("2.40.0", localProv))

		remote := NewLockfileState()
		remote.AddPackage("brew:git", NewPackageLockInfo("2.41.0", remoteProv))

		// Use manual resolution strategy to ensure we get a conflict
		manualEngine := NewSyncEngine(WithResolver(NewConflictResolver(StrategyManual)))
		result, err := manualEngine.Sync(SyncInput{Local: local, Remote: remote})
		require.NoError(t, err)
		require.NotEmpty(t, result.ManualConflicts, "Expected manual conflicts with StrategyManual")

		conflict := result.ManualConflicts[0]
		change, err := merger.ApplyResolution(manualEngine, result, conflict, ChooseRemote)
		require.NoError(t, err)

		assert.Equal(t, "brew:git", change.PackageKey)
		assert.Equal(t, ChangeUpdated, change.Type)
		assert.Equal(t, "2.41.0", change.After.Version())
	})

	t.Run("apply resolution not found", func(t *testing.T) {
		t.Parallel()

		result := &SyncResult{
			Merged:          NewLockfileState(),
			ManualConflicts: []LockConflict{},
		}

		nonExistent := NewLockConflict("brew:fake", VersionMismatch,
			NewPackageLockInfo("1.0.0", PackageProvenance{}),
			NewPackageLockInfo("2.0.0", PackageProvenance{}),
			PackageLockInfo{},
		)

		_, err := merger.ApplyResolution(engine, result, nonExistent, ChooseLocal)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conflict not found")
	})
}

func TestLockfileMerger_FromPackageLocks(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	merger := NewLockfileMerger(machineID, "test-host")

	packages := map[string]PackageLockInfo{
		"brew:ripgrep": NewPackageLockInfo("14.1.0", PackageProvenance{}),
		"brew:fd":      NewPackageLockInfo("9.0.0", PackageProvenance{}),
	}

	vector := NewVersionVector()
	vector = vector.Increment(machineID)
	meta := NewSyncMetadata(vector)

	state := merger.FromPackageLocks(packages, meta)

	assert.Len(t, state.Packages, 2)
	assert.Equal(t, "14.1.0", state.Packages["brew:ripgrep"].Version())
	assert.Equal(t, "9.0.0", state.Packages["brew:fd"].Version())
	assert.True(t, state.Metadata.Vector().Equals(vector))
}

func TestLockfileMerger_UpdateProvenance(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	merger := NewLockfileMerger(machineID, "test-host")

	vector := NewVersionVector()
	vector = vector.Increment(machineID)

	info := NewPackageLockInfo("14.1.0", PackageProvenance{})
	updated := merger.UpdateProvenance(info, vector)

	assert.Equal(t, "14.1.0", updated.Version())
	assert.Equal(t, machineID.String(), updated.Provenance().ModifiedBy())
	assert.True(t, updated.Provenance().VectorAtChange().Equals(vector))
}

func TestLockfileMerger_IncrementVector(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	merger := NewLockfileMerger(machineID, "test-host")

	vector := NewVersionVector()
	assert.Equal(t, uint64(0), vector.Get(machineID.String()))

	newVector := merger.IncrementVector(vector)
	assert.Equal(t, uint64(1), newVector.Get(machineID.String()))

	newVector = merger.IncrementVector(newVector)
	assert.Equal(t, uint64(2), newVector.Get(machineID.String()))
}

func TestLockfileMerger_PrepareForCommit(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	merger := NewLockfileMerger(machineID, "test-host")

	t.Run("nil state returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, merger.PrepareForCommit(nil))
	})

	t.Run("prepares state for commit", func(t *testing.T) {
		t.Parallel()

		state := NewLockfileState()
		state.AddPackage("brew:git", NewPackageLockInfo("2.40.0", PackageProvenance{}))

		prepared := merger.PrepareForCommit(state)

		// Vector should be incremented
		assert.Equal(t, uint64(1), prepared.Metadata.Vector().Get(machineID.String()))

		// Packages should be preserved
		assert.Len(t, prepared.Packages, 1)
		assert.Equal(t, "2.40.0", prepared.Packages["brew:git"].Version())

		// Lineage should include this machine
		lineage, ok := prepared.Metadata.GetLineage(machineID.String())
		assert.True(t, ok)
		assert.Equal(t, "test-host", lineage.Hostname())
	})

	t.Run("preserves existing lineage", func(t *testing.T) {
		t.Parallel()

		otherMachine, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440001")
		existingLineage := NewMachineLineage(otherMachine.String(), "other-host", time.Now().Add(-time.Hour))

		meta := NewSyncMetadata(NewVersionVector()).AddLineage(existingLineage)
		state := NewLockfileStateWithMetadata(meta)

		prepared := merger.PrepareForCommit(state)

		// Both machines should be in lineage
		_, hasOther := prepared.Metadata.GetLineage(otherMachine.String())
		_, hasSelf := prepared.Metadata.GetLineage(machineID.String())
		assert.True(t, hasOther)
		assert.True(t, hasSelf)
	})
}

func TestLockfileMerger_Diff(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	merger := NewLockfileMerger(machineID, "test-host")

	t.Run("nil states handled gracefully", func(t *testing.T) {
		t.Parallel()
		changes := merger.Diff(nil, nil)
		assert.Empty(t, changes)
	})

	t.Run("additions detected", func(t *testing.T) {
		t.Parallel()

		before := NewLockfileState()
		after := NewLockfileState()
		after.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", PackageProvenance{}))

		changes := merger.Diff(before, after)

		require.Len(t, changes, 1)
		assert.Equal(t, "brew:ripgrep", changes[0].PackageKey)
		assert.Equal(t, ChangeAdded, changes[0].Type)
	})

	t.Run("removals detected", func(t *testing.T) {
		t.Parallel()

		before := NewLockfileState()
		before.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", PackageProvenance{}))
		after := NewLockfileState()

		changes := merger.Diff(before, after)

		require.Len(t, changes, 1)
		assert.Equal(t, "brew:ripgrep", changes[0].PackageKey)
		assert.Equal(t, ChangeRemoved, changes[0].Type)
	})

	t.Run("updates detected", func(t *testing.T) {
		t.Parallel()

		before := NewLockfileState()
		before.AddPackage("brew:ripgrep", NewPackageLockInfo("14.0.0", PackageProvenance{}))

		after := NewLockfileState()
		after.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", PackageProvenance{}))

		changes := merger.Diff(before, after)

		require.Len(t, changes, 1)
		assert.Equal(t, "brew:ripgrep", changes[0].PackageKey)
		assert.Equal(t, ChangeUpdated, changes[0].Type)
		assert.Equal(t, "14.0.0", changes[0].Before.Version())
		assert.Equal(t, "14.1.0", changes[0].After.Version())
	})

	t.Run("unchanged packages not reported", func(t *testing.T) {
		t.Parallel()

		before := NewLockfileState()
		before.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", PackageProvenance{}))

		after := NewLockfileState()
		after.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", PackageProvenance{}))

		changes := merger.Diff(before, after)
		assert.Empty(t, changes)
	})

	t.Run("mixed changes detected", func(t *testing.T) {
		t.Parallel()

		before := NewLockfileState()
		before.AddPackage("brew:ripgrep", NewPackageLockInfo("14.0.0", PackageProvenance{}))
		before.AddPackage("brew:removed", NewPackageLockInfo("1.0.0", PackageProvenance{}))

		after := NewLockfileState()
		after.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", PackageProvenance{}))
		after.AddPackage("brew:added", NewPackageLockInfo("2.0.0", PackageProvenance{}))

		changes := merger.Diff(before, after)

		assert.Len(t, changes, 3)

		// Find each change type
		var added, removed, updated, kept int
		for _, c := range changes {
			switch c.Type {
			case ChangeAdded:
				added++
			case ChangeRemoved:
				removed++
			case ChangeUpdated:
				updated++
			case ChangeKept:
				kept++
			}
		}
		assert.Equal(t, 1, added)
		assert.Equal(t, 1, removed)
		assert.Equal(t, 1, updated)
		assert.Equal(t, 0, kept)
	})
}

func TestPackageLockInfo_Equals(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineID)
	prov := NewPackageProvenance(machineID, vector)

	t.Run("equal values", func(t *testing.T) {
		t.Parallel()
		a := NewPackageLockInfo("1.0.0", prov)
		b := NewPackageLockInfo("1.0.0", prov)
		assert.True(t, a.Equals(b))
	})

	t.Run("different versions", func(t *testing.T) {
		t.Parallel()
		a := NewPackageLockInfo("1.0.0", prov)
		b := NewPackageLockInfo("2.0.0", prov)
		assert.False(t, a.Equals(b))
	})

	t.Run("different provenance", func(t *testing.T) {
		t.Parallel()
		otherMachine, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440001")
		otherProv := NewPackageProvenance(otherMachine, vector)

		a := NewPackageLockInfo("1.0.0", prov)
		b := NewPackageLockInfo("1.0.0", otherProv)
		assert.False(t, a.Equals(b))
	})
}

func TestPackageLockInfo_With(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineID)
	prov := NewPackageProvenance(machineID, vector)

	t.Run("WithProvenance", func(t *testing.T) {
		t.Parallel()

		info := NewPackageLockInfo("1.0.0", PackageProvenance{})
		updated := info.WithProvenance(prov)

		assert.Equal(t, "1.0.0", updated.Version())
		assert.Equal(t, machineID.String(), updated.Provenance().ModifiedBy())
	})

	t.Run("WithModifiedAt", func(t *testing.T) {
		t.Parallel()

		info := NewPackageLockInfo("1.0.0", PackageProvenance{})
		targetTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
		updated := info.WithModifiedAt(targetTime)

		assert.Equal(t, "1.0.0", updated.Version())
		assert.Equal(t, targetTime, updated.ModifiedAt())
	})
}

func TestPackageProvenance_Equals(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440001")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)

	t.Run("equal provenances", func(t *testing.T) {
		t.Parallel()
		a := NewPackageProvenance(machineA, vectorA)
		b := NewPackageProvenance(machineA, vectorA)
		assert.True(t, a.Equals(b))
	})

	t.Run("different machines", func(t *testing.T) {
		t.Parallel()
		a := NewPackageProvenance(machineA, vectorA)
		b := NewPackageProvenance(machineB, vectorA)
		assert.False(t, a.Equals(b))
	})

	t.Run("different vectors", func(t *testing.T) {
		t.Parallel()
		a := NewPackageProvenance(machineA, vectorA)
		b := NewPackageProvenance(machineA, vectorB)
		assert.False(t, a.Equals(b))
	})
}

func TestSyncMetadata_WithLineage(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440001")

	lineageA := NewMachineLineage(machineA.String(), "host-a", time.Now())
	lineageB := NewMachineLineage(machineB.String(), "host-b", time.Now())

	t.Run("sets lineage from map", func(t *testing.T) {
		t.Parallel()

		lineageMap := map[string]MachineLineage{
			machineA.String(): lineageA,
			machineB.String(): lineageB,
		}

		meta := NewSyncMetadata(NewVersionVector()).WithLineage(lineageMap)

		resultLineage := meta.Lineage()
		assert.Len(t, resultLineage, 2)

		la, ok := meta.GetLineage(machineA.String())
		assert.True(t, ok)
		assert.Equal(t, "host-a", la.Hostname())
	})

	t.Run("creates defensive copy", func(t *testing.T) {
		t.Parallel()

		lineageMap := map[string]MachineLineage{
			machineA.String(): lineageA,
		}

		meta := NewSyncMetadata(NewVersionVector()).WithLineage(lineageMap)

		// Modify original map
		lineageMap[machineB.String()] = lineageB

		// Meta should not be affected
		resultLineage := meta.Lineage()
		assert.Len(t, resultLineage, 1)
	})
}
