package sync

import (
	"fmt"
	"time"
)

// LockfileMerger applies sync results to create merged lockfile states.
// It handles the conversion between sync domain types and lockfile structures.
type LockfileMerger struct {
	machineID MachineID
	hostname  string
}

// NewLockfileMerger creates a new LockfileMerger with the given machine identity.
func NewLockfileMerger(machineID MachineID, hostname string) *LockfileMerger {
	return &LockfileMerger{
		machineID: machineID,
		hostname:  hostname,
	}
}

// MergeResult contains the outcome of a merge operation.
type MergeResult struct {
	// State is the merged lockfile state
	State *LockfileState
	// Changes is a list of changes made during merge
	Changes []MergeChange
	// Stats contains merge statistics
	Stats MergeStats
}

// MergeChange represents a single change made during merge.
type MergeChange struct {
	// PackageKey identifies the package
	PackageKey string
	// Type is the type of change
	Type MergeChangeType
	// Before is the state before merge (may be zero for additions)
	Before PackageLockInfo
	// After is the state after merge (may be zero for removals)
	After PackageLockInfo
	// Reason explains why this change was made
	Reason string
}

// MergeChangeType categorizes merge changes.
type MergeChangeType int

const (
	// ChangeAdded indicates a package was added.
	ChangeAdded MergeChangeType = iota
	// ChangeRemoved indicates a package was removed.
	ChangeRemoved
	// ChangeUpdated indicates a package version was updated.
	ChangeUpdated
	// ChangeKept indicates a package was kept unchanged.
	ChangeKept
)

// String returns a human-readable name for the change type.
func (t MergeChangeType) String() string {
	switch t {
	case ChangeAdded:
		return "added"
	case ChangeRemoved:
		return "removed"
	case ChangeUpdated:
		return "updated"
	case ChangeKept:
		return "kept"
	default:
		return "unknown"
	}
}

// MergeStats contains statistics about the merge operation.
type MergeStats struct {
	// TotalPackages is the count of packages in the merged state
	TotalPackages int
	// Added is the count of packages added
	Added int
	// Removed is the count of packages removed
	Removed int
	// Updated is the count of packages updated
	Updated int
	// Kept is the count of packages unchanged
	Kept int
}

// Merge applies a sync result to create the final merged state.
// This takes the SyncResult from a sync operation and produces
// a clean MergeResult with detailed change tracking.
func (m *LockfileMerger) Merge(result *SyncResult) (*MergeResult, error) {
	if result == nil {
		return nil, fmt.Errorf("sync result is required")
	}
	if result.Merged == nil {
		return nil, fmt.Errorf("sync result has no merged state")
	}

	// Collect changes from resolutions
	changes := make([]MergeChange, 0, len(result.Resolutions))
	stats := MergeStats{}

	for _, res := range result.Resolutions {
		conflict := res.Conflict()
		change := MergeChange{
			PackageKey: conflict.PackageKey(),
			Before:     conflict.Local(),
			After:      res.Result(),
			Reason:     res.Reason(),
		}

		switch {
		case res.IsSkipped():
			change.Type = ChangeKept
			change.After = conflict.Local()
			stats.Kept++
		case res.IsDelete():
			change.Type = ChangeRemoved
			stats.Removed++
		case conflict.Local().IsZero():
			change.Type = ChangeAdded
			stats.Added++
		default:
			change.Type = ChangeUpdated
			stats.Updated++
		}

		changes = append(changes, change)
	}

	stats.TotalPackages = len(result.Merged.Packages)

	return &MergeResult{
		State:   result.Merged,
		Changes: changes,
		Stats:   stats,
	}, nil
}

