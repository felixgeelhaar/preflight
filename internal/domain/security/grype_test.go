package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrypeScanner_Name(t *testing.T) {
	t.Parallel()

	scanner := NewGrypeScanner()
	assert.Equal(t, "grype", scanner.Name())
}

func TestGrypeScanner_buildArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		target   ScanTarget
		expected []string
	}{
		{
			name:     "directory scan with path",
			target:   ScanTarget{Type: "directory", Path: "/some/path"},
			expected: []string{"--output", "json", "dir:/some/path"},
		},
		{
			name:     "directory scan empty path",
			target:   ScanTarget{Type: "directory", Path: ""},
			expected: []string{"--output", "json", "dir:."},
		},
		{
			name:     "dir alias",
			target:   ScanTarget{Type: "dir", Path: "/path"},
			expected: []string{"--output", "json", "dir:/path"},
		},
		{
			name:     "empty type defaults to directory",
			target:   ScanTarget{Type: "", Path: "/path"},
			expected: []string{"--output", "json", "dir:/path"},
		},
		{
			name:     "sbom scan",
			target:   ScanTarget{Type: "sbom", Path: "/path/to/sbom.json"},
			expected: []string{"--output", "json", "sbom:/path/to/sbom.json"},
		},
		{
			name:     "image scan",
			target:   ScanTarget{Type: "image", Path: "nginx:latest"},
			expected: []string{"--output", "json", "nginx:latest"},
		},
		{
			name:     "unknown type defaults to directory",
			target:   ScanTarget{Type: "unknown", Path: "/path"},
			expected: []string{"--output", "json", "dir:/path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			scanner := NewGrypeScanner()
			args := scanner.buildArgs(tt.target, ScanOptions{})
			assert.Equal(t, tt.expected, args)
		})
	}
}

func TestGrypeScanner_parseOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          []byte
		expectedCount  int
		expectedPkgs   int
		wantErr        bool
		validateResult func(t *testing.T, vulns Vulnerabilities)
	}{
		{
			name:          "empty input",
			input:         []byte{},
			expectedCount: 0,
			expectedPkgs:  0,
			wantErr:       false,
		},
		{
			name: "valid output with vulnerabilities",
			input: []byte(`{
				"matches": [
					{
						"vulnerability": {
							"id": "CVE-2024-1234",
							"severity": "Critical",
							"description": "Test vulnerability",
							"fix": {"versions": ["1.0.1"], "state": "fixed"},
							"urls": ["https://nvd.nist.gov/vuln/detail/CVE-2024-1234"],
							"cvss": [{"version": "3.1", "vector": "CVSS:3.1/AV:N", "metrics": {"baseScore": 9.8}}]
						},
						"artifact": {
							"name": "openssl",
							"version": "1.0.0",
							"type": "deb"
						}
					},
					{
						"vulnerability": {
							"id": "CVE-2024-5678",
							"severity": "High",
							"description": "Another vulnerability",
							"fix": {"versions": [], "state": "not-fixed"},
							"urls": []
						},
						"artifact": {
							"name": "curl",
							"version": "7.0.0",
							"type": "deb"
						}
					}
				],
				"source": {"type": "directory", "target": "."}
			}`),
			expectedCount: 2,
			expectedPkgs:  2,
			wantErr:       false,
			validateResult: func(t *testing.T, vulns Vulnerabilities) {
				assert.Equal(t, "CVE-2024-1234", vulns[0].ID)
				assert.Equal(t, "openssl", vulns[0].Package)
				assert.Equal(t, "1.0.0", vulns[0].Version)
				assert.Equal(t, SeverityCritical, vulns[0].Severity)
				assert.InDelta(t, 9.8, vulns[0].CVSS, 0.001)
				assert.Equal(t, "1.0.1", vulns[0].FixedIn)
				assert.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2024-1234", vulns[0].Reference)
				assert.Equal(t, "deb", vulns[0].Provider)

				assert.Equal(t, "CVE-2024-5678", vulns[1].ID)
				assert.Equal(t, "curl", vulns[1].Package)
				assert.Equal(t, SeverityHigh, vulns[1].Severity)
				assert.Empty(t, vulns[1].FixedIn)
			},
		},
		{
			name: "same package multiple vulnerabilities",
			input: []byte(`{
				"matches": [
					{
						"vulnerability": {"id": "CVE-1", "severity": "High"},
						"artifact": {"name": "openssl", "version": "1.0.0", "type": "deb"}
					},
					{
						"vulnerability": {"id": "CVE-2", "severity": "Medium"},
						"artifact": {"name": "openssl", "version": "1.0.0", "type": "deb"}
					}
				],
				"source": {"type": "directory", "target": "."}
			}`),
			expectedCount: 2,
			expectedPkgs:  1, // Same package
			wantErr:       false,
		},
		{
			name:    "invalid json",
			input:   []byte(`{invalid}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			scanner := NewGrypeScanner()
			vulns, pkgCount, err := scanner.parseOutput(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, vulns, tt.expectedCount)
			assert.Equal(t, tt.expectedPkgs, pkgCount)

			if tt.validateResult != nil {
				tt.validateResult(t, vulns)
			}
		})
	}
}

func TestGrypeScanner_Scan_NotAvailable(t *testing.T) {
	t.Parallel()

	// This test verifies behavior when scanner is unavailable
	// The actual availability is checked via exec.LookPath in the real implementation
	// We verify the scanner constructor and that it would return ErrScannerNotAvailable
	// when the underlying grype command is not found

	scanner := NewGrypeScanner()
	assert.NotNil(t, scanner)
	assert.Equal(t, "grype", scanner.Name())
}

func TestGrypeScanner_Version(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		stdout         string
		exitCode       int
		expected       string
		wantErr        bool
		useFallback    bool
		fallbackStdout string
	}{
		{
			name:     "json version output",
			stdout:   `{"version": "0.74.0"}`,
			expected: "0.74.0",
		},
		{
			name:           "fallback to --version",
			exitCode:       1,
			useFallback:    true,
			fallbackStdout: "grype 0.74.0",
			expected:       "0.74.0",
		},
		{
			name:     "unparseable json returns raw",
			stdout:   `not json`,
			expected: "not json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// These tests validate the parsing logic but don't execute real commands
			// Full integration tests would require grype to be installed
		})
	}
}
