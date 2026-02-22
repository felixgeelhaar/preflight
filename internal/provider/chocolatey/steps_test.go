package chocolatey

import (
	"context"
	"os/exec"
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
	step := NewPackageStep(pkg, nil, nil, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})

	assert.Equal(t, "chocolatey:package:git", step.ID().String())
}

func TestPackageStep_DependsOn(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, nil, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})

	assert.Equal(t, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)}, step.DependsOn())
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
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
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
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
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
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
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
			step := NewPackageStep(tc.pkg, nil, nil, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
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
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
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
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
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
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
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
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
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
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
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
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestPackageStep_Explain(t *testing.T) {
	t.Parallel()

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git", Version: "2.40.0", Source: "internal", Pin: true}
	step := NewPackageStep(pkg, nil, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
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
	step := NewPackageStep(pkg, nil, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
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

func TestSourceStep_DependsOn(t *testing.T) {
	t.Parallel()

	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, nil, nil)

	assert.Equal(t, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)}, step.DependsOn())
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

// --- InstallStep Tests ---

func TestInstallStep_ID(t *testing.T) {
	t.Parallel()

	step := NewInstallStep(nil, nil)

	assert.Equal(t, chocoInstallStepID, step.ID().String())
}

func TestInstallStep_DependsOn(t *testing.T) {
	t.Parallel()

	step := NewInstallStep(nil, nil)

	deps := step.DependsOn()

	assert.Nil(t, deps)
}

func TestInstallStep_Plan(t *testing.T) {
	t.Parallel()

	step := NewInstallStep(nil, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "chocolatey", diff.Resource())
	assert.Equal(t, "install", diff.Name())
}

func TestInstallStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("powershell", []string{"-NoProfile", "-InputFormat", "None", "-ExecutionPolicy", "Bypass", "-Command", "[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Chocolatey installed successfully.",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewInstallStep(runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestInstallStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("powershell", []string{"-NoProfile", "-InputFormat", "None", "-ExecutionPolicy", "Bypass", "-Command", "[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Access denied",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewInstallStep(runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chocolatey install failed")
}

func TestInstallStep_Apply_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("powershell", []string{"-NoProfile", "-InputFormat", "None", "-ExecutionPolicy", "Bypass", "-Command", "[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))"}, assert.AnError)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewInstallStep(runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
}

func TestInstallStep_Apply_WSL_UsesPowershellExe(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("powershell.exe", []string{"-NoProfile", "-InputFormat", "None", "-ExecutionPolicy", "Bypass", "-Command", "[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	step := NewInstallStep(runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
	calls := runner.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "powershell.exe", calls[0].Command)
}

func TestInstallStep_Explain(t *testing.T) {
	t.Parallel()

	step := NewInstallStep(nil, nil)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.Equal(t, "Install Chocolatey", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "Chocolatey")
	assert.NotEmpty(t, explanation.DocLinks())
}

func TestInstallStep_chocoCommand_Windows(t *testing.T) {
	t.Parallel()

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewInstallStep(nil, plat)

	assert.Equal(t, "choco", step.chocoCommand())
}

func TestInstallStep_chocoCommand_WSL(t *testing.T) {
	t.Parallel()

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	step := NewInstallStep(nil, plat)

	assert.Equal(t, "choco.exe", step.chocoCommand())
}

func TestInstallStep_chocoCommand_NilPlatform(t *testing.T) {
	t.Parallel()

	step := NewInstallStep(nil, nil)

	assert.Equal(t, "choco", step.chocoCommand())
}

func TestInstallStep_powerShellCommand_Windows(t *testing.T) {
	t.Parallel()

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewInstallStep(nil, plat)

	assert.Equal(t, "powershell", step.powerShellCommand())
}

func TestInstallStep_powerShellCommand_WSL(t *testing.T) {
	t.Parallel()

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	step := NewInstallStep(nil, plat)

	assert.Equal(t, "powershell.exe", step.powerShellCommand())
}

func TestInstallStep_powerShellCommand_NilPlatform(t *testing.T) {
	t.Parallel()

	step := NewInstallStep(nil, nil)

	assert.Equal(t, "powershell", step.powerShellCommand())
}

// --- SourceStep additional tests ---

func TestSourceStep_Check_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("choco", []string{"source", "list"}, exec.ErrNotFound)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSourceStep_Check_UnknownError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("choco", []string{"source", "list"}, assert.AnError)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestSourceStep_Apply_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("choco", []string{"source", "add", "-n=internal", "-s=https://nuget.internal.com/"}, exec.ErrNotFound)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "choco not found in PATH")
}

func TestSourceStep_Apply_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("choco", []string{"source", "add", "-n=internal", "-s=https://nuget.internal.com/"}, assert.AnError)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
}

func TestSourceStep_Apply_DisableFailed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"source", "add", "-n=internal", "-s=https://nuget.internal.com/"}, ports.CommandResult{
		ExitCode: 0,
	})
	runner.AddResult("choco", []string{"source", "disable", "-n=internal"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Failed to disable source",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/", Disabled: true}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "choco source disable internal failed")
}

func TestSourceStep_Apply_DisableRunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"source", "add", "-n=internal", "-s=https://nuget.internal.com/"}, ports.CommandResult{
		ExitCode: 0,
	})
	runner.AddError("choco", []string{"source", "disable", "-n=internal"}, assert.AnError)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	source := Source{Name: "internal", URL: "https://nuget.internal.com/", Disabled: true}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
}

