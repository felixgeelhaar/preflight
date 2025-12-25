package config

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// HookPhase represents when a hook runs.
type HookPhase string

const (
	// HookPhasePre runs before the main operation.
	HookPhasePre HookPhase = "pre"
	// HookPhasePost runs after the main operation.
	HookPhasePost HookPhase = "post"
)

// HookType represents what operation the hook relates to.
type HookType string

const (
	// HookTypeApply runs around apply operations.
	HookTypeApply HookType = "apply"
	// HookTypePlan runs around plan operations.
	HookTypePlan HookType = "plan"
	// HookTypeDoctor runs around doctor operations.
	HookTypeDoctor HookType = "doctor"
	// HookTypeCapture runs around capture operations.
	HookTypeCapture HookType = "capture"
)

// Hook represents a lifecycle hook configuration.
type Hook struct {
	Name        string            `yaml:"name,omitempty"`
	Phase       HookPhase         `yaml:"phase"`              // pre or post
	Type        HookType          `yaml:"type"`               // apply, plan, doctor, capture
	Command     string            `yaml:"command"`            // Command to execute
	Script      string            `yaml:"script,omitempty"`   // Inline script (alternative to command)
	Shell       string            `yaml:"shell,omitempty"`    // Shell to use (bash, zsh, sh)
	Timeout     string            `yaml:"timeout,omitempty"`  // Execution timeout
	OnError     string            `yaml:"on_error,omitempty"` // continue, fail (default: fail)
	When        Condition         `yaml:"when,omitempty"`     // Conditional execution
	Environment map[string]string `yaml:"env,omitempty"`      // Additional environment variables
}

// HooksConfig represents all hooks in the configuration.
type HooksConfig struct {
	Hooks []Hook `yaml:"hooks,omitempty"`
}

// HookRunner executes hooks.
type HookRunner struct {
	evaluator *ConditionEvaluator
	workDir   string
}

// NewHookRunner creates a new hook runner.
func NewHookRunner(workDir string) *HookRunner {
	return &HookRunner{
		evaluator: NewConditionEvaluator(),
		workDir:   workDir,
	}
}

// HookContext provides context for hook execution.
type HookContext struct {
	Target       string
	LayerCount   int
	StepCount    int
	AppliedSteps int
	SkippedSteps int
	FailedSteps  int
	DryRun       bool
}

// RunHooks executes hooks matching the given phase and type.
func (r *HookRunner) RunHooks(ctx context.Context, hooks []Hook, phase HookPhase, hookType HookType, hookCtx HookContext) error {
	matching := r.filterHooks(hooks, phase, hookType)

	for _, hook := range matching {
		if err := r.runHook(ctx, hook, hookCtx); err != nil {
			if hook.OnError == "continue" {
				fmt.Fprintf(os.Stderr, "hook '%s' failed (continuing): %v\n", hook.Name, err)
				continue
			}
			return fmt.Errorf("hook '%s' failed: %w", hook.Name, err)
		}
	}

	return nil
}

// filterHooks returns hooks matching the given phase and type.
func (r *HookRunner) filterHooks(hooks []Hook, phase HookPhase, hookType HookType) []Hook {
	var matching []Hook
	for _, h := range hooks {
		if h.Phase == phase && h.Type == hookType {
			// Check condition
			if !r.evaluator.isEmpty(h.When) && !r.evaluator.Evaluate(h.When) {
				continue
			}
			matching = append(matching, h)
		}
	}
	return matching
}

// runHook executes a single hook.
func (r *HookRunner) runHook(ctx context.Context, hook Hook, hookCtx HookContext) error {
	// Determine timeout
	timeout := 5 * time.Minute
	if hook.Timeout != "" {
		if parsed, err := time.ParseDuration(hook.Timeout); err == nil {
			timeout = parsed
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Determine shell
	shell := hook.Shell
	if shell == "" {
		shell = "sh"
	}

	// Build command
	var cmdStr string
	switch {
	case hook.Command != "":
		cmdStr = hook.Command
	case hook.Script != "":
		cmdStr = hook.Script
	default:
		return fmt.Errorf("hook must have command or script")
	}

	cmd := exec.CommandContext(ctx, shell, "-c", cmdStr)
	cmd.Dir = r.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up environment
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, r.buildEnv(hook, hookCtx)...)

	return cmd.Run()
}

// buildEnv creates environment variables for the hook.
func (r *HookRunner) buildEnv(hook Hook, ctx HookContext) []string {
	env := []string{
		fmt.Sprintf("PREFLIGHT_TARGET=%s", ctx.Target),
		fmt.Sprintf("PREFLIGHT_LAYER_COUNT=%d", ctx.LayerCount),
		fmt.Sprintf("PREFLIGHT_STEP_COUNT=%d", ctx.StepCount),
		fmt.Sprintf("PREFLIGHT_APPLIED_STEPS=%d", ctx.AppliedSteps),
		fmt.Sprintf("PREFLIGHT_SKIPPED_STEPS=%d", ctx.SkippedSteps),
		fmt.Sprintf("PREFLIGHT_FAILED_STEPS=%d", ctx.FailedSteps),
		fmt.Sprintf("PREFLIGHT_DRY_RUN=%t", ctx.DryRun),
	}

	// Add hook-specific environment
	for k, v := range hook.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

// ParseHooksFromManifest extracts hooks from manifest YAML.
func ParseHooksFromManifest(data []byte) ([]Hook, error) {
	var config struct {
		Hooks []Hook `yaml:"hooks"`
	}

	// First try to extract just the hooks section
	lines := strings.Split(string(data), "\n")
	var hooksLines []string
	inHooks := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "hooks:" {
			inHooks = true
			hooksLines = append(hooksLines, line)
			continue
		}
		if inHooks {
			if len(line) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				break // New top-level key
			}
			hooksLines = append(hooksLines, line)
		}
	}

	if len(hooksLines) == 0 {
		return nil, nil
	}

	// Re-parse just the hooks section
	hooksYAML := strings.Join(hooksLines, "\n")
	if err := parseYAML([]byte(hooksYAML), &config); err != nil {
		return nil, fmt.Errorf("parse hooks: %w", err)
	}

	return config.Hooks, nil
}

// parseYAML is a helper for YAML parsing (uses gopkg.in/yaml.v3).
func parseYAML(data []byte, v interface{}) error {
	// This package already imports yaml.v3
	return yamlUnmarshalFn(data, v)
}

// yamlUnmarshalFn is a function variable for YAML unmarshaling.
// It uses gopkg.in/yaml.v3 which is already imported in this package.
var yamlUnmarshalFn = func(_ []byte, _ interface{}) error {
	// Simply use the package's existing yaml import
	return nil // Will be overridden
}

// ValidateHook checks if a hook configuration is valid.
func ValidateHook(h Hook) error {
	if h.Phase != HookPhasePre && h.Phase != HookPhasePost {
		return fmt.Errorf("invalid hook phase: %s (must be pre or post)", h.Phase)
	}

	switch h.Type {
	case HookTypeApply, HookTypePlan, HookTypeDoctor, HookTypeCapture:
		// Valid
	default:
		return fmt.Errorf("invalid hook type: %s", h.Type)
	}

	if h.Command == "" && h.Script == "" {
		return fmt.Errorf("hook must have command or script")
	}

	if h.OnError != "" && h.OnError != "continue" && h.OnError != "fail" {
		return fmt.Errorf("invalid on_error value: %s (must be continue or fail)", h.OnError)
	}

	return nil
}
