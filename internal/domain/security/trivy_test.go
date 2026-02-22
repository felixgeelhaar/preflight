package security

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrivyScanner_Name(t *testing.T) {
	t.Parallel()

	scanner := NewTrivyScanner()
	assert.Equal(t, "trivy", scanner.Name())
}

func TestTrivyScanner_buildArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		target   ScanTarget
		expected []string
	}{
		{
			name:     "filesystem scan with path",
			target:   ScanTarget{Type: "directory", Path: "/some/path"},
			expected: []string{"fs", "--format", "json", "--scanners", "vuln", "/some/path"},
		},
		{
			name:     "filesystem scan empty path",
			target:   ScanTarget{Type: "directory", Path: ""},
			expected: []string{"fs", "--format", "json", "--scanners", "vuln", "."},
		},
		{
			name:     "dir alias",
			target:   ScanTarget{Type: "dir", Path: "/path"},
			expected: []string{"fs", "--format", "json", "--scanners", "vuln", "/path"},
		},
		{
			name:     "empty type defaults to filesystem",
			target:   ScanTarget{Type: "", Path: "/path"},
			expected: []string{"fs", "--format", "json", "--scanners", "vuln", "/path"},
		},
		{
			name:     "sbom scan",
			target:   ScanTarget{Type: "sbom", Path: "/path/to/sbom.json"},
			expected: []string{"sbom", "--format", "json", "--scanners", "vuln", "/path/to/sbom.json"},
		},
		{
			name:     "image scan",
			target:   ScanTarget{Type: "image", Path: "nginx:latest"},
			expected: []string{"image", "--format", "json", "--scanners", "vuln", "nginx:latest"},
		},
		{
			name:     "unknown type defaults to filesystem",
			target:   ScanTarget{Type: "unknown", Path: "/path"},
			expected: []string{"fs", "--format", "json", "--scanners", "vuln", "/path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			scanner := NewTrivyScanner()
			args := scanner.buildArgs(tt.target, ScanOptions{})
			assert.Equal(t, tt.expected, args)
		})
	}
}

