package lock

import (
	"errors"
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
)

// LockfileVersion is the current lockfile format version.
const LockfileVersion = 1

// Lockfile errors.
var (
	ErrPackageExists      = errors.New("package already exists in lockfile")
	ErrPackageNotFound    = errors.New("package not found in lockfile")
	ErrInvalidPackageLock = errors.New("invalid package lock")
)

// Lockfile is the aggregate root for lockfile management.
// It tracks locked versions and integrity for reproducible builds.
type Lockfile struct {
	version     int
	mode        config.ReproducibilityMode
	machineInfo MachineInfo
	packages    map[string]PackageLock
}

// NewLockfile creates a new Lockfile with the given mode and machine info.
func NewLockfile(mode config.ReproducibilityMode, machineInfo MachineInfo) *Lockfile {
	return &Lockfile{
		version:     LockfileVersion,
		mode:        mode,
		machineInfo: machineInfo,
		packages:    make(map[string]PackageLock),
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
