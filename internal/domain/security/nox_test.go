package security

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoxScanner_Name(t *testing.T) {
	t.Parallel()

	scanner := NewNoxScanner()
	assert.Equal(t, "nox", scanner.Name())
}

func TestNoxScanner_Version_Success(t *testing.T) {
	t.Parallel()

	scanner := &NoxScanner{
		execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.Command("echo", "nox 0.7.0")
		},
	}

	version, err := scanner.Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "0.7.0", version)
}

func TestNoxScanner_Version_SingleWord(t *testing.T) {
	t.Parallel()

	scanner := &NoxScanner{
		execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.Command("echo", "0.7.0")
		},
	}

	version, err := scanner.Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "0.7.0", version)
}

func TestNoxScanner_Version_Failure(t *testing.T) {
	t.Parallel()

	scanner := &NoxScanner{
		execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.Command("false")
		},
	}

	_, err := scanner.Version(context.Background())
	assert.Error(t, err)
}

func TestNoxScanner_parseOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          []byte
		expectedCount  int
		wantErr        bool
		validateResult func(t *testing.T, vulns Vulnerabilities)
	}{
		{
			name:          "empty input",
			input:         []byte{},
			expectedCount: 0,
			wantErr:       false,
		},
		{
			name: "valid output with findings",
			input: []byte(`{
				"findings": [
					{
						"ID": "SEC-004:path/file.go:42",
						"RuleID": "SEC-004",
						"Severity": "critical",
						"Confidence": "high",
						"Location": {
							"FilePath": "path/file.go",
							"StartLine": 42
						},
						"Message": "Private key header detected",
						"Metadata": {
							"cwe": "CWE-798"
						}
					},
					{
						"ID": "SEC-002:main.go:10",
						"RuleID": "SEC-002",
						"Severity": "high",
						"Confidence": "medium",
						"Location": {
							"FilePath": "main.go",
							"StartLine": 10
						},
						"Message": "Hardcoded credential found",
						"Metadata": {
							"cwe": "CWE-259"
						}
					}
				]
			}`),
			expectedCount: 2,
			wantErr:       false,
			validateResult: func(t *testing.T, vulns Vulnerabilities) {
				assert.Equal(t, "SEC-004:path/file.go:42", vulns[0].ID)
				assert.Equal(t, "path/file.go", vulns[0].Package)
				assert.Equal(t, SeverityCritical, vulns[0].Severity)
				assert.Equal(t, "Private key header detected", vulns[0].Description)
				assert.Equal(t, "SEC-004", vulns[0].Provider)
				assert.Equal(t, "path/file.go:42", vulns[0].Reference)

				assert.Equal(t, "SEC-002:main.go:10", vulns[1].ID)
				assert.Equal(t, "main.go", vulns[1].Package)
				assert.Equal(t, SeverityHigh, vulns[1].Severity)
				assert.Equal(t, "Hardcoded credential found", vulns[1].Description)
			},
		},
		{
			name: "empty findings array",
			input: []byte(`{
				"findings": []
			}`),
			expectedCount: 0,
			wantErr:       false,
		},
		{
			name: "unknown severity maps to unknown",
			input: []byte(`{
				"findings": [
					{
						"ID": "SEC-099:file.go:1",
						"RuleID": "SEC-099",
						"Severity": "info",
						"Confidence": "low",
						"Location": {"FilePath": "file.go", "StartLine": 1},
						"Message": "Informational finding"
					}
				]
			}`),
			expectedCount: 1,
			wantErr:       false,
			validateResult: func(t *testing.T, vulns Vulnerabilities) {
				// "info" does not map to any known severity
				assert.Equal(t, SeverityUnknown, vulns[0].Severity)
			},
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
			scanner := NewNoxScanner()
			vulns, err := scanner.parseOutput(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, vulns, tt.expectedCount)

			if tt.validateResult != nil {
				tt.validateResult(t, vulns)
			}
		})
	}
}

