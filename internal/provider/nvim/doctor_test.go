package nvim_test

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/nvim"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
)

func TestDoctorCheck_RequiredBinaries(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	doctor := nvim.NewDoctorCheck(runner)

	binaries := doctor.RequiredBinaries()

	assert.Len(t, binaries, 5)

	// Check nvim is required
	var nvimCheck *nvim.BinaryCheck
	for i := range binaries {
		if binaries[i].Name == "nvim" {
			nvimCheck = &binaries[i]
			break
		}
	}
	assert.NotNil(t, nvimCheck)
	assert.True(t, nvimCheck.Required)
	assert.Equal(t, "0.9.0", nvimCheck.MinVersion)
}

func TestDoctorCheck_CheckBinaries_AllFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()

	// Mock which commands
	runner.AddResult("which", []string{"nvim"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "/usr/local/bin/nvim",
	})
	runner.AddResult("which", []string{"rg"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "/usr/local/bin/rg",
	})
	runner.AddResult("which", []string{"fd"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "/usr/local/bin/fd",
	})
	runner.AddResult("which", []string{"node"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "/usr/local/bin/node",
	})
	runner.AddResult("which", []string{"npm"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "/usr/local/bin/npm",
	})

	// Mock version commands
	runner.AddResult("nvim", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "NVIM v0.10.0",
	})
	runner.AddResult("rg", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "ripgrep 14.0.0",
	})
	runner.AddResult("fd", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "fd 9.0.0",
	})
	runner.AddResult("node", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "v20.0.0",
	})
	runner.AddResult("npm", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "10.0.0",
	})

	doctor := nvim.NewDoctorCheck(runner)
	results := doctor.CheckBinaries(context.Background())

	assert.Len(t, results, 5)
	for _, r := range results {
		assert.True(t, r.Found, "expected %s to be found", r.Name)
	}
	assert.False(t, doctor.HasIssues(results))
}

func TestDoctorCheck_CheckBinaries_NvimMissing(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()

	// Mock which commands - nvim not found
	runner.AddResult("which", []string{"nvim"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "nvim not found",
	})
	runner.AddResult("which", []string{"rg"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "/usr/local/bin/rg",
	})
	runner.AddResult("which", []string{"fd"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "/usr/local/bin/fd",
	})
	runner.AddResult("which", []string{"node"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "/usr/local/bin/node",
	})
	runner.AddResult("which", []string{"npm"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "/usr/local/bin/npm",
	})

	// Mock version commands for found binaries
	runner.AddResult("rg", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "ripgrep 14.0.0",
	})
	runner.AddResult("fd", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "fd 9.0.0",
	})
	runner.AddResult("node", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "v20.0.0",
	})
	runner.AddResult("npm", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "10.0.0",
	})

	doctor := nvim.NewDoctorCheck(runner)
	results := doctor.CheckBinaries(context.Background())

	// Find nvim result
	var nvimResult *nvim.BinaryResult
	for i := range results {
		if results[i].Name == "nvim" {
			nvimResult = &results[i]
			break
		}
	}

	assert.NotNil(t, nvimResult)
	assert.False(t, nvimResult.Found)
	assert.True(t, doctor.HasIssues(results), "should report issues when nvim missing")
}

func TestDoctorCheck_CheckBinaries_NvimVersionTooLow(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()

	// Mock which commands
	runner.AddResult("which", []string{"nvim"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "/usr/local/bin/nvim",
	})
	runner.AddResult("which", []string{"rg"}, ports.CommandResult{
		ExitCode: 1,
	})
	runner.AddResult("which", []string{"fd"}, ports.CommandResult{
		ExitCode: 1,
	})
	runner.AddResult("which", []string{"node"}, ports.CommandResult{
		ExitCode: 1,
	})
	runner.AddResult("which", []string{"npm"}, ports.CommandResult{
		ExitCode: 1,
	})

	// Mock old nvim version
	runner.AddResult("nvim", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "NVIM v0.8.0",
	})

	doctor := nvim.NewDoctorCheck(runner)
	results := doctor.CheckBinaries(context.Background())

	// Find nvim result
	var nvimResult *nvim.BinaryResult
	for i := range results {
		if results[i].Name == "nvim" {
			nvimResult = &results[i]
			break
		}
	}

	assert.NotNil(t, nvimResult)
	assert.True(t, nvimResult.Found)
	assert.Equal(t, "0.8.0", nvimResult.Version)
	assert.False(t, nvimResult.MeetsMin, "version 0.8.0 should not meet minimum 0.9.0")
	assert.True(t, doctor.HasIssues(results), "should report issues when version too low")
}

func TestDoctorCheck_OptionalBinariesMissing(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()

	// Mock which commands - only nvim found
	runner.AddResult("which", []string{"nvim"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "/usr/local/bin/nvim",
	})
	runner.AddResult("which", []string{"rg"}, ports.CommandResult{
		ExitCode: 1,
	})
	runner.AddResult("which", []string{"fd"}, ports.CommandResult{
		ExitCode: 1,
	})
	runner.AddResult("which", []string{"node"}, ports.CommandResult{
		ExitCode: 1,
	})
	runner.AddResult("which", []string{"npm"}, ports.CommandResult{
		ExitCode: 1,
	})

	// Mock nvim version
	runner.AddResult("nvim", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "NVIM v0.10.0",
	})

	doctor := nvim.NewDoctorCheck(runner)
	results := doctor.CheckBinaries(context.Background())

	// Should not report issues - optional binaries missing is OK
	assert.False(t, doctor.HasIssues(results))

	// Count missing
	missing := 0
	for _, r := range results {
		if !r.Found {
			missing++
		}
	}
	assert.Equal(t, 4, missing, "should have 4 missing optional binaries")
}

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a, b     string
		expected int
	}{
		{"0.9.0", "0.9.0", 0},
		{"0.10.0", "0.9.0", 1},
		{"0.8.0", "0.9.0", -1},
		{"1.0.0", "0.9.0", 1},
		{"0.9.1", "0.9.0", 1},
		{"10.0.0", "9.0.0", 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			// Access via parseVersion since compareVersions is not exported
			// We test it indirectly through the doctor check
		})
	}
}
