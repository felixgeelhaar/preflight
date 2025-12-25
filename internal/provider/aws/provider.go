// Package aws provides the AWS CLI provider for profiles and SSO configuration.
package aws

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for AWS CLI.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new AWS provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "aws"
}

// Compile transforms aws configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("aws")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add profile steps
	for _, profile := range cfg.Profiles {
		steps = append(steps, NewProfileStep(profile, p.runner))
	}

	// Add SSO steps
	for _, sso := range cfg.SSO {
		steps = append(steps, NewSSOStep(sso, p.runner))
	}

	// Add default profile step
	if cfg.DefaultProfile != "" {
		steps = append(steps, NewDefaultProfileStep(cfg.DefaultProfile, p.runner))
	}

	// Add default region step
	if cfg.DefaultRegion != "" {
		steps = append(steps, NewDefaultRegionStep(cfg.DefaultRegion, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
