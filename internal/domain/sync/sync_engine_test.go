package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSyncEngine(t *testing.T) {
	t.Parallel()

	engine := NewSyncEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.detector)
	assert.NotNil(t, engine.resolver)
}

func TestNewSyncEngine_WithOptions(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	resolver := NewConflictResolver(StrategyLocalWins)

	engine := NewSyncEngine(
		WithResolver(resolver),
		WithMachineID(machineID, "test-host"),
	)

	assert.Equal(t, StrategyLocalWins, engine.resolver.Strategy())
	assert.Equal(t, machineID, engine.machineID)
	assert.Equal(t, "test-host", engine.hostname)
}

func TestLockfileState(t *testing.T) {
	t.Parallel()

	state := NewLockfileState()
	assert.True(t, state.IsEmpty())

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	state.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", prov))
	assert.False(t, state.IsEmpty())
	assert.Len(t, state.Packages, 1)
}

func TestLockfileState_WithMetadata(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	meta := NewSyncMetadata(vector)

	state := NewLockfileStateWithMetadata(meta)
	assert.False(t, state.Metadata.IsZero())
	assert.Equal(t, uint64(1), state.Metadata.Vector().Get(machineA.String()))
}

func TestSyncEngine_Sync_NoConflicts(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	local := NewLockfileState()
	local.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", prov))
	local.AddPackage("brew:fd", NewPackageLockInfo("9.0.0", prov))

	remote := NewLockfileState()
	remote.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", prov))
	remote.AddPackage("brew:fd", NewPackageLockInfo("9.0.0", prov))

	engine := NewSyncEngine()
	result, err := engine.Sync(SyncInput{Local: local, Remote: remote})

	require.NoError(t, err)
	assert.True(t, result.IsClean())
	assert.Empty(t, result.ManualConflicts)
	assert.Len(t, result.Merged.Packages, 2)
	assert.Equal(t, 2, result.Stats.PackagesUnchanged)
}

func TestSyncEngine_Sync_LocalAddition(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	local := NewLockfileState()
	local.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", prov))
	local.AddPackage("brew:fd", NewPackageLockInfo("9.0.0", prov)) // Local only

	remote := NewLockfileState()
	remote.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", prov))

	engine := NewSyncEngine()
	result, err := engine.Sync(SyncInput{Local: local, Remote: remote})

	require.NoError(t, err)
	assert.True(t, result.IsClean())
	assert.Len(t, result.Merged.Packages, 2)
	assert.Contains(t, result.Merged.Packages, "brew:fd")
}

func TestSyncEngine_Sync_RemoteAddition(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")
	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)
	provA := NewPackageProvenance(machineA, vectorA)
	provB := NewPackageProvenance(machineB, vectorB)

	local := NewLockfileState()
	local.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", provA))

	remote := NewLockfileState()
	remote.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", provA))
	remote.AddPackage("brew:bat", NewPackageLockInfo("0.24.0", provB)) // Remote only

	engine := NewSyncEngine()
	result, err := engine.Sync(SyncInput{Local: local, Remote: remote})

	require.NoError(t, err)
	assert.True(t, result.IsClean())
	assert.Len(t, result.Merged.Packages, 2)
	assert.Contains(t, result.Merged.Packages, "brew:bat")
}

func TestSyncEngine_Sync_SequentialUpdate(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := vectorA.Increment(machineB) // B happened after A

	provA := NewPackageProvenance(machineA, vectorA)
	provB := NewPackageProvenance(machineB, vectorB)

	local := NewLockfileState()
	local.AddPackage("brew:ripgrep", NewPackageLockInfo("14.0.0", provA))

	remote := NewLockfileState()
	remote.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", provB))

	engine := NewSyncEngine()
	result, err := engine.Sync(SyncInput{Local: local, Remote: remote})

	require.NoError(t, err)
	assert.True(t, result.IsClean())
	// Remote is newer, should take remote version
	assert.Equal(t, "14.1.0", result.Merged.Packages["brew:ripgrep"].Version())
}

func TestSyncEngine_Sync_ConcurrentConflict(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB) // Concurrent

	provA := NewPackageProvenance(machineA, vectorA)
	provB := NewPackageProvenance(machineB, vectorB)

	local := NewLockfileState()
	local.AddPackage("brew:ripgrep", NewPackageLockInfo("14.0.0", provA))

	remote := NewLockfileState()
	remote.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", provB))

	engine := NewSyncEngine()
	result, err := engine.Sync(SyncInput{Local: local, Remote: remote})

	require.NoError(t, err)
	assert.True(t, result.HasManualConflicts())
	assert.Len(t, result.ManualConflicts, 1)
	assert.Equal(t, "brew:ripgrep", result.ManualConflicts[0].PackageKey())
}

