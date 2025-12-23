package lock

import (
	"errors"
	"fmt"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
)

// Resolver errors.
var (
	ErrVersionNotLocked = errors.New("version not locked")
)

// ResolutionSource indicates where a version resolution came from.
type ResolutionSource string

const (
	// ResolutionSourceNone indicates resolution failed.
	ResolutionSourceNone ResolutionSource = ""
	// ResolutionSourceLatest indicates version from latest available.
	ResolutionSourceLatest ResolutionSource = "latest"
	// ResolutionSourceLockfile indicates version from lockfile.
	ResolutionSourceLockfile ResolutionSource = "lockfile"
)

// Resolution represents the result of version resolution.
type Resolution struct {
	Provider         string
	Name             string
	Version          string
	Source           ResolutionSource
	Locked           bool
	LockedVersion    string
	AvailableVersion string
	Drifted          bool
	Updated          bool
	Failed           bool
	Error            error
}

// String returns a human-readable representation of the resolution.
func (r Resolution) String() string {
	if r.Failed {
		return fmt.Sprintf("%s:%s (failed: %v)", r.Provider, r.Name, r.Error)
	}

	var suffix string
	switch r.Source {
	case ResolutionSourceLatest:
		suffix = "latest"
	case ResolutionSourceLockfile:
		suffix = "locked"
	case ResolutionSourceNone:
		suffix = "unknown"
	}

	return fmt.Sprintf("%s:%s@%s (%s)", r.Provider, r.Name, r.Version, suffix)
}

// Key returns the package key.
func (r Resolution) Key() string {
	return r.Provider + ":" + r.Name
}

// Resolver handles version resolution based on reproducibility mode.
type Resolver struct {
	lockfile *Lockfile
}

// NewResolver creates a new Resolver with the given lockfile.
// If lockfile is nil, creates an empty one with intent mode.
func NewResolver(lockfile *Lockfile) *Resolver {
	if lockfile == nil {
		lockfile = NewLockfile(config.ModeIntent, MachineInfoFromSystem())
	}
	return &Resolver{
		lockfile: lockfile,
	}
}

// Mode returns the current reproducibility mode.
func (r *Resolver) Mode() config.ReproducibilityMode {
	return r.lockfile.Mode()
}

// Lockfile returns the underlying lockfile.
func (r *Resolver) Lockfile() *Lockfile {
	return r.lockfile
}

// Resolve determines which version to use based on the reproducibility mode.
// latestVersion is the latest available version from the provider.
func (r *Resolver) Resolve(provider, name, latestVersion string) Resolution {
	res := Resolution{
		Provider:         provider,
		Name:             name,
		AvailableVersion: latestVersion,
	}

	locked, hasLock := r.lockfile.GetPackage(provider, name)
	if hasLock {
		res.LockedVersion = locked.Version()
	}

	switch r.lockfile.Mode() {
	case config.ModeIntent:
		// Intent mode: always use latest
		res.Version = latestVersion
		res.Source = ResolutionSourceLatest
		res.Locked = false

	case config.ModeLocked:
		if hasLock {
			// Use locked version
			res.Version = locked.Version()
			res.Source = ResolutionSourceLockfile
			res.Locked = true
		} else {
			// No lock, use latest
			res.Version = latestVersion
			res.Source = ResolutionSourceLatest
			res.Locked = false
		}

	case config.ModeFrozen:
		if !hasLock {
			// Frozen mode requires a lock
			res.Failed = true
			res.Error = fmt.Errorf("%w: %s:%s", ErrVersionNotLocked, provider, name)
			res.Source = ResolutionSourceNone
		} else {
			// Use locked version, mark drift if different
			res.Version = locked.Version()
			res.Source = ResolutionSourceLockfile
			res.Locked = true
			if locked.Version() != latestVersion {
				res.Drifted = true
			}
		}
	}

	return res
}

// ResolveWithUpdate forces use of latest version regardless of mode.
// This is used when explicitly updating the lockfile.
func (r *Resolver) ResolveWithUpdate(provider, name, latestVersion string) Resolution {
	res := Resolution{
		Provider:         provider,
		Name:             name,
		Version:          latestVersion,
		AvailableVersion: latestVersion,
		Source:           ResolutionSourceLatest,
		Updated:          true,
	}

	if locked, hasLock := r.lockfile.GetPackage(provider, name); hasLock {
		res.LockedVersion = locked.Version()
	}

	return res
}

// Lock adds or updates a package in the lockfile.
func (r *Resolver) Lock(provider, name, version string, integrity Integrity) error {
	pkg, err := NewPackageLock(provider, name, version, integrity, time.Now())
	if err != nil {
		return err
	}
	return r.lockfile.SetPackage(pkg)
}

// Unlock removes a package from the lockfile.
func (r *Resolver) Unlock(provider, name string) bool {
	return r.lockfile.RemovePackage(provider, name)
}

// IsLocked returns true if the package is in the lockfile.
func (r *Resolver) IsLocked(provider, name string) bool {
	return r.lockfile.HasPackage(provider, name)
}

// GetLockedVersion returns the locked version if it exists.
func (r *Resolver) GetLockedVersion(provider, name string) (string, bool) {
	if pkg, exists := r.lockfile.GetPackage(provider, name); exists {
		return pkg.Version(), true
	}
	return "", false
}
