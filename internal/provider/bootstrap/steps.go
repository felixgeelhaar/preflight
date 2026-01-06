package bootstrap

import (
	"fmt"
	"os/exec"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	tooldeps "github.com/felixgeelhaar/preflight/internal/domain/deps"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/commandutil"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// ToolStep ensures a runtime toolchain is installed.
type ToolStep struct {
	tool        tooldeps.Tool
	manager     string
	packageName string
	id          compiler.StepID
	deps        []compiler.StepID
	runner      ports.CommandRunner
	platform    *platform.Platform
}

// NewToolStep creates a new ToolStep.
func NewToolStep(tool tooldeps.Tool, manager, packageName string, id compiler.StepID, dep compiler.StepID, runner ports.CommandRunner, plat *platform.Platform) *ToolStep {
	deps := []compiler.StepID{}
	if dep.String() != "" {
		deps = append(deps, dep)
	}
	return &ToolStep{
		tool:        tool,
		manager:     manager,
		packageName: packageName,
		id:          id,
		deps:        deps,
		runner:      runner,
		platform:    plat,
	}
}

// ID returns the step identifier.
func (s *ToolStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ToolStep) DependsOn() []compiler.StepID {
	return s.deps
}

// Check determines if the toolchain is already installed.
func (s *ToolStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if _, err := exec.LookPath(toolBinary(s.tool)); err == nil {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ToolStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "bootstrap", string(s.tool), "", s.packageName), nil
}

// Apply installs the toolchain using the selected package manager.
func (s *ToolStep) Apply(ctx compiler.RunContext) error {
	switch s.manager {
	case "brew":
		if err := validation.ValidatePackageName(s.packageName); err != nil {
			return fmt.Errorf("invalid brew package name: %w", err)
		}
		return s.runCommand(ctx, "brew", "install", s.packageName)
	case "apt":
		if err := validation.ValidatePackageName(s.packageName); err != nil {
			return fmt.Errorf("invalid apt package name: %w", err)
		}
		return s.runCommand(ctx, "sudo", "apt-get", "install", "-y", s.packageName)
	case "winget":
		if err := validation.ValidateWingetID(s.packageName); err != nil {
			return fmt.Errorf("invalid winget ID: %w", err)
		}
		cmd := "winget"
		if s.platform != nil && s.platform.IsWSL() {
			cmd = "winget.exe"
		}
		return s.runCommand(ctx, cmd, "install", "--id", s.packageName, "--exact", "--accept-source-agreements", "--accept-package-agreements", "--silent")
	case "chocolatey":
		if err := validation.ValidateChocoPackage(s.packageName); err != nil {
			return fmt.Errorf("invalid chocolatey package name: %w", err)
		}
		cmd := "choco"
		if s.platform != nil && s.platform.IsWSL() {
			cmd = "choco.exe"
		}
		return s.runCommand(ctx, cmd, "install", s.packageName, "-y", "--no-progress")
	case "scoop":
		if err := validation.ValidatePackageName(s.packageName); err != nil {
			return fmt.Errorf("invalid scoop package name: %w", err)
		}
		cmd := "scoop"
		if s.platform != nil && s.platform.IsWSL() {
			cmd = "scoop.cmd"
		}
		return s.runCommand(ctx, cmd, "install", s.packageName)
	default:
		return fmt.Errorf("unsupported bootstrap manager: %s", s.manager)
	}
}

// Explain provides a human-readable explanation.
func (s *ToolStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Installs %s toolchain via %s.", s.tool, s.manager)
	return compiler.NewExplanation(
		"Bootstrap Toolchain",
		desc,
		nil,
	)
}

// LockInfo returns lockfile information for this toolchain install.
func (s *ToolStep) LockInfo() (compiler.LockInfo, bool) {
	return compiler.LockInfo{
		Provider: s.manager,
		Name:     s.packageName,
		Version:  "",
	}, true
}

func (s *ToolStep) runCommand(ctx compiler.RunContext, cmd string, args ...string) error {
	result, err := s.runner.Run(ctx.Context(), cmd, args...)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return fmt.Errorf("%s not found in PATH; bootstrap the package manager first", cmd)
		}
		return err
	}
	if !result.Success() {
		return fmt.Errorf("%s %s failed: %s", cmd, args[0], result.Stderr)
	}
	return nil
}

func toolBinary(tool tooldeps.Tool) string {
	switch tool {
	case tooldeps.ToolNode:
		return "node"
	case tooldeps.ToolPython:
		return "python3"
	case tooldeps.ToolRuby:
		return "ruby"
	case tooldeps.ToolGo:
		return "go"
	case tooldeps.ToolRust:
		return "cargo"
	default:
		return string(tool)
	}
}
