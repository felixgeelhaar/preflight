package sync

import (
	"fmt"
	"time"
)

// ConflictType represents the type of conflict between local and remote lock entries.
type ConflictType int

const (
	// VersionMismatch indicates local and remote have different versions.
	VersionMismatch ConflictType = iota
	// LocalOnly indicates the package exists locally but not remotely.
	LocalOnly
	// RemoteOnly indicates the package exists remotely but not locally.
	RemoteOnly
	// BothModified indicates both local and remote modified the package concurrently.
	BothModified
)

// String returns a human-readable description of the conflict type.
func (ct ConflictType) String() string {
	switch ct {
	case VersionMismatch:
		return "version_mismatch"
	case LocalOnly:
		return "local_only"
	case RemoteOnly:
		return "remote_only"
	case BothModified:
		return "both_modified"
	default:
		return "unknown"
	}
}

// PackageLockInfo represents the state of a package lock at a point in time.
// This is a lightweight representation used during conflict detection.
type PackageLockInfo struct {
	version    string
	provenance PackageProvenance
	modifiedAt time.Time
}

// NewPackageLockInfo creates a new PackageLockInfo.
func NewPackageLockInfo(version string, provenance PackageProvenance) PackageLockInfo {
	return PackageLockInfo{
		version:    version,
		provenance: provenance,
		modifiedAt: time.Now().UTC(),
	}
}

// NewPackageLockInfoWithTime creates a new PackageLockInfo with explicit modification time.
func NewPackageLockInfoWithTime(version string, provenance PackageProvenance, modifiedAt time.Time) PackageLockInfo {
	return PackageLockInfo{
		version:    version,
		provenance: provenance,
		modifiedAt: modifiedAt,
	}
}

// Version returns the package version.
func (p PackageLockInfo) Version() string {
	return p.version
}

// Provenance returns the package provenance information.
func (p PackageLockInfo) Provenance() PackageProvenance {
	return p.provenance
}

// ModifiedAt returns when the package was modified.
func (p PackageLockInfo) ModifiedAt() time.Time {
	return p.modifiedAt
}

// IsZero returns true if this is a zero-value PackageLockInfo.
func (p PackageLockInfo) IsZero() bool {
	return p.version == "" && p.provenance.IsZero()
}

// LockConflict represents a conflict between local and remote lock states.
// It contains all the information needed to resolve the conflict.
type LockConflict struct {
	packageKey   string
	conflictType ConflictType
	local        PackageLockInfo
	remote       PackageLockInfo
	base         PackageLockInfo // Common ancestor (if known)
}

// NewLockConflict creates a new LockConflict.
func NewLockConflict(
	packageKey string,
	conflictType ConflictType,
	local, remote, base PackageLockInfo,
) LockConflict {
	return LockConflict{
		packageKey:   packageKey,
		conflictType: conflictType,
		local:        local,
		remote:       remote,
		base:         base,
	}
}

// PackageKey returns the unique key for the conflicting package (provider:name).
func (c LockConflict) PackageKey() string {
	return c.packageKey
}

// Type returns the type of conflict.
func (c LockConflict) Type() ConflictType {
	return c.conflictType
}

// Local returns the local lock state.
func (c LockConflict) Local() PackageLockInfo {
	return c.local
}

// Remote returns the remote lock state.
func (c LockConflict) Remote() PackageLockInfo {
	return c.remote
}

// Base returns the common ancestor lock state (if known).
func (c LockConflict) Base() PackageLockInfo {
	return c.base
}

// IsZero returns true if this is a zero-value LockConflict.
func (c LockConflict) IsZero() bool {
	return c.packageKey == ""
}

// HasBase returns true if a common ancestor (base) is available.
// Three-way merge is possible when a base exists.
func (c LockConflict) HasBase() bool {
	return !c.base.IsZero()
}

// IsResolvable returns true if this conflict can be automatically resolved.
// A conflict is resolvable if:
// - It's a LocalOnly or RemoteOnly conflict (simple add/remove)
// - The changes are not concurrent (one happened-before the other)
// - It's NOT a delete/modify conflict (BothModified with one side missing)
func (c LockConflict) IsResolvable() bool {
	// LocalOnly and RemoteOnly are always resolvable
	if c.conflictType == LocalOnly || c.conflictType == RemoteOnly {
		return true
	}

	// BothModified with delete/modify (one side deleted) requires manual resolution
	if c.conflictType == BothModified {
		// If either side is completely missing (deleted), it's a delete/modify conflict
		// These always require manual resolution
		if c.local.IsZero() || c.remote.IsZero() {
			return false
		}
	}

	// VersionMismatch: If either side has no provenance, fall back to timestamp-based resolution
	if c.conflictType == VersionMismatch {
		if c.local.provenance.IsZero() || c.remote.provenance.IsZero() {
			return true // Fall back to timestamp-based resolution
		}
	}

	// Both sides have provenance - check if changes are concurrent using version vectors
	if !c.local.provenance.IsZero() && !c.remote.provenance.IsZero() {
		localVector := c.local.provenance.VectorAtChange()
		remoteVector := c.remote.provenance.VectorAtChange()

		// If one happened-before the other, it's resolvable (take the later one)
		relation := localVector.Compare(remoteVector)
		return relation != Concurrent
	}

	// Default: not auto-resolvable
	return false
}

// String returns a human-readable description of the conflict.
func (c LockConflict) String() string {
	return fmt.Sprintf("%s: %s (local=%s, remote=%s)",
		c.packageKey, c.conflictType.String(), c.local.version, c.remote.version)
}
