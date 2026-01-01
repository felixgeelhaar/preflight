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

// TrivyScanner implements the Scanner interface using Trivy.
type TrivyScanner struct {
	execCommand func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// NewTrivyScanner creates a new Trivy scanner.
func NewTrivyScanner() *TrivyScanner {
	return &TrivyScanner{
		execCommand: exec.CommandContext,
	}
}

// Name returns the scanner name.
func (t *TrivyScanner) Name() string {
	return "trivy"
}

// Available returns true if trivy is installed.
func (t *TrivyScanner) Available() bool {
	_, err := exec.LookPath("trivy")
	return err == nil
}

// Version returns the trivy version.
func (t *TrivyScanner) Version(ctx context.Context) (string, error) {
	cmd := t.execCommand(ctx, "trivy", "--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get trivy version: %w", err)
	}

	// Parse "Version: 0.50.0" format
	output := strings.TrimSpace(stdout.String())
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "Version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Version:")), nil
		}
	}
	return output, nil
}

// Scan performs a vulnerability scan using trivy.
func (t *TrivyScanner) Scan(ctx context.Context, target ScanTarget, opts ScanOptions) (*ScanResult, error) {
	if !t.Available() {
		return nil, ErrScannerNotAvailable
	}

	version, _ := t.Version(ctx)

	result := &ScanResult{
		Scanner:         t.Name(),
		Version:         version,
		Vulnerabilities: make(Vulnerabilities, 0),
	}

	// Build trivy command based on target type
	args := t.buildArgs(target, opts)

	cmd := t.execCommand(ctx, "trivy", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if it's just because vulnerabilities were found
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Trivy returns non-zero when vulnerabilities are found with --exit-code
			// but we're not using that, so any exit error is a real error
			if exitErr.ExitCode() != 0 && stdout.Len() == 0 {
				return nil, fmt.Errorf("%w: %s", ErrScanFailed, stderr.String())
			}
		} else {
			return nil, fmt.Errorf("failed to run trivy: %w", err)
		}
	}

	// Parse JSON output
	vulns, packagesScanned, err := t.parseOutput(stdout.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to parse trivy output: %w", err)
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

// buildArgs constructs trivy command arguments.
func (t *TrivyScanner) buildArgs(target ScanTarget, _ ScanOptions) []string {
	args := []string{"--format", "json", "--scanners", "vuln"}

	switch target.Type {
	case "directory", "dir", "":
		path := target.Path
		if path == "" {
			path = "."
		}
		args = append([]string{"fs"}, args...)
		args = append(args, path)
	case "sbom":
		args = append([]string{"sbom"}, args...)
		args = append(args, target.Path)
	case "image":
		args = append([]string{"image"}, args...)
		args = append(args, target.Path)
	default:
		// Default to filesystem scan
		path := target.Path
		if path == "" {
			path = "."
		}
		args = append([]string{"fs"}, args...)
		args = append(args, path)
	}

	return args
}

// trivyOutput represents the JSON output from trivy.
type trivyOutput struct {
	Results []trivyResult `json:"Results"`
}

type trivyResult struct {
	Target          string      `json:"Target"`
	Type            string      `json:"Type"`
	Vulnerabilities []trivyVuln `json:"Vulnerabilities"`
}

type trivyVuln struct {
	VulnerabilityID  string               `json:"VulnerabilityID"`
	PkgName          string               `json:"PkgName"`
	InstalledVersion string               `json:"InstalledVersion"`
	FixedVersion     string               `json:"FixedVersion"`
	Severity         string               `json:"Severity"`
	Title            string               `json:"Title"`
	Description      string               `json:"Description"`
	References       []string             `json:"References"`
	CVSS             map[string]trivyCVSS `json:"CVSS"`
}

type trivyCVSS struct {
	V3Score float64 `json:"V3Score"`
}

// parseOutput parses trivy JSON output.
func (t *TrivyScanner) parseOutput(data []byte) (Vulnerabilities, int, error) {
	if len(data) == 0 {
		return Vulnerabilities{}, 0, nil
	}

	var output trivyOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, 0, err
	}

	// Track unique packages
	packages := make(map[string]bool)

	vulns := make(Vulnerabilities, 0)
	for _, result := range output.Results {
		for _, v := range result.Vulnerabilities {
			packages[v.PkgName] = true

			severity, _ := ParseSeverity(v.Severity)

			// Get CVSS score (prefer NVD)
			var cvss float64
			if nvd, ok := v.CVSS["nvd"]; ok {
				cvss = nvd.V3Score
			} else if redhat, ok := v.CVSS["redhat"]; ok {
				cvss = redhat.V3Score
			}

			// Get reference URL
			var reference string
			if len(v.References) > 0 {
				reference = v.References[0]
			}

			vuln := NewVulnerability(v.VulnerabilityID, v.PkgName).
				WithVersion(v.InstalledVersion).
				WithSeverity(severity).
				WithCVSS(cvss).
				WithTitle(v.Title).
				WithDescription(v.Description).
				WithFixedIn(v.FixedVersion).
				WithReference(reference).
				WithProvider(result.Type).
				Build()

			vuln.DetectedAt = time.Now()
			vulns = append(vulns, vuln)
		}
	}

	return vulns, len(packages), nil
}