func TestSyncEngine_Sync_ThreeWay(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	baseVector := NewVersionVector()
	vectorA := baseVector.Increment(machineA)
	vectorB := baseVector.Increment(machineB)

	baseProv := NewPackageProvenance(machineA, baseVector)
	provA := NewPackageProvenance(machineA, vectorA)
	provB := NewPackageProvenance(machineB, vectorB)

	base := NewLockfileState()
	base.AddPackage("brew:ripgrep", NewPackageLockInfo("13.0.0", baseProv))

	local := NewLockfileState()
	local.AddPackage("brew:ripgrep", NewPackageLockInfo("14.0.0", provA))

	remote := NewLockfileState()
	remote.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", provB))

	engine := NewSyncEngine()
	result, err := engine.ThreeWaySync(local, remote, base)

	require.NoError(t, err)
	// Both modified from base - concurrent conflict
	assert.True(t, result.HasManualConflicts())
}

func TestSyncEngine_Sync_NilInputs(t *testing.T) {
	t.Parallel()

	engine := NewSyncEngine()

	_, err := engine.Sync(SyncInput{Local: nil, Remote: NewLockfileState()})
	assert.ErrorIs(t, err, ErrNoLocalState)

	_, err = engine.Sync(SyncInput{Local: NewLockfileState(), Remote: nil})
	assert.ErrorIs(t, err, ErrNoRemoteState)
}

func TestSyncEngine_ResolveManualConflict(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)

	provA := NewPackageProvenance(machineA, vectorA)
	provB := NewPackageProvenance(machineB, vectorB)

	local := NewLockfileState()
	local.AddPackage("brew:ripgrep", NewPackageLockInfo("14.0.0", provA))

	remote := NewLockfileState()
	remote.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", provB))

	engine := NewSyncEngine()
	result, err := engine.Sync(SyncInput{Local: local, Remote: remote})
	require.NoError(t, err)
	require.Len(t, result.ManualConflicts, 1)

	// Resolve choosing remote
	conflict := result.ManualConflicts[0]
	err = engine.ResolveManualConflict(result, conflict, ChooseRemote)

	require.NoError(t, err)
	assert.Empty(t, result.ManualConflicts)
	assert.True(t, result.IsClean())
	assert.Equal(t, "14.1.0", result.Merged.Packages["brew:ripgrep"].Version())
}

func TestSyncEngine_ResolveManualConflict_NotFound(t *testing.T) {
	t.Parallel()

	engine := NewSyncEngine()
	result := &SyncResult{
		Merged:          NewLockfileState(),
		ManualConflicts: []LockConflict{},
	}

	fakeConflict := NewLockConflict("brew:fake", LocalOnly, PackageLockInfo{}, PackageLockInfo{}, PackageLockInfo{})
	err := engine.ResolveManualConflict(result, fakeConflict, ChooseLocal)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSyncEngine_CompareStates(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name     string
		localFn  func() *LockfileState
		remoteFn func() *LockfileState
		want     CausalRelation
	}{
		{
			name: "equal states",
			localFn: func() *LockfileState {
				v := NewVersionVector().Increment(machineA)
				return NewLockfileStateWithMetadata(NewSyncMetadata(v))
			},
			remoteFn: func() *LockfileState {
				v := NewVersionVector().Increment(machineA)
				return NewLockfileStateWithMetadata(NewSyncMetadata(v))
			},
			want: Equal,
		},
		{
			name: "local ahead",
			localFn: func() *LockfileState {
				v := NewVersionVector().Increment(machineA).Increment(machineA)
				return NewLockfileStateWithMetadata(NewSyncMetadata(v))
			},
			remoteFn: func() *LockfileState {
				v := NewVersionVector().Increment(machineA)
				return NewLockfileStateWithMetadata(NewSyncMetadata(v))
			},
			want: After,
		},
		{
			name: "local behind",
			localFn: func() *LockfileState {
				v := NewVersionVector().Increment(machineA)
				return NewLockfileStateWithMetadata(NewSyncMetadata(v))
			},
			remoteFn: func() *LockfileState {
				v := NewVersionVector().Increment(machineA).Increment(machineA)
				return NewLockfileStateWithMetadata(NewSyncMetadata(v))
			},
			want: Before,
		},
		{
			name: "concurrent",
			localFn: func() *LockfileState {
				v := NewVersionVector().Increment(machineA)
				return NewLockfileStateWithMetadata(NewSyncMetadata(v))
			},
			remoteFn: func() *LockfileState {
				v := NewVersionVector().Increment(machineB)
				return NewLockfileStateWithMetadata(NewSyncMetadata(v))
			},
			want: Concurrent,
		},
		{
			name: "nil local",
			localFn: func() *LockfileState {
				return nil
			},
			remoteFn: NewLockfileState,
			want:     Concurrent,
		},
	}

	engine := NewSyncEngine()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			local := tt.localFn()
			remote := tt.remoteFn()
			assert.Equal(t, tt.want, engine.CompareStates(local, remote))
		})
	}
}

