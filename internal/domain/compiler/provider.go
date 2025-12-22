package compiler

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
	}
}
