package security

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// NoxScanner implements the Scanner interface using nox.
type NoxScanner struct {
	execCommand func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// NewNoxScanner creates a new nox scanner.
func NewNoxScanner() *NoxScanner {
	return &NoxScanner{
		execCommand: exec.CommandContext,
	}
}

// Name returns the scanner name.
func (n *NoxScanner) Name() string {
	return "nox"
}

// Available returns true if nox is installed.
func (n *NoxScanner) Available() bool {
	_, err := exec.LookPath("nox")
	return err == nil
}

// Version returns the nox version.
func (n *NoxScanner) Version(ctx context.Context) (string, error) {
	cmd := n.execCommand(ctx, "nox", "--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get nox version: %w", err)
	}

	// Parse "nox 0.7.0" format
	output := strings.TrimSpace(stdout.String())
	parts := strings.Fields(output)
	if len(parts) >= 2 {
		return parts[1], nil
	}
	return output, nil
}

// Scan performs a vulnerability scan using nox.
func (n *NoxScanner) Scan(ctx context.Context, target ScanTarget, opts ScanOptions) (*ScanResult, error) {
	if !n.Available() {
		return nil, ErrScannerNotAvailable
	}

	version, _ := n.Version(ctx)

	result := &ScanResult{
		Scanner:         n.Name(),
		Version:         version,
		Vulnerabilities: make(Vulnerabilities, 0),
	}

	// Create a temp file for nox output
	tmpFile, err := os.CreateTemp("", "nox-scan-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file for nox output: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	// Build nox command
	args := n.buildArgs(target, tmpPath)

	cmd := n.execCommand(ctx, "nox", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	// nox exit code 1 means findings found (not an error)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() != 1 {
				return nil, fmt.Errorf("%w: %s", ErrScanFailed, stderr.String())
			}
		} else {
			return nil, fmt.Errorf("failed to run nox: %w", err)
		}
	}

	// Read the output file
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		// No output file means no findings
		return result, nil
	}

	// Parse nox output
	vulns, err := n.parseOutput(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nox output: %w", err)
	}

	result.Vulnerabilities = vulns
	result.PackagesScanned = n.countUniqueFiles(vulns)

	// Apply filters
	if len(opts.IgnoreIDs) > 0 {
		result.Vulnerabilities = result.Vulnerabilities.ExcludeIDs(opts.IgnoreIDs)
	}

	if opts.MinSeverity != "" {
		result.Vulnerabilities = result.Vulnerabilities.BySeverity(opts.MinSeverity)
	}

	return result, nil
}

// buildArgs constructs nox command arguments.
func (n *NoxScanner) buildArgs(target ScanTarget, outFile string) []string {
	path := target.Path
	if path == "" {
		path = "."
	}

	return []string{"scan", path, "-o", outFile}
}

// noxOutput represents the JSON output from nox.
type noxOutput struct {
	Findings []noxFinding `json:"findings"`
}

type noxFinding struct {
	ID         string      `json:"ID"`
	RuleID     string      `json:"RuleID"`
	Severity   string      `json:"Severity"`
	Confidence string      `json:"Confidence"`
	Location   noxLocation `json:"Location"`
	Message    string      `json:"Message"`
	Metadata   noxMetadata `json:"Metadata"`
}

type noxLocation struct {
	FilePath  string `json:"FilePath"`
	StartLine int    `json:"StartLine"`
}

type noxMetadata struct {
	CWE string `json:"cwe"`
}

// parseOutput parses nox JSON output.
func (n *NoxScanner) parseOutput(data []byte) (Vulnerabilities, error) {
	if len(data) == 0 {
		return Vulnerabilities{}, nil
	}

	var output noxOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, err
	}

	vulns := make(Vulnerabilities, 0, len(output.Findings))
	for _, finding := range output.Findings {
		severity := n.mapSeverity(finding.Severity)

		// Build a file:line reference
		reference := fmt.Sprintf("%s:%d", finding.Location.FilePath, finding.Location.StartLine)

		vuln := NewVulnerability(finding.ID, finding.Location.FilePath).
			WithSeverity(severity).
			WithDescription(finding.Message).
			WithReference(reference).
			WithProvider(finding.RuleID).
			Build()

		vuln.DetectedAt = time.Now()
		vulns = append(vulns, vuln)
	}

	return vulns, nil
}

// mapSeverity maps nox severity strings to security.Severity.
func (n *NoxScanner) mapSeverity(s string) Severity {
	switch strings.ToLower(s) {
	case "critical":
		return SeverityCritical
	case "high":
		return SeverityHigh
	case "medium":
		return SeverityMedium
	case "low":
		return SeverityLow
	default:
		return SeverityUnknown
	}
}

// countUniqueFiles counts unique file paths in vulnerabilities.
func (n *NoxScanner) countUniqueFiles(vulns Vulnerabilities) int {
	files := make(map[string]bool)
	for _, v := range vulns {
		files[v.Package] = true
	}
	return len(files)
}
