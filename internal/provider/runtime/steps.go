package runtime

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// ToolVersionStep manages the .tool-versions file.
type ToolVersionStep struct {
	cfg *Config
	id  compiler.StepID
	fs  ports.FileSystem
}

// NewToolVersionStep creates a new ToolVersionStep.
func NewToolVersionStep(cfg *Config, fs ports.FileSystem) *ToolVersionStep {
	id := compiler.MustNewStepID("runtime:tool-versions")
	return &ToolVersionStep{
		cfg: cfg,
		id:  id,
		fs:  fs,
	}
}

// ID returns the step identifier.
func (s *ToolVersionStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *ToolVersionStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if the .tool-versions file is up to date.
func (s *ToolVersionStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	path := s.resolvedPath()
	if !s.fs.Exists(path) {
		return compiler.StatusNeedsApply, nil
	}

	existing, err := s.fs.ReadFile(path)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // file read error means needs apply
	}

	desired := s.generateContent()
	if bytes.Equal(existing, desired) {
		return compiler.StatusSatisfied, nil
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ToolVersionStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"toolversions",
		s.cfg.ToolVersionsPath(),
		"",
		fmt.Sprintf("%d tools configured", len(s.cfg.Tools)),
	), nil
}

// Apply writes the .tool-versions file.
func (s *ToolVersionStep) Apply(_ compiler.RunContext) error {
	path := s.resolvedPath()
	content := s.generateContent()
	return s.fs.WriteFile(path, content, 0o644)
}

// Explain provides context for this step.
func (s *ToolVersionStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	tools := make([]string, 0, len(s.cfg.Tools))
	for _, t := range s.cfg.Tools {
		tools = append(tools, fmt.Sprintf("%s@%s", t.Name, t.Version))
	}
	return compiler.NewExplanation(
		"Manage Tool Versions",
		fmt.Sprintf("Write %s with: %v", s.cfg.ToolVersionsPath(), tools),
		nil,
	)
}

func (s *ToolVersionStep) resolvedPath() string {
	path := s.cfg.ToolVersionsPath()
	if path == "~/.tool-versions" {
		return ports.ExpandPath(path)
	}
	return path
}

func (s *ToolVersionStep) generateContent() []byte {
	var buf bytes.Buffer
	for _, tool := range s.cfg.Tools {
		fmt.Fprintf(&buf, "%s %s\n", tool.Name, tool.Version)
	}
	return buf.Bytes()
}

// PluginStep manages an asdf/rtx plugin.
type PluginStep struct {
	plugin  PluginConfig
	backend string
	id      compiler.StepID
	runner  ports.CommandRunner
}

// NewPluginStep creates a new PluginStep without a runner (backward compatible).
func NewPluginStep(plugin PluginConfig) *PluginStep {
	return NewPluginStepWith(plugin, "", nil)
}

// NewPluginStepWith creates a new PluginStep with a command runner and backend.
func NewPluginStepWith(plugin PluginConfig, backend string, runner ports.CommandRunner) *PluginStep {
	id := compiler.MustNewStepID(fmt.Sprintf("runtime:plugin:%s", plugin.Name))
	return &PluginStep{
		plugin:  plugin,
		backend: backend,
		id:      id,
		runner:  runner,
	}
}

// ID returns the step identifier.
func (s *PluginStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *PluginStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if the plugin is installed by querying the runtime backend.
func (s *PluginStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	if s.runner == nil {
		return compiler.StatusNeedsApply, nil
	}

	bin := s.backendBinary()
	result, err := s.runner.Run(ctx.Context(), bin, "plugin", "list")
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // command failure means needs apply
	}

	for _, line := range strings.Split(result.Stdout, "\n") {
		if strings.TrimSpace(line) == s.plugin.Name {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *PluginStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	desc := fmt.Sprintf("Install plugin: %s", s.plugin.Name)
	if s.plugin.URL != "" {
		desc = fmt.Sprintf("%s from %s", desc, s.plugin.URL)
	}
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"plugin",
		s.plugin.Name,
		"",
		desc,
	), nil
}

// Apply installs the plugin using the runtime backend.
func (s *PluginStep) Apply(ctx compiler.RunContext) error {
	if s.runner == nil {
		return fmt.Errorf("command runner not configured for runtime plugin step")
	}

	bin := s.backendBinary()
	args := []string{"plugin", "add", s.plugin.Name}
	if s.plugin.URL != "" {
		args = append(args, s.plugin.URL)
	}

	result, err := s.runner.Run(ctx.Context(), bin, args...)
	if err != nil {
		return fmt.Errorf("%s plugin add %s failed: %w", bin, s.plugin.Name, err)
	}
	if !result.Success() {
		return fmt.Errorf("%s plugin add %s failed: %s", bin, s.plugin.Name, result.Stderr)
	}
	return nil
}

// Explain provides context for this step.
func (s *PluginStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Plugin",
		fmt.Sprintf("Install asdf plugin for %s version management", s.plugin.Name),
		nil,
	)
}

// backendBinary returns the binary name for the configured backend.
func (s *PluginStep) backendBinary() string {
	switch s.backend {
	case "rtx", "mise":
		return "mise"
	default:
		return "asdf"
	}
}
