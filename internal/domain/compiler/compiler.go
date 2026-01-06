// Package compiler transforms configuration into executable steps.
// It provides the core compilation pipeline: Config → Provider → StepGraph.
package compiler

import (
	"fmt"
)

// Compiler orchestrates providers to build a StepGraph from configuration.
type Compiler struct {
	providers []Provider
}

// NewCompiler creates a new Compiler.
func NewCompiler() *Compiler {
	return &Compiler{
		providers: make([]Provider, 0),
	}
}

// RegisterProvider adds a provider to the compiler.
// Providers are called in registration order during compilation.
func (c *Compiler) RegisterProvider(provider Provider) {
	c.providers = append(c.providers, provider)
}

// Providers returns all registered providers.
func (c *Compiler) Providers() []Provider {
	return c.providers
}

// Compile transforms configuration into a validated StepGraph.
// It calls each provider's Compile method and aggregates the results.
// Returns an error if:
// - Any provider fails to compile
// - Duplicate step IDs are detected
// - Dependencies are missing
// - Cyclic dependencies are detected
func (c *Compiler) Compile(config map[string]interface{}) (*StepGraph, error) {
	return c.CompileWithContext(NewCompileContext(config))
}

// CompileWithContext transforms configuration into a validated StepGraph using
// the provided compilation context.
func (c *Compiler) CompileWithContext(ctx CompileContext) (*StepGraph, error) {
	graph := NewStepGraph()

	// Compile each provider
	for _, provider := range c.providers {
		steps, err := provider.Compile(ctx)
		if err != nil {
			return nil, fmt.Errorf("provider %q: %w", provider.Name(), err)
		}

		// Add steps to graph
		for _, step := range steps {
			if err := graph.Add(step); err != nil {
				return nil, fmt.Errorf("provider %q, step %q: %w",
					provider.Name(), step.ID().String(), err)
			}
		}
	}

	// Validate the graph
	if err := graph.Validate(); err != nil {
		return nil, err
	}

	// Check for cycles by attempting topological sort
	if _, err := graph.TopologicalSort(); err != nil {
		return nil, err
	}

	return graph, nil
}
