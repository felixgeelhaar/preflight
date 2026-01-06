package winget

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

func TestPackageStep_ID(t *testing.T) {
	t.Parallel()

	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, nil, nil)

	assert.Equal(t, "winget:package:Microsoft.VisualStudioCode", step.ID().String())
}

func TestPackageStep_DependsOn(t *testing.T) {
	t.Parallel()

	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, nil, nil)

	deps := step.DependsOn()

	assert.Equal(t, []compiler.StepID{compiler.MustNewStepID(wingetReadyStepID)}, deps)
}

func TestPackageStep_Check_Installed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("winget", []string{"list", "--id", "Microsoft.VisualStudioCode", "--exact", "--accept-source-agreements"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Name                          Id                           Version\n-------------------------------------------------------------------\nMicrosoft Visual Studio Code  Microsoft.VisualStudioCode   1.85.0\n",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPackageStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("winget", []string{"list", "--id", "Microsoft.VisualStudioCode", "--exact", "--accept-source-agreements"}, ports.CommandResult{
		ExitCode: 1,
		Stdout:   "No installed package found matching input criteria.\n",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPackageStep_Check_WSL_UsesWingetExe(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("winget.exe", []string{"list", "--id", "Microsoft.VisualStudioCode", "--exact", "--accept-source-agreements"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Name                          Id                           Version\n-------------------------------------------------------------------\nMicrosoft Visual Studio Code  Microsoft.VisualStudioCode   1.85.0\n",
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPackageStep_Plan(t *testing.T) {
	t.Parallel()

	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, nil, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "Microsoft.VisualStudioCode", diff.Name())
	assert.Equal(t, "latest", diff.NewValue())
}

func TestPackageStep_Plan_WithVersion(t *testing.T) {
	t.Parallel()

	pkg := Package{ID: "Microsoft.VisualStudioCode", Version: "1.85.0"}
	step := NewPackageStep(pkg, nil, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, "1.85.0", diff.NewValue())
}

func TestPackageStep_Apply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("winget", []string{"install", "--id", "Microsoft.VisualStudioCode", "--exact", "--accept-source-agreements", "--accept-package-agreements", "--silent"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
	calls := runner.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "winget", calls[0].Command)
}

func TestPackageStep_Apply_WithVersion(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("winget", []string{"install", "--id", "Microsoft.VisualStudioCode", "--exact", "--accept-source-agreements", "--accept-package-agreements", "--silent", "--version", "1.85.0"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{ID: "Microsoft.VisualStudioCode", Version: "1.85.0"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestPackageStep_Apply_WithSource(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("winget", []string{"install", "--id", "Microsoft.VisualStudioCode", "--exact", "--accept-source-agreements", "--accept-package-agreements", "--silent", "--source", "winget"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{ID: "Microsoft.VisualStudioCode", Source: "winget"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestPackageStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("winget", []string{"install", "--id", "Invalid.Package", "--exact", "--accept-source-agreements", "--accept-package-agreements", "--silent"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "No package found matching input criteria.",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{ID: "Invalid.Package"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "winget install Invalid.Package failed")
}

func TestPackageStep_Apply_InvalidID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	// Use a package ID that's valid for step ID but fails winget validation
	// (doesn't have the Publisher.Package format)
	pkg := Package{ID: "invalidpackage"} // Missing dot in Publisher.Package format
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package ID")
}

func TestPackageStep_Apply_InvalidSource(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	// Use a source name that fails validation (starts with number)
	pkg := Package{ID: "Microsoft.VisualStudioCode", Source: "123invalid"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source")
}

func TestPackageStep_Apply_WSL_UsesWingetExe(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("winget.exe", []string{"install", "--id", "Microsoft.VisualStudioCode", "--exact", "--accept-source-agreements", "--accept-package-agreements", "--silent"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
	calls := runner.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "winget.exe", calls[0].Command)
}

func TestPackageStep_Explain(t *testing.T) {
	t.Parallel()

	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, nil, nil)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.NotEmpty(t, explanation.Summary())
	assert.NotEmpty(t, explanation.Detail())
	assert.Contains(t, explanation.Detail(), "Microsoft.VisualStudioCode")
}

func TestPackageStep_Explain_WithVersion(t *testing.T) {
	t.Parallel()

	pkg := Package{ID: "Microsoft.VisualStudioCode", Version: "1.85.0"}
	step := NewPackageStep(pkg, nil, nil)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.Contains(t, explanation.Detail(), "1.85.0")
}

func TestPackageStep_Explain_WSL(t *testing.T) {
	t.Parallel()

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, nil, plat)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	tradeoffs := explanation.Tradeoffs()
	hasWSLTradeoff := false
	for _, t := range tradeoffs {
		if t == "+ Installs Windows applications accessible from WSL" {
			hasWSLTradeoff = true
			break
		}
	}
	assert.True(t, hasWSLTradeoff, "Should include WSL-specific tradeoff")
}

func TestPackageStep_wingetCommand_Windows(t *testing.T) {
	t.Parallel()

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{ID: "Test.Package"}
	step := NewPackageStep(pkg, nil, plat)

	assert.Equal(t, "winget", step.wingetCommand())
}

func TestPackageStep_wingetCommand_WSL(t *testing.T) {
	t.Parallel()

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	pkg := Package{ID: "Test.Package"}
	step := NewPackageStep(pkg, nil, plat)

	assert.Equal(t, "winget.exe", step.wingetCommand())
}

func TestPackageStep_wingetCommand_NilPlatform(t *testing.T) {
	t.Parallel()

	pkg := Package{ID: "Test.Package"}
	step := NewPackageStep(pkg, nil, nil)

	assert.Equal(t, "winget", step.wingetCommand())
}
