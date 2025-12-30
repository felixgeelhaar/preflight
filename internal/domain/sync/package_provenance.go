package sync

import (
	"encoding/json"
	"fmt"
)

// PackageProvenance tracks who modified a package and when (in terms of vector time).
// This enables detecting concurrent modifications across machines.
type PackageProvenance struct {
	modifiedBy     string        // Machine ID that last modified this package
	vectorAtChange VersionVector // Version vector at time of change
}

// NewPackageProvenance creates a new PackageProvenance.
func NewPackageProvenance(modifiedBy MachineID, vectorAtChange VersionVector) PackageProvenance {
	return PackageProvenance{
		modifiedBy:     modifiedBy.String(),
		vectorAtChange: vectorAtChange,
	}
}

// ModifiedBy returns the machine ID that last modified this package.
func (p PackageProvenance) ModifiedBy() string {
	return p.modifiedBy
}

// VectorAtChange returns the version vector at the time of the change.
func (p PackageProvenance) VectorAtChange() VersionVector {
	return p.vectorAtChange
}

// IsZero returns true if this is a zero-value PackageProvenance.
func (p PackageProvenance) IsZero() bool {
	return p.modifiedBy == "" && p.vectorAtChange.IsEmpty()
}

// WithModifiedBy returns a new PackageProvenance with an updated modifier.
func (p PackageProvenance) WithModifiedBy(machineID MachineID) PackageProvenance {
	return PackageProvenance{
		modifiedBy:     machineID.String(),
		vectorAtChange: p.vectorAtChange,
	}
}

// WithVectorAtChange returns a new PackageProvenance with an updated vector.
func (p PackageProvenance) WithVectorAtChange(vector VersionVector) PackageProvenance {
	return PackageProvenance{
		modifiedBy:     p.modifiedBy,
		vectorAtChange: vector,
	}
}

// IsConcurrentWith returns true if this provenance is concurrent with another.
// Two changes are concurrent if neither happened-before the other.
func (p PackageProvenance) IsConcurrentWith(other PackageProvenance) bool {
	return p.vectorAtChange.IsConcurrentWith(other.vectorAtChange)
}

// HappenedBefore returns true if this change happened before another.
func (p PackageProvenance) HappenedBefore(other PackageProvenance) bool {
	return p.vectorAtChange.Compare(other.vectorAtChange) == Before
}

// HappenedAfter returns true if this change happened after another.
func (p PackageProvenance) HappenedAfter(other PackageProvenance) bool {
	return p.vectorAtChange.Compare(other.vectorAtChange) == After
}

// String returns a human-readable representation.
func (p PackageProvenance) String() string {
	shortID := p.modifiedBy
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return fmt.Sprintf("modified by %s at %s", shortID, p.vectorAtChange.String())
}

// MarshalJSON implements json.Marshaler.
func (p PackageProvenance) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ModifiedBy     string        `json:"modified_by"`
		VectorAtChange VersionVector `json:"vector_at_change"`
	}{
		ModifiedBy:     p.modifiedBy,
		VectorAtChange: p.vectorAtChange,
	})
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *PackageProvenance) UnmarshalJSON(data []byte) error {
	var aux struct {
		ModifiedBy     string        `json:"modified_by"`
		VectorAtChange VersionVector `json:"vector_at_change"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.modifiedBy = aux.ModifiedBy
	p.vectorAtChange = aux.VectorAtChange
	return nil
}
