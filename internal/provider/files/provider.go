package files

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for file management.
type Provider struct {
	fs        ports.FileSystem
	lifecycle ports.FileLifecycle
}

// NewProvider creates a new files provider.
func NewProvider(fs ports.FileSystem) *Provider {
	return &Provider{
		fs:        fs,
		lifecycle: &ports.NoopLifecycle{},
	}
}

// WithLifecycle sets the lifecycle manager for the provider.
func (p *Provider) WithLifecycle(lifecycle ports.FileLifecycle) *Provider {
	if lifecycle != nil {
		p.lifecycle = lifecycle
	}
	return p
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "files"
}

// Compile transforms files configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("files")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add link steps first
	for _, link := range cfg.Links {
		steps = append(steps, NewLinkStep(link, p.fs, p.lifecycle))
	}

	// Add template steps
	for _, tmpl := range cfg.Templates {
		steps = append(steps, NewTemplateStep(tmpl, p.fs, p.lifecycle))
	}

	// Add copy steps
	for _, cp := range cfg.Copies {
		steps = append(steps, NewCopyStep(cp, p.fs, p.lifecycle))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
