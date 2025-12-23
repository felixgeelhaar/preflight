package lock

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestResolver(t *testing.T, mode config.ReproducibilityMode) *Resolver {
	t.Helper()
	machineInfo := createTestMachineInfo(t)
	lockfile := NewLockfile(mode, machineInfo)
	return NewResolver(lockfile)
}

func TestNewResolver(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	resolver := NewResolver(lockfile)

	assert.NotNil(t, resolver)
	assert.Equal(t, config.ModeLocked, resolver.Mode())
}

func TestNewResolver_NilLockfile(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(nil)

	assert.NotNil(t, resolver)
	// Should create empty lockfile with default mode
	assert.Equal(t, config.ModeIntent, resolver.Mode())
}

func TestResolver_Resolve_Intent_NoLock(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeIntent)

	resolution := resolver.Resolve("brew", "ripgrep", "14.1.0")

	assert.Equal(t, "14.1.0", resolution.Version)
	assert.Equal(t, ResolutionSourceLatest, resolution.Source)
	assert.False(t, resolution.Locked)
	assert.Empty(t, resolution.LockedVersion)
}

func TestResolver_Resolve_Intent_WithLock(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeIntent)
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
	_ = resolver.lockfile.AddPackage(pkg)

	// Intent mode ignores lock, uses latest
	resolution := resolver.Resolve("brew", "ripgrep", "14.1.0")

	assert.Equal(t, "14.1.0", resolution.Version)
	assert.Equal(t, ResolutionSourceLatest, resolution.Source)
	assert.False(t, resolution.Locked)
	assert.Equal(t, "14.0.0", resolution.LockedVersion) // Shows what was locked
}

func TestResolver_Resolve_Locked_NoLock(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeLocked)

	// No lock exists, use latest and suggest locking
	resolution := resolver.Resolve("brew", "ripgrep", "14.1.0")

	assert.Equal(t, "14.1.0", resolution.Version)
	assert.Equal(t, ResolutionSourceLatest, resolution.Source)
	assert.False(t, resolution.Locked)
}

func TestResolver_Resolve_Locked_WithLock(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeLocked)
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
	_ = resolver.lockfile.AddPackage(pkg)

	// Locked mode uses locked version
	resolution := resolver.Resolve("brew", "ripgrep", "14.1.0")

	assert.Equal(t, "14.0.0", resolution.Version)
	assert.Equal(t, ResolutionSourceLockfile, resolution.Source)
	assert.True(t, resolution.Locked)
	assert.Equal(t, "14.0.0", resolution.LockedVersion)
}

func TestResolver_Resolve_Frozen_NoLock(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeFrozen)

	// Frozen mode fails if no lock exists
	resolution := resolver.Resolve("brew", "ripgrep", "14.1.0")

	assert.Empty(t, resolution.Version)
	assert.Equal(t, ResolutionSourceNone, resolution.Source)
	assert.True(t, resolution.Failed)
	assert.ErrorIs(t, resolution.Error, ErrVersionNotLocked)
}

func TestResolver_Resolve_Frozen_VersionMismatch(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeFrozen)
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
	_ = resolver.lockfile.AddPackage(pkg)

	// Frozen mode fails if latest differs from lock
	resolution := resolver.Resolve("brew", "ripgrep", "14.1.0")

	assert.Equal(t, "14.0.0", resolution.Version) // Still returns locked version
	assert.Equal(t, ResolutionSourceLockfile, resolution.Source)
	assert.True(t, resolution.Locked)
	assert.True(t, resolution.Drifted) // Marks as drifted
	assert.Equal(t, "14.1.0", resolution.AvailableVersion)
}

func TestResolver_Resolve_Frozen_VersionMatch(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeFrozen)
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")
	_ = resolver.lockfile.AddPackage(pkg)

	// Frozen mode succeeds when versions match
	resolution := resolver.Resolve("brew", "ripgrep", "14.1.0")

	assert.Equal(t, "14.1.0", resolution.Version)
	assert.Equal(t, ResolutionSourceLockfile, resolution.Source)
	assert.True(t, resolution.Locked)
	assert.False(t, resolution.Drifted)
	assert.False(t, resolution.Failed)
}

func TestResolver_ResolveWithUpdate(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeLocked)
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
	_ = resolver.lockfile.AddPackage(pkg)

	// Force update even in locked mode
	resolution := resolver.ResolveWithUpdate("brew", "ripgrep", "14.1.0")

	assert.Equal(t, "14.1.0", resolution.Version)
	assert.Equal(t, ResolutionSourceLatest, resolution.Source)
	assert.False(t, resolution.Locked)
	assert.True(t, resolution.Updated)
}

func TestResolver_Lock(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeLocked)

	integrity := createTestIntegrity(t)
	err := resolver.Lock("brew", "ripgrep", "14.1.0", integrity)

	require.NoError(t, err)

	pkg, exists := resolver.lockfile.GetPackage("brew", "ripgrep")
	assert.True(t, exists)
	assert.Equal(t, "14.1.0", pkg.Version())
}

func TestResolver_Lock_Update(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeLocked)
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
	_ = resolver.lockfile.AddPackage(pkg)

	integrity := createTestIntegrity(t)
	err := resolver.Lock("brew", "ripgrep", "14.1.0", integrity)

	require.NoError(t, err)

	updated, _ := resolver.lockfile.GetPackage("brew", "ripgrep")
	assert.Equal(t, "14.1.0", updated.Version())
}

func TestResolver_Unlock(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeLocked)
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
	_ = resolver.lockfile.AddPackage(pkg)

	removed := resolver.Unlock("brew", "ripgrep")

	assert.True(t, removed)
	assert.False(t, resolver.lockfile.HasPackage("brew", "ripgrep"))
}

func TestResolver_IsLocked(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeLocked)
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
	_ = resolver.lockfile.AddPackage(pkg)

	assert.True(t, resolver.IsLocked("brew", "ripgrep"))
	assert.False(t, resolver.IsLocked("brew", "fd"))
}

func TestResolver_GetLockedVersion(t *testing.T) {
	t.Parallel()

	resolver := createTestResolver(t, config.ModeLocked)
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
	_ = resolver.lockfile.AddPackage(pkg)

	version, ok := resolver.GetLockedVersion("brew", "ripgrep")
	assert.True(t, ok)
	assert.Equal(t, "14.0.0", version)

	_, ok = resolver.GetLockedVersion("brew", "fd")
	assert.False(t, ok)
}

func TestResolver_Lockfile(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	resolver := NewResolver(lockfile)

	assert.Same(t, lockfile, resolver.Lockfile())
}

func TestResolution_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		resolution Resolution
		expected   string
	}{
		{
			name: "from latest",
			resolution: Resolution{
				Provider: "brew",
				Name:     "ripgrep",
				Version:  "14.1.0",
				Source:   ResolutionSourceLatest,
			},
			expected: "brew:ripgrep@14.1.0 (latest)",
		},
		{
			name: "from lockfile",
			resolution: Resolution{
				Provider: "brew",
				Name:     "ripgrep",
				Version:  "14.0.0",
				Source:   ResolutionSourceLockfile,
				Locked:   true,
			},
			expected: "brew:ripgrep@14.0.0 (locked)",
		},
		{
			name: "failed",
			resolution: Resolution{
				Provider: "brew",
				Name:     "ripgrep",
				Source:   ResolutionSourceNone,
				Failed:   true,
				Error:    ErrVersionNotLocked,
			},
			expected: "brew:ripgrep (failed: version not locked)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.resolution.String())
		})
	}
}
