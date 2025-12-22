package compiler

import "context"

// RunContext provides context for step execution (Check, Plan, Apply).
type RunContext struct {
	ctx    context.Context
	dryRun bool
}

// NewRunContext creates a new RunContext with the given context.
func NewRunContext(ctx context.Context) RunContext {
	return RunContext{
		ctx:    ctx,
		dryRun: false,
	}
}

// Context returns the underlying context.Context.
func (r RunContext) Context() context.Context {
	return r.ctx
}

// DryRun returns whether this is a dry-run execution.
func (r RunContext) DryRun() bool {
	return r.dryRun
}

// WithDryRun returns a new RunContext with the dry-run flag set.
func (r RunContext) WithDryRun(dryRun bool) RunContext {
	return RunContext{
		ctx:    r.ctx,
		dryRun: dryRun,
	}
}

// ExplainContext provides context for generating step explanations.
type ExplainContext struct {
	verbose bool
}

// NewExplainContext creates a new ExplainContext.
func NewExplainContext() ExplainContext {
	return ExplainContext{
		verbose: false,
	}
}

// Verbose returns whether verbose explanations are requested.
func (e ExplainContext) Verbose() bool {
	return e.verbose
}

// WithVerbose returns a new ExplainContext with verbose mode set.
func (e ExplainContext) WithVerbose(verbose bool) ExplainContext {
	return ExplainContext{
		verbose: verbose,
	}
}
