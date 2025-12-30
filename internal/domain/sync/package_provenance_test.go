package sync

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPackageProvenance(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineID)

	prov := NewPackageProvenance(machineID, vector)

	assert.Equal(t, machineID.String(), prov.ModifiedBy())
	assert.True(t, prov.VectorAtChange().Equals(vector))
	assert.False(t, prov.IsZero())
}

func TestPackageProvenance_IsZero(t *testing.T) {
	t.Parallel()

	var zeroProv PackageProvenance
	assert.True(t, zeroProv.IsZero())

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	nonZeroProv := NewPackageProvenance(machineID, NewVersionVector())
	assert.False(t, nonZeroProv.IsZero())
}

func TestPackageProvenance_WithModifiedBy(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	original := NewPackageProvenance(machineA, NewVersionVector().Increment(machineA))
	updated := original.WithModifiedBy(machineB)

	// Original unchanged
	assert.Equal(t, machineA.String(), original.ModifiedBy())
	// Updated has new modifier
	assert.Equal(t, machineB.String(), updated.ModifiedBy())
}

func TestPackageProvenance_WithVectorAtChange(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector1 := NewVersionVector().Increment(machineID)
	vector2 := vector1.Increment(machineID)

	original := NewPackageProvenance(machineID, vector1)
	updated := original.WithVectorAtChange(vector2)

	// Original unchanged
	assert.Equal(t, uint64(1), original.VectorAtChange().GetByMachineID(machineID))
	// Updated has new vector
	assert.Equal(t, uint64(2), updated.VectorAtChange().GetByMachineID(machineID))
}

func TestPackageProvenance_IsConcurrentWith(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	// Create two provenance records that are concurrent
	vectorA := NewVersionVector().Increment(machineA)
	vectorB := NewVersionVector().Increment(machineB)

	provA := NewPackageProvenance(machineA, vectorA)
	provB := NewPackageProvenance(machineB, vectorB)

	assert.True(t, provA.IsConcurrentWith(provB))
	assert.True(t, provB.IsConcurrentWith(provA))
}

func TestPackageProvenance_HappenedBefore(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")

	vector1 := NewVersionVector().Increment(machineA)
	vector2 := vector1.Increment(machineA)

	provOlder := NewPackageProvenance(machineA, vector1)
	provNewer := NewPackageProvenance(machineA, vector2)

	assert.True(t, provOlder.HappenedBefore(provNewer))
	assert.False(t, provNewer.HappenedBefore(provOlder))
}

func TestPackageProvenance_HappenedAfter(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")

	vector1 := NewVersionVector().Increment(machineA)
	vector2 := vector1.Increment(machineA)

	provOlder := NewPackageProvenance(machineA, vector1)
	provNewer := NewPackageProvenance(machineA, vector2)

	assert.False(t, provOlder.HappenedAfter(provNewer))
	assert.True(t, provNewer.HappenedAfter(provOlder))
}

func TestPackageProvenance_JSON(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineID)
	prov := NewPackageProvenance(machineID, vector)

	// Marshal
	data, err := json.Marshal(prov)
	require.NoError(t, err)

	// Unmarshal
	var loaded PackageProvenance
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, machineID.String(), loaded.ModifiedBy())
	assert.Equal(t, uint64(1), loaded.VectorAtChange().Get(machineID.String()))
}

func TestPackageProvenance_String(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineID)
	prov := NewPackageProvenance(machineID, vector)

	s := prov.String()
	assert.Contains(t, s, "550e8400")
}
