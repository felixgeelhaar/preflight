package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConflictDetector(t *testing.T) {
	t.Parallel()

	detector := NewConflictDetector()
	assert.NotNil(t, detector)
}

func TestConflictDetector_Detect_NoConflicts(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	detector := NewConflictDetector()
	input := DetectInput{
		Local: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", prov),
			"brew:fd":      NewPackageLockInfo("9.0.0", prov),
		},
		Remote: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", prov),
			"brew:fd":      NewPackageLockInfo("9.0.0", prov),
		},
	}

	result := detector.Detect(input)

	assert.False(t, result.HasConflicts())
	assert.False(t, result.HasAutoResolvable())
	assert.Len(t, result.Clean, 2)
	assert.Contains(t, result.Clean, "brew:ripgrep")
	assert.Contains(t, result.Clean, "brew:fd")
}

func TestConflictDetector_Detect_LocalOnly(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	detector := NewConflictDetector()
	input := DetectInput{
		Local: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", prov),
			"brew:fd":      NewPackageLockInfo("9.0.0", prov), // Only local
		},
		Remote: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", prov),
		},
	}

	result := detector.Detect(input)

	assert.False(t, result.HasConflicts())
	assert.True(t, result.HasAutoResolvable())
	require.Len(t, result.AutoResolvable, 1)
	assert.Equal(t, "brew:fd", result.AutoResolvable[0].PackageKey())
	assert.Equal(t, LocalOnly, result.AutoResolvable[0].Type())
}

func TestConflictDetector_Detect_RemoteOnly(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	detector := NewConflictDetector()
	input := DetectInput{
		Local: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", prov),
		},
		Remote: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", prov),
			"brew:bat":     NewPackageLockInfo("0.24.0", prov), // Only remote
		},
	}

	result := detector.Detect(input)

	assert.False(t, result.HasConflicts())
	assert.True(t, result.HasAutoResolvable())
	require.Len(t, result.AutoResolvable, 1)
	assert.Equal(t, "brew:bat", result.AutoResolvable[0].PackageKey())
	assert.Equal(t, RemoteOnly, result.AutoResolvable[0].Type())
}

func TestConflictDetector_Detect_VersionMismatch_Sequential(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	// Machine A makes a change, then Machine B builds on it
	vectorA := NewVersionVector().Increment(machineA)
	vectorB := vectorA.Increment(machineB) // B knows about A's change

	provA := NewPackageProvenance(machineA, vectorA)
	provB := NewPackageProvenance(machineB, vectorB)

	detector := NewConflictDetector()
	input := DetectInput{
		Local: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.0.0", provA), // Local has older version
		},
		Remote: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", provB), // Remote has newer version
		},
	}

	result := detector.Detect(input)

	// Sequential changes are auto-resolvable
	assert.False(t, result.HasConflicts())
	assert.True(t, result.HasAutoResolvable())
	require.Len(t, result.AutoResolvable, 1)
	assert.Equal(t, "brew:ripgrep", result.AutoResolvable[0].PackageKey())
	assert.Equal(t, VersionMismatch, result.AutoResolvable[0].Type())
}

func TestConflictDetector_Detect_VersionMismatch_Concurrent(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	// Both machines make changes independently (concurrent)
	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)

	provA := NewPackageProvenance(machineA, vectorA)
	provB := NewPackageProvenance(machineB, vectorB)

	detector := NewConflictDetector()
	input := DetectInput{
		Local: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.0.0", provA),
		},
		Remote: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", provB),
		},
	}

	result := detector.Detect(input)

	// Concurrent changes require manual resolution
	assert.True(t, result.HasConflicts())
	assert.False(t, result.HasAutoResolvable())
	require.Len(t, result.Conflicts, 1)
	assert.Equal(t, "brew:ripgrep", result.Conflicts[0].PackageKey())
	assert.Equal(t, BothModified, result.Conflicts[0].Type())
}

func TestConflictDetector_Detect_ThreeWayMerge_LocalDeleted(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	baseVector := NewVersionVector()
	vectorB := baseVector.Increment(machineB)

	baseProv := NewPackageProvenance(machineA, baseVector)
	provB := NewPackageProvenance(machineB, vectorB)

	detector := NewConflictDetector()
	input := DetectInput{
		Local: map[string]PackageLockInfo{
			// brew:ripgrep deleted locally
		},
		Remote: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", provB), // Still in remote
		},
		Base: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.0.0", baseProv), // Existed in base
		},
	}

	result := detector.Detect(input)

	// Local deleted, remote still has it - conflict
	assert.True(t, result.HasConflicts())
	require.Len(t, result.Conflicts, 1)
	assert.Equal(t, "brew:ripgrep", result.Conflicts[0].PackageKey())
	assert.Equal(t, BothModified, result.Conflicts[0].Type())
	assert.True(t, result.Conflicts[0].HasBase())
}

