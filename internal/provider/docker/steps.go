package docker

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// InstallStep represents Docker Desktop installation.
type InstallStep struct {
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewInstallStep creates a new Docker Desktop installation step.
func NewInstallStep(runner ports.CommandRunner) *InstallStep {
	id := compiler.MustNewStepID("docker:install")
	return &InstallStep{
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *InstallStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *InstallStep) DependsOn() []compiler.StepID {
	// Docker Desktop installation depends on Homebrew on macOS
	if runtime.GOOS == "darwin" {
		return []compiler.StepID{compiler.MustNewStepID("brew:cask:docker")}
	}
	return nil
}

// Check determines if Docker is installed.
func (s *InstallStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "docker", "--version")
	if err != nil {
		// Command not found means Docker needs to be installed
		return compiler.StatusNeedsApply, nil //nolint:nilerr // intentional: command failure = needs apply
	}
	if result.Success() && strings.Contains(result.Stdout, "Docker version") {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *InstallStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "docker", "Docker Desktop", "", "installed"), nil
}

// Apply executes Docker Desktop installation.
func (s *InstallStep) Apply(ctx compiler.RunContext) error {
	switch runtime.GOOS {
	case "darwin":
		// Docker Desktop is installed via brew cask
		result, err := s.runner.Run(ctx.Context(), "brew", "install", "--cask", "docker")
		if err != nil {
			return err
		}
		if !result.Success() {
			return fmt.Errorf("brew install docker failed: %s", result.Stderr)
		}
	case "linux":
		// Install Docker Engine on Linux using convenience script
		result, err := s.runner.Run(ctx.Context(), "sh", "-c",
			"curl -fsSL https://get.docker.com | sh")
		if err != nil {
			return err
		}
		if !result.Success() {
			return fmt.Errorf("docker installation failed: %s", result.Stderr)
		}
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *InstallStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Docker Desktop",
		"Installs Docker Desktop, providing the Docker runtime, Docker Compose, and container management tools.",
		[]string{
			"https://docs.docker.com/desktop/",
			"https://docs.docker.com/get-started/",
		},
	).WithTradeoffs([]string{
		"+ Enables containerized development and deployment",
		"+ Includes Docker Compose for multi-container applications",
		"+ Provides a GUI for container management",
		"- Uses significant system resources when running",
		"- Requires periodic updates for security patches",
	})
}

// BuildKitStep configures BuildKit for improved Docker builds.
type BuildKitStep struct {
	id             compiler.StepID
	requiresDocker bool
	runner         ports.CommandRunner
}

// NewBuildKitStep creates a BuildKit configuration step.
func NewBuildKitStep(requiresDocker bool, runner ports.CommandRunner) *BuildKitStep {
	id := compiler.MustNewStepID("docker:buildkit")
	return &BuildKitStep{
		id:             id,
		requiresDocker: requiresDocker,
		runner:         runner,
	}
}

// ID returns the step identifier.
func (s *BuildKitStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *BuildKitStep) DependsOn() []compiler.StepID {
	if s.requiresDocker {
		return []compiler.StepID{compiler.MustNewStepID("docker:install")}
	}
	return nil
}

// Check determines if BuildKit is enabled.
func (s *BuildKitStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "docker", "buildx", "version")
	if err != nil {
		// buildx not available means BuildKit needs to be enabled
		return compiler.StatusNeedsApply, nil //nolint:nilerr // intentional: command failure = needs apply
	}
	if result.Success() {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *BuildKitStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "buildkit", "BuildKit", "disabled", "enabled"), nil
}

// Apply enables BuildKit.
func (s *BuildKitStep) Apply(ctx compiler.RunContext) error {
	// Ensure buildx is the default builder
	result, err := s.runner.Run(ctx.Context(), "docker", "buildx", "install")
	if err != nil {
		return err
	}
	if !result.Success() {
		// buildx install might not be available in older versions, try creating a builder
		_, err = s.runner.Run(ctx.Context(), "docker", "buildx", "create", "--use", "--name", "preflight-builder")
		if err != nil {
			return err
		}
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *BuildKitStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Enable Docker BuildKit",
		"Enables BuildKit, Docker's next-generation build system with improved performance, caching, and build features.",
		[]string{
			"https://docs.docker.com/build/buildkit/",
			"https://docs.docker.com/build/",
		},
	).WithTradeoffs([]string{
		"+ Faster builds with better caching",
		"+ Support for advanced Dockerfile features",
		"+ Parallel build stages",
		"+ Improved security with rootless builds",
	})
}

// KubernetesStep enables Kubernetes in Docker Desktop.
type KubernetesStep struct {
	id             compiler.StepID
	requiresDocker bool
	runner         ports.CommandRunner
}

// NewKubernetesStep creates a Kubernetes enablement step.
func NewKubernetesStep(requiresDocker bool, runner ports.CommandRunner) *KubernetesStep {
	id := compiler.MustNewStepID("docker:kubernetes")
	return &KubernetesStep{
		id:             id,
		requiresDocker: requiresDocker,
		runner:         runner,
	}
}

