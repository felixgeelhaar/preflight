package winget

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

// --- ReadyStep Tests ---

func TestReadyStep_ID(t *testing.T) {
	t.Parallel()

	step := NewReadyStep(nil)

	assert.Equal(t, wingetReadyStepID, step.ID().String())
}

func TestReadyStep_DependsOn(t *testing.T) {
	t.Parallel()

	step := NewReadyStep(nil)

	deps := step.DependsOn()

	assert.Nil(t, deps)
}

func TestReadyStep_Plan(t *testing.T) {
	t.Parallel()

	step := NewReadyStep(nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "winget", diff.Resource())
	assert.Equal(t, "ready", diff.Name())
}

func TestReadyStep_Apply(t *testing.T) {
	t.Parallel()

	step := NewReadyStep(nil)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "winget not found in PATH")
}

func TestReadyStep_Explain(t *testing.T) {
	t.Parallel()

	step := NewReadyStep(nil)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.Equal(t, "Ensure winget Available", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "winget")
	assert.NotEmpty(t, explanation.DocLinks())
}

func TestReadyStep_wingetCommand_NilPlatform(t *testing.T) {
	t.Parallel()

	step := NewReadyStep(nil)

	assert.Equal(t, "winget", step.wingetCommand())
}

func TestReadyStep_wingetCommand_Windows(t *testing.T) {
	t.Parallel()

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewReadyStep(plat)

	assert.Equal(t, "winget", step.wingetCommand())
}

func TestReadyStep_wingetCommand_WSL(t *testing.T) {
	t.Parallel()

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	step := NewReadyStep(plat)

	assert.Equal(t, "winget.exe", step.wingetCommand())
}

// --- PackageStep additional Check tests ---

func TestPackageStep_Check_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("winget", []string{"list", "--id", "Microsoft.VisualStudioCode", "--exact", "--accept-source-agreements"}, exec.ErrNotFound)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPackageStep_Check_UnknownError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("winget", []string{"list", "--id", "Microsoft.VisualStudioCode", "--exact", "--accept-source-agreements"}, assert.AnError)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

// --- PackageStep additional Apply tests ---

func TestPackageStep_Apply_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("winget", []string{"install", "--id", "Microsoft.VisualStudioCode", "--exact", "--accept-source-agreements", "--accept-package-agreements", "--silent"}, exec.ErrNotFound)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "winget not found in PATH")
}

func TestPackageStep_Apply_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("winget", []string{"install", "--id", "Microsoft.VisualStudioCode", "--exact", "--accept-source-agreements", "--accept-package-agreements", "--silent"}, assert.AnError)

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{ID: "Microsoft.VisualStudioCode"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
}

// --- PackageStep Explain with Source ---

func TestPackageStep_Explain_WithSource(t *testing.T) {
	t.Parallel()

	pkg := Package{ID: "Microsoft.VisualStudioCode", Source: "msstore"}
	step := NewPackageStep(pkg, nil, nil)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.Contains(t, explanation.Detail(), "msstore")
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
			pkg:             Package{ID: "Microsoft.VisualStudioCode"},
			expectedName:    "Microsoft.VisualStudioCode",
			expectedVersion: "",
		},
		{
			name:            "with version",
			pkg:             Package{ID: "Git.Git", Version: "2.43.0"},
			expectedName:    "Git.Git",
			expectedVersion: "2.43.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			step := NewPackageStep(tc.pkg, nil, nil)

			info, ok := step.LockInfo()

			assert.True(t, ok)
			assert.Equal(t, "winget", info.Provider)
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
			pkg:  Package{ID: "Microsoft.VisualStudioCode"},
			plat: platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:  "winget",
			result: ports.CommandResult{
				ExitCode: 0,
				Stdout:   "Name                          Id                           Version\n-------------------------------------------------------------------\nMicrosoft Visual Studio Code  Microsoft.VisualStudioCode   1.85.0\n",
			},
			expectedVersion: "1.85.0",
			expectedFound:   true,
		},
		{
			name: "not found - empty output",
			pkg:  Package{ID: "Microsoft.VisualStudioCode"},
			plat: platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:  "winget",
			result: ports.CommandResult{
				ExitCode: 0,
				Stdout:   "Name  Id  Version\n---\n",
			},
			expectedVersion: "",
			expectedFound:   false,
		},
		{
			name:            "command not found",
			pkg:             Package{ID: "Microsoft.VisualStudioCode"},
			plat:            platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:             "winget",
			err:             exec.ErrNotFound,
			expectedVersion: "",
			expectedFound:   false,
		},
		{
			name: "run failure - not success",
			pkg:  Package{ID: "Microsoft.VisualStudioCode"},
			plat: platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:  "winget",
			result: ports.CommandResult{
				ExitCode: 1,
				Stderr:   "Error",
			},
			expectedVersion: "",
			expectedFound:   false,
		},
		{
			name:            "runner error - not command not found",
			pkg:             Package{ID: "Microsoft.VisualStudioCode"},
			plat:            platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:             "winget",
			err:             assert.AnError,
			expectedVersion: "",
			expectedFound:   false,
			expectedErr:     true,
		},
		{
			name: "WSL uses winget.exe",
			pkg:  Package{ID: "Microsoft.VisualStudioCode"},
			plat: platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c"),
			cmd:  "winget.exe",
			result: ports.CommandResult{
				ExitCode: 0,
				Stdout:   "Name                          Id                           Version\n-------------------------------------------------------------------\nMicrosoft Visual Studio Code  Microsoft.VisualStudioCode   1.85.0\n",
			},
			expectedVersion: "1.85.0",
			expectedFound:   true,
		},
		{
			name: "package listed but no version column",
			pkg:  Package{ID: "Microsoft.VisualStudioCode"},
			plat: platform.New(platform.OSWindows, "amd64", platform.EnvNative),
			cmd:  "winget",
			result: ports.CommandResult{
				ExitCode: 0,
				Stdout:   "Microsoft.VisualStudioCode\n",
			},
			expectedVersion: "",
			expectedFound:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runner := mocks.NewCommandRunner()
			args := []string{"list", "--id", tc.pkg.ID, "--exact", "--accept-source-agreements"}
			if tc.err != nil {
				runner.AddError(tc.cmd, args, tc.err)
			} else {
				runner.AddResult(tc.cmd, args, tc.result)
			}

			step := NewPackageStep(tc.pkg, runner, tc.plat)
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
