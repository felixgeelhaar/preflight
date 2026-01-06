// Package app provides the main application logic for preflight.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/felixgeelhaar/preflight/internal/adapters/command"
	"github.com/felixgeelhaar/preflight/internal/adapters/filesystem"
	lockadapter "github.com/felixgeelhaar/preflight/internal/adapters/lockfile"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/domain/policy"
	"github.com/felixgeelhaar/preflight/internal/provider/apt"
	"github.com/felixgeelhaar/preflight/internal/provider/bootstrap"
	"github.com/felixgeelhaar/preflight/internal/provider/brew"
	"github.com/felixgeelhaar/preflight/internal/provider/cargo"
	"github.com/felixgeelhaar/preflight/internal/provider/chocolatey"
	"github.com/felixgeelhaar/preflight/internal/provider/files"
	"github.com/felixgeelhaar/preflight/internal/provider/gem"
	"github.com/felixgeelhaar/preflight/internal/provider/git"
	"github.com/felixgeelhaar/preflight/internal/provider/gotools"
	"github.com/felixgeelhaar/preflight/internal/provider/npm"
	"github.com/felixgeelhaar/preflight/internal/provider/nvim"
	"github.com/felixgeelhaar/preflight/internal/provider/pip"
	"github.com/felixgeelhaar/preflight/internal/provider/runtime"
	"github.com/felixgeelhaar/preflight/internal/provider/scoop"
	"github.com/felixgeelhaar/preflight/internal/provider/shell"
	"github.com/felixgeelhaar/preflight/internal/provider/ssh"
	"github.com/felixgeelhaar/preflight/internal/provider/vscode"
	"github.com/felixgeelhaar/preflight/internal/provider/winget"
)

// Preflight is the main application orchestrator.
type Preflight struct {
	compiler          *compiler.Compiler
	planner           *execution.Planner
	executor          *execution.Executor
	lockRepo          lock.Repository
	mode              config.ReproducibilityMode
	modeSet           bool
	rollbackOnFailure bool
	out               io.Writer
}

// New creates a new Preflight application.
func New(out io.Writer) *Preflight {
	// Create real implementations
	cmdRunner := command.NewRealRunner()
	fs := filesystem.NewRealFileSystem()

	// Detect platform for platform-aware providers
	plat, _ := platform.Detect()

	// Create compiler with providers
	comp := compiler.NewCompiler()
	comp.RegisterProvider(bootstrap.NewProvider(cmdRunner, plat))
	comp.RegisterProvider(apt.NewProvider(cmdRunner))
	comp.RegisterProvider(brew.NewProvider(cmdRunner))
	comp.RegisterProvider(cargo.NewProvider(cmdRunner))
	comp.RegisterProvider(chocolatey.NewProvider(cmdRunner, plat))
	comp.RegisterProvider(files.NewProvider(fs))
	comp.RegisterProvider(gem.NewProvider(cmdRunner))
	comp.RegisterProvider(git.NewProvider(fs))
	comp.RegisterProvider(gotools.NewProvider(cmdRunner))
	comp.RegisterProvider(npm.NewProvider(cmdRunner))
	comp.RegisterProvider(nvim.NewProvider(fs, cmdRunner))
	comp.RegisterProvider(pip.NewProvider(cmdRunner))
	comp.RegisterProvider(runtime.NewProvider(fs))
	comp.RegisterProvider(scoop.NewProvider(cmdRunner, plat))
	comp.RegisterProvider(shell.NewProvider(fs))
	comp.RegisterProvider(ssh.NewProvider(fs))
	comp.RegisterProvider(vscode.NewProvider(fs, cmdRunner, plat))
	comp.RegisterProvider(winget.NewProvider(cmdRunner, plat))

	return &Preflight{
		compiler: comp,
		planner:  execution.NewPlanner(),
		executor: execution.NewExecutor(),
		lockRepo: lockadapter.NewYAMLRepository(),
		out:      out,
	}
}

// WithMode sets a reproducibility mode override for planning and applying.
func (p *Preflight) WithMode(mode config.ReproducibilityMode) *Preflight {
	p.mode = mode
	p.modeSet = true
	return p
}

// WithRollbackOnFailure enables rollback on failed apply.
func (p *Preflight) WithRollbackOnFailure(enabled bool) *Preflight {
	p.rollbackOnFailure = enabled
	return p
}

