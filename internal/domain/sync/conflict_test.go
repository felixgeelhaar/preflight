package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConflictType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ct   ConflictType
		want string
	}{
		{VersionMismatch, "version_mismatch"},
		{LocalOnly, "local_only"},
		{RemoteOnly, "remote_only"},
		{BothModified, "both_modified"},
		{ConflictType(99), "unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.ct.String())
	}
}

func TestNewLockConflict(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)

	local := NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vectorA))
	remote := NewPackageLockInfo("14.1.0", NewPackageProvenance(machineB, vectorB))
	base := NewPackageLockInfo("13.0.0", NewPackageProvenance(machineA, NewVersionVector()))

	conflict := NewLockConflict("brew:ripgrep", BothModified, local, remote, base)

	assert.Equal(t, "brew:ripgrep", conflict.PackageKey())
	assert.Equal(t, BothModified, conflict.Type())
	assert.Equal(t, "14.0.0", conflict.Local().Version())
	assert.Equal(t, "14.1.0", conflict.Remote().Version())
	assert.Equal(t, "13.0.0", conflict.Base().Version())
	assert.False(t, conflict.IsZero())
}

func TestLockConflict_IsZero(t *testing.T) {
	t.Parallel()

	var zeroConflict LockConflict
	assert.True(t, zeroConflict.IsZero())

	nonZeroConflict := NewLockConflict("brew:ripgrep", LocalOnly, PackageLockInfo{}, PackageLockInfo{}, PackageLockInfo{})
	assert.False(t, nonZeroConflict.IsZero())
}

func TestLockConflict_HasBase(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)

	// With base
	withBase := NewLockConflict(
		"brew:ripgrep",
		BothModified,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vector)),
		NewPackageLockInfo("14.1.0", NewPackageProvenance(machineA, vector)),
		NewPackageLockInfo("13.0.0", NewPackageProvenance(machineA, NewVersionVector())),
	)
	assert.True(t, withBase.HasBase())

	// Without base
	withoutBase := NewLockConflict(
		"brew:ripgrep",
		LocalOnly,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vector)),
		PackageLockInfo{},
		PackageLockInfo{},
	)
	assert.False(t, withoutBase.HasBase())
}

func TestLockConflict_IsResolvable(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)

	tests := []struct {
		name       string
		conflictFn func() LockConflict
		want       bool
	}{
		{
			name: "local only is resolvable",
			conflictFn: func() LockConflict {
				return NewLockConflict(
					"brew:ripgrep",
					LocalOnly,
					NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vectorA)),
					PackageLockInfo{},
					PackageLockInfo{},
				)
			},
			want: true,
		},
		{
			name: "remote only is resolvable",
			conflictFn: func() LockConflict {
				return NewLockConflict(
					"brew:ripgrep",
					RemoteOnly,
					PackageLockInfo{},
					NewPackageLockInfo("14.0.0", NewPackageProvenance(machineB, vectorB)),
					PackageLockInfo{},
				)
			},
			want: true,
		},
		{
			name: "version mismatch without concurrent changes is resolvable",
			conflictFn: func() LockConflict {
				// Sequential changes (A happened before B)
				vecA := NewVersionVector().Increment(machineA)
				vecB := vecA.Increment(machineB) // B knows about A's change

				return NewLockConflict(
					"brew:ripgrep",
					VersionMismatch,
					NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vecA)),
					NewPackageLockInfo("14.1.0", NewPackageProvenance(machineB, vecB)),
					PackageLockInfo{},
				)
			},
			want: true,
		},
		{
			name: "both modified with concurrent changes is not auto-resolvable",
			conflictFn: func() LockConflict {
				// Concurrent changes (neither knows about the other)
				return NewLockConflict(
					"brew:ripgrep",
					BothModified,
					NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vectorA)),
					NewPackageLockInfo("14.1.0", NewPackageProvenance(machineB, vectorB)),
					NewPackageLockInfo("13.0.0", PackageProvenance{}),
				)
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			conflict := tt.conflictFn()
			assert.Equal(t, tt.want, conflict.IsResolvable())
		})
	}
}

func TestLockConflict_String(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)

	conflict := NewLockConflict(
		"brew:ripgrep",
		VersionMismatch,
		NewPackageLockInfo("14.0.0", NewPackageProvenance(machineA, vector)),
		NewPackageLockInfo("14.1.0", NewPackageProvenance(machineA, vector)),
		PackageLockInfo{},
	)

	s := conflict.String()
	assert.Contains(t, s, "brew:ripgrep")
	assert.Contains(t, s, "version_mismatch")
}

func TestPackageLockInfo(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	info := NewPackageLockInfo("14.1.0", prov)

	assert.Equal(t, "14.1.0", info.Version())
	assert.Equal(t, machineA.String(), info.Provenance().ModifiedBy())
	assert.False(t, info.IsZero())
}

func TestPackageLockInfo_IsZero(t *testing.T) {
	t.Parallel()

	var zeroInfo PackageLockInfo
	assert.True(t, zeroInfo.IsZero())

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)

	nonZeroInfo := NewPackageLockInfo("14.1.0", NewPackageProvenance(machineA, vector))
	assert.False(t, nonZeroInfo.IsZero())
}

func TestPackageLockInfo_ModifiedAt(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineA)
	prov := NewPackageProvenance(machineA, vector)

	info := NewPackageLockInfoWithTime("14.1.0", prov, time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))

	assert.Equal(t, time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC), info.ModifiedAt())
}