// ApplyResolution applies a manual resolution to a pending sync result.
// This is a convenience method that wraps SyncEngine.ResolveManualConflict
// and tracks the change.
func (m *LockfileMerger) ApplyResolution(
	engine *SyncEngine,
	result *SyncResult,
	conflict LockConflict,
	choice ResolutionChoice,
) (*MergeChange, error) {
	// Capture before state
	before := conflict.Local()

	// Apply the resolution
	if err := engine.ResolveManualConflict(result, conflict, choice); err != nil {
		return nil, err
	}

	// Find the resolution that was just applied
	var resolution Resolution
	found := false
	for _, res := range result.Resolutions {
		if res.Conflict().PackageKey() == conflict.PackageKey() {
			resolution = res
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("internal error: resolution for %s not found after applying", conflict.PackageKey())
	}

	// Build change record
	change := &MergeChange{
		PackageKey: conflict.PackageKey(),
		Before:     before,
		After:      resolution.Result(),
		Reason:     resolution.Reason(),
	}

	switch {
	case resolution.IsSkipped():
		change.Type = ChangeKept
		change.After = before
	case resolution.IsDelete():
		change.Type = ChangeRemoved
	case before.IsZero():
		change.Type = ChangeAdded
	default:
		change.Type = ChangeUpdated
	}

	return change, nil
}

// FromPackageLocks converts a map of package locks to a LockfileState.
// This is useful when converting from the lock domain to the sync domain.
func (m *LockfileMerger) FromPackageLocks(
	packages map[string]PackageLockInfo,
	metadata SyncMetadata,
) *LockfileState {
	state := NewLockfileStateWithMetadata(metadata)
	for key, info := range packages {
		state.AddPackage(key, info)
	}
	return state
}

// UpdateProvenance updates a package's provenance for the current machine.
// This should be called when a package is modified locally.
func (m *LockfileMerger) UpdateProvenance(info PackageLockInfo, vector VersionVector) PackageLockInfo {
	provenance := NewPackageProvenance(m.machineID, vector)
	return info.WithProvenance(provenance).WithModifiedAt(time.Now())
}

// IncrementVector increments the version vector for the current machine.
// This should be called after making local changes.
func (m *LockfileMerger) IncrementVector(vector VersionVector) VersionVector {
	return vector.Increment(m.machineID)
}

// PrepareForCommit prepares a lockfile state for committing.
// This increments the version vector and updates metadata.
func (m *LockfileMerger) PrepareForCommit(state *LockfileState) *LockfileState {
	if state == nil {
		return nil
	}

	// RecordActivity increments vector, updates lineage, and records timestamp
	// We start with current metadata and record new activity
	newMeta := state.Metadata.RecordActivity(m.machineID, m.hostname)

	// Create new state with updated metadata
	newState := NewLockfileStateWithMetadata(newMeta)
	for key, info := range state.Packages {
		newState.AddPackage(key, info)
	}

	return newState
}

// Diff compares two lockfile states and returns the changes.
// This is useful for showing what changed between states.
func (m *LockfileMerger) Diff(before, after *LockfileState) []MergeChange {
	if before == nil {
		before = NewLockfileState()
	}
	if after == nil {
		after = NewLockfileState()
	}

	changes := make([]MergeChange, 0)

	// Find added and updated packages
	for key, afterInfo := range after.Packages {
		beforeInfo, existed := before.Packages[key]
		if !existed {
			changes = append(changes, MergeChange{
				PackageKey: key,
				Type:       ChangeAdded,
				After:      afterInfo,
				Reason:     "added",
			})
		} else if !beforeInfo.Equals(afterInfo) {
			changes = append(changes, MergeChange{
				PackageKey: key,
				Type:       ChangeUpdated,
				Before:     beforeInfo,
				After:      afterInfo,
				Reason:     "updated",
			})
		}
	}

	// Find removed packages
	for key, beforeInfo := range before.Packages {
		if _, exists := after.Packages[key]; !exists {
			changes = append(changes, MergeChange{
				PackageKey: key,
				Type:       ChangeRemoved,
				Before:     beforeInfo,
				Reason:     "removed",
			})
		}
	}

	return changes
}