// WithLockRepo sets the lock repository for lockfile operations.
func (p *Preflight) WithLockRepo(repo lock.Repository) *Preflight {
	p.lockRepo = repo
	return p
}

// Plan loads configuration and creates an execution plan.
func (p *Preflight) Plan(ctx context.Context, configPath, target string) (*execution.Plan, error) {
	mode, err := p.resolveMode(configPath)
	if err != nil {
		return nil, err
	}

	resolver, err := p.buildResolver(ctx, configPath, mode)
	if err != nil {
		return nil, err
	}

	// Load configuration
	cfg, err := p.loadConfig(configPath, target)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Compile to step graph
	configRoot := filepath.Dir(configPath)
	compileCtx := compiler.NewCompileContext(cfg).
		WithResolver(resolver).
		WithConfigRoot(configRoot).
		WithTarget(target)
	graph, err := p.compiler.CompileWithContext(compileCtx)
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
	executor := p.executor.WithDryRun(dryRun).WithRollbackOnFailure(p.rollbackOnFailure)
	return executor.Execute(ctx, plan)
}

// UpdateLockFromPlan updates the lockfile based on lockable steps in the plan.
func (p *Preflight) UpdateLockFromPlan(ctx context.Context, configPath string, plan *execution.Plan) error {
	if plan == nil {
		return fmt.Errorf("plan is required to update lockfile")
	}

	mode, err := p.resolveMode(configPath)
	if err != nil {
		return err
	}

	lockPath := strings.TrimSuffix(configPath, filepath.Ext(configPath)) + ".lock"
	if p.lockRepo == nil {
		return fmt.Errorf("lockfile repository not configured")
	}

	lockfile, err := p.lockRepo.Load(ctx, lockPath)
	if err != nil {
		if errors.Is(err, lock.ErrLockfileNotFound) {
			lockfile = lock.NewLockfile(mode, lock.MachineInfoFromSystem())
		} else {
			return fmt.Errorf("failed to load lockfile: %w", err)
		}
	}

	lockfile = lockfile.WithMode(mode)

	lockedKeys := make(map[string]struct{})
	lockedProviders := make(map[string]struct{})
	runCtx := compiler.NewRunContext(ctx)
	for _, entry := range plan.Entries() {
		lockable, ok := entry.Step().(compiler.LockableStep)
		if !ok {
			continue
		}
		info, ok := lockable.LockInfo()
		if !ok {
			continue
		}

		provider := strings.TrimSpace(info.Provider)
		name := strings.TrimSpace(info.Name)
		version := strings.TrimSpace(info.Version)
		if provider == "" || name == "" {
			continue
		}
		if version == "" || version == "latest" {
			if versioned, ok := entry.Step().(compiler.VersionedStep); ok {
				installed, ok, err := versioned.InstalledVersion(runCtx)
				if err != nil {
					return fmt.Errorf("failed to resolve installed version for %s:%s: %w", provider, name, err)
				}
				installed = strings.TrimSpace(installed)
				if ok && installed != "" {
					version = installed
				}
			}
		}
		if version == "" {
			version = "latest"
		}

		integrity := lock.IntegrityFromData(lock.AlgorithmSHA256, []byte(provider+":"+name+"@"+version))
		pkg, err := lock.NewPackageLock(provider, name, version, integrity, time.Now())
		if err != nil {
			return fmt.Errorf("failed to lock %s:%s: %w", provider, name, err)
		}
		if err := lockfile.SetPackage(pkg); err != nil {
			return fmt.Errorf("failed to update lockfile: %w", err)
		}

		key := provider + ":" + name
		lockedKeys[key] = struct{}{}
		lockedProviders[provider] = struct{}{}
	}

	for key, pkg := range lockfile.Packages() {
		if _, ok := lockedProviders[pkg.Provider()]; !ok {
			continue
		}
		if _, ok := lockedKeys[key]; ok {
			continue
		}
		lockfile.RemovePackage(pkg.Provider(), pkg.Name())
	}

	if err := p.lockRepo.Save(ctx, lockPath, lockfile); err != nil {
		return fmt.Errorf("failed to save lockfile: %w", err)
	}

	p.printf("Lockfile updated: %s\n", lockPath)
	return nil
}

// LoadMergedConfig loads and merges configuration, returning the raw map.
func (p *Preflight) LoadMergedConfig(_ context.Context, configPath, targetName string) (map[string]interface{}, error) {
	return p.loadConfig(configPath, targetName)
}

