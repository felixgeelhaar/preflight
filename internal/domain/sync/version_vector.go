package sync

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// CausalRelation represents the causal relationship between two version vectors.
type CausalRelation int

const (
	// Equal indicates the vectors are identical.
	Equal CausalRelation = iota
	// Before indicates this vector happened-before the other.
	Before
	// After indicates this vector happened-after the other.
	After
	// Concurrent indicates neither vector happened-before the other.
	Concurrent
)

// String returns a human-readable description of the causal relation.
func (r CausalRelation) String() string {
	switch r {
	case Equal:
		return "equal"
	case Before:
		return "before"
	case After:
		return "after"
	case Concurrent:
		return "concurrent"
	default:
		return "unknown"
	}
}

// VersionVector implements vector clocks for causal ordering.
// Each entry maps a machine ID to its logical timestamp.
// Vector clocks allow detection of concurrent modifications that
// cannot be ordered using simple wall-clock timestamps.
type VersionVector struct {
	entries map[string]uint64
}

// NewVersionVector creates an empty version vector.
func NewVersionVector() VersionVector {
	return VersionVector{entries: make(map[string]uint64)}
}

// FromMap creates a version vector from an existing map.
// The map is copied to ensure immutability.
func FromMap(m map[string]uint64) VersionVector {
	entries := make(map[string]uint64, len(m))
	for k, v := range m {
		entries[k] = v
	}
	return VersionVector{entries: entries}
}

// Get returns the timestamp for a machine ID.
// Returns 0 if the machine has no entry.
func (v VersionVector) Get(machineID string) uint64 {
	return v.entries[machineID]
}

// GetByMachineID returns the timestamp for a MachineID.
func (v VersionVector) GetByMachineID(id MachineID) uint64 {
	return v.entries[id.String()]
}

// Set returns a new vector with the timestamp for machineID set to value.
// This method is immutable - it returns a new vector.
func (v VersionVector) Set(machineID string, value uint64) VersionVector {
	newEntries := maps.Clone(v.entries)
	if newEntries == nil {
		newEntries = make(map[string]uint64)
	}
	newEntries[machineID] = value
	return VersionVector{entries: newEntries}
}

// Increment advances the timestamp for the given machine ID by 1.
// Returns a new vector (immutable operation).
func (v VersionVector) Increment(machineID MachineID) VersionVector {
	id := machineID.String()
	return v.Set(id, v.entries[id]+1)
}

// IncrementByString advances the timestamp for the given machine ID string by 1.
func (v VersionVector) IncrementByString(machineID string) VersionVector {
	return v.Set(machineID, v.entries[machineID]+1)
}

// Merge combines two vectors by taking the element-wise maximum.
// This is used when incorporating knowledge from another machine.
// Returns a new vector (immutable operation).
func (v VersionVector) Merge(other VersionVector) VersionVector {
	newEntries := make(map[string]uint64)

	// Copy all entries from this vector
	for k, val := range v.entries {
		newEntries[k] = val
	}

	// Merge entries from other vector (taking max)
	for k, val := range other.entries {
		if val > newEntries[k] {
			newEntries[k] = val
		}
	}

	return VersionVector{entries: newEntries}
}

// Compare determines the causal relationship between two vectors.
//
// Returns:
//   - Equal: vectors are identical
//   - Before: v happened-before other (v < other)
//   - After: v happened-after other (v > other)
//   - Concurrent: neither happened-before the other
func (v VersionVector) Compare(other VersionVector) CausalRelation {
	// Collect all machine IDs from both vectors
	allKeys := make(map[string]struct{})
	for k := range v.entries {
		allKeys[k] = struct{}{}
	}
	for k := range other.entries {
		allKeys[k] = struct{}{}
	}

	// If both are empty, they're equal
	if len(allKeys) == 0 {
		return Equal
	}

	hasLess := false // v has at least one entry less than other
	hasMore := false // v has at least one entry greater than other

	for k := range allKeys {
		vVal := v.entries[k]
		oVal := other.entries[k]

		if vVal < oVal {
			hasLess = true
		} else if vVal > oVal {
			hasMore = true
		}
	}

	switch {
	case !hasLess && !hasMore:
		return Equal
	case hasLess && !hasMore:
		return Before
	case !hasLess && hasMore:
		return After
	default: // hasLess && hasMore
		return Concurrent
	}
}

