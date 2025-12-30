package sync

import (
	"errors"
	"fmt"
)

// SyncEngine orchestrates lockfile synchronization between machines.
// It coordinates conflict detection, resolution, and lockfile merging.
//
//nolint:revive // SyncEngine is the canonical name for this type
type SyncEngine struct {
	detector  *ConflictDetector
	resolver  *ConflictResolver
	machineID MachineID
	hostname  string
}

// SyncEngineOption configures a SyncEngine.
//
//nolint:revive // SyncEngineOption follows Go options pattern naming
type SyncEngineOption func(*SyncEngine)

// WithResolver sets a custom conflict resolver.
func WithResolver(resolver *ConflictResolver) SyncEngineOption {
	return func(e *SyncEngine) {
		e.resolver = resolver
	}
}

// WithMachineID sets the local machine identity.
func WithMachineID(id MachineID, hostname string) SyncEngineOption {
	return func(e *SyncEngine) {
		e.machineID = id
		e.hostname = hostname
	}
}

// NewSyncEngine creates a new SyncEngine with the given options.
func NewSyncEngine(opts ...SyncEngineOption) *SyncEngine {
	e := &SyncEngine{
		detector: NewConflictDetector(),
		resolver: NewConflictResolver(StrategyAuto),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// SyncInput contains the inputs for a sync operation.
//
//nolint:revive // SyncInput is clear and descriptive
type SyncInput struct {
	// Local is the local lockfile state
	Local *LockfileState
	// Remote is the remote lockfile state
	Remote *LockfileState
	// Base is the common ancestor (optional, enables three-way merge)
	Base *LockfileState
}

// LockfileState represents a lockfile's package state for syncing.
// This is a simplified view focused on sync-relevant data.
type LockfileState struct {
	// Packages maps package keys to their lock info
	Packages map[string]PackageLockInfo
	// Metadata contains sync metadata (version vector, lineage)
	Metadata SyncMetadata
}

// NewLockfileState creates a new empty LockfileState.
func NewLockfileState() *LockfileState {
	return &LockfileState{
		Packages: make(map[string]PackageLockInfo),
		Metadata: NewSyncMetadata(NewVersionVector()),
	}
}

// NewLockfileStateWithMetadata creates a LockfileState with existing metadata.
func NewLockfileStateWithMetadata(meta SyncMetadata) *LockfileState {
	return &LockfileState{
		Packages: make(map[string]PackageLockInfo),
		Metadata: meta,
	}
}

// AddPackage adds a package to the state.
func (s *LockfileState) AddPackage(key string, info PackageLockInfo) {
	s.Packages[key] = info
}

// IsEmpty returns true if the state has no packages.
func (s *LockfileState) IsEmpty() bool {
	return s == nil || len(s.Packages) == 0
}

// SyncResult contains the result of a sync operation.
//
//nolint:revive // SyncResult is clear and descriptive
type SyncResult struct {
	// Merged is the resulting merged state
	Merged *LockfileState
	// Resolutions are the conflict resolutions applied
	Resolutions []Resolution
	// ManualConflicts are conflicts that need user intervention
	ManualConflicts []LockConflict
	// Stats contains sync statistics
	Stats SyncStats
}

// SyncStats contains statistics about the sync operation.
//
//nolint:revive // SyncStats is clear and descriptive
type SyncStats struct {
	// PackagesUnchanged is the count of packages that didn't change
	PackagesUnchanged int
	// PackagesAdded is the count of packages added
	PackagesAdded int
	// PackagesRemoved is the count of packages removed
	PackagesRemoved int
	// PackagesUpdated is the count of packages with version changes
	PackagesUpdated int
	// ConflictsAutoResolved is the count of auto-resolved conflicts
	ConflictsAutoResolved int
	// ConflictsManual is the count of conflicts needing manual resolution
	ConflictsManual int
}

// HasManualConflicts returns true if there are unresolved conflicts.
func (r *SyncResult) HasManualConflicts() bool {
	return len(r.ManualConflicts) > 0
}

// IsClean returns true if the sync completed without manual intervention needed.
func (r *SyncResult) IsClean() bool {
	return !r.HasManualConflicts()
}

// Sync errors.
var (
	ErrNoLocalState    = errors.New("local state is required")
	ErrNoRemoteState   = errors.New("remote state is required")
	ErrMergeIncomplete = errors.New("merge incomplete: manual conflicts remain")
)

// Sync performs a synchronization between local and remote states.
// Returns the merged result or an error if sync cannot proceed.
func (e *SyncEngine) Sync(input SyncInput) (*SyncResult, error) {
	if input.Local == nil {
		return nil, ErrNoLocalState
	}
	if input.Remote == nil {
		return nil, ErrNoRemoteState
	}

	// Build detection input
	detectInput := DetectInput{
		Local:  input.Local.Packages,
		Remote: input.Remote.Packages,
	}
	if input.Base != nil {
		detectInput.Base = input.Base.Packages
	}

	// Detect conflicts
	detectResult := e.detector.Detect(detectInput)

	// Resolve what we can
	allConflicts := detectResult.AllConflicts()
	resolved, manual := e.resolver.ResolveAll(allConflicts)

	// Build merged state
	merged, stats := e.buildMergedState(input, detectResult, resolved)

	// Update stats
	stats.ConflictsAutoResolved = len(resolved)
	stats.ConflictsManual = len(manual)

	return &SyncResult{
		Merged:          merged,
		Resolutions:     resolved,
		ManualConflicts: manual,
		Stats:           stats,
	}, nil
}

// buildMergedState creates the merged lockfile state from sync results.
func (e *SyncEngine) buildMergedState(
	input SyncInput,
	detectResult DetectResult,
	resolutions []Resolution,
) (*LockfileState, SyncStats) {
	stats := SyncStats{}

	// Start with merged metadata
	mergedMeta := input.Local.Metadata.Merge(input.Remote.Metadata)
	if !e.machineID.IsZero() {
		mergedMeta = mergedMeta.RecordActivity(e.machineID, e.hostname)
	}

	merged := NewLockfileStateWithMetadata(mergedMeta)

	// Track which packages we've handled via resolutions
	resolvedKeys := make(map[string]bool)
	for _, res := range resolutions {
		key := res.Conflict().PackageKey()
		resolvedKeys[key] = true

		if res.IsSkipped() {
			// Keep local version for skipped
			if info, ok := input.Local.Packages[key]; ok {
				merged.AddPackage(key, info)
			}
			continue
		}

		result := res.Result()
		if result.IsZero() {
			// Package removed
			stats.PackagesRemoved++
			continue
		}

		merged.AddPackage(key, result)

		// Determine if this was an add or update
		_, inLocal := input.Local.Packages[key]
		_, inRemote := input.Remote.Packages[key]
		switch {
		case !inLocal && inRemote:
			// Remote addition
			stats.PackagesAdded++
		case inLocal && !inRemote:
			// Local-only kept
			stats.PackagesAdded++
		default:
			stats.PackagesUpdated++
		}
	}

	// Add clean packages (unchanged in both)
	for _, key := range detectResult.Clean {
		if !resolvedKeys[key] {
			if info, ok := input.Local.Packages[key]; ok {
				merged.AddPackage(key, info)
				stats.PackagesUnchanged++
			}
		}
	}

	// Handle packages that weren't in conflict (exist in only one side, not detected as conflict)
	allKeys := make(map[string]bool)
	for key := range input.Local.Packages {
		allKeys[key] = true
	}
	for key := range input.Remote.Packages {
		allKeys[key] = true
	}

	for key := range allKeys {
		if resolvedKeys[key] {
			continue
		}
		// Check if already in clean
		isClean := false
		for _, cleanKey := range detectResult.Clean {
			if cleanKey == key {
				isClean = true
				break
			}
		}
		if isClean {
			continue
		}

		// This package wasn't detected as a conflict and isn't clean
		// This shouldn't normally happen, but handle gracefully
		if info, ok := input.Local.Packages[key]; ok {
			merged.AddPackage(key, info)
		} else if info, ok := input.Remote.Packages[key]; ok {
			merged.AddPackage(key, info)
		}
	}

	return merged, stats
}

// Pull fetches remote changes and merges them into local.
// This is a convenience method that assumes remote takes precedence for additions.
func (e *SyncEngine) Pull(local, remote *LockfileState) (*SyncResult, error) {
	return e.Sync(SyncInput{
		Local:  local,
		Remote: remote,
	})
}

// Push prepares local changes for pushing to remote.
// Returns what the remote state should become after accepting local changes.
func (e *SyncEngine) Push(local, remote *LockfileState) (*SyncResult, error) {
	// For push, we flip the perspective - local changes should take precedence
	// But we still detect conflicts the same way
	return e.Sync(SyncInput{
		Local:  local,
		Remote: remote,
	})
}

// ThreeWaySync performs a three-way merge using a common ancestor.
func (e *SyncEngine) ThreeWaySync(local, remote, base *LockfileState) (*SyncResult, error) {
	return e.Sync(SyncInput{
		Local:  local,
		Remote: remote,
		Base:   base,
	})
}

// ResolveManualConflict applies a manual resolution to a pending conflict.
func (e *SyncEngine) ResolveManualConflict(
	result *SyncResult,
	conflict LockConflict,
	choice ResolutionChoice,
) error {
	// Find and remove the conflict from manual list
	found := false
	newManual := make([]LockConflict, 0, len(result.ManualConflicts))
	for _, c := range result.ManualConflicts {
		if c.PackageKey() == conflict.PackageKey() {
			found = true
			continue
		}
		newManual = append(newManual, c)
	}

	if !found {
		return fmt.Errorf("conflict not found: %s", conflict.PackageKey())
	}

	// Create resolution
	resolution := ResolveManually(conflict, choice)
	result.Resolutions = append(result.Resolutions, resolution)
	result.ManualConflicts = newManual

	// Apply to merged state
	if !resolution.IsSkipped() {
		if resolution.IsDelete() {
			delete(result.Merged.Packages, conflict.PackageKey())
			result.Stats.PackagesRemoved++
		} else {
			result.Merged.AddPackage(conflict.PackageKey(), resolution.Result())
			result.Stats.PackagesUpdated++
		}
	}

	result.Stats.ConflictsManual--
	// Note: We don't increment ConflictsAutoResolved here since this was
	// a manual resolution, not auto-resolved. The caller should track
	// manual resolutions separately if needed.

	return nil
}

// CompareStates returns the causal relationship between two lockfile states.
func (e *SyncEngine) CompareStates(local, remote *LockfileState) CausalRelation {
	if local == nil || remote == nil {
		return Concurrent // Can't compare nil states
	}
	return local.Metadata.Vector().Compare(remote.Metadata.Vector())
}

// NeedsMerge returns true if the states have diverged and need merging.
func (e *SyncEngine) NeedsMerge(local, remote *LockfileState) bool {
	relation := e.CompareStates(local, remote)
	return relation == Concurrent
}

// IsAhead returns true if local is strictly ahead of remote.
func (e *SyncEngine) IsAhead(local, remote *LockfileState) bool {
	relation := e.CompareStates(local, remote)
	return relation == After
}

// IsBehind returns true if local is strictly behind remote.
func (e *SyncEngine) IsBehind(local, remote *LockfileState) bool {
	relation := e.CompareStates(local, remote)
	return relation == Before
}

// IsInSync returns true if local and remote are identical.
func (e *SyncEngine) IsInSync(local, remote *LockfileState) bool {
	relation := e.CompareStates(local, remote)
	return relation == Equal
}