// LoadManifest loads the manifest file without merging layers.
func (p *Preflight) LoadManifest(_ context.Context, configPath string) (*config.Manifest, error) {
	loader := config.NewLoader()
	return loader.LoadManifest(configPath)
}

// CaptureSystemState captures the current system state for comparison.
func (p *Preflight) CaptureSystemState(ctx context.Context) (map[string]interface{}, error) {
	findings, err := p.Capture(ctx, CaptureOptions{
		IncludeSecrets: false,
	})
	if err != nil {
		return nil, err
	}

	// Convert findings to a map structure
	result := make(map[string]interface{})

	// Group items by provider
	byProvider := make(map[string][]interface{})
	for _, item := range findings.Items {
		byProvider[item.Provider] = append(byProvider[item.Provider], item.Name)
	}

	// Build provider-specific structures
	if formulae, ok := byProvider["brew"]; ok {
		result["brew"] = map[string]interface{}{
			"formulae": formulae,
		}
	}
	if extensions, ok := byProvider["vscode"]; ok {
		result["vscode"] = map[string]interface{}{
			"extensions": extensions,
		}
	}

	return result, nil
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

		stepID := entry.Step().ID().String()
		if entry.Status() == compiler.StatusNeedsApply && IsBootstrapStep(stepID) {
			p.printf("  %s %s (bootstrap)\n", status, stepID)
		} else {
			p.printf("  %s %s\n", status, stepID)
		}

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
	for i := range results {
		switch results[i].Status() {
		case compiler.StatusSatisfied:
			succeeded++
			p.printf("  ✓ %s\n", results[i].StepID().String())
		case compiler.StatusFailed:
			failed++
			p.printf("  ✗ %s: %v\n", results[i].StepID().String(), results[i].Error())
		case compiler.StatusSkipped:
			skipped++
			p.printf("  - %s (skipped)\n", results[i].StepID().String())
		case compiler.StatusNeedsApply:
			p.printf("  + %s (needs apply)\n", results[i].StepID().String())
		case compiler.StatusUnknown:
			p.printf("  ? %s (unknown)\n", results[i].StepID().String())
		}
	}

	p.printf("\nSummary: %d succeeded, %d failed, %d skipped\n",
		succeeded, failed, skipped)
}

// printf is a helper that writes to the output writer, ignoring errors.
func (p *Preflight) printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(p.out, format, args...)
}

// ValidationResult contains the results of configuration validation.
type ValidationResult struct {
	Errors           []string
	Warnings         []string
	Info             []string
	PolicyViolations []string
}

// ValidateOptions configures validation behavior.
type ValidateOptions struct {
	// PolicyFile is an optional path to a policy YAML file (allow/deny rules)
	PolicyFile string
	// OrgPolicyFile is an optional path to an org policy YAML file
	OrgPolicyFile string
}

// Validate checks the configuration for errors without making changes.
func (p *Preflight) Validate(ctx context.Context, configPath, targetName string) (*ValidationResult, error) {
	return p.ValidateWithOptions(ctx, configPath, targetName, ValidateOptions{})
}

// ValidateWithOptions checks the configuration with additional options.
func (p *Preflight) ValidateWithOptions(ctx context.Context, configPath, targetName string, opts ValidateOptions) (*ValidationResult, error) {
	result := &ValidationResult{}

	mode, err := p.resolveMode(configPath)
	if err != nil {
		return nil, err
	}

	resolver, err := p.buildResolver(ctx, configPath, mode)
	if err != nil {
		return nil, err
	}

	// Load configuration
	cfg, err := p.loadConfig(configPath, targetName)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Add info about loaded config
	result.Info = append(result.Info, fmt.Sprintf("Loaded config from %s", configPath))
	result.Info = append(result.Info, fmt.Sprintf("Target: %s", targetName))

	// Try to compile - this validates providers and dependencies
	configRoot := filepath.Dir(configPath)
	compileCtx := compiler.NewCompileContext(cfg).
		WithResolver(resolver).
		WithConfigRoot(configRoot).
		WithTarget(targetName)
	graph, err := p.compiler.CompileWithContext(compileCtx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Compilation failed: %v", err))
		return result, nil
	}

	// Get step count
	steps := graph.Steps()
	result.Info = append(result.Info, fmt.Sprintf("Compiled %d steps", len(steps)))

	// Check for potential issues
	p.validateSteps(ctx, graph, result)

	// Check policies (allow/deny rules)
	p.validatePolicies(ctx, cfg, graph, opts, result)

	// Check org policies (required/forbidden patterns)
	p.validateOrgPolicies(ctx, cfg, graph, opts, result)

	return result, nil
}

