package shell

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for shell configuration.
type Provider struct {
	fs     ports.FileSystem
	runner ports.CommandRunner
}

// NewProvider creates a new shell provider with filesystem only (backward compatible).
func NewProvider(fs ports.FileSystem) *Provider {
	return &Provider{fs: fs}
}

// NewProviderWith creates a new shell provider with all dependencies.
func NewProviderWith(fs ports.FileSystem, runner ports.CommandRunner) *Provider {
	return &Provider{fs: fs, runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "shell"
}

// Compile transforms shell configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("shell")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	// Check if there's any actual configuration
	if len(cfg.Shells) == 0 && !cfg.Starship.Enabled && len(cfg.Env) == 0 && len(cfg.Aliases) == 0 {
		return nil, nil
	}

	// Estimate capacity: frameworks + plugins + custom plugins + starship + env + aliases
	capacity := len(cfg.Shells) * 3 // rough estimate
	if cfg.Starship.Enabled {
		capacity++
	}
	if len(cfg.Env) > 0 {
		capacity++
	}
	if len(cfg.Aliases) > 0 {
		capacity++
	}
	steps := make([]compiler.Step, 0, capacity)

	// Add framework and plugin steps for each shell
	for _, shell := range cfg.Shells {
		// Framework step (if framework is specified)
		if shell.Framework != "" {
			steps = append(steps, NewFrameworkStepWith(shell, p.fs, p.runner))

			// Add plugin steps based on framework
			if shell.Framework == "fisher" {
				// Fisher plugins
				for _, plugin := range shell.Plugins {
					steps = append(steps, NewFisherPluginStepWith(plugin, p.fs, p.runner))
				}
			} else {
				// Standard plugins (oh-my-zsh, etc.)
				for _, plugin := range shell.Plugins {
					steps = append(steps, NewPluginStepWithFS(shell.Name, shell.Framework, plugin, p.fs))
				}

				// Custom plugins (git cloned)
				for _, plugin := range shell.CustomPlugins {
					steps = append(steps, NewCustomPluginStepWith(shell.Name, shell.Framework, plugin, p.fs, p.runner))
				}
			}
		}
	}

	// Add starship step if enabled
	if cfg.Starship.Enabled {
		steps = append(steps, NewStarshipStepWithFS(cfg.Starship, p.fs))
	}

	// Add env step if there are environment variables
	if len(cfg.Env) > 0 && len(cfg.Shells) > 0 {
		// Use the first shell's name for env step
		steps = append(steps, NewEnvStepWithFS(cfg.Shells[0].Name, cfg.Env, p.fs))
	}

	// Add aliases step if there are aliases
	if len(cfg.Aliases) > 0 && len(cfg.Shells) > 0 {
		// Use the first shell's name for aliases step
		steps = append(steps, NewAliasStepWithFS(cfg.Shells[0].Name, cfg.Aliases, p.fs))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
