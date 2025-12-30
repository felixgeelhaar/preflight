package sync

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSyncMetadata(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	vector := NewVersionVector().Increment(machineID)

	meta := NewSyncMetadata(vector)

	assert.False(t, meta.IsZero())
	assert.True(t, meta.Vector().Equals(vector))
	assert.Empty(t, meta.Lineage())
}

func TestSyncMetadata_IsZero(t *testing.T) {
	t.Parallel()

	var zeroMeta SyncMetadata
	assert.True(t, zeroMeta.IsZero())

	nonZeroMeta := NewSyncMetadata(NewVersionVector())
	assert.False(t, nonZeroMeta.IsZero())
}

func TestSyncMetadata_WithVector(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	initialVector := NewVersionVector()
	meta := NewSyncMetadata(initialVector)

	newVector := initialVector.Increment(machineID)
	newMeta := meta.WithVector(newVector)

	// Original is unchanged
	assert.True(t, meta.Vector().IsEmpty())
	// New has updated vector
	assert.True(t, newMeta.Vector().Equals(newVector))
}

func TestMachineLineage(t *testing.T) {
	t.Parallel()

	machineID := "550e8400-e29b-41d4-a716-446655440000"
	hostname := "work-macbook"
	lastSeen := time.Now().UTC().Truncate(time.Second)

	lineage := NewMachineLineage(machineID, hostname, lastSeen)

	assert.Equal(t, machineID, lineage.MachineID())
	assert.Equal(t, hostname, lineage.Hostname())
	assert.Equal(t, lastSeen, lineage.LastSeen())
	assert.False(t, lineage.IsZero())
}

func TestMachineLineage_IsZero(t *testing.T) {
	t.Parallel()

	var zeroLineage MachineLineage
	assert.True(t, zeroLineage.IsZero())

	nonZeroLineage := NewMachineLineage("id", "host", time.Now())
	assert.False(t, nonZeroLineage.IsZero())
}

func TestMachineLineage_WithLastSeen(t *testing.T) {
	t.Parallel()

	original := NewMachineLineage("id", "host", time.Now().Add(-time.Hour))
	newTime := time.Now().UTC()

	updated := original.WithLastSeen(newTime)

	// Original unchanged
	assert.NotEqual(t, newTime.Unix(), original.LastSeen().Unix())
	// Updated has new time
	assert.Equal(t, newTime.Unix(), updated.LastSeen().Unix())
}

func TestSyncMetadata_AddLineage(t *testing.T) {
	t.Parallel()

	meta := NewSyncMetadata(NewVersionVector())
	lineage := NewMachineLineage("machine-1", "laptop", time.Now())

	newMeta := meta.AddLineage(lineage)

	// Original unchanged
	assert.Empty(t, meta.Lineage())
	// New has lineage
	assert.Len(t, newMeta.Lineage(), 1)
	assert.Equal(t, "laptop", newMeta.Lineage()["machine-1"].Hostname())
}

func TestSyncMetadata_GetLineage(t *testing.T) {
	t.Parallel()

	meta := NewSyncMetadata(NewVersionVector())
	lineage := NewMachineLineage("machine-1", "laptop", time.Now())
	meta = meta.AddLineage(lineage)

	// Found
	found, ok := meta.GetLineage("machine-1")
	assert.True(t, ok)
	assert.Equal(t, "laptop", found.Hostname())

	// Not found
	_, ok = meta.GetLineage("machine-2")
	assert.False(t, ok)
}

func TestSyncMetadata_UpdateLineage(t *testing.T) {
	t.Parallel()

	meta := NewSyncMetadata(NewVersionVector())
	lineage1 := NewMachineLineage("machine-1", "laptop", time.Now().Add(-time.Hour))
	meta = meta.AddLineage(lineage1)

	// Update existing
	lineage2 := NewMachineLineage("machine-1", "laptop-new", time.Now())
	newMeta := meta.UpdateLineage(lineage2)

	assert.Equal(t, "laptop-new", newMeta.Lineage()["machine-1"].Hostname())
}

func TestSyncMetadata_RecordActivity(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	hostname := "work-macbook"

	meta := NewSyncMetadata(NewVersionVector())
	newMeta := meta.RecordActivity(machineID, hostname)

	// Vector incremented
	assert.Equal(t, uint64(1), newMeta.Vector().GetByMachineID(machineID))

	// Lineage added/updated
	lineage, ok := newMeta.GetLineage(machineID.String())
	assert.True(t, ok)
	assert.Equal(t, hostname, lineage.Hostname())
}

func TestSyncMetadata_RecordActivity_UpdatesExisting(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	hostname := "work-macbook"

	meta := NewSyncMetadata(NewVersionVector())
	meta = meta.RecordActivity(machineID, hostname)
	meta = meta.RecordActivity(machineID, hostname)

	// Vector incremented twice
	assert.Equal(t, uint64(2), meta.Vector().GetByMachineID(machineID))

	// Lineage still present with updated time
	lineage, ok := meta.GetLineage(machineID.String())
	assert.True(t, ok)
	assert.Equal(t, hostname, lineage.Hostname())
}

func TestSyncMetadata_Merge(t *testing.T) {
	t.Parallel()

	machineA, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	metaA := NewSyncMetadata(NewVersionVector()).RecordActivity(machineA, "laptop-a")
	metaB := NewSyncMetadata(NewVersionVector()).RecordActivity(machineB, "laptop-b")

	merged := metaA.Merge(metaB)

	// Vector merged
	assert.Equal(t, uint64(1), merged.Vector().GetByMachineID(machineA))
	assert.Equal(t, uint64(1), merged.Vector().GetByMachineID(machineB))

	// Lineage merged
	assert.Len(t, merged.Lineage(), 2)
	lineageA, ok := merged.GetLineage(machineA.String())
	assert.True(t, ok)
	assert.Equal(t, "laptop-a", lineageA.Hostname())
}

func TestSyncMetadata_JSON(t *testing.T) {
	t.Parallel()

	machineID, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	lastSeen := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	meta := NewSyncMetadata(NewVersionVector().Increment(machineID))
	meta = meta.AddLineage(NewMachineLineage(machineID.String(), "work-macbook", lastSeen))

	// Marshal
	data, err := json.Marshal(meta)
	require.NoError(t, err)

	// Unmarshal
	var loaded SyncMetadata
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, uint64(1), loaded.Vector().Get(machineID.String()))
	lineage, ok := loaded.GetLineage(machineID.String())
	assert.True(t, ok)
	assert.Equal(t, "work-macbook", lineage.Hostname())
	assert.True(t, lastSeen.Equal(lineage.LastSeen()))
}

func TestMachineLineage_JSON(t *testing.T) {
	t.Parallel()

	lastSeen := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	lineage := NewMachineLineage("machine-1", "laptop", lastSeen)

	// Marshal
	data, err := json.Marshal(lineage)
	require.NoError(t, err)

	// Unmarshal
	var loaded MachineLineage
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, "machine-1", loaded.MachineID())
	assert.Equal(t, "laptop", loaded.Hostname())
	assert.True(t, lastSeen.Equal(loaded.LastSeen()))
}