// validateSteps performs additional validation on compiled steps.
func (p *Preflight) validateSteps(_ context.Context, graph *compiler.StepGraph, result *ValidationResult) {
	steps := graph.Steps()

	// Check for duplicate step IDs (shouldn't happen but good to verify)
	seen := make(map[string]bool)
	for _, step := range steps {
		id := step.ID().String()
		if seen[id] {
			result.Errors = append(result.Errors, fmt.Sprintf("Duplicate step ID: %s", id))
		}
		seen[id] = true
	}

	// Check for missing dependencies
	for _, step := range steps {
		for _, dep := range step.DependsOn() {
			if _, exists := graph.Get(dep); !exists {
				result.Errors = append(result.Errors, fmt.Sprintf("Step %s depends on missing step: %s", step.ID(), dep))
			}
		}
	}

	// Check for empty providers
	providerCounts := make(map[string]int)
	for _, step := range steps {
		provider := step.ID().Provider()
		providerCounts[provider]++
	}

	if len(providerCounts) == 0 {
		result.Warnings = append(result.Warnings, "No steps generated - configuration may be empty")
	}
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

func (p *Preflight) resolveMode(configPath string) (config.ReproducibilityMode, error) {
	loader := config.NewLoader()
	manifest, err := loader.LoadManifest(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to load manifest: %w", err)
	}

	mode := manifest.Defaults.Mode
	if mode == "" {
		mode = config.ModeIntent
	}
	if p.modeSet {
		mode = p.mode
	}

	switch mode {
	case config.ModeIntent, config.ModeLocked, config.ModeFrozen:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid reproducibility mode: %s", mode)
	}
}

func (p *Preflight) buildResolver(ctx context.Context, configPath string, mode config.ReproducibilityMode) (compiler.VersionResolver, error) {
	lockPath := strings.TrimSuffix(configPath, filepath.Ext(configPath)) + ".lock"
	if p.lockRepo == nil {
		if mode == config.ModeIntent {
			lockfile := lock.NewLockfile(mode, lock.MachineInfoFromSystem())
			return versionResolverAdapter{resolver: lock.NewResolver(lockfile)}, nil
		}
		return nil, fmt.Errorf("lockfile repository not configured")
	}

	lockfile, err := p.lockRepo.Load(ctx, lockPath)
	if err != nil {
		if errors.Is(err, lock.ErrLockfileNotFound) {
			if mode != config.ModeIntent {
				return nil, fmt.Errorf("lockfile not found: %s (run 'preflight lock update')", lockPath)
			}
			lockfile = lock.NewLockfile(mode, lock.MachineInfoFromSystem())
		} else {
			return nil, fmt.Errorf("failed to load lockfile: %w", err)
		}
	}

	lockfile = lockfile.WithMode(mode)
	return versionResolverAdapter{resolver: lock.NewResolver(lockfile)}, nil
}

type versionResolverAdapter struct {
	resolver *lock.Resolver
}

func (a versionResolverAdapter) Resolve(provider, name, latestVersion string) compiler.Resolution {
	res := a.resolver.Resolve(provider, name, latestVersion)
	return compiler.Resolution{
		Provider:         res.Provider,
		Name:             res.Name,
		Version:          res.Version,
		Source:           compiler.ResolutionSource(res.Source),
		Locked:           res.Locked,
		LockedVersion:    res.LockedVersion,
		AvailableVersion: res.AvailableVersion,
		Drifted:          res.Drifted,
		Updated:          res.Updated,
		Failed:           res.Failed,
		Error:            res.Error,
	}
}

// validatePolicies checks compiled steps against policy constraints.
func (p *Preflight) validatePolicies(_ context.Context, cfg map[string]interface{}, graph *compiler.StepGraph, opts ValidateOptions, result *ValidationResult) {
	var policies []policy.Policy

	// Load policies from config
	configPolicies, err := policy.ParseFromConfig(cfg)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to parse inline policies: %v", err))
	} else if len(configPolicies) > 0 {
		policies = append(policies, configPolicies...)
		result.Info = append(result.Info, fmt.Sprintf("Loaded %d inline policies", len(configPolicies)))
	}

	// Load policies from external file if specified
	if opts.PolicyFile != "" {
		filePolicies, err := policy.LoadFromFile(opts.PolicyFile)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to load policy file: %v", err))
		} else if len(filePolicies) > 0 {
			policies = append(policies, filePolicies...)
			result.Info = append(result.Info, fmt.Sprintf("Loaded %d policies from %s", len(filePolicies), opts.PolicyFile))
		}
	}

	// If no policies, skip evaluation
	if len(policies) == 0 {
		return
	}

	// Extract step IDs for policy evaluation
	steps := graph.Steps()
	stepIDs := make([]string, len(steps))
	for i, step := range steps {
		stepIDs[i] = step.ID().String()
	}

	// Evaluate policies
	evaluator := policy.NewEvaluator(policies...)
	policyResult := evaluator.EvaluateSteps(stepIDs)

	// Add violations to result
	for _, violation := range policyResult.Violations {
		result.PolicyViolations = append(result.PolicyViolations, violation.Error())
	}

	if len(policyResult.Violations) > 0 {
		result.Info = append(result.Info, fmt.Sprintf("Policy check: %d violations, %d allowed",
			len(policyResult.Violations), len(policyResult.Allowed)))
	} else {
		result.Info = append(result.Info, fmt.Sprintf("Policy check: all %d steps allowed", len(policyResult.Allowed)))
	}
}

