// Package app provides the main application logic for preflight.
package app

import (
	"context"
	"fmt"
	"io"

	lockadapter "github.com/felixgeelhaar/preflight/internal/adapters/lockfile"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/apt"
	"github.com/felixgeelhaar/preflight/internal/provider/brew"
	"github.com/felixgeelhaar/preflight/internal/provider/files"
	"github.com/felixgeelhaar/preflight/internal/provider/git"
	"github.com/felixgeelhaar/preflight/internal/provider/nvim"
	"github.com/felixgeelhaar/preflight/internal/provider/runtime"
	"github.com/felixgeelhaar/preflight/internal/provider/shell"
	"github.com/felixgeelhaar/preflight/internal/provider/ssh"
	"github.com/felixgeelhaar/preflight/internal/provider/vscode"
)

// Preflight is the main application orchestrator.
type Preflight struct {
	compiler *compiler.Compiler
	planner  *execution.Planner
	executor *execution.Executor
	lockRepo lock.Repository
	out      io.Writer
}

// New creates a new Preflight application.
func New(out io.Writer) *Preflight {
	// Create real implementations
	cmdRunner := ports.NewRealCommandRunner()
	fs := ports.NewRealFileSystem()

	// Create compiler with providers
	comp := compiler.NewCompiler()
	comp.RegisterProvider(apt.NewProvider(cmdRunner))
	comp.RegisterProvider(brew.NewProvider(cmdRunner))
	comp.RegisterProvider(files.NewProvider(fs))
	comp.RegisterProvider(git.NewProvider(fs))
	comp.RegisterProvider(ssh.NewProvider(fs))
	comp.RegisterProvider(runtime.NewProvider(fs))
	comp.RegisterProvider(shell.NewProvider(fs))
	comp.RegisterProvider(nvim.NewProvider(fs, cmdRunner))
	comp.RegisterProvider(vscode.NewProvider(fs, cmdRunner))

	return &Preflight{
		compiler: comp,
		planner:  execution.NewPlanner(),
		executor: execution.NewExecutor(),
		lockRepo: lockadapter.NewYAMLRepository(),
		out:      out,
	}
}

// WithLockRepo sets the lock repository for lockfile operations.
func (p *Preflight) WithLockRepo(repo lock.Repository) *Preflight {
	p.lockRepo = repo
	return p
}

// Plan loads configuration and creates an execution plan.
func (p *Preflight) Plan(ctx context.Context, configPath, target string) (*execution.Plan, error) {
	// Load configuration
	cfg, err := p.loadConfig(configPath, target)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Compile to step graph
	graph, err := p.compiler.Compile(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to compile: %w", err)
	}

	// Create execution plan
	plan, err := p.planner.Plan(ctx, graph)
	if err != nil {
		return nil, fmt.Errorf("failed to plan: %w", err)
	}

	return plan, nil
}

// Apply executes the plan.
func (p *Preflight) Apply(ctx context.Context, plan *execution.Plan, dryRun bool) ([]execution.StepResult, error) {
	executor := p.executor.WithDryRun(dryRun)
	return executor.Execute(ctx, plan)
}

// PrintPlan outputs a human-readable plan summary.
func (p *Preflight) PrintPlan(plan *execution.Plan) {
	summary := plan.Summary()

	p.printf("\nPreflight Plan\n")
	p.printf("==============\n\n")

	if !plan.HasChanges() {
		p.printf("No changes needed. Your system is up to date.\n")
		return
	}

	p.printf("Steps: %d total, %d to apply, %d satisfied\n\n",
		summary.Total, summary.NeedsApply, summary.Satisfied)

	for _, entry := range plan.Entries() {
		status := "✓"
		if entry.Status() == compiler.StatusNeedsApply {
			status = "+"
		}

		p.printf("  %s %s\n", status, entry.Step().ID().String())

		diff := entry.Diff()
		if !diff.IsEmpty() {
			p.printf("      %s\n", diff.Summary())
		}
	}

	p.printf("\nRun 'preflight apply' to execute this plan.\n")
}

// PrintResults outputs execution results.
func (p *Preflight) PrintResults(results []execution.StepResult) {
	p.printf("\nExecution Results\n")
	p.printf("=================\n\n")

	var succeeded, failed, skipped int
	for _, r := range results {
		switch r.Status() {
		case compiler.StatusSatisfied:
			succeeded++
			p.printf("  ✓ %s\n", r.StepID().String())
		case compiler.StatusFailed:
			failed++
			p.printf("  ✗ %s: %v\n", r.StepID().String(), r.Error())
		case compiler.StatusSkipped:
			skipped++
			p.printf("  - %s (skipped)\n", r.StepID().String())
		case compiler.StatusNeedsApply:
			p.printf("  + %s (needs apply)\n", r.StepID().String())
		case compiler.StatusUnknown:
			p.printf("  ? %s (unknown)\n", r.StepID().String())
		}
	}

	p.printf("\nSummary: %d succeeded, %d failed, %d skipped\n",
		succeeded, failed, skipped)
}

// printf is a helper that writes to the output writer, ignoring errors.
func (p *Preflight) printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(p.out, format, args...)
}

// loadConfig loads and merges configuration from the given path.
func (p *Preflight) loadConfig(configPath, targetName string) (map[string]interface{}, error) {
	loader := config.NewLoader()

	// Parse target name
	target, err := config.NewTargetName(targetName)
	if err != nil {
		return nil, fmt.Errorf("invalid target name: %w", err)
	}

	// Load and merge configuration
	merged, err := loader.Load(configPath, target)
	if err != nil {
		return nil, err
	}

	return merged.Raw(), nil
}