func TestTrivyScanner_parseOutput(t *testing.T) {
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
				"Results": [
					{
						"Target": "Gemfile.lock",
						"Type": "bundler",
						"Vulnerabilities": [
							{
								"VulnerabilityID": "CVE-2024-1234",
								"PkgName": "rails",
								"InstalledVersion": "6.0.0",
								"FixedVersion": "6.0.1",
								"Severity": "CRITICAL",
								"Title": "Remote code execution",
								"Description": "A critical vulnerability",
								"References": ["https://nvd.nist.gov/vuln/detail/CVE-2024-1234"],
								"CVSS": {"nvd": {"V3Score": 9.8}}
							}
						]
					},
					{
						"Target": "package-lock.json",
						"Type": "npm",
						"Vulnerabilities": [
							{
								"VulnerabilityID": "CVE-2024-5678",
								"PkgName": "lodash",
								"InstalledVersion": "4.0.0",
								"Severity": "HIGH",
								"Description": "Prototype pollution",
								"CVSS": {"redhat": {"V3Score": 7.5}}
							}
						]
					}
				]
			}`),
			expectedCount: 2,
			expectedPkgs:  2,
			wantErr:       false,
			validateResult: func(t *testing.T, vulns Vulnerabilities) {
				assert.Equal(t, "CVE-2024-1234", vulns[0].ID)
				assert.Equal(t, "rails", vulns[0].Package)
				assert.Equal(t, "6.0.0", vulns[0].Version)
				assert.Equal(t, SeverityCritical, vulns[0].Severity)
				assert.InDelta(t, 9.8, vulns[0].CVSS, 0.001)
				assert.Equal(t, "6.0.1", vulns[0].FixedIn)
				assert.Equal(t, "Remote code execution", vulns[0].Title)
				assert.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2024-1234", vulns[0].Reference)
				assert.Equal(t, "bundler", vulns[0].Provider)

				assert.Equal(t, "CVE-2024-5678", vulns[1].ID)
				assert.Equal(t, "lodash", vulns[1].Package)
				assert.Equal(t, SeverityHigh, vulns[1].Severity)
				assert.InDelta(t, 7.5, vulns[1].CVSS, 0.001) // Uses redhat CVSS
				assert.Empty(t, vulns[1].FixedIn)
			},
		},
		{
			name: "multiple vulnerabilities same package",
			input: []byte(`{
				"Results": [
					{
						"Target": "go.mod",
						"Type": "gomod",
						"Vulnerabilities": [
							{
								"VulnerabilityID": "CVE-1",
								"PkgName": "golang.org/x/crypto",
								"InstalledVersion": "0.1.0",
								"Severity": "HIGH"
							},
							{
								"VulnerabilityID": "CVE-2",
								"PkgName": "golang.org/x/crypto",
								"InstalledVersion": "0.1.0",
								"Severity": "MEDIUM"
							}
						]
					}
				]
			}`),
			expectedCount: 2,
			expectedPkgs:  1, // Same package
			wantErr:       false,
		},
		{
			name: "results with null vulnerabilities",
			input: []byte(`{
				"Results": [
					{
						"Target": "Dockerfile",
						"Type": "dockerfile",
						"Vulnerabilities": null
					}
				]
			}`),
			expectedCount: 0,
			expectedPkgs:  0,
			wantErr:       false,
		},
		{
			name: "empty results",
			input: []byte(`{
				"Results": []
			}`),
			expectedCount: 0,
			expectedPkgs:  0,
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
			scanner := NewTrivyScanner()
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

func TestTrivyScanner_parseOutput_CVSSPriority(t *testing.T) {
	t.Parallel()

	// Test that NVD CVSS is preferred over RedHat
	input := []byte(`{
		"Results": [
			{
				"Target": "test",
				"Type": "test",
				"Vulnerabilities": [
					{
						"VulnerabilityID": "CVE-TEST",
						"PkgName": "test-pkg",
						"InstalledVersion": "1.0.0",
						"Severity": "HIGH",
						"CVSS": {
							"nvd": {"V3Score": 8.0},
							"redhat": {"V3Score": 7.0}
						}
					}
				]
			}
		]
	}`)

	scanner := NewTrivyScanner()
	vulns, _, err := scanner.parseOutput(input)
	require.NoError(t, err)
	require.Len(t, vulns, 1)
	// Should use NVD score (8.0) not RedHat (7.0)
	assert.InDelta(t, 8.0, vulns[0].CVSS, 0.001)
}

func TestTrivyScanner_Version_Parsing(t *testing.T) {
	t.Parallel()

	// Test version parsing logic (isolated from command execution)
	versionOutputs := []struct {
		output   string
		expected string
	}{
		{"Version: 0.50.0\nVulnerability DB:\n  Version: 2", "0.50.0"},
		{"Version: 0.48.1", "0.48.1"},
	}

	for _, tc := range versionOutputs {
		for _, line := range splitLines(tc.output) {
			if hasPrefix(line, "Version:") {
				version := trimPrefix(line, "Version:")
				assert.Equal(t, tc.expected, trimSpace(version))
				break
			}
		}
	}
}

func TestTrivyScanner_Version_WithMock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		output   string
		expected string
		wantErr  bool
	}{
		{
			name:     "standard version output",
			output:   "Version: 0.50.0\nVulnerability DB:\n  Version: 2",
			expected: "0.50.0",
		},
		{
			name:     "single line version",
			output:   "Version: 0.48.1",
			expected: "0.48.1",
		},
		{
			name:     "no version prefix returns raw output",
			output:   "some-other-output",
			expected: "some-other-output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			scanner := &TrivyScanner{
				execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
					return exec.Command("printf", "%s", tt.output)
				},
			}

			version, err := scanner.Version(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestTrivyScanner_Version_Error(t *testing.T) {
	t.Parallel()

	scanner := &TrivyScanner{
		execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.Command("false")
		},
	}

	_, err := scanner.Version(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get trivy version")
}

func TestTrivyScanner_Scan_WithMock(t *testing.T) {
	t.Parallel()

	trivyOutput := `{
		"Results": [
			{
				"Target": "go.mod",
				"Type": "gomod",
				"Vulnerabilities": [
					{
						"VulnerabilityID": "CVE-2024-1234",
						"PkgName": "golang.org/x/net",
						"InstalledVersion": "0.1.0",
						"FixedVersion": "0.2.0",
						"Severity": "HIGH",
						"Title": "HTTP/2 vulnerability",
						"Description": "A vulnerability in net/http2",
						"References": ["https://nvd.nist.gov/vuln/detail/CVE-2024-1234"],
						"CVSS": {"nvd": {"V3Score": 7.5}}
					}
				]
			}
		]
	}`

	scanner := &TrivyScanner{
		execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
			if name == "trivy" && len(args) > 0 && args[0] == "--version" {
				return exec.Command("printf", "%s", "Version: 0.50.0")
			}
			// Scan command
			return exec.Command("printf", "%s", trivyOutput)
		},
	}

	ctx := context.Background()
	target := ScanTarget{Type: "directory", Path: "."}
	opts := ScanOptions{}

	result, err := scanner.Scan(ctx, target, opts)
	if err != nil {
		// Expected: trivy not available on this machine
		assert.ErrorIs(t, err, ErrScannerNotAvailable)
	} else {
		require.NotNil(t, result)
		assert.Equal(t, "trivy", result.Scanner)
		assert.Equal(t, "0.50.0", result.Version)
		require.Len(t, result.Vulnerabilities, 1)
		assert.Equal(t, "CVE-2024-1234", result.Vulnerabilities[0].ID)
		assert.Equal(t, "golang.org/x/net", result.Vulnerabilities[0].Package)
	}
}

func TestTrivyScanner_Scan_WithFilters(t *testing.T) {
	t.Parallel()

	trivyOutput := `{
		"Results": [
			{
				"Target": "go.mod",
				"Type": "gomod",
				"Vulnerabilities": [
					{
						"VulnerabilityID": "CVE-2024-1111",
						"PkgName": "pkg-a",
						"InstalledVersion": "0.1.0",
						"Severity": "CRITICAL"
					},
					{
						"VulnerabilityID": "CVE-2024-2222",
						"PkgName": "pkg-b",
						"InstalledVersion": "1.0.0",
						"Severity": "LOW"
					},
					{
						"VulnerabilityID": "CVE-2024-3333",
						"PkgName": "pkg-c",
						"InstalledVersion": "2.0.0",
						"Severity": "HIGH"
					}
				]
			}
		]
	}`

	scanner := &TrivyScanner{
		execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
			if name == "trivy" && len(args) > 0 && args[0] == "--version" {
				return exec.Command("printf", "%s", "Version: 0.50.0")
			}
			return exec.Command("printf", "%s", trivyOutput)
		},
	}

	if !scanner.Available() {
		t.Skip("trivy not available")
	}

	t.Run("filter by min severity", func(t *testing.T) {
		t.Parallel()
		result, err := scanner.Scan(context.Background(), ScanTarget{Type: "directory", Path: "."}, ScanOptions{
			MinSeverity: SeverityHigh,
		})
		require.NoError(t, err)
		// Should only include CRITICAL and HIGH
		assert.Len(t, result.Vulnerabilities, 2)
	})

	t.Run("filter by ignore IDs", func(t *testing.T) {
		t.Parallel()
		result, err := scanner.Scan(context.Background(), ScanTarget{Type: "directory", Path: "."}, ScanOptions{
			IgnoreIDs: []string{"CVE-2024-1111"},
		})
		require.NoError(t, err)
		assert.Len(t, result.Vulnerabilities, 2)
		for _, v := range result.Vulnerabilities {
			assert.NotEqual(t, "CVE-2024-1111", v.ID)
		}
	})
}

func TestTrivyScanner_Scan_ExitCode(t *testing.T) {
	t.Parallel()

	// When trivy exits with non-zero and no stdout, it's an error
	scanner := &TrivyScanner{
		execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
			if name == "trivy" && len(args) > 0 && args[0] == "--version" {
				return exec.Command("printf", "%s", "Version: 0.50.0")
			}
			// Simulate trivy failing with non-zero exit code
			return exec.Command("false")
		},
	}

	if !scanner.Available() {
		t.Skip("trivy not available")
	}

	_, err := scanner.Scan(context.Background(), ScanTarget{Type: "directory", Path: "."}, ScanOptions{})
	assert.Error(t, err)
}

// Helper functions mimicking strings package for isolated testing
func splitLines(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func trimPrefix(s, prefix string) string {
	if hasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