// validateOrgPolicies checks compiled steps against org policy constraints.
func (p *Preflight) validateOrgPolicies(_ context.Context, cfg map[string]interface{}, graph *compiler.StepGraph, opts ValidateOptions, result *ValidationResult) {
	var orgPolicies []*policy.OrgPolicy

	// Load org policy from config (inline)
	inlineOrgPolicy, err := policy.ParseOrgPolicyFromConfig(cfg)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to parse inline org policy: %v", err))
	} else if inlineOrgPolicy != nil {
		orgPolicies = append(orgPolicies, inlineOrgPolicy)
		result.Info = append(result.Info, fmt.Sprintf("Loaded inline org policy: %s", inlineOrgPolicy.Name))
	}

	// Load org policy from external file if specified
	if opts.OrgPolicyFile != "" {
		fileOrgPolicy, err := policy.LoadOrgPolicyFromFile(opts.OrgPolicyFile)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to load org policy file: %v", err))
		} else if fileOrgPolicy != nil {
			orgPolicies = append(orgPolicies, fileOrgPolicy)
			result.Info = append(result.Info, fmt.Sprintf("Loaded org policy from %s: %s", opts.OrgPolicyFile, fileOrgPolicy.Name))
		}
	}

	// If no org policies, skip evaluation
	if len(orgPolicies) == 0 {
		return
	}

	// Merge org policies
	mergedOrgPolicy := policy.MergeOrgPolicies(orgPolicies...)
	if mergedOrgPolicy == nil {
		return
	}

	// Extract step IDs for org policy evaluation
	steps := graph.Steps()
	stepIDs := make([]string, len(steps))
	for i, step := range steps {
		stepIDs[i] = step.ID().String()
	}

	// Evaluate org policy
	evaluator := policy.NewOrgEvaluator(mergedOrgPolicy)
	orgResult := evaluator.Evaluate(stepIDs)

	// Add violations based on enforcement mode
	if orgResult.HasViolations() {
		for _, violation := range orgResult.Violations {
			result.PolicyViolations = append(result.PolicyViolations, violation.Error())
		}
		result.Info = append(result.Info, fmt.Sprintf("Org policy: %d violations (enforcement: %s)",
			len(orgResult.Violations), orgResult.Enforcement))
	}

	// Add warnings
	if orgResult.HasWarnings() {
		for _, warning := range orgResult.Warnings {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Org policy warning: %s", warning.Error()))
		}
		result.Info = append(result.Info, fmt.Sprintf("Org policy: %d warnings (enforcement: %s)",
			len(orgResult.Warnings), orgResult.Enforcement))
	}

	// Report overrides applied
	if len(orgResult.OverridesApplied) > 0 {
		result.Info = append(result.Info, fmt.Sprintf("Org policy: %d overrides applied", len(orgResult.OverridesApplied)))
	}
}