// ID returns the step identifier.
func (s *KubernetesStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *KubernetesStep) DependsOn() []compiler.StepID {
	if s.requiresDocker {
		return []compiler.StepID{compiler.MustNewStepID("docker:install")}
	}
	return nil
}

// Check determines if Kubernetes is enabled in Docker Desktop.
func (s *KubernetesStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	// Check if kubectl can connect to docker-desktop context
	result, err := s.runner.Run(ctx.Context(), "kubectl", "config", "get-contexts", "-o", "name")
	if err != nil {
		// kubectl not available or failed means Kubernetes needs to be enabled
		return compiler.StatusNeedsApply, nil //nolint:nilerr // intentional: command failure = needs apply
	}
	if result.Success() && strings.Contains(result.Stdout, "docker-desktop") {
		// Verify cluster is accessible
		checkResult, err := s.runner.Run(ctx.Context(), "kubectl", "--context", "docker-desktop", "cluster-info")
		if err == nil && checkResult.Success() {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *KubernetesStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "kubernetes", "Docker Desktop Kubernetes", "disabled", "enabled"), nil
}

// Apply enables Kubernetes in Docker Desktop.
func (s *KubernetesStep) Apply(_ compiler.RunContext) error {
	// Docker Desktop Kubernetes must be enabled through the GUI or settings file
	// We can't directly enable it via CLI, so we inform the user
	return fmt.Errorf("kubernetes must be enabled through Docker Desktop settings. " +
		"Open Docker Desktop → Settings → Kubernetes → Enable Kubernetes")
}

// Explain provides a human-readable explanation.
func (s *KubernetesStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Enable Docker Desktop Kubernetes",
		"Enables the built-in Kubernetes cluster in Docker Desktop for local Kubernetes development.",
		[]string{
			"https://docs.docker.com/desktop/kubernetes/",
			"https://kubernetes.io/docs/home/",
		},
	).WithTradeoffs([]string{
		"+ Local Kubernetes cluster for development",
		"+ Integrated with Docker Desktop networking",
		"+ Easy to reset and reconfigure",
		"- Uses additional system resources (2-4GB RAM)",
		"- Single-node cluster only",
	})
}

// ContextStep creates a Docker context for multi-host management.
type ContextStep struct {
	context        Context
	id             compiler.StepID
	requiresDocker bool
	runner         ports.CommandRunner
}

// NewContextStep creates a Docker context configuration step.
func NewContextStep(context Context, requiresDocker bool, runner ports.CommandRunner) *ContextStep {
	id := compiler.MustNewStepID("docker:context:" + context.Name)
	return &ContextStep{
		context:        context,
		id:             id,
		requiresDocker: requiresDocker,
		runner:         runner,
	}
}

// ID returns the step identifier.
func (s *ContextStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ContextStep) DependsOn() []compiler.StepID {
	if s.requiresDocker {
		return []compiler.StepID{compiler.MustNewStepID("docker:install")}
	}
	return nil
}

// Check determines if the context exists.
func (s *ContextStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "docker", "context", "ls", "--format", "{{.Name}}")
	if err != nil {
		return compiler.StatusUnknown, err
	}
	if result.Success() {
		contexts := strings.Split(strings.TrimSpace(result.Stdout), "\n")
		for _, c := range contexts {
			if c == s.context.Name {
				return compiler.StatusSatisfied, nil
			}
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ContextStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "context", s.context.Name, "", s.context.Host), nil
}

// Apply creates the Docker context.
func (s *ContextStep) Apply(ctx compiler.RunContext) error {
	args := []string{"context", "create", s.context.Name, "--docker", "host=" + s.context.Host}
	if s.context.Description != "" {
		args = append(args, "--description", s.context.Description)
	}

	result, err := s.runner.Run(ctx.Context(), "docker", args...)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("docker context create failed: %s", result.Stderr)
	}

	// Set as default if specified
	if s.context.Default {
		useResult, err := s.runner.Run(ctx.Context(), "docker", "context", "use", s.context.Name)
		if err != nil {
			return err
		}
		if !useResult.Success() {
			return fmt.Errorf("docker context use failed: %s", useResult.Stderr)
		}
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *ContextStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Create Docker Context",
		fmt.Sprintf("Creates a Docker context '%s' pointing to %s for multi-host Docker management.",
			s.context.Name, s.context.Host),
		[]string{
			"https://docs.docker.com/engine/context/working-with-contexts/",
		},
	).WithTradeoffs([]string{
		"+ Enables managing multiple Docker hosts from one machine",
		"+ Simplifies switching between local and remote Docker",
		"+ Useful for production deployments from development machine",
	})
}

// Ensure all steps implement compiler.Step.
var (
	_ compiler.Step = (*InstallStep)(nil)
	_ compiler.Step = (*BuildKitStep)(nil)
	_ compiler.Step = (*KubernetesStep)(nil)
	_ compiler.Step = (*ContextStep)(nil)
)
