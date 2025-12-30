package lock

import (
	"errors"
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/sync"
)

// LockfileVersion is the current lockfile format version.
const LockfileVersion = 1

// LockfileVersionV2 is the v2 format with multi-machine sync support.
const LockfileVersionV2 = 2

// Lockfile errors.
var (
	ErrPackageExists      = errors.New("package already exists in lockfile")
	ErrPackageNotFound    = errors.New("package not found in lockfile")
	ErrInvalidPackageLock = errors.New("invalid package lock")
)

// Lockfile is the aggregate root for lockfile management.
// It tracks locked versions and integrity for reproducible builds.
// V2 lockfiles include sync metadata for multi-machine support.
type Lockfile struct {
	version      int
	mode         config.ReproducibilityMode
	machineInfo  MachineInfo
	packages     map[string]PackageLock
	syncMetadata sync.SyncMetadata // V2: Multi-machine sync metadata
}

// NewLockfile creates a new v1 Lockfile with the given mode and machine info.
// For new lockfiles, prefer NewLockfileV2 for multi-machine sync support.
func NewLockfile(mode config.ReproducibilityMode, machineInfo MachineInfo) *Lockfile {
	return &Lockfile{
		version:     LockfileVersion,
		mode:        mode,
		machineInfo: machineInfo,
		packages:    make(map[string]PackageLock),
	}
}

// NewLockfileV2 creates a new v2 Lockfile with multi-machine sync support.
// This is the recommended constructor for new lockfiles.
func NewLockfileV2(mode config.ReproducibilityMode, machineInfo MachineInfo) *Lockfile {
	return &Lockfile{
		version:      LockfileVersionV2,
		mode:         mode,
		machineInfo:  machineInfo,
		packages:     make(map[string]PackageLock),
		syncMetadata: sync.NewSyncMetadata(sync.NewVersionVector()),
	}
}

// Version returns the lockfile format version.
func (l *Lockfile) Version() int {
	return l.version
}

// Mode returns the reproducibility mode.
func (l *Lockfile) Mode() config.ReproducibilityMode {
	return l.mode
}

// MachineInfo returns the machine info snapshot.
func (l *Lockfile) MachineInfo() MachineInfo {
	return l.machineInfo
}

// SyncMetadata returns the multi-machine sync metadata (v2 only).
// For v1 lockfiles, returns a zero-value SyncMetadata.
func (l *Lockfile) SyncMetadata() sync.SyncMetadata {
	return l.syncMetadata
}

// WithSyncMetadata returns a new Lockfile with updated sync metadata.
// The packages are copied, not shared.
func (l *Lockfile) WithSyncMetadata(meta sync.SyncMetadata) *Lockfile {
	newLock := &Lockfile{
		version:      l.version,
		mode:         l.mode,
		machineInfo:  l.machineInfo,
		packages:     make(map[string]PackageLock, len(l.packages)),
		syncMetadata: meta,
	}
	for k, v := range l.packages {
		newLock.packages[k] = v
	}
	return newLock
}

// RecordChange records a change made by the given machine.
// This increments the version vector and updates the machine's lineage.
// This method mutates the lockfile in place.
func (l *Lockfile) RecordChange(machineID sync.MachineID, hostname string) {
	l.syncMetadata = l.syncMetadata.RecordActivity(machineID, hostname)
}

// Packages returns a copy of all locked packages.
func (l *Lockfile) Packages() map[string]PackageLock {
	result := make(map[string]PackageLock, len(l.packages))
	for k, v := range l.packages {
		result[k] = v
	}
	return result
}

// AddPackage adds a new package to the lockfile.
// Returns an error if the package already exists.
func (l *Lockfile) AddPackage(pkg PackageLock) error {
	if pkg.IsZero() {
		return ErrInvalidPackageLock
	}

	key := pkg.Key()
	if _, exists := l.packages[key]; exists {
		return fmt.Errorf("%w: %s", ErrPackageExists, key)
	}

	l.packages[key] = pkg
	return nil
}

// UpdatePackage updates an existing package in the lockfile.
// Returns an error if the package does not exist.
func (l *Lockfile) UpdatePackage(pkg PackageLock) error {
	if pkg.IsZero() {
		return ErrInvalidPackageLock
	}

	key := pkg.Key()
	if _, exists := l.packages[key]; !exists {
		return fmt.Errorf("%w: %s", ErrPackageNotFound, key)
	}

	l.packages[key] = pkg
	return nil
}

// SetPackage adds or updates a package in the lockfile.
// This is an upsert operation.
func (l *Lockfile) SetPackage(pkg PackageLock) error {
	if pkg.IsZero() {
		return ErrInvalidPackageLock
	}

	l.packages[pkg.Key()] = pkg
	return nil
}

// RemovePackage removes a package from the lockfile.
// Returns true if the package was removed, false if it didn't exist.
func (l *Lockfile) RemovePackage(provider, name string) bool {
	key := provider + ":" + name
	if _, exists := l.packages[key]; !exists {
		return false
	}
	delete(l.packages, key)
	return true
}

// GetPackage returns a package lock by provider and name.
// Returns the package and true if found, zero value and false otherwise.
func (l *Lockfile) GetPackage(provider, name string) (PackageLock, bool) {
	key := provider + ":" + name
	pkg, exists := l.packages[key]
	return pkg, exists
}

// HasPackage returns true if the package exists in the lockfile.
func (l *Lockfile) HasPackage(provider, name string) bool {
	key := provider + ":" + name
	_, exists := l.packages[key]
	return exists
}

// PackageCount returns the number of locked packages.
func (l *Lockfile) PackageCount() int {
	return len(l.packages)
}

// PackagesByProvider returns all packages for a given provider.
func (l *Lockfile) PackagesByProvider(provider string) []PackageLock {
	var result []PackageLock
	for _, pkg := range l.packages {
		if pkg.Provider() == provider {
			result = append(result, pkg)
		}
	}
	return result
}

// WithMode returns a new Lockfile with a different mode.
// The packages are copied, not shared.
func (l *Lockfile) WithMode(mode config.ReproducibilityMode) *Lockfile {
	newLock := NewLockfile(mode, l.machineInfo)
	for k, v := range l.packages {
		newLock.packages[k] = v
	}
	return newLock
}

// WithMachineInfo returns a new Lockfile with different machine info.
// The packages are copied, not shared.
func (l *Lockfile) WithMachineInfo(info MachineInfo) *Lockfile {
	newLock := NewLockfile(l.mode, info)
	for k, v := range l.packages {
		newLock.packages[k] = v
	}
	return newLock
}

// IsEmpty returns true if the lockfile has no packages.
func (l *Lockfile) IsEmpty() bool {
	return len(l.packages) == 0
}

// Clear removes all packages from the lockfile.
func (l *Lockfile) Clear() {
	l.packages = make(map[string]PackageLock)
}
