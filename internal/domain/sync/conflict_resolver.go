package sync

import "errors"

// ResolutionStrategy defines how conflicts should be resolved.
type ResolutionStrategy int

const (
	// StrategyManual requires user intervention for all conflicts.
	StrategyManual ResolutionStrategy = iota
	// StrategyNewest takes the version with the later modification time or higher vector.
	StrategyNewest
	// StrategyLocalWins always prefers the local version.
	StrategyLocalWins
	// StrategyRemoteWins always prefers the remote version.
	StrategyRemoteWins
	// StrategyAuto uses the most appropriate strategy based on conflict type.
	StrategyAuto
)

// String returns a human-readable name for the strategy.
func (s ResolutionStrategy) String() string {
	switch s {
	case StrategyManual:
		return "manual"
	case StrategyNewest:
		return "newest"
	case StrategyLocalWins:
		return "local-wins"
	case StrategyRemoteWins:
		return "remote-wins"
	case StrategyAuto:
		return "auto"
	default:
		return "unknown"
	}
}

// ParseResolutionStrategy parses a string into a ResolutionStrategy.
func ParseResolutionStrategy(s string) (ResolutionStrategy, error) {
	switch s {
	case "manual":
		return StrategyManual, nil
	case "newest":
		return StrategyNewest, nil
	case "local-wins", "local":
		return StrategyLocalWins, nil
	case "remote-wins", "remote":
		return StrategyRemoteWins, nil
	case "auto":
		return StrategyAuto, nil
	default:
		return StrategyManual, errors.New("unknown resolution strategy: " + s)
	}
}

// ResolutionChoice represents the user's or auto choice for a conflict.
type ResolutionChoice int

const (
	// ChooseLocal selects the local version.
	ChooseLocal ResolutionChoice = iota
	// ChooseRemote selects the remote version.
	ChooseRemote
	// ChooseBase selects the base version (revert both changes).
	ChooseBase
	// ChooseSkip skips this conflict (leave unresolved).
	ChooseSkip
)

// String returns a human-readable name for the choice.
func (c ResolutionChoice) String() string {
	switch c {
	case ChooseLocal:
		return "local"
	case ChooseRemote:
		return "remote"
	case ChooseBase:
		return "base"
	case ChooseSkip:
		return "skip"
	default:
		return "unknown"
	}
}

// Resolution represents a resolved conflict with the chosen outcome.
type Resolution struct {
	conflict LockConflict
	choice   ResolutionChoice
	result   PackageLockInfo
	reason   string
}

// NewResolution creates a new Resolution.
func NewResolution(conflict LockConflict, choice ResolutionChoice, result PackageLockInfo, reason string) Resolution {
	return Resolution{
		conflict: conflict,
		choice:   choice,
		result:   result,
		reason:   reason,
	}
}

// Conflict returns the original conflict.
func (r Resolution) Conflict() LockConflict {
	return r.conflict
}

// Choice returns the resolution choice made.
func (r Resolution) Choice() ResolutionChoice {
	return r.choice
}

// Result returns the chosen package lock info.
// Returns zero value for skip or delete resolutions.
func (r Resolution) Result() PackageLockInfo {
	return r.result
}

// Reason returns the human-readable reason for this resolution.
func (r Resolution) Reason() string {
	return r.reason
}

// IsZero returns true if this is a zero-value Resolution.
func (r Resolution) IsZero() bool {
	return r.conflict.IsZero()
}

// IsSkipped returns true if this conflict was skipped.
func (r Resolution) IsSkipped() bool {
	return r.choice == ChooseSkip
}

// IsDelete returns true if the resolution is to delete the package.
// This happens when choosing the side that doesn't have the package.
func (r Resolution) IsDelete() bool {
	return r.result.IsZero() && r.choice != ChooseSkip
}

// ConflictResolver resolves conflicts using a specified strategy.
type ConflictResolver struct {
	strategy ResolutionStrategy
}

// NewConflictResolver creates a new ConflictResolver with the given strategy.
func NewConflictResolver(strategy ResolutionStrategy) *ConflictResolver {
	return &ConflictResolver{strategy: strategy}
}

// Strategy returns the current resolution strategy.
func (r *ConflictResolver) Strategy() ResolutionStrategy {
	return r.strategy
}

// ResolveAll attempts to resolve all conflicts using the configured strategy.
// Returns resolutions for auto-resolvable conflicts and remaining manual conflicts.
func (r *ConflictResolver) ResolveAll(conflicts []LockConflict) (resolved []Resolution, manual []LockConflict) {
	resolved = make([]Resolution, 0, len(conflicts))
	manual = make([]LockConflict, 0)

	for _, c := range conflicts {
		resolution, needsManual := r.Resolve(c)
		if needsManual {
			manual = append(manual, c)
		} else {
			resolved = append(resolved, resolution)
		}
	}

	return resolved, manual
}

