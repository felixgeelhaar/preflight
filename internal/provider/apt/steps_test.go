package apt_test

import (
	"context"
	"os/exec"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/apt"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPPAStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewPPAStep("ppa:git-core/ppa", runner)

	assert.Equal(t, "apt:ppa:ppa-git-core/ppa", step.ID().String())
}

func TestPPAStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewPPAStep("ppa:git-core/ppa", runner)

	assert.Equal(t, []compiler.StepID{compiler.MustNewStepID("apt:ready")}, step.DependsOn())
}

func TestPPAStep_Check_NotAdded(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("apt-cache", []string{"policy"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "",
		Stderr:   "",
	})

	step := apt.NewPPAStep("ppa:git-core/ppa", runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPPAStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewPPAStep("ppa:git-core/ppa", runner)

	ctx := compiler.NewRunContext(context.TODO())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, diff.Summary())
	assert.Contains(t, diff.Summary(), "ppa:git-core/ppa")
}

func TestPPAStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewPPAStep("ppa:git-core/ppa", runner)

	ctx := compiler.NewExplainContext()
	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "ppa:git-core/ppa")
}

func TestPackageStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)

	assert.Equal(t, "apt:package:git", step.ID().String())
}

func TestPackageStep_ID_WithVersion(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	pkg := apt.Package{Name: "nodejs", Version: "18.0.0"}
	step := apt.NewPackageStep(pkg, runner)

	assert.Equal(t, "apt:package:nodejs", step.ID().String())
}

func TestPackageStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)

	assert.Equal(t, []compiler.StepID{compiler.MustNewStepID("apt:update")}, step.DependsOn())
}

func TestPackageStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("dpkg-query", []string{"-W", "-f=${Package}\t${Version}\t${db:Status-Status}\n", "git"}, ports.CommandResult{
		ExitCode: 1,
		Stdout:   "",
		Stderr:   "dpkg-query: no packages found matching git",
	})

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPackageStep_Check_Installed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("dpkg-query", []string{"-W", "-f=${Package}\t${Version}\t${db:Status-Status}\n", "git"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "git	2.39.0	installed",
		Stderr:   "",
	})

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPackageStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewRunContext(context.TODO())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, diff.Summary())
}

func TestPackageStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewExplainContext()
	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "git")
}