func TestNoxScanner_buildArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		target   ScanTarget
		outFile  string
		expected []string
	}{
		{
			name:     "directory scan with path",
			target:   ScanTarget{Type: "directory", Path: "/some/path"},
			outFile:  "/tmp/nox-out.json",
			expected: []string{"scan", "/some/path", "-o", "/tmp/nox-out.json"},
		},
		{
			name:     "directory scan empty path defaults to dot",
			target:   ScanTarget{Type: "directory", Path: ""},
			outFile:  "/tmp/out.json",
			expected: []string{"scan", ".", "-o", "/tmp/out.json"},
		},
		{
			name:     "dir alias",
			target:   ScanTarget{Type: "dir", Path: "/path"},
			outFile:  "/tmp/out.json",
			expected: []string{"scan", "/path", "-o", "/tmp/out.json"},
		},
		{
			name:     "empty type defaults to directory",
			target:   ScanTarget{Type: "", Path: "/path"},
			outFile:  "/tmp/out.json",
			expected: []string{"scan", "/path", "-o", "/tmp/out.json"},
		},
		{
			name:     "unknown type defaults to directory",
			target:   ScanTarget{Type: "unknown", Path: "/path"},
			outFile:  "/tmp/out.json",
			expected: []string{"scan", "/path", "-o", "/tmp/out.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			scanner := NewNoxScanner()
			args := scanner.buildArgs(tt.target, tt.outFile)
			assert.Equal(t, tt.expected, args)
		})
	}
}

func TestNoxScanner_Scan_WithMockExec(t *testing.T) {
	t.Parallel()

	scanner := &NoxScanner{
		execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
			if name == "nox" && len(args) > 0 && args[0] == "--version" {
				return exec.Command("echo", "nox 0.7.0")
			}
			// Main scan command
			return exec.Command("echo", "scan done")
		},
	}

	ctx := context.Background()
	target := ScanTarget{Type: "directory", Path: "."}
	opts := ScanOptions{}

	// If nox is not available, the Scan() method returns ErrScannerNotAvailable
	result, err := scanner.Scan(ctx, target, opts)
	if err != nil {
		assert.ErrorIs(t, err, ErrScannerNotAvailable)
	} else {
		require.NotNil(t, result)
		assert.Equal(t, "nox", result.Scanner)
	}
}

func TestNoxScanner_parseOutput_LowSeverity(t *testing.T) {
	t.Parallel()

	input := []byte(`{
		"findings": [
			{
				"ID": "SEC-010:config.go:5",
				"RuleID": "SEC-010",
				"Severity": "low",
				"Confidence": "high",
				"Location": {"FilePath": "config.go", "StartLine": 5},
				"Message": "Low severity finding"
			}
		]
	}`)

	scanner := NewNoxScanner()
	vulns, err := scanner.parseOutput(input)
	require.NoError(t, err)
	require.Len(t, vulns, 1)
	assert.Equal(t, SeverityLow, vulns[0].Severity)
}

func TestNoxScanner_parseOutput_MediumSeverity(t *testing.T) {
	t.Parallel()

	input := []byte(`{
		"findings": [
			{
				"ID": "SEC-020:app.go:15",
				"RuleID": "SEC-020",
				"Severity": "medium",
				"Confidence": "high",
				"Location": {"FilePath": "app.go", "StartLine": 15},
				"Message": "Medium severity finding"
			}
		]
	}`)

	scanner := NewNoxScanner()
	vulns, err := scanner.parseOutput(input)
	require.NoError(t, err)
	require.Len(t, vulns, 1)
	assert.Equal(t, SeverityMedium, vulns[0].Severity)
}

func TestNoxScanner_parseOutput_MetadataWithCWE(t *testing.T) {
	t.Parallel()

	input := []byte(`{
		"findings": [
			{
				"ID": "SEC-004:secret.go:1",
				"RuleID": "SEC-004",
				"Severity": "critical",
				"Confidence": "high",
				"Location": {"FilePath": "secret.go", "StartLine": 1},
				"Message": "Secret detected",
				"Metadata": {"cwe": "CWE-798"}
			}
		]
	}`)

	scanner := NewNoxScanner()
	vulns, err := scanner.parseOutput(input)
	require.NoError(t, err)
	require.Len(t, vulns, 1)
	assert.Equal(t, "SEC-004", vulns[0].Provider)
	assert.Equal(t, "secret.go:1", vulns[0].Reference)
}