// Resolve attempts to resolve a single conflict.
// Returns the resolution and a boolean indicating if manual intervention is needed.
func (r *ConflictResolver) Resolve(c LockConflict) (Resolution, bool) {
	switch r.strategy {
	case StrategyManual:
		// All conflicts require manual resolution
		return Resolution{}, true

	case StrategyLocalWins:
		return r.resolveLocalWins(c), false

	case StrategyRemoteWins:
		return r.resolveRemoteWins(c), false

	case StrategyNewest:
		return r.resolveNewest(c)

	case StrategyAuto:
		return r.resolveAuto(c)

	default:
		return Resolution{}, true
	}
}

// resolveLocalWins always chooses the local version.
func (r *ConflictResolver) resolveLocalWins(c LockConflict) Resolution {
	return NewResolution(c, ChooseLocal, c.Local(), "strategy: local-wins")
}

// resolveRemoteWins always chooses the remote version.
func (r *ConflictResolver) resolveRemoteWins(c LockConflict) Resolution {
	return NewResolution(c, ChooseRemote, c.Remote(), "strategy: remote-wins")
}

// resolveNewest chooses the version with the later modification or higher vector.
func (r *ConflictResolver) resolveNewest(c LockConflict) (Resolution, bool) {
	local := c.Local()
	remote := c.Remote()

	// If one side is empty (deleted), we need manual resolution for newest strategy
	if local.IsZero() || remote.IsZero() {
		return Resolution{}, true
	}

	// Try to use version vectors for causal ordering
	if !local.Provenance().IsZero() && !remote.Provenance().IsZero() {
		localVec := local.Provenance().VectorAtChange()
		remoteVec := remote.Provenance().VectorAtChange()

		switch localVec.Compare(remoteVec) {
		case After:
			return NewResolution(c, ChooseLocal, local, "newest: local has later vector"), false
		case Before:
			return NewResolution(c, ChooseRemote, remote, "newest: remote has later vector"), false
		case Equal:
			// Same vector - fallback to timestamp
		case Concurrent:
			// Can't determine newest - need manual
			return Resolution{}, true
		}
	}

	// Fallback to timestamp comparison
	if local.ModifiedAt().After(remote.ModifiedAt()) {
		return NewResolution(c, ChooseLocal, local, "newest: local has later timestamp"), false
	}
	if remote.ModifiedAt().After(local.ModifiedAt()) {
		return NewResolution(c, ChooseRemote, remote, "newest: remote has later timestamp"), false
	}

	// Same timestamp - need manual
	return Resolution{}, true
}

// resolveAuto uses intelligent resolution based on conflict type.
func (r *ConflictResolver) resolveAuto(c LockConflict) (Resolution, bool) {
	switch c.Type() {
	case LocalOnly:
		// Package only exists locally - keep it (local addition)
		return NewResolution(c, ChooseLocal, c.Local(), "auto: local addition"), false

	case RemoteOnly:
		// Package only exists remotely - take it (remote addition)
		return NewResolution(c, ChooseRemote, c.Remote(), "auto: remote addition"), false

	case VersionMismatch:
		// Try to determine causality
		return r.resolveVersionMismatchAuto(c)

	case BothModified:
		// Both modified - check if we can determine causality
		if c.Local().IsZero() || c.Remote().IsZero() {
			// Delete/modify conflict - needs manual resolution
			return Resolution{}, true
		}
		return r.resolveVersionMismatchAuto(c)

	default:
		return Resolution{}, true
	}
}

// resolveVersionMismatchAuto tries to resolve version mismatches automatically.
func (r *ConflictResolver) resolveVersionMismatchAuto(c LockConflict) (Resolution, bool) {
	local := c.Local()
	remote := c.Remote()

	// Both need to exist for version comparison
	if local.IsZero() || remote.IsZero() {
		return Resolution{}, true
	}

	// Check version vectors if available
	if !local.Provenance().IsZero() && !remote.Provenance().IsZero() {
		localVec := local.Provenance().VectorAtChange()
		remoteVec := remote.Provenance().VectorAtChange()

		switch localVec.Compare(remoteVec) {
		case After:
			return NewResolution(c, ChooseLocal, local, "auto: local happened-after remote"), false
		case Before:
			return NewResolution(c, ChooseRemote, remote, "auto: remote happened-after local"), false
		case Equal:
			// Same vector with different versions - shouldn't happen, need manual
			return Resolution{}, true
		case Concurrent:
			// Concurrent changes - need manual
			return Resolution{}, true
		}
	}

	// No provenance - fall back to timestamp
	if local.ModifiedAt().After(remote.ModifiedAt()) {
		return NewResolution(c, ChooseLocal, local, "auto: local has later timestamp"), false
	}
	if remote.ModifiedAt().After(local.ModifiedAt()) {
		return NewResolution(c, ChooseRemote, remote, "auto: remote has later timestamp"), false
	}

	// Can't determine - need manual
	return Resolution{}, true
}

// ResolveManually creates a resolution for a manual choice.
func ResolveManually(c LockConflict, choice ResolutionChoice) Resolution {
	var result PackageLockInfo
	var reason string

	switch choice {
	case ChooseLocal:
		result = c.Local()
		reason = "manually selected local"
	case ChooseRemote:
		result = c.Remote()
		reason = "manually selected remote"
	case ChooseBase:
		result = c.Base()
		reason = "manually selected base"
	case ChooseSkip:
		reason = "manually skipped"
	}

	return NewResolution(c, choice, result, reason)
}
