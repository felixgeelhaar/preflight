// Package lockfile provides adapters for lockfile persistence.
package lockfile

import (
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
	"github.com/felixgeelhaar/preflight/internal/domain/sync"
)

// SyncAdapter converts between the lock domain and sync domain types.
// This enables the sync engine to work with lockfiles.
type SyncAdapter struct{}

// NewSyncAdapter creates a new SyncAdapter.
func NewSyncAdapter() *SyncAdapter {
	return &SyncAdapter{}
}

// ToLockfileState converts a lock.Lockfile to a sync.LockfileState.
// This is the primary conversion for sync operations.
func (a *SyncAdapter) ToLockfileState(lockfile *lock.Lockfile) *sync.LockfileState {
	if lockfile == nil {
		return sync.NewLockfileState()
	}

	state := sync.NewLockfileStateWithMetadata(lockfile.SyncMetadata())

	for key, pkg := range lockfile.Packages() {
		info := a.PackageLockToInfo(pkg)
		state.AddPackage(key, info)
	}

	return state
}

// PackageLockToInfo converts a lock.PackageLock to a sync.PackageLockInfo.
func (a *SyncAdapter) PackageLockToInfo(pkg lock.PackageLock) sync.PackageLockInfo {
	// Note: lock.PackageLock doesn't have provenance info yet,
	// so we create a zero-value provenance. This will be upgraded
	// when provenance is added to PackageLock.
	return sync.NewPackageLockInfoWithTime(
		pkg.Version(),
		sync.PackageProvenance{}, // Zero value - no provenance available
		pkg.InstalledAt(),
	)
}

// ApplyMergeResult applies a sync.MergeResult back to a lock.Lockfile.
// This updates the lockfile with the merged sync state.
// If remoteLockfile is provided, new packages from remote can be added.
func (a *SyncAdapter) ApplyMergeResult(
	lockfile *lock.Lockfile,
	result *sync.MergeResult,
) (*lock.Lockfile, error) {
	return a.ApplyMergeResultWithRemote(lockfile, result, nil)
}

// ApplyMergeResultWithRemote applies a sync.MergeResult with access to remote lockfile.
// This allows adding packages that only exist in remote.
func (a *SyncAdapter) ApplyMergeResultWithRemote(
	lockfile *lock.Lockfile,
	result *sync.MergeResult,
	remoteLockfile *lock.Lockfile,
) (*lock.Lockfile, error) {
	if result == nil || result.State == nil {
		return lockfile, nil
	}

	// Update sync metadata
	newLockfile := lockfile.WithSyncMetadata(result.State.Metadata)

	// Apply changes
	for _, change := range result.Changes {
		provider, name, err := lock.ParsePackageKey(change.PackageKey)
		if err != nil {
			return nil, err
		}

		switch change.Type {
		case sync.ChangeRemoved:
			newLockfile.RemovePackage(provider, name)

		case sync.ChangeAdded:
			// For added packages, try to get from remote lockfile
			if existing, ok := newLockfile.GetPackage(provider, name); ok {
				// Package already exists locally - just update version
				updated, err := existing.WithVersion(
					change.After.Version(),
					existing.Integrity(),
					change.After.ModifiedAt(),
				)
				if err != nil {
					return nil, err
				}
				if err := newLockfile.UpdatePackage(updated); err != nil {
					return nil, err
				}
			} else if remoteLockfile != nil {
				// Copy package from remote
				if remotePkg, ok := remoteLockfile.GetPackage(provider, name); ok {
					if err := newLockfile.AddPackage(remotePkg); err != nil {
						return nil, err
					}
				}
			}
			// If no remote lockfile, we can't add the package - skip silently

		case sync.ChangeUpdated:
			// Update existing package
			if existing, ok := newLockfile.GetPackage(provider, name); ok {
				updated, err := existing.WithVersion(
					change.After.Version(),
					existing.Integrity(),
					change.After.ModifiedAt(),
				)
				if err != nil {
					return nil, err
				}
				if err := newLockfile.UpdatePackage(updated); err != nil {
					return nil, err
				}
			}

		case sync.ChangeKept:
			// No change needed
		}
	}

	return newLockfile, nil
}

// CompareStates determines the causal relationship between local and remote lockfiles.
func (a *SyncAdapter) CompareStates(local, remote *lock.Lockfile) sync.CausalRelation {
	if local == nil || remote == nil {
		return sync.Concurrent
	}

	localVector := local.SyncMetadata().Vector()
	remoteVector := remote.SyncMetadata().Vector()

	return localVector.Compare(remoteVector)
}

// NeedsMerge returns true if the lockfiles have diverged and need merging.
func (a *SyncAdapter) NeedsMerge(local, remote *lock.Lockfile) bool {
	return a.CompareStates(local, remote) == sync.Concurrent
}

// IsAhead returns true if local is strictly ahead of remote.
func (a *SyncAdapter) IsAhead(local, remote *lock.Lockfile) bool {
	return a.CompareStates(local, remote) == sync.After
}

// IsBehind returns true if local is strictly behind remote.
func (a *SyncAdapter) IsBehind(local, remote *lock.Lockfile) bool {
	return a.CompareStates(local, remote) == sync.Before
}

// IsInSync returns true if local and remote are identical.
func (a *SyncAdapter) IsInSync(local, remote *lock.Lockfile) bool {
	return a.CompareStates(local, remote) == sync.Equal
}
