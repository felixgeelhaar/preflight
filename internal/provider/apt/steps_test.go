package apt_test

import (
	"context"
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
