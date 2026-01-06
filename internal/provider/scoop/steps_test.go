package scoop

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

// BucketStep tests

func TestBucketStep_ID(t *testing.T) {
	t.Parallel()

	bucket := Bucket{Name: "extras"}
	step := NewBucketStep(bucket, nil, nil)

	assert.Equal(t, "scoop:bucket:extras", step.ID().String())
}

func TestBucketStep_DependsOn(t *testing.T) {
	t.Parallel()

	bucket := Bucket{Name: "extras"}
	step := NewBucketStep(bucket, nil, nil)

	deps := step.DependsOn()

	assert.Equal(t, []compiler.StepID{compiler.MustNewStepID(scoopInstallStepID)}, deps)
}

func TestBucketStep_Check_AlreadyAdded(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop", []string{"bucket", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Name    Source                              Updated\n----    ------                              -------\nextras  https://github.com/ScoopInstaller/Extras  2024-01-01\nmain    https://github.com/ScoopInstaller/Main    2024-01-01\n",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	bucket := Bucket{Name: "extras"}
	step := NewBucketStep(bucket, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestBucketStep_Check_NotAdded(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop", []string{"bucket", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Name    Source                              Updated\n----    ------                              -------\nmain    https://github.com/ScoopInstaller/Main    2024-01-01\n",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	bucket := Bucket{Name: "extras"}
	step := NewBucketStep(bucket, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestBucketStep_Check_WSL_UsesScoopCmd(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop.cmd", []string{"bucket", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "extras\n",
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	bucket := Bucket{Name: "extras"}
	step := NewBucketStep(bucket, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestBucketStep_Plan(t *testing.T) {
	t.Parallel()

	bucket := Bucket{Name: "extras"}
	step := NewBucketStep(bucket, nil, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "extras", diff.Name())
}

func TestBucketStep_Apply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop", []string{"bucket", "add", "extras"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	bucket := Bucket{Name: "extras"}
	step := NewBucketStep(bucket, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
	calls := runner.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "scoop", calls[0].Command)
}

func TestBucketStep_Apply_WithURL(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop", []string{"bucket", "add", "custom", "https://github.com/user/bucket"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	bucket := Bucket{Name: "custom", URL: "https://github.com/user/bucket"}
	step := NewBucketStep(bucket, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestBucketStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop", []string{"bucket", "add", "invalid"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Bucket not found",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	bucket := Bucket{Name: "invalid"}
	step := NewBucketStep(bucket, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scoop bucket add invalid failed")
}

func TestBucketStep_Apply_InvalidName(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	// Bucket name that fails validation (starts with number)
	bucket := Bucket{Name: "123invalid"}
	step := NewBucketStep(bucket, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bucket name")
}

func TestBucketStep_Explain(t *testing.T) {
	t.Parallel()

	bucket := Bucket{Name: "extras"}
	step := NewBucketStep(bucket, nil, nil)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.NotEmpty(t, explanation.Summary())
	assert.NotEmpty(t, explanation.Detail())
	assert.Contains(t, explanation.Detail(), "extras")
}

func TestBucketStep_scoopCommand_Windows(t *testing.T) {
	t.Parallel()

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	bucket := Bucket{Name: "extras"}
	step := NewBucketStep(bucket, nil, plat)

	assert.Equal(t, "scoop", step.scoopCommand())
}

func TestBucketStep_scoopCommand_WSL(t *testing.T) {
	t.Parallel()

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	bucket := Bucket{Name: "extras"}
	step := NewBucketStep(bucket, nil, plat)

	assert.Equal(t, "scoop.cmd", step.scoopCommand())
}

// PackageStep tests

func TestPackageStep_ID(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, nil)

	assert.Equal(t, "scoop:package:git", step.ID().String())
}

func TestPackageStep_ID_WithBucket(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "neovim", Bucket: "extras"}
	step := NewPackageStep(pkg, nil, nil)

	assert.Equal(t, "scoop:package:extras/neovim", step.ID().String())
}

func TestPackageStep_DependsOn_NoBucket(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, nil)

	deps := step.DependsOn()

	assert.Equal(t, []compiler.StepID{compiler.MustNewStepID(scoopInstallStepID)}, deps)
}

func TestPackageStep_DependsOn_WithBucket(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "neovim", Bucket: "extras"}
	step := NewPackageStep(pkg, nil, nil)

	deps := step.DependsOn()

	require.Len(t, deps, 2)
	assert.Equal(t, scoopInstallStepID, deps[0].String())
	assert.Equal(t, "scoop:bucket:extras", deps[1].String())
}

func TestPackageStep_Check_Installed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop", []string{"list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Installed apps:\n\ngit (2.43.0) [main]\ncurl (8.5.0) [main]\n",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPackageStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop", []string{"list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Installed apps:\n\ncurl (8.5.0) [main]\n",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPackageStep_Plan(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "git", diff.Name())
	assert.Equal(t, "latest", diff.NewValue())
}

func TestPackageStep_Plan_WithVersion(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "git", Version: "2.43.0"}
	step := NewPackageStep(pkg, nil, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, "2.43.0", diff.NewValue())
}

func TestPackageStep_Apply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop", []string{"install", "git"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
	calls := runner.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "scoop", calls[0].Command)
}

func TestPackageStep_Apply_WithBucket(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop", []string{"install", "extras/neovim"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "neovim", Bucket: "extras"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestPackageStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop", []string{"install", "invalid-pkg"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Package not found",
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "invalid-pkg"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scoop install invalid-pkg failed")
}

func TestPackageStep_Apply_InvalidName(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	// Package name with slash is valid for step ID but invalid for package name validation
	// (slash is allowed in step IDs but not in package names)
	pkg := Package{Name: "invalid/name"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestPackageStep_Apply_WSL_UsesScoopCmd(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("scoop.cmd", []string{"install", "git"}, ports.CommandResult{
		ExitCode: 0,
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
	calls := runner.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "scoop.cmd", calls[0].Command)
}

func TestPackageStep_Explain(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, nil)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.NotEmpty(t, explanation.Summary())
	assert.NotEmpty(t, explanation.Detail())
	assert.Contains(t, explanation.Detail(), "git")
}

func TestPackageStep_Explain_WithBucket(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "neovim", Bucket: "extras"}
	step := NewPackageStep(pkg, nil, nil)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.Contains(t, explanation.Detail(), "extras")
}

func TestPackageStep_Explain_WSL(t *testing.T) {
	t.Parallel()

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	pkg := Package{Name: "git"}
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

func TestPackageStep_scoopCommand_Windows(t *testing.T) {
	t.Parallel()

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, plat)

	assert.Equal(t, "scoop", step.scoopCommand())
}

func TestPackageStep_scoopCommand_WSL(t *testing.T) {
	t.Parallel()

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, plat)

	assert.Equal(t, "scoop.cmd", step.scoopCommand())
}

func TestPackageStep_scoopCommand_NilPlatform(t *testing.T) {
	t.Parallel()

	pkg := Package{Name: "git"}
	step := NewPackageStep(pkg, nil, nil)

	assert.Equal(t, "scoop", step.scoopCommand())
}
