package compiler

import "github.com/felixgeelhaar/preflight/internal/domain/lock"

// Provider compiles a section of configuration into executable steps.
// Each provider handles a specific type of resource (brew, apt, files, etc.).
type Provider interface {
	// Name returns the provider's identifier (e.g., "brew", "apt", "files").
	Name() string

	// Compile transforms configuration into a list of steps.
	// The steps should be independent within the provider; cross-provider
	// dependencies are expressed through Step.DependsOn().
	Compile(ctx CompileContext) ([]Step, error)
}

// CompileContext provides configuration data and metadata to providers during compilation.
type CompileContext struct {
	config     map[string]interface{}
	provenance string
	resolver   *lock.Resolver
	configRoot string // Root directory of preflight configuration
	target     string // Current target name (e.g., "work", "personal")
}

// NewCompileContext creates a new CompileContext with the given configuration.
func NewCompileContext(config map[string]interface{}) CompileContext {
	return CompileContext{
		config: config,
	}
}

// Config returns the full merged configuration.
func (c CompileContext) Config() map[string]interface{} {
	return c.config
}

// GetSection returns a specific section of the configuration by key.
// Returns nil if the section doesn't exist or isn't a map.
func (c CompileContext) GetSection(key string) map[string]interface{} {
	if c.config == nil {
		return nil
	}
	section, ok := c.config[key]
	if !ok {
		return nil
	}
	sectionMap, ok := section.(map[string]interface{})
	if !ok {
		return nil
	}
	return sectionMap
}

// Provenance returns the source layer that defined this configuration.
func (c CompileContext) Provenance() string {
	return c.provenance
}

// WithProvenance returns a new CompileContext with provenance set.
func (c CompileContext) WithProvenance(provenance string) CompileContext {
	return CompileContext{
		config:     c.config,
		provenance: provenance,
		resolver:   c.resolver,
		configRoot: c.configRoot,
		target:     c.target,
	}
}

// Resolver returns the lock resolver for version resolution.
// Returns nil if no resolver is set.
func (c CompileContext) Resolver() *lock.Resolver {
	return c.resolver
}

// WithResolver returns a new CompileContext with the resolver set.
func (c CompileContext) WithResolver(resolver *lock.Resolver) CompileContext {
	return CompileContext{
		config:     c.config,
		provenance: c.provenance,
		resolver:   resolver,
		configRoot: c.configRoot,
		target:     c.target,
	}
}

// ConfigRoot returns the root directory of the preflight configuration.
func (c CompileContext) ConfigRoot() string {
	return c.configRoot
}

// WithConfigRoot returns a new CompileContext with the config root set.
func (c CompileContext) WithConfigRoot(configRoot string) CompileContext {
	return CompileContext{
		config:     c.config,
		provenance: c.provenance,
		resolver:   c.resolver,
		configRoot: configRoot,
		target:     c.target,
	}
}

// Target returns the current target name.
func (c CompileContext) Target() string {
	return c.target
}

// WithTarget returns a new CompileContext with the target set.
func (c CompileContext) WithTarget(target string) CompileContext {
	return CompileContext{
		config:     c.config,
		provenance: c.provenance,
		resolver:   c.resolver,
		configRoot: c.configRoot,
		target:     target,
	}
}

// ResolveVersion resolves a package version using the lockfile.
// If no resolver is set, returns the latest version unchanged.
func (c CompileContext) ResolveVersion(provider, name, latestVersion string) lock.Resolution {
	if c.resolver == nil {
		return lock.Resolution{
			Provider: provider,
			Name:     name,
			Version:  latestVersion,
			Source:   lock.ResolutionSourceLatest,
		}
	}
	return c.resolver.Resolve(provider, name, latestVersion)
}