// Dominates returns true if v >= other for all entries.
// This is equivalent to Compare returning Equal or After.
func (v VersionVector) Dominates(other VersionVector) bool {
	relation := v.Compare(other)
	return relation == Equal || relation == After
}

// IsConcurrentWith returns true if neither vector dominates the other.
func (v VersionVector) IsConcurrentWith(other VersionVector) bool {
	return v.Compare(other) == Concurrent
}

// IsEmpty returns true if the vector has no entries.
func (v VersionVector) IsEmpty() bool {
	return len(v.entries) == 0
}

// Size returns the number of machine entries in the vector.
func (v VersionVector) Size() int {
	return len(v.entries)
}

// MachineIDs returns all machine IDs in the vector, sorted.
func (v VersionVector) MachineIDs() []string {
	keys := make([]string, 0, len(v.entries))
	for k := range v.entries {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// Sum returns the total of all timestamps (useful for debugging).
func (v VersionVector) Sum() uint64 {
	var sum uint64
	for _, val := range v.entries {
		sum += val
	}
	return sum
}

// ToMap returns a copy of the internal map.
func (v VersionVector) ToMap() map[string]uint64 {
	return maps.Clone(v.entries)
}

// String returns a compact string representation for debugging.
func (v VersionVector) String() string {
	if len(v.entries) == 0 {
		return "{}"
	}

	keys := v.MachineIDs()
	parts := make([]string, len(keys))
	for i, k := range keys {
		// Use short ID for readability
		shortKey := k
		if len(k) > 8 {
			shortKey = k[:8]
		}
		parts[i] = fmt.Sprintf("%s:%d", shortKey, v.entries[k])
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// MarshalJSON implements json.Marshaler for serialization.
func (v VersionVector) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.entries)
}

// UnmarshalJSON implements json.Unmarshaler for deserialization.
func (v *VersionVector) UnmarshalJSON(data []byte) error {
	if v.entries == nil {
		v.entries = make(map[string]uint64)
	}
	return json.Unmarshal(data, &v.entries)
}

// Clone returns a deep copy of the version vector.
func (v VersionVector) Clone() VersionVector {
	return FromMap(v.entries)
}

// Equals returns true if two vectors have identical entries.
func (v VersionVector) Equals(other VersionVector) bool {
	return v.Compare(other) == Equal
}

// Max returns the maximum timestamp in the vector.
func (v VersionVector) Max() uint64 {
	var maxVal uint64
	for _, val := range v.entries {
		if val > maxVal {
			maxVal = val
		}
	}
	return maxVal
}

// ContainsMachine returns true if the vector has an entry for the machine.
func (v VersionVector) ContainsMachine(machineID string) bool {
	_, ok := v.entries[machineID]
	return ok
}

// WithoutMachine returns a new vector with the specified machine removed.
func (v VersionVector) WithoutMachine(machineID string) VersionVector {
	newEntries := make(map[string]uint64, len(v.entries))
	for k, val := range v.entries {
		if k != machineID {
			newEntries[k] = val
		}
	}
	return VersionVector{entries: newEntries}
}

// Diff returns machine IDs where the vectors differ.
func (v VersionVector) Diff(other VersionVector) []string {
	diff := []string{}

	// Check all keys in v
	for k, vVal := range v.entries {
		if oVal := other.entries[k]; vVal != oVal {
			diff = append(diff, k)
		}
	}

	// Check keys only in other
	for k := range other.entries {
		if _, ok := v.entries[k]; !ok {
			diff = append(diff, k)
		}
	}

	slices.Sort(diff)
	return diff
}
