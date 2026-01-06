package bootstrap

import (
	"context"
	"os/exec"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	tooldeps "github.com/felixgeelhaar/preflight/internal/domain/deps"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/require"
)

func TestToolStep_CheckMissingBinary(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewToolStep(tooldeps.Tool("definitely-not-a-tool"), "brew", "node", compiler.MustNewStepID("bootstrap:tool:missing"), compiler.StepID{}, runner, nil)

	status, err := step.Check(compiler.NewRunContext(context.Background()))
	require.NoError(t, err)
	require.Equal(t, compiler.StatusNeedsApply, status)
}

func TestToolStep_Apply_CommandFailure(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"install", "node"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "boom",
	})
	step := NewToolStep(tooldeps.ToolNode, "brew", "node", compiler.MustNewStepID("bootstrap:tool:node"), compiler.StepID{}, runner, nil)

	err := step.Apply(compiler.NewRunContext(context.Background()))
	require.Error(t, err)
	require.Contains(t, err.Error(), "brew install failed")
}

func TestToolStep_LockInfo(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewToolStep(tooldeps.ToolGo, "brew", "go", compiler.MustNewStepID("bootstrap:tool:go"), compiler.StepID{}, runner, nil)

	info, ok := step.LockInfo()
	require.True(t, ok)
	require.Equal(t, "brew", info.Provider)
	require.Equal(t, "go", info.Name)
	require.Equal(t, "", info.Version)
}

func TestToolStep_Apply_InvalidPackageName(t *testing.T) {
	runner := mocks.NewCommandRunner()
	stepID := compiler.MustNewStepID("bootstrap:tool:bad")

	tests := []struct {
		name    string
		manager string
		pkg     string
		wantErr string
	}{
		{"brew", "brew", "bad name", "invalid brew package name"},
		{"apt", "apt", "bad name", "invalid apt package name"},
		{"winget", "winget", "bad name", "invalid winget ID"},
		{"chocolatey", "chocolatey", "bad name", "invalid chocolatey package name"},
		{"scoop", "scoop", "bad name", "invalid scoop package name"},
		{"unknown", "unknown", "bad", "unsupported bootstrap manager"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := NewToolStep(tooldeps.Tool("custom"), tt.manager, tt.pkg, stepID, compiler.StepID{}, runner, nil)
			err := step.Apply(compiler.NewRunContext(context.Background()))
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestToolBinary(t *testing.T) {
	tests := []struct {
		tool tooldeps.Tool
		want string
	}{
		{tooldeps.ToolNode, "node"},
		{tooldeps.ToolPython, "python3"},
		{tooldeps.ToolRuby, "ruby"},
		{tooldeps.ToolGo, "go"},
		{tooldeps.ToolRust, "cargo"},
		{tooldeps.Tool("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.tool), func(t *testing.T) {
			require.Equal(t, tt.want, toolBinary(tt.tool))
		})
	}
}

func TestToolStep_PlanAndDependsOn(t *testing.T) {
	runner := mocks.NewCommandRunner()
	dep := compiler.MustNewStepID("brew:install")
	step := NewToolStep(tooldeps.ToolNode, "brew", "node", compiler.MustNewStepID("bootstrap:tool:node"), dep, runner, nil)

	require.Equal(t, []compiler.StepID{dep}, step.DependsOn())

	diff, err := step.Plan(compiler.NewRunContext(context.Background()))
	require.NoError(t, err)
	require.Equal(t, compiler.DiffTypeAdd, diff.Type())
	require.Equal(t, "bootstrap", diff.Resource())
	require.Equal(t, "node", diff.Name())
}

func TestToolStep_Explain(t *testing.T) {
	step := NewToolStep(tooldeps.ToolNode, "brew", "node", compiler.MustNewStepID("bootstrap:tool:node"), compiler.StepID{}, mocks.NewCommandRunner(), nil)

	expl := step.Explain(compiler.NewExplainContext())
	require.Equal(t, "Bootstrap Toolchain", expl.Summary())
	require.Contains(t, expl.Detail(), "node")
}

func TestToolStep_Apply_WSLCommands(t *testing.T) {
	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")

	tests := []struct {
		name    string
		manager string
		cmd     string
		args    []string
		pkg     string
	}{
		{
			name:    "winget",
			manager: "winget",
			cmd:     "winget.exe",
			args: []string{
				"install", "--id", "Git.Git", "--exact",
				"--accept-source-agreements", "--accept-package-agreements", "--silent",
			},
			pkg: "Git.Git",
		},
		{
			name:    "chocolatey",
			manager: "chocolatey",
			cmd:     "choco.exe",
			args:    []string{"install", "git", "-y", "--no-progress"},
			pkg:     "git",
		},
		{
			name:    "scoop",
			manager: "scoop",
			cmd:     "scoop.cmd",
			args:    []string{"install", "git"},
			pkg:     "git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := mocks.NewCommandRunner()
			runner.AddResult(tt.cmd, tt.args, ports.CommandResult{ExitCode: 0})
			step := NewToolStep(tooldeps.Tool("custom"), tt.manager, tt.pkg, compiler.MustNewStepID("bootstrap:tool:test"), compiler.StepID{}, runner, plat)

			err := step.Apply(compiler.NewRunContext(context.Background()))
			require.NoError(t, err)
		})
	}
}

type errRunner struct{}

func (errRunner) Run(_ context.Context, _ string, _ ...string) (ports.CommandResult, error) {
	return ports.CommandResult{}, exec.ErrNotFound
}

func TestToolStep_Apply_CommandNotFound(t *testing.T) {
	step := NewToolStep(tooldeps.ToolNode, "brew", "node", compiler.MustNewStepID("bootstrap:tool:node"), compiler.StepID{}, errRunner{}, nil)

	err := step.Apply(compiler.NewRunContext(context.Background()))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found in PATH")
}
