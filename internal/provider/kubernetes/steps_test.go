package kubernetes_test

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/kubernetes"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PluginStep Tests
// =============================================================================

func TestPluginStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := kubernetes.NewPluginStep("ctx", runner)

	assert.Equal(t, "kubernetes:plugin:ctx", step.ID().String())
}

func TestPluginStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := kubernetes.NewPluginStep("ctx", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestPluginStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"krew", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "ctx\nns\nstern\n",
	})

	step := kubernetes.NewPluginStep("ctx", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPluginStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"krew", "list"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "ns\nstern\n",
	})

	step := kubernetes.NewPluginStep("ctx", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPluginStep_Check_KrewNotInstalled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// Don't register any result - the mock will return an error

	step := kubernetes.NewPluginStep("ctx", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err) // Should not error, just needs apply
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPluginStep_Check_CommandFailed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"krew", "list"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "permission denied",
	})

	step := kubernetes.NewPluginStep("ctx", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestPluginStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := kubernetes.NewPluginStep("ctx", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "plugin", diff.Resource())
	assert.Equal(t, "ctx", diff.Name())
}

func TestPluginStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"krew", "install", "ctx"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := kubernetes.NewPluginStep("ctx", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestPluginStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"krew", "install", "ctx"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "plugin not found",
	})

	step := kubernetes.NewPluginStep("ctx", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin not found")
}

func TestPluginStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := kubernetes.NewPluginStep("ctx", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Install kubectl Plugin", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "ctx")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// ContextStep Tests
// =============================================================================

func TestContextStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	ctx := kubernetes.Context{Name: "dev", Cluster: "dev-cluster"}
	step := kubernetes.NewContextStep(ctx, runner)

	assert.Equal(t, "kubernetes:context:dev", step.ID().String())
}

func TestContextStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	ctx := kubernetes.Context{Name: "dev", Cluster: "dev-cluster"}
	step := kubernetes.NewContextStep(ctx, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestContextStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "get-contexts", "-o", "name"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "dev\nprod\n",
	})

	k8sCtx := kubernetes.Context{Name: "dev", Cluster: "dev-cluster"}
	step := kubernetes.NewContextStep(k8sCtx, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestContextStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "get-contexts", "-o", "name"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "prod\n",
	})

	k8sCtx := kubernetes.Context{Name: "dev", Cluster: "dev-cluster"}
	step := kubernetes.NewContextStep(k8sCtx, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestContextStep_Check_CommandFailed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "get-contexts", "-o", "name"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "kubeconfig not found",
	})

	k8sCtx := kubernetes.Context{Name: "dev", Cluster: "dev-cluster"}
	step := kubernetes.NewContextStep(k8sCtx, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestContextStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	k8sCtx := kubernetes.Context{Name: "dev", Cluster: "dev-cluster"}
	step := kubernetes.NewContextStep(k8sCtx, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "context", diff.Resource())
	assert.Equal(t, "dev", diff.Name())
}

func TestContextStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "set-context", "dev", "--cluster=dev-cluster"}, ports.CommandResult{
		ExitCode: 0,
	})

	k8sCtx := kubernetes.Context{Name: "dev", Cluster: "dev-cluster"}
	step := kubernetes.NewContextStep(k8sCtx, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestContextStep_Apply_WithAllOptions(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "set-context", "dev", "--cluster=dev-cluster", "--user=dev-admin", "--namespace=development"}, ports.CommandResult{
		ExitCode: 0,
	})

	k8sCtx := kubernetes.Context{Name: "dev", Cluster: "dev-cluster", User: "dev-admin", Namespace: "development"}
	step := kubernetes.NewContextStep(k8sCtx, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestContextStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "set-context", "dev", "--cluster=dev-cluster"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "cluster not found",
	})

	k8sCtx := kubernetes.Context{Name: "dev", Cluster: "dev-cluster"}
	step := kubernetes.NewContextStep(k8sCtx, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster not found")
}

func TestContextStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	k8sCtx := kubernetes.Context{Name: "dev", Cluster: "dev-cluster"}
	step := kubernetes.NewContextStep(k8sCtx, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Kubernetes Context", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "dev")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// NamespaceStep Tests
// =============================================================================

func TestNamespaceStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := kubernetes.NewNamespaceStep("my-namespace", runner)

	assert.Equal(t, "kubernetes:namespace:my-namespace", step.ID().String())
}

func TestNamespaceStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := kubernetes.NewNamespaceStep("my-namespace", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestNamespaceStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "view", "--minify", "-o", "jsonpath={..namespace}"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "my-namespace",
	})

	step := kubernetes.NewNamespaceStep("my-namespace", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestNamespaceStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "view", "--minify", "-o", "jsonpath={..namespace}"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "default",
	})

	step := kubernetes.NewNamespaceStep("my-namespace", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestNamespaceStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "view", "--minify", "-o", "jsonpath={..namespace}"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "default",
	})

	step := kubernetes.NewNamespaceStep("my-namespace", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "namespace", diff.Resource())
	assert.Equal(t, "default", diff.OldValue())
	assert.Equal(t, "my-namespace", diff.NewValue())
}

func TestNamespaceStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "set-context", "--current", "--namespace=my-namespace"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := kubernetes.NewNamespaceStep("my-namespace", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestNamespaceStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("kubectl", []string{"config", "set-context", "--current", "--namespace=my-namespace"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "no current context",
	})

	step := kubernetes.NewNamespaceStep("my-namespace", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no current context")
}

func TestNamespaceStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := kubernetes.NewNamespaceStep("my-namespace", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Set Default Namespace", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "my-namespace")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}
