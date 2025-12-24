package chocolatey

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- PackageStep Tests ---

func TestPackageStep_ID(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, nil)

	assert.Equal(t, "chocolatey:package:git", step.ID().String())
}

func TestPackageStep_DependsOn(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, nil)

	assert.Empty(t, step.DependsOn())
}

func TestPackageStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"list", "--local-only", "--exact", "git"}, ports.CommandResult{
		Stdout:   "Chocolatey v1.4.0\ngit 2.40.0\n1 package installed.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPackageStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"list", "--local-only", "--exact", "git"}, ports.CommandResult{
		Stdout:   "Chocolatey v1.4.0\n0 packages installed.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPackageStep_Check_WSL(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco.exe", []string{"list", "--local-only", "--exact", "git"}, ports.CommandResult{
		Stdout:   "Chocolatey v1.4.0\ngit 2.40.0\n1 package installed.",
		ExitCode: 0,
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPackageStep_Plan(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		pkg             Package
		expectedVersion string
	}{
		{
			name:            "no version specified",
			pkg:             Package{Name: "git"},
			expectedVersion: "latest",
		},
		{
			name:            "specific version",
			pkg:             Package{Name: "git", Version: "2.40.0"},
			expectedVersion: "2.40.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			step := NewPackageStep(tc.pkg, nil, nil)
			ctx := compiler.NewRunContext(context.Background())

			diff, err := step.Plan(ctx)
			require.NoError(t, err)
			assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
			assert.Equal(t, tc.pkg.Name, diff.Name())
			assert.Equal(t, tc.expectedVersion, diff.NewValue())
		})
	}
}

func TestPackageStep_Apply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"install", "git", "-y", "--no-progress"}, ports.CommandResult{
		Stdout:   "Installing git...\nPackage installed successfully.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestPackageStep_Apply_WithVersion(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"install", "git", "-y", "--no-progress", "--version=2.40.0"}, ports.CommandResult{
		Stdout:   "Installing git...\nPackage installed successfully.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git", Version: "2.40.0"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestPackageStep_Apply_WithSource(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"install", "git", "-y", "--no-progress", "--source=internal"}, ports.CommandResult{
		Stdout:   "Installing git...\nPackage installed successfully.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git", Source: "internal"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestPackageStep_Apply_WithPin(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"install", "git", "-y", "--no-progress"}, ports.CommandResult{
		Stdout:   "Installing git...\nPackage installed successfully.",
		ExitCode: 0,
	})
	runner.AddResult("choco", []string{"pin", "add", "-n=git"}, ports.CommandResult{
		Stdout:   "Pin added for git.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git", Pin: true}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestPackageStep_Apply_Failed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"install", "nonexistent", "-y", "--no-progress"}, ports.CommandResult{
		Stderr:   "Package not found.",
		ExitCode: 1,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "nonexistent"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "choco install nonexistent failed")
}

func TestPackageStep_Apply_InvalidName(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	// Package name with slash is invalid for chocolatey (valid for step ID but not choco)
	pkg := Package{Name: "git/malicious"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestPackageStep_Explain(t *testing.T) {
	t.Parallel()

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git", Version: "2.40.0", Source: "internal", Pin: true}
	step := NewPackageStep(pkg, nil, plat)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)
	assert.Equal(t, "Install Chocolatey Package", exp.Summary())
	assert.Contains(t, exp.Detail(), "git")
	assert.Contains(t, exp.Detail(), "2.40.0")
	assert.Contains(t, exp.Detail(), "internal")
	assert.Contains(t, exp.Detail(), "Pinned")
}

func TestPackageStep_Explain_WSL(t *testing.T) {
	t.Parallel()

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, plat)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)
	assert.Contains(t, exp.Detail(), "git")

	// Verify WSL tradeoffs are included
	tradeoffs := exp.Tradeoffs()
	hasWSLTradeoff := false
	for _, to := range tradeoffs {
		if to == "+ Installs Windows applications accessible from WSL" {
			hasWSLTradeoff = true
			break
		}
	}
	assert.True(t, hasWSLTradeoff, "Should include WSL tradeoff")
}

// --- SourceStep Tests ---

func TestSourceStep_ID(t *testing.T) {
	t.Parallel()

	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, nil, nil)

	assert.Equal(t, "chocolatey:source:internal", step.ID().String())
}

func TestSourceStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"source", "list"}, ports.CommandResult{
		Stdout:   "internal - https://nuget.internal.com/ | Priority: 0\nchocolatey - https://community.chocolatey.org/api/v2/ | Priority: 0",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestSourceStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"source", "list"}, ports.CommandResult{
		Stdout:   "chocolatey - https://community.chocolatey.org/api/v2/ | Priority: 0",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSourceStep_Check_WSL(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco.exe", []string{"source", "list"}, ports.CommandResult{
		Stdout:   "internal - https://nuget.internal.com/ | Priority: 0",
		ExitCode: 0,
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestSourceStep_Plan(t *testing.T) {
	t.Parallel()

	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, nil, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "internal", diff.Name())
	assert.Equal(t, "https://nuget.internal.com/", diff.NewValue())
}

func TestSourceStep_Apply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"source", "add", "-n=internal", "-s=https://nuget.internal.com/"}, ports.CommandResult{
		Stdout:   "Source added successfully.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestSourceStep_Apply_WithPriority(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"source", "add", "-n=internal", "-s=https://nuget.internal.com/", "--priority=1"}, ports.CommandResult{
		Stdout:   "Source added successfully.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/", Priority: 1}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestSourceStep_Apply_WithDisabled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"source", "add", "-n=internal", "-s=https://nuget.internal.com/"}, ports.CommandResult{
		Stdout:   "Source added successfully.",
		ExitCode: 0,
	})
	runner.AddResult("choco", []string{"source", "disable", "-n=internal"}, ports.CommandResult{
		Stdout:   "Source disabled.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/", Disabled: true}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestSourceStep_Apply_Failed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"source", "add", "-n=internal", "-s=https://nuget.internal.com/"}, ports.CommandResult{
		Stderr:   "Source already exists.",
		ExitCode: 1,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "choco source add internal failed")
}

func TestSourceStep_Apply_InvalidName(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	// Source name with slash is invalid for chocolatey (valid for step ID but not choco)
	source := Source{Name: "internal/malicious", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source name")
}

func TestSourceStep_Apply_InvalidURL(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "not-a-url"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source URL")
}

func TestSourceStep_Explain(t *testing.T) {
	t.Parallel()

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/", Priority: 1, Disabled: true}
	step := NewSourceStep(source, nil, plat)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)
	assert.Equal(t, "Configure Chocolatey Source", exp.Summary())
	assert.Contains(t, exp.Detail(), "internal")
	assert.Contains(t, exp.Detail(), "https://nuget.internal.com/")
	assert.Contains(t, exp.Detail(), "Priority: 1")
	assert.Contains(t, exp.Detail(), "(Disabled)")
}