func TestSyncEngine_StateHelpers(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	engine := NewSyncEngine()

	// Equal states
	v1 := NewVersionVector().Increment(machineA)
	equal1 := NewLockfileStateWithMetadata(NewSyncMetadata(v1))
	equal2 := NewLockfileStateWithMetadata(NewSyncMetadata(v1))
	assert.True(t, engine.IsInSync(equal1, equal2))
	assert.False(t, engine.NeedsMerge(equal1, equal2))

	// Local ahead
	v2 := v1.Increment(machineA)
	ahead := NewLockfileStateWithMetadata(NewSyncMetadata(v2))
	behind := NewLockfileStateWithMetadata(NewSyncMetadata(v1))
	assert.True(t, engine.IsAhead(ahead, behind))
	assert.True(t, engine.IsBehind(behind, ahead))
	assert.False(t, engine.NeedsMerge(ahead, behind))

	// Concurrent
	concA := NewLockfileStateWithMetadata(NewSyncMetadata(NewVersionVector().Increment(machineA)))
	concB := NewLockfileStateWithMetadata(NewSyncMetadata(NewVersionVector().Increment(machineB)))
	assert.True(t, engine.NeedsMerge(concA, concB))
	assert.False(t, engine.IsAhead(concA, concB))
	assert.False(t, engine.IsBehind(concA, concB))
}

func TestSyncEngine_Pull(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	local := NewLockfileState()
	local.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", prov))

	remote := NewLockfileState()
	remote.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", prov))
	remote.AddPackage("brew:bat", NewPackageLockInfo("0.24.0", prov))

	engine := NewSyncEngine()
	result, err := engine.Pull(local, remote)

	require.NoError(t, err)
	assert.True(t, result.IsClean())
	assert.Len(t, result.Merged.Packages, 2)
}

func TestSyncEngine_Push(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	local := NewLockfileState()
	local.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", prov))
	local.AddPackage("brew:fd", NewPackageLockInfo("9.0.0", prov))

	remote := NewLockfileState()
	remote.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", prov))

	engine := NewSyncEngine()
	result, err := engine.Push(local, remote)

	require.NoError(t, err)
	assert.True(t, result.IsClean())
	assert.Len(t, result.Merged.Packages, 2)
}

func TestSyncEngine_MergedMetadata(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)

	metaA := NewSyncMetadata(vectorA)
	metaB := NewSyncMetadata(vectorB)

	local := NewLockfileStateWithMetadata(metaA)
	remote := NewLockfileStateWithMetadata(metaB)

	engine := NewSyncEngine(WithMachineID(machineA, "local-host"))
	result, err := engine.Sync(SyncInput{Local: local, Remote: remote})

	require.NoError(t, err)

	// Merged metadata should contain both vectors merged, plus local activity
	mergedVec := result.Merged.Metadata.Vector()
	assert.GreaterOrEqual(t, mergedVec.Get(machineA.String()), uint64(1))
	assert.Equal(t, uint64(1), mergedVec.Get(machineB.String()))
}

func TestSyncResult_Stats(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := vectorA.Increment(machineB)

	provA := NewPackageProvenance(machineA, vectorA)
	provB := NewPackageProvenance(machineB, vectorB)

	local := NewLockfileState()
	local.AddPackage("brew:ripgrep", NewPackageLockInfo("14.0.0", provA)) // Will be updated
	local.AddPackage("brew:fd", NewPackageLockInfo("9.0.0", provA))       // Unchanged
	local.AddPackage("brew:jq", NewPackageLockInfo("1.7", provA))         // Local only

	remote := NewLockfileState()
	remote.AddPackage("brew:ripgrep", NewPackageLockInfo("14.1.0", provB)) // Newer
	remote.AddPackage("brew:fd", NewPackageLockInfo("9.0.0", provA))       // Same
	remote.AddPackage("brew:bat", NewPackageLockInfo("0.24.0", provB))     // Remote only

	engine := NewSyncEngine()
	result, err := engine.Sync(SyncInput{Local: local, Remote: remote})

	require.NoError(t, err)
	assert.True(t, result.IsClean())
	assert.Equal(t, 1, result.Stats.PackagesUnchanged) // fd
	assert.GreaterOrEqual(t, result.Stats.PackagesAdded, 1)
	assert.GreaterOrEqual(t, result.Stats.ConflictsAutoResolved, 1)
}

func TestSyncEngine_ResolveManualConflict_Delete(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)

	provA := NewPackageProvenance(machineA, vectorA)

	// Local has package, remote deleted it (simulated by not having it + base having it)
	local := NewLockfileState()
	local.AddPackage("brew:ripgrep", NewPackageLockInfo("14.0.0", provA))

	remote := NewLockfileStateWithMetadata(NewSyncMetadata(vectorB))
	// No ripgrep in remote

	base := NewLockfileState()
	base.AddPackage("brew:ripgrep", NewPackageLockInfo("13.0.0", PackageProvenance{}))

	engine := NewSyncEngine()
	result, err := engine.ThreeWaySync(local, remote, base)
	require.NoError(t, err)

	if len(result.ManualConflicts) > 0 {
		// Resolve by choosing remote (delete)
		conflict := result.ManualConflicts[0]
		err = engine.ResolveManualConflict(result, conflict, ChooseRemote)
		require.NoError(t, err)

		assert.NotContains(t, result.Merged.Packages, "brew:ripgrep")
	}
}

func TestLockfileState_IsEmpty_Nil(t *testing.T) {
	t.Parallel()

	var nilState *LockfileState
	assert.True(t, nilState.IsEmpty())
}