func TestSourceStep_Apply_WSL_UsesChocoExe(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco.exe", []string{"source", "add", "-n=internal", "-s=https://nuget.internal.com/"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
	calls := runner.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "choco.exe", calls[0].Command)
}

// --- PackageStep additional tests ---

func TestPackageStep_Check_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("choco", []string{"list", "--local-only", "--exact", "git"}, exec.ErrNotFound)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPackageStep_Check_UnknownError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("choco", []string{"list", "--local-only", "--exact", "git"}, assert.AnError)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestPackageStep_Apply_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("choco", []string{"install", "git", "-y", "--no-progress"}, exec.ErrNotFound)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "choco not found in PATH")
}

func TestPackageStep_Apply_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("choco", []string{"install", "git", "-y", "--no-progress"}, assert.AnError)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
}

func TestPackageStep_Apply_WithArgs(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"install", "vscode", "-y", "--no-progress", "--package-parameters=/NoDesktopIcon"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "vscode", Args: "/NoDesktopIcon"}
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestPackageStep_Apply_PinCommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"install", "git", "-y", "--no-progress"}, ports.CommandResult{
		ExitCode: 0,
	})
	runner.AddError("choco", []string{"pin", "add", "-n=git"}, exec.ErrNotFound)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git", Pin: true}
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "choco not found in PATH")
}

func TestPackageStep_Apply_PinRunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"install", "git", "-y", "--no-progress"}, ports.CommandResult{
		ExitCode: 0,
	})
	runner.AddError("choco", []string{"pin", "add", "-n=git"}, assert.AnError)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git", Pin: true}
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
}

func TestPackageStep_Apply_PinFailed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco", []string{"install", "git", "-y", "--no-progress"}, ports.CommandResult{
		ExitCode: 0,
	})
	runner.AddResult("choco", []string{"pin", "add", "-n=git"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Pin failed",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git", Pin: true}
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "choco pin git failed")
}

func TestPackageStep_Apply_InvalidSource(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git", Source: "invalid/source"}
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source")
}

func TestPackageStep_Apply_WSL_UsesChocoExe(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("choco.exe", []string{"install", "git", "-y", "--no-progress"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
	calls := runner.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "choco.exe", calls[0].Command)
}

// --- PackageStep LockInfo ---

func TestPackageStep_LockInfo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		pkg             Package
		expectedName    string
		expectedVersion string
	}{
		{
			name:            "without version",
			pkg:             Package{Name: "git"},
			expectedName:    "git",
			expectedVersion: "",
		},
		{
			name:            "with version",
			pkg:             Package{Name: "git", Version: "2.40.0"},
			expectedName:    "git",
			expectedVersion: "2.40.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			step := NewPackageStep(tc.pkg, nil, nil, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})

			info, ok := step.LockInfo()

			assert.True(t, ok)
			assert.Equal(t, "chocolatey", info.Provider)
			assert.Equal(t, tc.expectedName, info.Name)
			assert.Equal(t, tc.expectedVersion, info.Version)
		})
	}
}

