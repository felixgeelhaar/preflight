package sync

import (
	"encoding/json"
	"maps"
	"time"
)

// MachineLineage tracks information about a machine that has participated in sync.
// This helps identify which machines have contributed to the lockfile and when.
type MachineLineage struct {
	machineID string
	hostname  string
	lastSeen  time.Time
}

// NewMachineLineage creates a new machine lineage entry.
func NewMachineLineage(machineID, hostname string, lastSeen time.Time) MachineLineage {
	return MachineLineage{
		machineID: machineID,
		hostname:  hostname,
		lastSeen:  lastSeen,
	}
}

// MachineID returns the machine's unique identifier.
func (m MachineLineage) MachineID() string {
	return m.machineID
}

// Hostname returns the machine's hostname at last sync.
func (m MachineLineage) Hostname() string {
	return m.hostname
}

// LastSeen returns when this machine last contributed to the lockfile.
func (m MachineLineage) LastSeen() time.Time {
	return m.lastSeen
}

// IsZero returns true if this is a zero-value MachineLineage.
func (m MachineLineage) IsZero() bool {
	return m.machineID == "" && m.hostname == "" && m.lastSeen.IsZero()
}

// WithLastSeen returns a new MachineLineage with updated last seen time.
func (m MachineLineage) WithLastSeen(t time.Time) MachineLineage {
	return MachineLineage{
		machineID: m.machineID,
		hostname:  m.hostname,
		lastSeen:  t,
	}
}

// MarshalJSON implements json.Marshaler.
func (m MachineLineage) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		MachineID string    `json:"machine_id"`
		Hostname  string    `json:"hostname"`
		LastSeen  time.Time `json:"last_seen"`
	}{
		MachineID: m.machineID,
		Hostname:  m.hostname,
		LastSeen:  m.lastSeen,
	})
}

// UnmarshalJSON implements json.Unmarshaler.
func (m *MachineLineage) UnmarshalJSON(data []byte) error {
	var aux struct {
		MachineID string    `json:"machine_id"`
		Hostname  string    `json:"hostname"`
		LastSeen  time.Time `json:"last_seen"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.machineID = aux.MachineID
	m.hostname = aux.Hostname
	m.lastSeen = aux.LastSeen
	return nil
}

// SyncMetadata contains multi-machine synchronization metadata for a lockfile.
// It tracks the version vector for causal ordering and lineage information
// about all machines that have contributed to the lockfile.
//
//nolint:revive // "SyncMetadata" is clearer than just "Metadata" in multi-package context
type SyncMetadata struct {
	vector  VersionVector
	lineage map[string]MachineLineage
	// initialized tracks if this is a real SyncMetadata or zero value
	initialized bool
}

// NewSyncMetadata creates a new SyncMetadata with the given version vector.
func NewSyncMetadata(vector VersionVector) SyncMetadata {
	return SyncMetadata{
		vector:      vector,
		lineage:     make(map[string]MachineLineage),
		initialized: true,
	}
}

// Vector returns the version vector for causal ordering.
func (s SyncMetadata) Vector() VersionVector {
	return s.vector
}

// Lineage returns a copy of the machine lineage map.
func (s SyncMetadata) Lineage() map[string]MachineLineage {
	return maps.Clone(s.lineage)
}

// IsZero returns true if this is a zero-value SyncMetadata.
func (s SyncMetadata) IsZero() bool {
	return !s.initialized
}

// WithVector returns a new SyncMetadata with an updated version vector.
func (s SyncMetadata) WithVector(vector VersionVector) SyncMetadata {
	return SyncMetadata{
		vector:      vector,
		lineage:     maps.Clone(s.lineage),
		initialized: true,
	}
}

// AddLineage adds a machine lineage entry.
// Returns a new SyncMetadata with the lineage added.
func (s SyncMetadata) AddLineage(lineage MachineLineage) SyncMetadata {
	newLineage := maps.Clone(s.lineage)
	if newLineage == nil {
		newLineage = make(map[string]MachineLineage)
	}
	newLineage[lineage.MachineID()] = lineage
	return SyncMetadata{
		vector:      s.vector,
		lineage:     newLineage,
		initialized: true,
	}
}

// UpdateLineage updates an existing machine lineage entry.
// If the machine doesn't exist, it adds it.
// Returns a new SyncMetadata.
func (s SyncMetadata) UpdateLineage(lineage MachineLineage) SyncMetadata {
	return s.AddLineage(lineage)
}

// GetLineage returns the lineage for a machine ID.
// Returns the lineage and true if found, zero value and false otherwise.
func (s SyncMetadata) GetLineage(machineID string) (MachineLineage, bool) {
	lineage, ok := s.lineage[machineID]
	return lineage, ok
}

// RecordActivity increments the version vector for the given machine
// and updates its lineage with the current timestamp.
// This is the primary method for recording local changes.
func (s SyncMetadata) RecordActivity(machineID MachineID, hostname string) SyncMetadata {
	newVector := s.vector.Increment(machineID)
	newLineage := NewMachineLineage(machineID.String(), hostname, time.Now().UTC())
	return SyncMetadata{
		vector:      newVector,
		lineage:     maps.Clone(s.lineage),
		initialized: true,
	}.AddLineage(newLineage)
}

// Merge combines two SyncMetadata by merging their version vectors
// and lineage information. For lineage, the entry with the later
// LastSeen time wins.
func (s SyncMetadata) Merge(other SyncMetadata) SyncMetadata {
	// Merge vectors
	mergedVector := s.vector.Merge(other.vector)

	// Merge lineage (later LastSeen wins)
	mergedLineage := make(map[string]MachineLineage)
	for k, v := range s.lineage {
		mergedLineage[k] = v
	}
	for k, v := range other.lineage {
		if existing, ok := mergedLineage[k]; ok {
			// Take the one with later LastSeen
			if v.LastSeen().After(existing.LastSeen()) {
				mergedLineage[k] = v
			}
		} else {
			mergedLineage[k] = v
		}
	}

	return SyncMetadata{
		vector:      mergedVector,
		lineage:     mergedLineage,
		initialized: true,
	}
}

// MarshalJSON implements json.Marshaler.
func (s SyncMetadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Vector  VersionVector             `json:"vector"`
		Lineage map[string]MachineLineage `json:"lineage"`
	}{
		Vector:  s.vector,
		Lineage: s.lineage,
	})
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *SyncMetadata) UnmarshalJSON(data []byte) error {
	var aux struct {
		Vector  VersionVector             `json:"vector"`
		Lineage map[string]MachineLineage `json:"lineage"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.vector = aux.Vector
	s.lineage = aux.Lineage
	if s.lineage == nil {
		s.lineage = make(map[string]MachineLineage)
	}
	s.initialized = true
	return nil
}
