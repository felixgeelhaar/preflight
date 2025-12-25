package kubernetes

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// PluginStep represents a kubectl krew plugin installation step.
type PluginStep struct {
	plugin string
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewPluginStep creates a new PluginStep.
func NewPluginStep(plugin string, runner ports.CommandRunner) *PluginStep {
	id := compiler.MustNewStepID("kubernetes:plugin:" + plugin)
	return &PluginStep{
		plugin: plugin,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *PluginStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *PluginStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the plugin is already installed.
func (s *PluginStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "kubectl", "krew", "list")
	if err != nil {
		// krew might not be installed
		return compiler.StatusNeedsApply, nil //nolint:nilerr // krew not installed means we need to apply
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("kubectl krew list failed: %s", result.Stderr)
	}

	plugins := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, p := range plugins {
		if strings.TrimSpace(p) == s.plugin {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *PluginStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "plugin", s.plugin, "", s.plugin), nil
}

// Apply installs the plugin.
func (s *PluginStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "kubectl", "krew", "install", s.plugin)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("kubectl krew install failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *PluginStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install kubectl Plugin",
		fmt.Sprintf("Installs the %s plugin via krew", s.plugin),
		[]string{
			fmt.Sprintf("https://krew.sigs.k8s.io/plugins/?search=%s", s.plugin),
			"https://krew.sigs.k8s.io/",
		},
	).WithTradeoffs([]string{
		"+ Extends kubectl capabilities",
		"+ Easy updates via 'kubectl krew upgrade'",
		"- Requires krew to be installed",
	})
}

// ContextStep represents a Kubernetes context configuration step.
type ContextStep struct {
	context Context
	id      compiler.StepID
	runner  ports.CommandRunner
}

// NewContextStep creates a new ContextStep.
func NewContextStep(context Context, runner ports.CommandRunner) *ContextStep {
	id := compiler.MustNewStepID("kubernetes:context:" + context.Name)
	return &ContextStep{
		context: context,
		id:      id,
		runner:  runner,
	}
}

// ID returns the step identifier.
func (s *ContextStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ContextStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the context exists.
func (s *ContextStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "kubectl", "config", "get-contexts", "-o", "name")
	if err != nil {
		return compiler.StatusUnknown, err
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("kubectl config get-contexts failed: %s", result.Stderr)
	}

	contexts := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, c := range contexts {
		if strings.TrimSpace(c) == s.context.Name {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ContextStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "context", s.context.Name, "", s.context.Cluster), nil
}

// Apply creates the context.
func (s *ContextStep) Apply(ctx compiler.RunContext) error {
	args := []string{"config", "set-context", s.context.Name}

	if s.context.Cluster != "" {
		args = append(args, "--cluster="+s.context.Cluster)
	}
	if s.context.User != "" {
		args = append(args, "--user="+s.context.User)
	}
	if s.context.Namespace != "" {
		args = append(args, "--namespace="+s.context.Namespace)
	}

	result, err := s.runner.Run(ctx.Context(), "kubectl", args...)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("kubectl config set-context failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *ContextStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Kubernetes Context",
		fmt.Sprintf("Creates or updates the %s context", s.context.Name),
		[]string{
			"https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/",
		},
	).WithTradeoffs([]string{
		"+ Quick context switching between clusters",
		"+ Namespace defaults per context",
	})
}

// NamespaceStep represents a default namespace configuration step.
type NamespaceStep struct {
	namespace string
	id        compiler.StepID
	runner    ports.CommandRunner
}

// NewNamespaceStep creates a new NamespaceStep.
func NewNamespaceStep(namespace string, runner ports.CommandRunner) *NamespaceStep {
	id := compiler.MustNewStepID("kubernetes:namespace:" + namespace)
	return &NamespaceStep{
		namespace: namespace,
		id:        id,
		runner:    runner,
	}
}

// ID returns the step identifier.
func (s *NamespaceStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *NamespaceStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the namespace is already set as default.
func (s *NamespaceStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "kubectl", "config", "view", "--minify", "-o", "jsonpath={..namespace}")
	if err != nil {
		return compiler.StatusUnknown, err
	}

	currentNamespace := strings.TrimSpace(result.Stdout)
	if currentNamespace == s.namespace {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *NamespaceStep) Plan(ctx compiler.RunContext) (compiler.Diff, error) {
	result, _ := s.runner.Run(ctx.Context(), "kubectl", "config", "view", "--minify", "-o", "jsonpath={..namespace}")
	currentNamespace := strings.TrimSpace(result.Stdout)
	return compiler.NewDiff(compiler.DiffTypeModify, "namespace", "default", currentNamespace, s.namespace), nil
}

// Apply sets the default namespace.
func (s *NamespaceStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "kubectl", "config", "set-context", "--current", "--namespace="+s.namespace)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("kubectl config set-context --namespace failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *NamespaceStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Set Default Namespace",
		fmt.Sprintf("Sets the default namespace to %s for the current context", s.namespace),
		[]string{
			"https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/",
		},
	).WithTradeoffs([]string{
		"+ No need to specify -n flag for every command",
		"- Might accidentally affect wrong namespace",
	})
}
