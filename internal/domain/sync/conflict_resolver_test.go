package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolutionStrategy_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		strategy ResolutionStrategy
		want     string
	}{
		{StrategyManual, "manual"},
		{StrategyNewest, "newest"},
		{StrategyLocalWins, "local-wins"},
		{StrategyRemoteWins, "remote-wins"},
		{StrategyAuto, "auto"},
		{ResolutionStrategy(99), "unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.strategy.String())
	}
}

func TestParseResolutionStrategy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    ResolutionStrategy
		wantErr bool
	}{
		{"manual", StrategyManual, false},
		{"newest", StrategyNewest, false},
		{"local-wins", StrategyLocalWins, false},
		{"local", StrategyLocalWins, false},
		{"remote-wins", StrategyRemoteWins, false},
		{"remote", StrategyRemoteWins, false},
		{"auto", StrategyAuto, false},
		{"invalid", StrategyManual, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := ParseResolutionStrategy(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestResolutionChoice_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		choice ResolutionChoice
		want   string
	}{
		{ChooseLocal, "local"},
		{ChooseRemote, "remote"},
		{ChooseBase, "base"},
		{ChooseSkip, "skip"},
		{ResolutionChoice(99), "unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.choice.String())
	}
}

func TestResolution(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)
	localInfo := NewPackageLockInfo("14.0.0", prov)
	remoteInfo := NewPackageLockInfo("14.1.0", prov)

	conflict := NewLockConflict("brew:ripgrep", VersionMismatch, localInfo, remoteInfo, PackageLockInfo{})
	resolution := NewResolution(conflict, ChooseLocal, localInfo, "test reason")

	assert.Equal(t, conflict.PackageKey(), resolution.Conflict().PackageKey())
	assert.Equal(t, ChooseLocal, resolution.Choice())
	assert.Equal(t, "14.0.0", resolution.Result().Version())
	assert.Equal(t, "test reason", resolution.Reason())
	assert.False(t, resolution.IsZero())
	assert.False(t, resolution.IsSkipped())
	assert.False(t, resolution.IsDelete())
}

func TestResolution_IsSkipped(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfo("14.0.0", prov),
		NewPackageLockInfo("14.1.0", prov),
		PackageLockInfo{})

	resolution := NewResolution(conflict, ChooseSkip, PackageLockInfo{}, "skipped")
	assert.True(t, resolution.IsSkipped())
}

func TestResolution_IsDelete(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	// Conflict where remote deleted the package
	conflict := NewLockConflict("brew:ripgrep", BothModified,
		NewPackageLockInfo("14.0.0", prov),
		PackageLockInfo{}, // Remote deleted
		NewPackageLockInfo("13.0.0", prov))

	// Choose remote (deletion)
	resolution := NewResolution(conflict, ChooseRemote, PackageLockInfo{}, "deleted")
	assert.True(t, resolution.IsDelete())
}

func TestNewConflictResolver(t *testing.T) {
	t.Parallel()

	resolver := NewConflictResolver(StrategyAuto)
	assert.Equal(t, StrategyAuto, resolver.Strategy())
}

func TestConflictResolver_StrategyManual(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	resolver := NewConflictResolver(StrategyManual)

	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfo("14.0.0", prov),
		NewPackageLockInfo("14.1.0", prov),
		PackageLockInfo{})

	_, needsManual := resolver.Resolve(conflict)
	assert.True(t, needsManual, "manual strategy should always require manual resolution")
}

func TestConflictResolver_StrategyLocalWins(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	resolver := NewConflictResolver(StrategyLocalWins)

	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfo("14.0.0", prov),
		NewPackageLockInfo("14.1.0", prov),
		PackageLockInfo{})

	resolution, needsManual := resolver.Resolve(conflict)
	assert.False(t, needsManual)
	assert.Equal(t, ChooseLocal, resolution.Choice())
	assert.Equal(t, "14.0.0", resolution.Result().Version())
}

func TestConflictResolver_StrategyRemoteWins(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	resolver := NewConflictResolver(StrategyRemoteWins)

	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfo("14.0.0", prov),
		NewPackageLockInfo("14.1.0", prov),
		PackageLockInfo{})

	resolution, needsManual := resolver.Resolve(conflict)
	assert.False(t, needsManual)
	assert.Equal(t, ChooseRemote, resolution.Choice())
	assert.Equal(t, "14.1.0", resolution.Result().Version())
}

func TestConflictResolver_StrategyNewest_ByVector(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	// Local vector is ahead of remote
	vectorA := NewVersionVector().Increment(machineA).Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)

	resolver := NewConflictResolver(StrategyNewest)

	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vectorA)),
		NewPackageLockInfo("14.1.0", NewPackageProvenance(machineB, vectorB)),
		PackageLockInfo{})

	resolution, needsManual := resolver.Resolve(conflict)
	// Concurrent vectors - can't determine, needs manual
	assert.True(t, needsManual)
	assert.True(t, resolution.IsZero())
}

