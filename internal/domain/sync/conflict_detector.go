package sync

// ConflictDetector detects conflicts between local and remote lockfile states.
// It implements three-way merge logic when a common ancestor (base) is available.
type ConflictDetector struct{}

// NewConflictDetector creates a new ConflictDetector.
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// DetectInput contains the inputs for conflict detection.
type DetectInput struct {
	// Local packages from the local lockfile
	Local map[string]PackageLockInfo
	// Remote packages from the remote lockfile
	Remote map[string]PackageLockInfo
	// Base packages from the common ancestor (optional, enables three-way merge)
	Base map[string]PackageLockInfo
}

// DetectResult contains the results of conflict detection.
type DetectResult struct {
	// Conflicts that need resolution
	Conflicts []LockConflict
	// AutoResolvable conflicts that can be resolved automatically
	AutoResolvable []LockConflict
	// Clean packages that exist in both and are identical
	Clean []string
}

// Detect identifies conflicts between local and remote package states.
// If a base is provided, it performs three-way merge detection.
// Returns all conflicts found, categorized by resolvability.
func (d *ConflictDetector) Detect(input DetectInput) DetectResult {
	result := DetectResult{
		Conflicts:      []LockConflict{},
		AutoResolvable: []LockConflict{},
		Clean:          []string{},
	}

	// Collect all unique package keys
	allKeys := make(map[string]struct{})
	for key := range input.Local {
		allKeys[key] = struct{}{}
	}
	for key := range input.Remote {
		allKeys[key] = struct{}{}
	}

	for key := range allKeys {
		local, hasLocal := input.Local[key]
		remote, hasRemote := input.Remote[key]
		base, hasBase := input.Base[key]

		conflict := d.detectConflict(key, local, hasLocal, remote, hasRemote, base, hasBase)
		if conflict.IsZero() {
			// No conflict - packages are identical or both missing
			if hasLocal && hasRemote {
				result.Clean = append(result.Clean, key)
			}
			continue
		}

		if conflict.IsResolvable() {
			result.AutoResolvable = append(result.AutoResolvable, conflict)
		} else {
			result.Conflicts = append(result.Conflicts, conflict)
		}
	}

	return result
}

// detectConflict determines if there's a conflict for a single package.
func (d *ConflictDetector) detectConflict(
	key string,
	local PackageLockInfo, hasLocal bool,
	remote PackageLockInfo, hasRemote bool,
	base PackageLockInfo, hasBase bool,
) LockConflict {
	// Case 1: Package only exists locally
	if hasLocal && !hasRemote {
		if hasBase {
			// Package was deleted remotely - this is a conflict
			return NewLockConflict(key, BothModified, local, PackageLockInfo{}, base)
		}
		// Package was added locally - this is LocalOnly
		return NewLockConflict(key, LocalOnly, local, PackageLockInfo{}, PackageLockInfo{})
	}

	// Case 2: Package only exists remotely
	if !hasLocal && hasRemote {
		if hasBase {
			// Package was deleted locally - this is a conflict
			return NewLockConflict(key, BothModified, PackageLockInfo{}, remote, base)
		}
		// Package was added remotely - this is RemoteOnly
		return NewLockConflict(key, RemoteOnly, PackageLockInfo{}, remote, PackageLockInfo{})
	}

	// Case 3: Package exists in neither (shouldn't happen given how we iterate)
	if !hasLocal && !hasRemote {
		return LockConflict{} // No conflict
	}

	// Case 4: Package exists in both - check for version mismatch
	if local.Version() == remote.Version() {
		return LockConflict{} // Same version, no conflict
	}

	// Versions differ - need to determine conflict type
	return d.detectVersionConflict(key, local, remote, base, hasBase)
}

// detectVersionConflict determines the type of version conflict.
func (d *ConflictDetector) detectVersionConflict(
	key string,
	local, remote, base PackageLockInfo,
	hasBase bool,
) LockConflict {
	// Check if we can determine causality from provenance
	localProv := local.Provenance()
	remoteProv := remote.Provenance()

	// If either has no provenance, we can only do timestamp-based comparison
	if localProv.IsZero() || remoteProv.IsZero() {
		return NewLockConflict(key, VersionMismatch, local, remote, base)
	}

	// Use version vectors to determine causal relationship
	localVector := localProv.VectorAtChange()
	remoteVector := remoteProv.VectorAtChange()

	relation := localVector.Compare(remoteVector)

	switch relation {
	case Equal:
		// Same vector but different versions - shouldn't happen normally
		// Treat as version mismatch
		return NewLockConflict(key, VersionMismatch, local, remote, base)
	case Before:
		// Local happened before remote - remote is newer
		return NewLockConflict(key, VersionMismatch, local, remote, base)
	case After:
		// Local happened after remote - local is newer
		return NewLockConflict(key, VersionMismatch, local, remote, base)
	case Concurrent:
		// Concurrent modifications - need manual resolution
		if hasBase && base.Version() != local.Version() && base.Version() != remote.Version() {
			// Both modified from base
			return NewLockConflict(key, BothModified, local, remote, base)
		}
		return NewLockConflict(key, BothModified, local, remote, base)
	default:
		return NewLockConflict(key, VersionMismatch, local, remote, base)
	}
}

// HasConflicts returns true if there are any conflicts that require manual resolution.
func (r DetectResult) HasConflicts() bool {
	return len(r.Conflicts) > 0
}

// HasAutoResolvable returns true if there are conflicts that can be auto-resolved.
func (r DetectResult) HasAutoResolvable() bool {
	return len(r.AutoResolvable) > 0
}

// AllConflicts returns all conflicts (both manual and auto-resolvable).
func (r DetectResult) AllConflicts() []LockConflict {
	all := make([]LockConflict, 0, len(r.Conflicts)+len(r.AutoResolvable))
	all = append(all, r.Conflicts...)
	all = append(all, r.AutoResolvable...)
	return all
}

// TotalConflicts returns the total number of conflicts.
func (r DetectResult) TotalConflicts() int {
	return len(r.Conflicts) + len(r.AutoResolvable)
}
