package ssh

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for SSH configuration.
type Provider struct {
	fs ports.FileSystem
}

// NewProvider creates a new SSH provider.
func NewProvider(fs ports.FileSystem) *Provider {
	return &Provider{fs: fs}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "ssh"
}

// Compile transforms SSH configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("ssh")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	// Only create step if there's actual config to write
	if len(cfg.Hosts) == 0 && len(cfg.Matches) == 0 && !hasDefaults(cfg) && cfg.Include == "" {
		return nil, nil
	}

	steps := make([]compiler.Step, 0)
	steps = append(steps, NewConfigStep(cfg, p.fs))

	return steps, nil
}

// hasDefaults returns true if any default options are set.
func hasDefaults(cfg *Config) bool {
	d := cfg.Defaults
	return d.AddKeysToAgent || d.IdentitiesOnly || d.ForwardAgent ||
		d.ServerAliveInterval > 0 || d.ServerAliveCountMax > 0
}
