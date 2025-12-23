package nvim

import (
	"context"
	"regexp"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// BinaryCheck describes a binary requirement.
type BinaryCheck struct {
	Name       string
	Required   bool
	MinVersion string
	Purpose    string
}

// BinaryResult contains the result of checking a binary.
type BinaryResult struct {
	Name     string
	Found    bool
	Version  string
	Path     string
	MeetsMin bool
	Purpose  string
}

// DoctorCheck provides health checks for nvim.
type DoctorCheck struct {
	runner ports.CommandRunner
}

// NewDoctorCheck creates a new DoctorCheck.
func NewDoctorCheck(runner ports.CommandRunner) *DoctorCheck {
	return &DoctorCheck{
		runner: runner,
	}
}

// RequiredBinaries returns the list of binaries needed for nvim functionality.
func (d *DoctorCheck) RequiredBinaries() []BinaryCheck {
	return []BinaryCheck{
		{Name: "nvim", Required: true, MinVersion: "0.9.0", Purpose: "Neovim editor"},
		{Name: "rg", Required: false, MinVersion: "", Purpose: "Telescope grep (ripgrep)"},
		{Name: "fd", Required: false, MinVersion: "", Purpose: "Telescope find"},
		{Name: "node", Required: false, MinVersion: "", Purpose: "LSP support"},
		{Name: "npm", Required: false, MinVersion: "", Purpose: "Mason package manager"},
	}
}

// CheckBinaries verifies all required binaries are installed.
func (d *DoctorCheck) CheckBinaries(ctx context.Context) []BinaryResult {
	var results []BinaryResult
	for _, check := range d.RequiredBinaries() {
		result := d.checkBinary(ctx, check)
		results = append(results, result)
	}
	return results
}

// checkBinary checks a single binary.
func (d *DoctorCheck) checkBinary(ctx context.Context, check BinaryCheck) BinaryResult {
	result := BinaryResult{
		Name:    check.Name,
		Purpose: check.Purpose,
	}

	// Try to find the binary using which
	whichResult, err := d.runner.Run(ctx, "which", check.Name)
	if err != nil || !whichResult.Success() {
		result.Found = false
		return result
	}

	result.Found = true
	result.Path = strings.TrimSpace(whichResult.Stdout)

	// Get version
	version := d.getVersion(ctx, check.Name)
	result.Version = version

	// Check minimum version if specified
	if check.MinVersion != "" && version != "" {
		result.MeetsMin = compareVersions(version, check.MinVersion) >= 0
	} else {
		result.MeetsMin = true // No minimum required
	}

	return result
}

// getVersion attempts to get the version of a binary.
func (d *DoctorCheck) getVersion(ctx context.Context, name string) string {
	var args []string
	switch name {
	case "nvim":
		args = []string{"--version"}
	case "node":
		args = []string{"--version"}
	case "npm":
		args = []string{"--version"}
	case "rg":
		args = []string{"--version"}
	case "fd":
		args = []string{"--version"}
	default:
		args = []string{"--version"}
	}

	result, err := d.runner.Run(ctx, name, args...)
	if err != nil || !result.Success() {
		return ""
	}

	return parseVersion(result.Stdout)
}

// parseVersion extracts version number from command output.
func parseVersion(output string) string {
	// Common patterns: "v0.9.0", "0.9.0", "NVIM v0.9.0"
	re := regexp.MustCompile(`v?(\d+\.\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return strings.TrimSpace(strings.Split(output, "\n")[0])
}

// compareVersions compares two semantic versions.
// Returns: -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		aVal := 0
		bVal := 0

		if i < len(aParts) {
			aVal = parseVersionPart(aParts[i])
		}
		if i < len(bParts) {
			bVal = parseVersionPart(bParts[i])
		}

		if aVal < bVal {
			return -1
		}
		if aVal > bVal {
			return 1
		}
	}
	return 0
}

// parseVersionPart parses a single version component.
func parseVersionPart(s string) int {
	// Strip any non-numeric suffix (e.g., "0-beta")
	re := regexp.MustCompile(`^(\d+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) >= 2 {
		var val int
		for _, c := range matches[1] {
			val = val*10 + int(c-'0')
		}
		return val
	}
	return 0
}

// HasIssues returns true if any required binary is missing or doesn't meet version requirements.
func (d *DoctorCheck) HasIssues(results []BinaryResult) bool {
	for _, r := range results {
		for _, check := range d.RequiredBinaries() {
			if check.Name == r.Name && check.Required {
				if !r.Found || !r.MeetsMin {
					return true
				}
			}
		}
	}
	return false
}
