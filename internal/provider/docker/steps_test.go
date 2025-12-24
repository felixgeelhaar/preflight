package docker

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallStep_ID(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewInstallStep(runner)
	assert.Equal(t, "docker:install", step.ID().String())
}

func TestInstallStep_Check_Installed(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("docker", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Docker version 24.0.7, build afdd53b",
	})

	step := NewInstallStep(runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestInstallStep_Check_NotInstalled(t *testing.T) {
	runner := mocks.NewCommandRunner()
	// No result registered = command not found

	step := NewInstallStep(runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestInstallStep_Plan(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewInstallStep(runner)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "docker", diff.Resource())
}

func TestInstallStep_Explain(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewInstallStep(runner)

	explanation := step.Explain(compiler.ExplainContext{})
	assert.Equal(t, "Install Docker Desktop", explanation.Summary())
	assert.NotEmpty(t, explanation.Detail())
	assert.NotEmpty(t, explanation.DocLinks())
}

func TestBuildKitStep_ID(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewBuildKitStep(true, runner)
	assert.Equal(t, "docker:buildkit", step.ID().String())
}

func TestBuildKitStep_DependsOn_WithDocker(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewBuildKitStep(true, runner)
	deps := step.DependsOn()
	require.Len(t, deps, 1)
	assert.Equal(t, "docker:install", deps[0].String())
}

func TestBuildKitStep_DependsOn_WithoutDocker(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewBuildKitStep(false, runner)
	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestBuildKitStep_Check_Enabled(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("docker", []string{"buildx", "version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "github.com/docker/buildx v0.11.2",
	})

	step := NewBuildKitStep(true, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestBuildKitStep_Check_NotEnabled(t *testing.T) {
	runner := mocks.NewCommandRunner()
	// No result registered

	step := NewBuildKitStep(true, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestBuildKitStep_Plan(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewBuildKitStep(true, runner)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "buildkit", diff.Resource())
}

func TestBuildKitStep_Apply_Success(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("docker", []string{"buildx", "install"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := NewBuildKitStep(true, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestBuildKitStep_Explain(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewBuildKitStep(true, runner)

	explanation := step.Explain(compiler.ExplainContext{})
	assert.Equal(t, "Enable Docker BuildKit", explanation.Summary())
	assert.NotEmpty(t, explanation.Detail())
}

func TestKubernetesStep_ID(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewKubernetesStep(true, runner)
	assert.Equal(t, "docker:kubernetes", step.ID().String())
}

func TestKubernetesStep_DependsOn_WithDocker(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewKubernetesStep(true, runner)
	deps := step.DependsOn()
	require.Len(t, deps, 1)
	assert.Equal(t, "docker:install", deps[0].String())
}

func TestKubernetesStep_Check_Enabled(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "get-contexts", "-o", "name"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "docker-desktop\nminikube",
	})
	runner.AddResult("kubectl", []string{"--context", "docker-desktop", "cluster-info"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "Kubernetes master is running",
	})

	step := NewKubernetesStep(true, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestKubernetesStep_Check_NotEnabled(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "get-contexts", "-o", "name"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "minikube",
	})

	step := NewKubernetesStep(true, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKubernetesStep_Plan(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewKubernetesStep(true, runner)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "kubernetes", diff.Resource())
}

func TestKubernetesStep_Apply_RequiresGUI(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewKubernetesStep(true, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Docker Desktop settings")
}

func TestKubernetesStep_Explain(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewKubernetesStep(true, runner)

	explanation := step.Explain(compiler.ExplainContext{})
	assert.Equal(t, "Enable Docker Desktop Kubernetes", explanation.Summary())
	assert.NotEmpty(t, explanation.Detail())
}

func TestContextStep_ID(t *testing.T) {
	runner := mocks.NewCommandRunner()
	dockerContext := Context{Name: "production", Host: "ssh://user@host"}
	step := NewContextStep(dockerContext, true, runner)
	assert.Equal(t, "docker:context:production", step.ID().String())
}

func TestContextStep_DependsOn_WithDocker(t *testing.T) {
	runner := mocks.NewCommandRunner()
	dockerContext := Context{Name: "production", Host: "ssh://user@host"}
	step := NewContextStep(dockerContext, true, runner)
	deps := step.DependsOn()
	require.Len(t, deps, 1)
	assert.Equal(t, "docker:install", deps[0].String())
}

func TestContextStep_Check_Exists(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("docker", []string{"context", "ls", "--format", "{{.Name}}"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "default\nproduction\nstaging",
	})

	dockerContext := Context{Name: "production", Host: "ssh://user@host"}
	step := NewContextStep(dockerContext, true, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestContextStep_Check_NotExists(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("docker", []string{"context", "ls", "--format", "{{.Name}}"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "default",
	})

	dockerContext := Context{Name: "production", Host: "ssh://user@host"}
	step := NewContextStep(dockerContext, true, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestContextStep_Plan(t *testing.T) {
	runner := mocks.NewCommandRunner()
	dockerContext := Context{Name: "production", Host: "ssh://user@host"}
	step := NewContextStep(dockerContext, true, runner)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "context", diff.Resource())
}

func TestContextStep_Apply_Success(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("docker", []string{"context", "create", "production", "--docker", "host=ssh://user@host"}, ports.CommandResult{
		ExitCode: 0,
	})

	dockerContext := Context{Name: "production", Host: "ssh://user@host"}
	step := NewContextStep(dockerContext, true, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestContextStep_Apply_WithDescription(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("docker", []string{"context", "create", "production", "--docker", "host=ssh://user@host", "--description", "Production server"}, ports.CommandResult{
		ExitCode: 0,
	})

	dockerContext := Context{
		Name:        "production",
		Host:        "ssh://user@host",
		Description: "Production server",
	}
	step := NewContextStep(dockerContext, true, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestContextStep_Apply_SetDefault(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("docker", []string{"context", "create", "production", "--docker", "host=ssh://user@host"}, ports.CommandResult{
		ExitCode: 0,
	})
	runner.AddResult("docker", []string{"context", "use", "production"}, ports.CommandResult{
		ExitCode: 0,
	})

	dockerContext := Context{
		Name:    "production",
		Host:    "ssh://user@host",
		Default: true,
	}
	step := NewContextStep(dockerContext, true, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestContextStep_Explain(t *testing.T) {
	runner := mocks.NewCommandRunner()
	dockerContext := Context{Name: "production", Host: "ssh://user@host"}
	step := NewContextStep(dockerContext, true, runner)

	explanation := step.Explain(compiler.ExplainContext{})
	assert.Equal(t, "Create Docker Context", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "production")
	assert.Contains(t, explanation.Detail(), "ssh://user@host")
}
