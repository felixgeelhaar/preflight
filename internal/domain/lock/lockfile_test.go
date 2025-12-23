package lock

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestIntegrity(t *testing.T) Integrity {
	t.Helper()
	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, err := NewIntegrity("sha256", hash)
	require.NoError(t, err)
	return integrity
}

func createTestMachineInfo(t *testing.T) MachineInfo {
	t.Helper()
	info, err := NewMachineInfo("darwin", "arm64", "macbook.local", time.Now())
	require.NoError(t, err)
	return info
}

func createTestPackageLock(t *testing.T, provider, name, version string) PackageLock {
	t.Helper()
	lock, err := NewPackageLock(provider, name, version, createTestIntegrity(t), time.Now())
	require.NoError(t, err)
	return lock
}

func TestNewLockfile(t *testing.T) {
	t.Parallel()

	machineInfo := createTestMachineInfo(t)
	lockfile := NewLockfile(config.ModeLocked, machineInfo)

	assert.Equal(t, LockfileVersion, lockfile.Version())
	assert.Equal(t, config.ModeLocked, lockfile.Mode())
	assert.Equal(t, machineInfo, lockfile.MachineInfo())
	assert.Empty(t, lockfile.Packages())
}

func TestLockfile_AddPackage(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")

	err := lockfile.AddPackage(pkg)

	require.NoError(t, err)
	assert.Len(t, lockfile.Packages(), 1)

	retrieved, exists := lockfile.GetPackage("brew", "ripgrep")
	assert.True(t, exists)
	assert.Equal(t, pkg.Version(), retrieved.Version())
}

func TestLockfile_AddPackage_Duplicate(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	pkg1 := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
	pkg2 := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")

	err := lockfile.AddPackage(pkg1)
	require.NoError(t, err)

	err = lockfile.AddPackage(pkg2)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPackageExists)

	// Original still exists
	retrieved, _ := lockfile.GetPackage("brew", "ripgrep")
	assert.Equal(t, "14.0.0", retrieved.Version())
}

func TestLockfile_AddPackage_ZeroValue(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))

	err := lockfile.AddPackage(PackageLock{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidPackageLock)
}

func TestLockfile_UpdatePackage(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	pkg1 := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
	pkg2 := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")

	_ = lockfile.AddPackage(pkg1)
	err := lockfile.UpdatePackage(pkg2)

	require.NoError(t, err)
	retrieved, _ := lockfile.GetPackage("brew", "ripgrep")
	assert.Equal(t, "14.1.0", retrieved.Version())
}

func TestLockfile_UpdatePackage_NotFound(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")

	err := lockfile.UpdatePackage(pkg)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPackageNotFound)
}

func TestLockfile_SetPackage(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	pkg1 := createTestPackageLock(t, "brew", "ripgrep", "14.0.0")
	pkg2 := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")

	// First set (add)
	err := lockfile.SetPackage(pkg1)
	require.NoError(t, err)

	retrieved, _ := lockfile.GetPackage("brew", "ripgrep")
	assert.Equal(t, "14.0.0", retrieved.Version())

	// Second set (update)
	err = lockfile.SetPackage(pkg2)
	require.NoError(t, err)

	retrieved, _ = lockfile.GetPackage("brew", "ripgrep")
	assert.Equal(t, "14.1.0", retrieved.Version())
}

func TestLockfile_RemovePackage(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")

	_ = lockfile.AddPackage(pkg)

	removed := lockfile.RemovePackage("brew", "ripgrep")
	assert.True(t, removed)
	assert.Empty(t, lockfile.Packages())

	// Remove non-existent returns false
	removed = lockfile.RemovePackage("brew", "ripgrep")
	assert.False(t, removed)
}