func TestConflictResolver_StrategyNewest_ByVectorCausal(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	// B's vector includes A's changes (sequential)
	vectorA := NewVersionVector().Increment(machineA)
	vectorB := vectorA.Increment(machineB) // B happened after A

	resolver := NewConflictResolver(StrategyNewest)

	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vectorA)),
		NewPackageLockInfo("14.1.0", NewPackageProvenance(machineB, vectorB)),
		PackageLockInfo{})

	resolution, needsManual := resolver.Resolve(conflict)
	assert.False(t, needsManual)
	assert.Equal(t, ChooseRemote, resolution.Choice())
	assert.Contains(t, resolution.Reason(), "later vector")
}

func TestConflictResolver_StrategyNewest_ByTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	resolver := NewConflictResolver(StrategyNewest)

	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfoWithTime("14.0.0", PackageProvenance{}, earlier),
		NewPackageLockInfoWithTime("14.1.0", PackageProvenance{}, later),
		PackageLockInfo{})

	resolution, needsManual := resolver.Resolve(conflict)
	assert.False(t, needsManual)
	assert.Equal(t, ChooseRemote, resolution.Choice())
	assert.Contains(t, resolution.Reason(), "later timestamp")
}

func TestConflictResolver_StrategyNewest_DeleteNeedsManual(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)

	resolver := NewConflictResolver(StrategyNewest)

	// One side deleted
	conflict := NewLockConflict("brew:ripgrep", BothModified,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vector)),
		PackageLockInfo{}, // Deleted remotely
		PackageLockInfo{})

	_, needsManual := resolver.Resolve(conflict)
	assert.True(t, needsManual, "delete/modify conflicts need manual resolution with newest strategy")
}

func TestConflictResolver_StrategyAuto_LocalOnly(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)

	resolver := NewConflictResolver(StrategyAuto)

	conflict := NewLockConflict("brew:fd", LocalOnly,
		NewPackageLockInfo("9.0.0", NewPackageProvenance(machineA, vector)),
		PackageLockInfo{},
		PackageLockInfo{})

	resolution, needsManual := resolver.Resolve(conflict)
	assert.False(t, needsManual)
	assert.Equal(t, ChooseLocal, resolution.Choice())
	assert.Contains(t, resolution.Reason(), "local addition")
}

func TestConflictResolver_StrategyAuto_RemoteOnly(t *testing.T) {
	t.Parallel()

	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineB)

	resolver := NewConflictResolver(StrategyAuto)

	conflict := NewLockConflict("brew:bat", RemoteOnly,
		PackageLockInfo{},
		NewPackageLockInfo("0.24.0", NewPackageProvenance(machineB, vector)),
		PackageLockInfo{})

	resolution, needsManual := resolver.Resolve(conflict)
	assert.False(t, needsManual)
	assert.Equal(t, ChooseRemote, resolution.Choice())
	assert.Contains(t, resolution.Reason(), "remote addition")
}

func TestConflictResolver_StrategyAuto_VersionMismatch_Sequential(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := vectorA.Increment(machineB)

	resolver := NewConflictResolver(StrategyAuto)

	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vectorA)),
		NewPackageLockInfo("14.1.0", NewPackageProvenance(machineB, vectorB)),
		PackageLockInfo{})

	resolution, needsManual := resolver.Resolve(conflict)
	assert.False(t, needsManual)
	assert.Equal(t, ChooseRemote, resolution.Choice())
	assert.Contains(t, resolution.Reason(), "happened-after")
}

func TestConflictResolver_StrategyAuto_BothModified_Concurrent(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)

	resolver := NewConflictResolver(StrategyAuto)

	conflict := NewLockConflict("brew:ripgrep", BothModified,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vectorA)),
		NewPackageLockInfo("14.1.0", NewPackageProvenance(machineB, vectorB)),
		PackageLockInfo{})

	_, needsManual := resolver.Resolve(conflict)
	assert.True(t, needsManual, "concurrent modifications need manual resolution")
}

func TestConflictResolver_ResolveAll(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := vectorA.Increment(machineB) // Sequential
	concurrentB := NewVersionVector().Increment(machineB)

	resolver := NewConflictResolver(StrategyAuto)

	conflicts := []LockConflict{
		// Auto-resolvable: local addition
		NewLockConflict("brew:fd", LocalOnly,
			NewPackageLockInfo("9.0.0", NewPackageProvenance(machineA, vectorA)),
			PackageLockInfo{},
			PackageLockInfo{}),
		// Auto-resolvable: sequential change
		NewLockConflict("brew:ripgrep", VersionMismatch,
			NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vectorA)),
			NewPackageLockInfo("14.1.0", NewPackageProvenance(machineB, vectorB)),
			PackageLockInfo{}),
		// Needs manual: concurrent change
		NewLockConflict("brew:bat", BothModified,
			NewPackageLockInfo("0.23.0", NewPackageProvenance(machineA, vectorA)),
			NewPackageLockInfo("0.24.0", NewPackageProvenance(machineB, concurrentB)),
			PackageLockInfo{}),
	}

	resolved, manual := resolver.ResolveAll(conflicts)

	assert.Len(t, resolved, 2)
	assert.Len(t, manual, 1)
	assert.Equal(t, "brew:bat", manual[0].PackageKey())
}

