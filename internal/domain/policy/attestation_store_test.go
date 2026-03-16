package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAttestationStore(t *testing.T) {
	t.Parallel()

	store := NewAttestationStore("/tmp/attestations")
	assert.NotNil(t, store)
}

func TestAttestationStore_Save(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAttestationStore(dir)

	report := &ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			ComplianceScore: 100.0,
		},
	}

	att, err := NewComplianceAttestation(report, "machine-001", "dev-laptop")
	require.NoError(t, err)

	err = store.Save(att)
	require.NoError(t, err)

	// Verify file was created
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)

	// Verify file permissions
	info, err := entries[0].Info()
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Verify file content is valid JSON
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	require.NoError(t, err)

	var loaded ComplianceAttestation
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)
	assert.Equal(t, "machine-001", loaded.MachineID)
	assert.Equal(t, "dev-laptop", loaded.Hostname)
}

func TestAttestationStore_Save_CreatesDirectory(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nested", "attestations")
	store := NewAttestationStore(dir)

	report := &ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			ComplianceScore: 100.0,
		},
	}

	att, err := NewComplianceAttestation(report, "machine-001", "dev-laptop")
	require.NoError(t, err)

	err = store.Save(att)
	require.NoError(t, err)

	// Directory should have been created
	_, err = os.Stat(dir)
	assert.NoError(t, err)
}

func TestAttestationStore_Load(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAttestationStore(dir)

	report := &ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			ComplianceScore: 100.0,
		},
	}

	// Save two attestations for same machine
	att1, err := NewComplianceAttestation(report, "machine-001", "dev-laptop")
	require.NoError(t, err)
	att1.AttestedAt = time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	att1.ContentDigest = att1.Digest()
	err = store.Save(att1)
	require.NoError(t, err)

	att2, err := NewComplianceAttestation(report, "machine-001", "dev-laptop")
	require.NoError(t, err)
	att2.AttestedAt = time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC)
	att2.ContentDigest = att2.Digest()
	err = store.Save(att2)
	require.NoError(t, err)

	// Save one for a different machine
	att3, err := NewComplianceAttestation(report, "machine-002", "staging-box")
	require.NoError(t, err)
	err = store.Save(att3)
	require.NoError(t, err)

	// Load should return only machine-001 attestations
	loaded, err := store.Load("machine-001")
	require.NoError(t, err)
	assert.Len(t, loaded, 2)

	for _, a := range loaded {
		assert.Equal(t, "machine-001", a.MachineID)
	}
}

func TestAttestationStore_Load_NoAttestations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAttestationStore(dir)

	loaded, err := store.Load("nonexistent-machine")
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestAttestationStore_LoadLatest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAttestationStore(dir)

	report := &ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			ComplianceScore: 100.0,
		},
	}

	// Save older attestation
	att1, err := NewComplianceAttestation(report, "machine-001", "dev-laptop")
	require.NoError(t, err)
	att1.AttestedAt = time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	att1.ContentDigest = att1.Digest()
	err = store.Save(att1)
	require.NoError(t, err)

	// Save newer attestation
	att2, err := NewComplianceAttestation(report, "machine-001", "dev-laptop")
	require.NoError(t, err)
	att2.AttestedAt = time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	att2.ContentDigest = att2.Digest()
	err = store.Save(att2)
	require.NoError(t, err)

	latest, err := store.LoadLatest("machine-001")
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, att2.AttestedAt.Unix(), latest.AttestedAt.Unix())
}

func TestAttestationStore_LoadLatest_NoAttestations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAttestationStore(dir)

	latest, err := store.LoadLatest("nonexistent-machine")
	assert.NoError(t, err)
	assert.Nil(t, latest)
}

func TestAttestationStore_List(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAttestationStore(dir)

	report := &ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			ComplianceScore: 100.0,
		},
	}

	// Save attestations for different machines
	for _, machineID := range []string{"machine-001", "machine-002", "machine-001"} {
		att, err := NewComplianceAttestation(report, machineID, "host")
		require.NoError(t, err)
		err = store.Save(att)
		require.NoError(t, err)
	}

	machineIDs, err := store.List()
	require.NoError(t, err)

	// Should return unique machine IDs
	assert.Len(t, machineIDs, 2)
	assert.Contains(t, machineIDs, "machine-001")
	assert.Contains(t, machineIDs, "machine-002")
}

func TestAttestationStore_List_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAttestationStore(dir)

	machineIDs, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, machineIDs)
}

func TestAttestationStore_Save_FileNameFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewAttestationStore(dir)

	report := &ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			ComplianceScore: 100.0,
		},
	}

	att, err := NewComplianceAttestation(report, "machine-001", "dev-laptop")
	require.NoError(t, err)

	err = store.Save(att)
	require.NoError(t, err)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	name := entries[0].Name()
	assert.Contains(t, name, "machine-001_")
	assert.Contains(t, name, ".json")
}
