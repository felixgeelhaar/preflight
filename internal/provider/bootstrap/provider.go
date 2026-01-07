// Package bootstrap provides automatic toolchain installation for missing dependencies.
package bootstrap

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	tooldeps "github.com/felixgeelhaar/preflight/internal/domain/deps"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/apt"
	"github.com/felixgeelhaar/preflight/internal/provider/brew"
	"github.com/felixgeelhaar/preflight/internal/provider/chocolatey"
	"github.com/felixgeelhaar/preflight/internal/provider/scoop"
	"github.com/felixgeelhaar/preflight/internal/provider/winget"
)

// Provider installs required toolchains when they are not explicitly configured.
type Provider struct {
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewProvider creates a new bootstrap provider.
func NewProvider(runner ports.CommandRunner, plat *platform.Platform) *Provider {
	return &Provider{
		runner:   runner,
		platform: plat,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "bootstrap"
}

// Compile emits toolchain bootstrap steps for tool-dependent providers.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	tools := requiredTools(ctx)
	if len(tools) == 0 {
		return nil, nil
	}

	steps := make([]compiler.Step, 0, len(tools))
	added := make(map[string]struct{})

	for _, tool := range tools {
		plan, ok := tooldeps.ResolveToolBootstrap(ctx, p.platform, tool)
		if !ok {
			continue
		}

		dep := p.ensureManagerSteps(ctx, plan.Manager, &steps, added)
		toolStep := NewToolStep(plan.Tool, plan.Manager, plan.PackageName, plan.StepID, dep, p.runner, p.platform)
		appendStep(&steps, added, toolStep)
	}

	if len(steps) == 0 {
		return nil, nil
	}

	return steps, nil
}

func requiredTools(ctx compiler.CompileContext) []tooldeps.Tool {
	tools := make([]tooldeps.Tool, 0, 5)

	if hasList(ctx.GetSection("cargo"), "crates") {
		tools = append(tools, tooldeps.ToolRust)
	}
	if hasList(ctx.GetSection("npm"), "packages") {
		tools = append(tools, tooldeps.ToolNode)
	}
	if hasList(ctx.GetSection("pip"), "packages") {
		tools = append(tools, tooldeps.ToolPython)
	}
	if hasList(ctx.GetSection("gem"), "gems") {
		tools = append(tools, tooldeps.ToolRuby)
	}
	if hasList(ctx.GetSection("gotools"), "tools") {
		tools = append(tools, tooldeps.ToolGo)
	}

	return tools
}

func (p *Provider) ensureManagerSteps(ctx compiler.CompileContext, manager string, steps *[]compiler.Step, added map[string]struct{}) compiler.StepID {
	switch manager {
	case "brew":
		if !managerConfigured(ctx, "brew") {
			appendStep(steps, added, brew.NewInstallStep(p.runner))
		}
		return compiler.MustNewStepID("brew:install")
	case "apt":
		if !managerConfigured(ctx, "apt") {
			appendStep(steps, added, apt.NewReadyStep(p.runner))
			updateDeps := []compiler.StepID{compiler.MustNewStepID("apt:ready")}
			appendStep(steps, added, apt.NewUpdateStep(p.runner, updateDeps))
		}
		return compiler.MustNewStepID("apt:update")
	case "winget":
		if !managerConfigured(ctx, "winget") {
			appendStep(steps, added, winget.NewReadyStep(p.platform))
		}
		return compiler.MustNewStepID("winget:ready")
	case "chocolatey":
		if !managerConfigured(ctx, "chocolatey") {
			appendStep(steps, added, chocolatey.NewInstallStep(p.runner, p.platform))
		}
		return compiler.MustNewStepID("chocolatey:install")
	case "scoop":
		if !managerConfigured(ctx, "scoop") {
			appendStep(steps, added, scoop.NewInstallStep(p.runner, p.platform))
		}
		return compiler.MustNewStepID("scoop:install")
	default:
		return compiler.StepID{}
	}
}

func managerConfigured(ctx compiler.CompileContext, manager string) bool {
	switch manager {
	case "brew":
		section := ctx.GetSection("brew")
		return hasList(section, "formulae") || hasList(section, "casks") || hasList(section, "taps")
	case "apt":
		section := ctx.GetSection("apt")
		return hasList(section, "packages") || hasList(section, "ppas")
	case "winget":
		section := ctx.GetSection("winget")
		return hasList(section, "packages")
	case "chocolatey":
		section := ctx.GetSection("chocolatey")
		return hasList(section, "packages") || hasList(section, "sources")
	case "scoop":
		section := ctx.GetSection("scoop")
		return hasList(section, "packages") || hasList(section, "buckets")
	default:
		return false
	}
}

func hasList(section map[string]interface{}, key string) bool {
	if section == nil {
		return false
	}
	list, ok := section[key].([]interface{})
	return ok && len(list) > 0
}

func appendStep(steps *[]compiler.Step, added map[string]struct{}, step compiler.Step) {
	if step == nil {
		return
	}
	id := step.ID().String()
	if _, ok := added[id]; ok {
		return
	}
	added[id] = struct{}{}
	*steps = append(*steps, step)
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
