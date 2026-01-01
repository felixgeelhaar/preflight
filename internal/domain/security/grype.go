package security

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GrypeScanner implements the Scanner interface using Grype.
type GrypeScanner struct {
	execCommand func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// NewGrypeScanner creates a new Grype scanner.
func NewGrypeScanner() *GrypeScanner {
	return &GrypeScanner{
		execCommand: exec.CommandContext,
	}
}

// Name returns the scanner name.
func (g *GrypeScanner) Name() string {
	return "grype"
}

// Available returns true if grype is installed.
func (g *GrypeScanner) Available() bool {
	_, err := exec.LookPath("grype")
	return err == nil
}

// Version returns the grype version.
func (g *GrypeScanner) Version(ctx context.Context) (string, error) {
	cmd := g.execCommand(ctx, "grype", "version", "--output", "json")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// Try simple version output
		stdout.Reset()
		cmd = g.execCommand(ctx, "grype", "--version")
		cmd.Stdout = &stdout
		if errRetry := cmd.Run(); errRetry != nil {
			return "", fmt.Errorf("failed to get grype version: %w", errRetry)
		}
		// Parse "grype 0.74.0" format
		output := strings.TrimSpace(stdout.String())
		parts := strings.Fields(output)
		if len(parts) >= 2 {
			return parts[1], nil
		}
		return output, nil
	}

	// Parse JSON version output
	var versionInfo struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &versionInfo); err != nil {
		return strings.TrimSpace(stdout.String()), nil //nolint:nilerr // Fallback to raw output on parse failure
	}
	return versionInfo.Version, nil
}

// Scan performs a vulnerability scan using grype.
func (g *GrypeScanner) Scan(ctx context.Context, target ScanTarget, opts ScanOptions) (*ScanResult, error) {
	if !g.Available() {
		return nil, ErrScannerNotAvailable
	}

	version, _ := g.Version(ctx)

	result := &ScanResult{
		Scanner:         g.Name(),
		Version:         version,
		Vulnerabilities: make(Vulnerabilities, 0),
	}

	// Build grype command based on target type
	args := g.buildArgs(target, opts)

	cmd := g.execCommand(ctx, "grype", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	// Grype returns exit code 1 when vulnerabilities are found, which is not an error
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Exit code 1 = vulnerabilities found (not an error)
			if exitErr.ExitCode() != 1 {
				return nil, fmt.Errorf("%w: %s", ErrScanFailed, stderr.String())
			}
		} else {
			return nil, fmt.Errorf("failed to run grype: %w", err)
		}
	}

	// Parse JSON output
	vulns, packagesScanned, err := g.parseOutput(stdout.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to parse grype output: %w", err)
	}

	result.Vulnerabilities = vulns
	result.PackagesScanned = packagesScanned

	// Apply filters
	if len(opts.IgnoreIDs) > 0 {
		result.Vulnerabilities = result.Vulnerabilities.ExcludeIDs(opts.IgnoreIDs)
	}

	if opts.MinSeverity != "" {
		result.Vulnerabilities = result.Vulnerabilities.BySeverity(opts.MinSeverity)
	}

	return result, nil
}

// buildArgs constructs grype command arguments.
func (g *GrypeScanner) buildArgs(target ScanTarget, _ ScanOptions) []string {
	args := []string{"--output", "json"}

	switch target.Type {
	case "directory", "dir", "":
		path := target.Path
		if path == "" {
			path = "."
		}
		args = append(args, "dir:"+path)
	case "sbom":
		args = append(args, "sbom:"+target.Path)
	case "image":
		args = append(args, target.Path)
	default:
		// Default to directory scan
		path := target.Path
		if path == "" {
			path = "."
		}
		args = append(args, "dir:"+path)
	}

	return args
}

// grypeOutput represents the JSON output from grype.
type grypeOutput struct {
	Matches []grypeMatch `json:"matches"`
	Source  grypeSource  `json:"source"`
}

type grypeMatch struct {
	Vulnerability grypeVuln     `json:"vulnerability"`
	Artifact      grypeArtifact `json:"artifact"`
}

type grypeVuln struct {
	ID          string      `json:"id"`
	Severity    string      `json:"severity"`
	Description string      `json:"description"`
	Fix         grypeFix    `json:"fix"`
	URLs        []string    `json:"urls"`
	CVSS        []grypeCVSS `json:"cvss"`
}

type grypeFix struct {
	Versions []string `json:"versions"`
	State    string   `json:"state"`
}

type grypeCVSS struct {
	Version string           `json:"version"`
	Vector  string           `json:"vector"`
	Metrics grypeCVSSMetrics `json:"metrics"`
}

type grypeCVSSMetrics struct {
	BaseScore float64 `json:"baseScore"`
}

type grypeArtifact struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

type grypeSource struct {
	Type   string      `json:"type"`
	Target interface{} `json:"target"`
}

// parseOutput parses grype JSON output.
func (g *GrypeScanner) parseOutput(data []byte) (Vulnerabilities, int, error) {
	if len(data) == 0 {
		return Vulnerabilities{}, 0, nil
	}

	var output grypeOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, 0, err
	}

	// Track unique packages
	packages := make(map[string]bool)

	vulns := make(Vulnerabilities, 0, len(output.Matches))
	for _, match := range output.Matches {
		packages[match.Artifact.Name] = true

		severity, _ := ParseSeverity(match.Vulnerability.Severity)

		// Get CVSS score
		var cvss float64
		if len(match.Vulnerability.CVSS) > 0 {
			cvss = match.Vulnerability.CVSS[0].Metrics.BaseScore
		}

		// Get fixed version
		var fixedIn string
		if len(match.Vulnerability.Fix.Versions) > 0 {
			fixedIn = match.Vulnerability.Fix.Versions[0]
		}

		// Get reference URL
		var reference string
		if len(match.Vulnerability.URLs) > 0 {
			reference = match.Vulnerability.URLs[0]
		}

		vuln := NewVulnerability(match.Vulnerability.ID, match.Artifact.Name).
			WithVersion(match.Artifact.Version).
			WithSeverity(severity).
			WithCVSS(cvss).
			WithDescription(match.Vulnerability.Description).
			WithFixedIn(fixedIn).
			WithReference(reference).
			WithProvider(match.Artifact.Type).
			Build()

		vuln.DetectedAt = time.Now()
		vulns = append(vulns, vuln)
	}

	return vulns, len(packages), nil
}