func TestResolveManually(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	localInfo := NewPackageLockInfo("14.0.0", prov)
	remoteInfo := NewPackageLockInfo("14.1.0", prov)
	baseInfo := NewPackageLockInfo("13.0.0", prov)

	conflict := NewLockConflict("brew:ripgrep", BothModified, localInfo, remoteInfo, baseInfo)

	tests := []struct {
		name   string
		choice ResolutionChoice
		want   string
	}{
		{"local", ChooseLocal, "14.0.0"},
		{"remote", ChooseRemote, "14.1.0"},
		{"base", ChooseBase, "13.0.0"},
		{"skip", ChooseSkip, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resolution := ResolveManually(conflict, tt.choice)
			assert.Equal(t, tt.choice, resolution.Choice())
			assert.Equal(t, tt.want, resolution.Result().Version())
			assert.Contains(t, resolution.Reason(), "manually")
		})
	}
}

func TestResolution_IsZero(t *testing.T) {
	t.Parallel()

	var zero Resolution
	assert.True(t, zero.IsZero())

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	conflict := NewLockConflict("brew:ripgrep", LocalOnly,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vector)),
		PackageLockInfo{},
		PackageLockInfo{})
	resolution := NewResolution(conflict, ChooseLocal, conflict.Local(), "test")
	assert.False(t, resolution.IsZero())
}

func TestConflictResolver_StrategyNewest_EqualTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	resolver := NewConflictResolver(StrategyNewest)

	// Same timestamp - needs manual
	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfoWithTime("14.0.0", PackageProvenance{}, now),
		NewPackageLockInfoWithTime("14.1.0", PackageProvenance{}, now),
		PackageLockInfo{})

	_, needsManual := resolver.Resolve(conflict)
	assert.True(t, needsManual, "equal timestamps need manual resolution")
}

func TestConflictResolver_StrategyAuto_SameVectorDifferentVersion(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)

	resolver := NewConflictResolver(StrategyAuto)

	// Same vector but different versions - shouldn't happen normally
	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vector)),
		NewPackageLockInfo("14.1.0", NewPackageProvenance(machineA, vector)),
		PackageLockInfo{})

	_, needsManual := resolver.Resolve(conflict)
	assert.True(t, needsManual, "same vector with different versions needs manual resolution")
}

func TestConflictResolver_UnknownStrategy(t *testing.T) {
	t.Parallel()

	resolver := &ConflictResolver{strategy: ResolutionStrategy(99)}

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)

	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vector)),
		NewPackageLockInfo("14.1.0", NewPackageProvenance(machineA, vector)),
		PackageLockInfo{})

	_, needsManual := resolver.Resolve(conflict)
	assert.True(t, needsManual, "unknown strategy should fall back to manual")
}

func TestConflictResolver_StrategyAuto_BothModified_Delete(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)

	resolver := NewConflictResolver(StrategyAuto)

	// BothModified with one side deleted
	conflict := NewLockConflict("brew:ripgrep", BothModified,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vector)),
		PackageLockInfo{}, // Deleted
		PackageLockInfo{})

	_, needsManual := resolver.Resolve(conflict)
	assert.True(t, needsManual, "delete/modify conflicts need manual resolution")
}

func TestConflictResolver_StrategyAuto_TimestampFallback(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	resolver := NewConflictResolver(StrategyAuto)

	// No provenance, different timestamps
	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfoWithTime("14.0.0", PackageProvenance{}, earlier),
		NewPackageLockInfoWithTime("14.1.0", PackageProvenance{}, later),
		PackageLockInfo{})

	resolution, needsManual := resolver.Resolve(conflict)
	assert.False(t, needsManual)
	assert.Equal(t, ChooseRemote, resolution.Choice())
	assert.Contains(t, resolution.Reason(), "later timestamp")
}

func TestConflictResolver_StrategyAuto_UnknownConflictType(t *testing.T) {
	t.Parallel()

	resolver := NewConflictResolver(StrategyAuto)

	// Unknown conflict type (simulated by using the struct directly)
	conflict := LockConflict{
		packageKey:   "brew:test",
		conflictType: ConflictType(99), // Unknown type
	}

	_, needsManual := resolver.Resolve(conflict)
	assert.True(t, needsManual, "unknown conflict type should need manual resolution")
}

func TestConflictResolver_StrategyNewest_LocalVectorAfter(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	// Local's vector includes remote's changes (local happened after)
	vectorB := NewVersionVector().Increment(machineB)
	vectorA := vectorB.Increment(machineA) // A happened after B

	resolver := NewConflictResolver(StrategyNewest)

	conflict := NewLockConflict("brew:ripgrep", VersionMismatch,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vectorA)),
		NewPackageLockInfo("14.1.0", NewPackageProvenance(machineB, vectorB)),
		PackageLockInfo{})

	resolution, needsManual := resolver.Resolve(conflict)
	require.False(t, needsManual)
	assert.Equal(t, ChooseLocal, resolution.Choice())
	assert.Contains(t, resolution.Reason(), "later vector")
}