func TestConflictDetector_Detect_ThreeWayMerge_RemoteDeleted(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")

	baseVector := NewVersionVector()
	vectorA := baseVector.Increment(machineA)

	baseProv := NewPackageProvenance(machineA, baseVector)
	provA := NewPackageProvenance(machineA, vectorA)

	detector := NewConflictDetector()
	input := DetectInput{
		Local: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", provA), // Still locally
		},
		Remote: map[string]PackageLockInfo{
			// brew:ripgrep deleted remotely
		},
		Base: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.0.0", baseProv), // Existed in base
		},
	}

	result := detector.Detect(input)

	// Remote deleted, local still has it - conflict
	assert.True(t, result.HasConflicts())
	require.Len(t, result.Conflicts, 1)
	assert.Equal(t, "brew:ripgrep", result.Conflicts[0].PackageKey())
	assert.Equal(t, BothModified, result.Conflicts[0].Type())
	assert.True(t, result.Conflicts[0].HasBase())
}

func TestConflictDetector_Detect_NoProvenance(t *testing.T) {
	t.Parallel()

	// When packages have no provenance, we still detect version mismatches
	detector := NewConflictDetector()
	input := DetectInput{
		Local: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.0.0", PackageProvenance{}),
		},
		Remote: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", PackageProvenance{}),
		},
	}

	result := detector.Detect(input)

	// Without provenance, version mismatch is auto-resolvable (falls back to timestamp)
	assert.False(t, result.HasConflicts())
	assert.True(t, result.HasAutoResolvable())
	require.Len(t, result.AutoResolvable, 1)
	assert.Equal(t, VersionMismatch, result.AutoResolvable[0].Type())
}

func TestConflictDetector_Detect_MixedConflicts(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	// Sequential change
	vectorA := NewVersionVector().Increment(machineA)
	vectorB := vectorA.Increment(machineB)

	// Concurrent change
	concurrentA := NewVersionVector().Increment(machineA)
	concurrentB := NewVersionVector().Increment(machineB)

	provA := NewPackageProvenance(machineA, vectorA)
	provB := NewPackageProvenance(machineB, vectorB)
	provConcA := NewPackageProvenance(machineA, concurrentA)
	provConcB := NewPackageProvenance(machineB, concurrentB)

	detector := NewConflictDetector()
	input := DetectInput{
		Local: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.0.0", provA),    // Sequential - older
			"brew:fd":      NewPackageLockInfo("9.0.0", provConcA), // Concurrent
			"brew:bat":     NewPackageLockInfo("0.24.0", provA),    // Same version
			"brew:jq":      NewPackageLockInfo("1.7", provA),       // Local only
		},
		Remote: map[string]PackageLockInfo{
			"brew:ripgrep": NewPackageLockInfo("14.1.0", provB),    // Sequential - newer
			"brew:fd":      NewPackageLockInfo("9.1.0", provConcB), // Concurrent
			"brew:bat":     NewPackageLockInfo("0.24.0", provA),    // Same version
			"brew:delta":   NewPackageLockInfo("0.16.0", provB),    // Remote only
		},
	}

	result := detector.Detect(input)

	// Should have 1 unresolvable conflict (concurrent fd)
	assert.True(t, result.HasConflicts())
	require.Len(t, result.Conflicts, 1)
	assert.Equal(t, "brew:fd", result.Conflicts[0].PackageKey())

	// Should have 3 auto-resolvable (sequential ripgrep, local-only jq, remote-only delta)
	assert.True(t, result.HasAutoResolvable())
	assert.Len(t, result.AutoResolvable, 3)

	// Clean should have bat
	assert.Contains(t, result.Clean, "brew:bat")
}

func TestConflictDetector_Detect_EmptyInputs(t *testing.T) {
	t.Parallel()

	detector := NewConflictDetector()
	input := DetectInput{
		Local:  map[string]PackageLockInfo{},
		Remote: map[string]PackageLockInfo{},
	}

	result := detector.Detect(input)

	assert.False(t, result.HasConflicts())
	assert.False(t, result.HasAutoResolvable())
	assert.Empty(t, result.Clean)
	assert.Equal(t, 0, result.TotalConflicts())
}

func TestDetectResult_AllConflicts(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)

	provA := NewPackageProvenance(machineA, vectorA)
	provB := NewPackageProvenance(machineB, vectorB)

	result := DetectResult{
		Conflicts: []LockConflict{
			NewLockConflict("brew:fd", BothModified,
				NewPackageLockInfo("9.0.0", provA),
				NewPackageLockInfo("9.1.0", provB),
				PackageLockInfo{}),
		},
		AutoResolvable: []LockConflict{
			NewLockConflict("brew:jq", LocalOnly,
				NewPackageLockInfo("1.7", provA),
				PackageLockInfo{},
				PackageLockInfo{}),
		},
	}

	all := result.AllConflicts()
	assert.Len(t, all, 2)
	assert.Equal(t, 2, result.TotalConflicts())
}