func TestLockfile_GetPackage(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")
	_ = lockfile.AddPackage(pkg)

	// Found
	retrieved, exists := lockfile.GetPackage("brew", "ripgrep")
	assert.True(t, exists)
	assert.Equal(t, pkg.Version(), retrieved.Version())

	// Not found
	_, exists = lockfile.GetPackage("apt", "ripgrep")
	assert.False(t, exists)
}

func TestLockfile_HasPackage(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	pkg := createTestPackageLock(t, "brew", "ripgrep", "14.1.0")
	_ = lockfile.AddPackage(pkg)

	assert.True(t, lockfile.HasPackage("brew", "ripgrep"))
	assert.False(t, lockfile.HasPackage("apt", "ripgrep"))
}

func TestLockfile_PackageCount(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))

	assert.Equal(t, 0, lockfile.PackageCount())

	_ = lockfile.AddPackage(createTestPackageLock(t, "brew", "ripgrep", "14.1.0"))
	_ = lockfile.AddPackage(createTestPackageLock(t, "brew", "fd", "9.0.0"))
	_ = lockfile.AddPackage(createTestPackageLock(t, "apt", "curl", "7.88.0"))

	assert.Equal(t, 3, lockfile.PackageCount())
}

func TestLockfile_PackagesByProvider(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	_ = lockfile.AddPackage(createTestPackageLock(t, "brew", "ripgrep", "14.1.0"))
	_ = lockfile.AddPackage(createTestPackageLock(t, "brew", "fd", "9.0.0"))
	_ = lockfile.AddPackage(createTestPackageLock(t, "apt", "curl", "7.88.0"))

	brewPkgs := lockfile.PackagesByProvider("brew")
	assert.Len(t, brewPkgs, 2)

	aptPkgs := lockfile.PackagesByProvider("apt")
	assert.Len(t, aptPkgs, 1)

	npmPkgs := lockfile.PackagesByProvider("npm")
	assert.Empty(t, npmPkgs)
}

func TestLockfile_WithMode(t *testing.T) {
	t.Parallel()

	original := NewLockfile(config.ModeIntent, createTestMachineInfo(t))
	_ = original.AddPackage(createTestPackageLock(t, "brew", "ripgrep", "14.1.0"))

	frozen := original.WithMode(config.ModeFrozen)

	// Original unchanged
	assert.Equal(t, config.ModeIntent, original.Mode())

	// New lockfile has new mode but same packages
	assert.Equal(t, config.ModeFrozen, frozen.Mode())
	assert.Equal(t, original.PackageCount(), frozen.PackageCount())
}

func TestLockfile_WithMachineInfo(t *testing.T) {
	t.Parallel()

	info1 := createTestMachineInfo(t)
	original := NewLockfile(config.ModeLocked, info1)
	_ = original.AddPackage(createTestPackageLock(t, "brew", "ripgrep", "14.1.0"))

	info2, _ := NewMachineInfo("linux", "amd64", "server.local", time.Now())
	updated := original.WithMachineInfo(info2)

	// Original unchanged
	assert.Equal(t, info1, original.MachineInfo())

	// New lockfile has new machine info
	assert.Equal(t, info2, updated.MachineInfo())
	assert.Equal(t, original.PackageCount(), updated.PackageCount())
}

func TestLockfile_IsEmpty(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	assert.True(t, lockfile.IsEmpty())

	_ = lockfile.AddPackage(createTestPackageLock(t, "brew", "ripgrep", "14.1.0"))
	assert.False(t, lockfile.IsEmpty())
}

func TestLockfile_Clear(t *testing.T) {
	t.Parallel()

	lockfile := NewLockfile(config.ModeLocked, createTestMachineInfo(t))
	_ = lockfile.AddPackage(createTestPackageLock(t, "brew", "ripgrep", "14.1.0"))
	_ = lockfile.AddPackage(createTestPackageLock(t, "brew", "fd", "9.0.0"))

	lockfile.Clear()

	assert.True(t, lockfile.IsEmpty())
	assert.Equal(t, 0, lockfile.PackageCount())
}