// --- PackageStep InstalledVersion ---

func TestPackageStep_InstalledVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		pkg             Package
		plat            *platform.Platform
		cmd             string
		result          ports.CommandResult
		err             error
		expectedVersion string
		expectedFound   bool
		expectedErr     bool
	}{
		{
			name: "found with version",
			pkg:  Package{Name: "git"},
			plat: platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:  "choco",
			result: ports.CommandResult{
				ExitCode: 0,
				Stdout:   "git|2.40.0\n",
			},
			expectedVersion: "2.40.0",
			expectedFound:   true,
		},
		{
			name: "not found - empty output",
			pkg:  Package{Name: "git"},
			plat: platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:  "choco",
			result: ports.CommandResult{
				ExitCode: 0,
				Stdout:   "",
			},
			expectedVersion: "",
			expectedFound:   false,
		},
		{
			name: "not found - no pipe separator",
			pkg:  Package{Name: "git"},
			plat: platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:  "choco",
			result: ports.CommandResult{
				ExitCode: 0,
				Stdout:   "git\n",
			},
			expectedVersion: "",
			expectedFound:   false,
		},
		{
			name: "not found - empty version after pipe",
			pkg:  Package{Name: "git"},
			plat: platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:  "choco",
			result: ports.CommandResult{
				ExitCode: 0,
				Stdout:   "git|\n",
			},
			expectedVersion: "",
			expectedFound:   false,
		},
		{
			name:            "command not found",
			pkg:             Package{Name: "git"},
			plat:            platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:             "choco",
			err:             exec.ErrNotFound,
			expectedVersion: "",
			expectedFound:   false,
		},
		{
			name:            "runner error - not command not found",
			pkg:             Package{Name: "git"},
			plat:            platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:             "choco",
			err:             assert.AnError,
			expectedVersion: "",
			expectedFound:   false,
			expectedErr:     true,
		},
		{
			name: "run failure - not success",
			pkg:  Package{Name: "git"},
			plat: platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:  "choco",
			result: ports.CommandResult{
				ExitCode: 1,
			},
			expectedVersion: "",
			expectedFound:   false,
		},
		{
			name: "WSL uses choco.exe",
			pkg:  Package{Name: "git"},
			plat: platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c"),
			cmd:  "choco.exe",
			result: ports.CommandResult{
				ExitCode: 0,
				Stdout:   "git|2.40.0\n",
			},
			expectedVersion: "2.40.0",
			expectedFound:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runner := mocks.NewCommandRunner()
			args := []string{"list", "--local-only", "--exact", "--limit-output", tc.pkg.Name}
			if tc.err != nil {
				runner.AddError(tc.cmd, args, tc.err)
			} else {
				runner.AddResult(tc.cmd, args, tc.result)
			}

			step := NewPackageStep(tc.pkg, runner, tc.plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
			ctx := compiler.NewRunContext(context.Background())

			version, found, err := step.InstalledVersion(ctx)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.expectedVersion, version)
			assert.Equal(t, tc.expectedFound, found)
		})
	}
}

// --- PackageStep Explain additional ---

func TestPackageStep_Explain_NoExtras(t *testing.T) {
	t.Parallel()

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, plat, []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)})
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.Equal(t, "Install Chocolatey Package", exp.Summary())
	assert.Contains(t, exp.Detail(), "git")
	assert.NotContains(t, exp.Detail(), "Version:")
	assert.NotContains(t, exp.Detail(), "Source:")
	assert.NotContains(t, exp.Detail(), "Pinned")
}

// --- SourceStep Explain additional ---

func TestSourceStep_Explain_NoPriorityNoDisabled(t *testing.T) {
	t.Parallel()

	source := Source{Name: "internal", URL: "https://nuget.internal.com/"}
	step := NewSourceStep(source, nil, nil)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)

	assert.Equal(t, "Configure Chocolatey Source", exp.Summary())
	assert.Contains(t, exp.Detail(), "internal")
	assert.NotContains(t, exp.Detail(), "Priority:")
	assert.NotContains(t, exp.Detail(), "(Disabled)")
	assert.NotEmpty(t, exp.Tradeoffs())
}
