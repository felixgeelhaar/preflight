package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// AttestationStore persists attestations to a directory as JSON files.
type AttestationStore struct {
	basePath string
}

// NewAttestationStore creates a new store rooted at basePath.
func NewAttestationStore(basePath string) *AttestationStore {
	return &AttestationStore{basePath: basePath}
}

// Save persists an attestation as a JSON file with 0600 permissions.
// Files are named {machineID}_{timestamp}.json.
func (s *AttestationStore) Save(attestation *ComplianceAttestation) error {
	if err := os.MkdirAll(s.basePath, 0700); err != nil {
		return fmt.Errorf("creating attestation directory: %w", err)
	}

	data, err := json.MarshalIndent(attestation, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling attestation: %w", err)
	}

	timestamp := attestation.AttestedAt.UTC().Format("20060102T150405Z")
	filename := fmt.Sprintf("%s_%s.json", attestation.MachineID, timestamp)
	filePath := filepath.Join(s.basePath, filename)

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("writing attestation file: %w", err)
	}

	return nil
}

// Load returns all attestations for a given machine ID.
func (s *AttestationStore) Load(machineID string) ([]*ComplianceAttestation, error) {
	prefix := machineID + "_"
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading attestation directory: %w", err)
	}

	var attestations []*ComplianceAttestation
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		att, err := s.loadFile(filepath.Join(s.basePath, entry.Name()))
		if err != nil {
			continue // skip corrupted files
		}
		attestations = append(attestations, att)
	}

	return attestations, nil
}

// LoadLatest returns the most recent attestation for a machine ID, or nil if none exist.
func (s *AttestationStore) LoadLatest(machineID string) (*ComplianceAttestation, error) {
	attestations, err := s.Load(machineID)
	if err != nil {
		return nil, err
	}
	if len(attestations) == 0 {
		return nil, nil
	}

	sort.Slice(attestations, func(i, j int) bool {
		return attestations[i].AttestedAt.After(attestations[j].AttestedAt)
	})

	return attestations[0], nil
}

// List returns the unique machine IDs that have stored attestations.
func (s *AttestationStore) List() ([]string, error) {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading attestation directory: %w", err)
	}

	seen := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) >= 1 && parts[0] != "" {
			seen[parts[0]] = true
		}
	}

	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	return ids, nil
}

// loadFile reads and unmarshals a single attestation file.
func (s *AttestationStore) loadFile(path string) (*ComplianceAttestation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var att ComplianceAttestation
	if err := json.Unmarshal(data, &att); err != nil {
		return nil, err
	}

	return &att, nil
}