func TestPPAStep_Check_AlreadyAdded(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("apt-cache", []string{"policy"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "500 http://ppa.launchpad.net/git-core/ppa/ubuntu jammy/main amd64",
		Stderr:   "",
	})

	step := apt.NewPPAStep("ppa:git-core/ppa", runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPPAStep_Check_Error(t *testing.T) {
	t.Parallel()

	// Don't add any result - the mock will return an error
	runner := mocks.NewCommandRunner()

	step := apt.NewPPAStep("ppa:git-core/ppa", runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestPPAStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("sudo", []string{"add-apt-repository", "-y", "ppa:git-core/ppa"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "",
		Stderr:   "",
	})

	step := apt.NewPPAStep("ppa:git-core/ppa", runner)

	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestPPAStep_Apply_RunnerError(t *testing.T) {
	t.Parallel()

	// Don't add any result - the mock will return an error
	runner := mocks.NewCommandRunner()
	step := apt.NewPPAStep("ppa:git-core/ppa", runner)

	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.Error(t, err)
}

func TestPPAStep_Apply_CommandFailure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("sudo", []string{"add-apt-repository", "-y", "ppa:git-core/ppa"}, ports.CommandResult{
		ExitCode: 1,
		Stdout:   "",
		Stderr:   "Permission denied",
	})

	step := apt.NewPPAStep("ppa:git-core/ppa", runner)

	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestPackageStep_Check_Error(t *testing.T) {
	t.Parallel()

	// Don't add any result - the mock will return an error
	runner := mocks.NewCommandRunner()

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestPackageStep_Plan_WithVersion(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	pkg := apt.Package{Name: "nodejs", Version: "18.0.0"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewRunContext(context.TODO())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Contains(t, diff.Summary(), "18.0.0")
}

func TestPackageStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("sudo", []string{"apt-get", "install", "-y", "git"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "",
		Stderr:   "",
	})

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestPackageStep_Apply_WithVersion(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("sudo", []string{"apt-get", "install", "-y", "nodejs=18.0.0"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "",
		Stderr:   "",
	})

	pkg := apt.Package{Name: "nodejs", Version: "18.0.0"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestPackageStep_Apply_RunnerError(t *testing.T) {
	t.Parallel()

	// Don't add any result - the mock will return an error
	runner := mocks.NewCommandRunner()
	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.Error(t, err)
}

func TestPackageStep_Apply_CommandFailure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("sudo", []string{"apt-get", "install", "-y", "git"}, ports.CommandResult{
		ExitCode: 1,
		Stdout:   "",
		Stderr:   "Package not found",
	})

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewRunContext(context.TODO())
	err := step.Apply(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestPackageStep_Explain_WithVersion(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	pkg := apt.Package{Name: "nodejs", Version: "18.0.0"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewExplainContext()
	exp := step.Explain(ctx)

	assert.NotEmpty(t, exp.Summary())
	assert.Contains(t, exp.Detail(), "nodejs")
	assert.Contains(t, exp.Detail(), "18.0.0")
}

func TestPackageStep_Check_InstalledNotContainsInstalled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("dpkg-query", []string{"-W", "-f=${Package}\t${Version}\t${db:Status-Status}\n", "git"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "git	2.39.0	config-files",
		Stderr:   "",
	})

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)

	ctx := compiler.NewRunContext(context.TODO())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

// --- ReadyStep tests ---

func TestReadyStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewReadyStep(runner)

	assert.Equal(t, "apt:ready", step.ID().String())
}

func TestReadyStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewReadyStep(runner)

	assert.Nil(t, step.DependsOn())
}

func TestReadyStep_Check(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewReadyStep(runner)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	// On macOS, apt-get is not installed, so this should return NeedsApply.
	// On Linux with apt, it returns Satisfied. Both are valid.
	assert.Contains(t, []compiler.StepStatus{compiler.StatusSatisfied, compiler.StatusNeedsApply}, status)
}

func TestReadyStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewReadyStep(runner)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "ready", diff.Name())
}

func TestReadyStep_Apply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewReadyStep(runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apt-get not found")
}

func TestReadyStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewReadyStep(runner)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)
	assert.NotEmpty(t, exp.Summary())
	assert.NotEmpty(t, exp.Detail())
}

// --- UpdateStep tests ---

func TestUpdateStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewUpdateStep(runner, nil)

	assert.Equal(t, "apt:update", step.ID().String())
}

func TestUpdateStep_DependsOn_NoDeps(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewUpdateStep(runner, nil)

	assert.Nil(t, step.DependsOn())
}

func TestUpdateStep_DependsOn_WithDeps(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	deps := []compiler.StepID{
		compiler.MustNewStepID("apt:ready"),
		compiler.MustNewStepID("apt:ppa:ppa-git-core/ppa"),
	}
	step := apt.NewUpdateStep(runner, deps)

	result := step.DependsOn()
	assert.Len(t, result, 2)
	assert.Equal(t, "apt:ready", result[0].String())
	assert.Equal(t, "apt:ppa:ppa-git-core/ppa", result[1].String())
}

func TestUpdateStep_Check(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewUpdateStep(runner, nil)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	// On macOS, apt-get is not available, so this returns NeedsApply.
	// On Linux, depends on whether update-success-stamp is recent.
	assert.Contains(t, []compiler.StepStatus{compiler.StatusSatisfied, compiler.StatusNeedsApply}, status)
}

func TestUpdateStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewUpdateStep(runner, nil)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "update", diff.Name())
}

func TestUpdateStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("sudo", []string{"apt-get", "update"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Hit:1 http://archive.ubuntu.com/ubuntu jammy InRelease",
	})

	step := apt.NewUpdateStep(runner, nil)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestUpdateStep_Apply_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// No result registered, runner returns error

	step := apt.NewUpdateStep(runner, nil)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.Error(t, err)
}

func TestUpdateStep_Apply_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("sudo", []string{"apt-get", "update"}, &exec.Error{Name: "sudo", Err: exec.ErrNotFound})

	step := apt.NewUpdateStep(runner, nil)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apt-get not found")
}

func TestUpdateStep_Apply_CommandFailure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("sudo", []string{"apt-get", "update"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "E: Could not get lock /var/lib/apt/lists/lock",
	})

	step := apt.NewUpdateStep(runner, nil)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apt-get update failed")
}

func TestUpdateStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := apt.NewUpdateStep(runner, nil)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)
	assert.NotEmpty(t, exp.Summary())
	assert.NotEmpty(t, exp.Detail())
}

// --- PPAStep Check with command-not-found ---

func TestPPAStep_Check_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("apt-cache", []string{"policy"}, &exec.Error{Name: "apt-cache", Err: exec.ErrNotFound})

	step := apt.NewPPAStep("ppa:git-core/ppa", runner)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

// --- PPAStep Apply with command-not-found ---

func TestPPAStep_Apply_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("sudo", []string{"add-apt-repository", "-y", "ppa:git-core/ppa"}, &exec.Error{Name: "sudo", Err: exec.ErrNotFound})

	step := apt.NewPPAStep("ppa:git-core/ppa", runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add-apt-repository not found")
}

// --- PackageStep Check with command-not-found ---

func TestPackageStep_Check_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("dpkg-query", []string{"-W", "-f=${Package}\t${Version}\t${db:Status-Status}\n", "git"}, &exec.Error{Name: "dpkg-query", Err: exec.ErrNotFound})

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

// --- PackageStep Apply with command-not-found ---

func TestPackageStep_Apply_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("sudo", []string{"apt-get", "install", "-y", "git"}, &exec.Error{Name: "sudo", Err: exec.ErrNotFound})

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apt-get not found")
}

// --- PackageStep LockInfo tests ---

func TestPackageStep_LockInfo(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	pkg := apt.Package{Name: "git", Version: "2.39.0"}
	step := apt.NewPackageStep(pkg, runner)

	info, ok := step.LockInfo()
	assert.True(t, ok)
	assert.Equal(t, "apt", info.Provider)
	assert.Equal(t, "git", info.Name)
	assert.Equal(t, "2.39.0", info.Version)
}

func TestPackageStep_LockInfo_NoVersion(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	pkg := apt.Package{Name: "curl"}
	step := apt.NewPackageStep(pkg, runner)

	info, ok := step.LockInfo()
	assert.True(t, ok)
	assert.Equal(t, "apt", info.Provider)
	assert.Equal(t, "curl", info.Name)
	assert.Equal(t, "", info.Version)
}

// --- PackageStep InstalledVersion tests ---

func TestPackageStep_InstalledVersion_Found(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("dpkg-query", []string{"-W", "-f=${Version}\n", "git"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "1:2.39.2-1ubuntu1.1\n",
	})

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)
	ctx := compiler.NewRunContext(context.TODO())

	version, found, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1:2.39.2-1ubuntu1.1", version)
}

func TestPackageStep_InstalledVersion_NotInstalled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("dpkg-query", []string{"-W", "-f=${Version}\n", "nonexistent"}, ports.CommandResult{
		ExitCode: 1,
		Stdout:   "",
	})

	pkg := apt.Package{Name: "nonexistent"}
	step := apt.NewPackageStep(pkg, runner)
	ctx := compiler.NewRunContext(context.TODO())

	version, found, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, "", version)
}

func TestPackageStep_InstalledVersion_CommandNotFound(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("dpkg-query", []string{"-W", "-f=${Version}\n", "git"}, &exec.Error{Name: "dpkg-query", Err: exec.ErrNotFound})

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)
	ctx := compiler.NewRunContext(context.TODO())

	version, found, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, "", version)
}

func TestPackageStep_InstalledVersion_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// No result registered, returns generic error

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)
	ctx := compiler.NewRunContext(context.TODO())

	_, _, err := step.InstalledVersion(ctx)
	require.Error(t, err)
}

func TestPackageStep_InstalledVersion_EmptyOutput(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("dpkg-query", []string{"-W", "-f=${Version}\n", "git"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "\n",
	})

	pkg := apt.Package{Name: "git"}
	step := apt.NewPackageStep(pkg, runner)
	ctx := compiler.NewRunContext(context.TODO())

	version, found, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, "", version)
}

// --- Additional config edge case tests ---

func TestParseConfig_PackageMapMissingName(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			map[string]interface{}{
				"version": "1.0.0",
			},
		},
	}
	_, err := apt.ParseConfig(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestParseConfig_PackageInvalidType(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"packages": []interface{}{
			123, // invalid type
		},
	}
	_, err := apt.ParseConfig(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "string or object")
}

func TestParseConfig_PPANotString(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"ppas": []interface{}{
			123, // invalid type
		},
	}
	_, err := apt.ParseConfig(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "string")
}
