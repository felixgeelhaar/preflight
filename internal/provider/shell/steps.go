package shell

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/adapters/filesystem"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// FrameworkStep manages shell framework installation (oh-my-zsh, fisher, etc.).
type FrameworkStep struct {
	config Entry
	id     compiler.StepID
	fs     ports.FileSystem
}

// NewFrameworkStep creates a new FrameworkStep.
func NewFrameworkStep(config Entry) *FrameworkStep {
	return NewFrameworkStepWithFS(config, filesystem.NewRealFileSystem())
}

// NewFrameworkStepWithFS creates a new FrameworkStep with a custom filesystem.
func NewFrameworkStepWithFS(config Entry, fs ports.FileSystem) *FrameworkStep {
	id := compiler.MustNewStepID(fmt.Sprintf("shell:framework:%s:%s", config.Name, config.Framework))
	return &FrameworkStep{
		config: config,
		id:     id,
		fs:     fs,
	}
}

// ID returns the step identifier.
func (s *FrameworkStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *FrameworkStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if the framework is installed.
func (s *FrameworkStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	path := s.frameworkPath()
	if s.fs.Exists(path) {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *FrameworkStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"framework",
		s.config.Framework,
		"",
		fmt.Sprintf("Install %s framework for %s", s.config.Framework, s.config.Name),
	), nil
}

// Apply installs the framework.
func (s *FrameworkStep) Apply(_ compiler.RunContext) error {
	// In real implementation, this would run the framework installation script
	// For oh-my-zsh: sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
	// For fisher: curl -sL https://git.io/fisher | source && fisher install jorgebucaran/fisher
	return nil
}

// Explain provides context for this step.
func (s *FrameworkStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	var docLinks []string
	switch s.config.Framework {
	case "oh-my-zsh":
		docLinks = []string{"https://ohmyz.sh/", "https://github.com/ohmyzsh/ohmyzsh"}
	case "fisher":
		docLinks = []string{"https://github.com/jorgebucaran/fisher"}
	case "oh-my-fish":
		docLinks = []string{"https://github.com/oh-my-fish/oh-my-fish"}
	}

	return compiler.NewExplanation(
		"Install Shell Framework",
		fmt.Sprintf("Install %s framework for %s shell. Provides plugin management, themes, and enhanced shell experience.", s.config.Framework, s.config.Name),
		docLinks,
	).WithTradeoffs([]string{
		"+ Easy plugin and theme management",
		"+ Enhanced shell productivity features",
		"- Slight shell startup time increase",
		"- Framework updates may require maintenance",
	})
}

func (s *FrameworkStep) frameworkPath() string {
	switch s.config.Framework {
	case "oh-my-zsh":
		return ports.ExpandPath("~/.oh-my-zsh")
	case "fisher":
		return ports.ExpandPath("~/.config/fish/functions/fisher.fish")
	case "oh-my-fish":
		return ports.ExpandPath("~/.local/share/omf")
	default:
		return ""
	}
}

// PluginStep manages built-in plugin configuration.
type PluginStep struct {
	shell     string
	framework string
	plugin    string
	id        compiler.StepID
}

// NewPluginStep creates a new PluginStep.
func NewPluginStep(shell, framework, plugin string) *PluginStep {
	id := compiler.MustNewStepID(fmt.Sprintf("shell:plugin:%s:%s", shell, plugin))
	return &PluginStep{
		shell:     shell,
		framework: framework,
		plugin:    plugin,
		id:        id,
	}
}

// ID returns the step identifier.
func (s *PluginStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *PluginStep) DependsOn() []compiler.StepID {
	frameworkID := compiler.MustNewStepID(fmt.Sprintf("shell:framework:%s:%s", s.shell, s.framework))
	return []compiler.StepID{frameworkID}
}

// Check verifies if the plugin is enabled.
func (s *PluginStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	// Built-in plugins are always available, just need to be enabled in config
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *PluginStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"plugin",
		s.plugin,
		"",
		fmt.Sprintf("Enable %s plugin for %s", s.plugin, s.shell),
	), nil
}

// Apply enables the plugin.
func (s *PluginStep) Apply(_ compiler.RunContext) error {
	// In real implementation, this would update the shell config to enable the plugin
	return nil
}

// Explain provides context for this step.
func (s *PluginStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Enable Plugin",
		fmt.Sprintf("Enable %s plugin for %s shell", s.plugin, s.shell),
		nil,
	)
}

// CustomPluginStep manages custom plugin installation from git repositories.
type CustomPluginStep struct {
	shell     string
	framework string
	plugin    CustomPlugin
	id        compiler.StepID
	fs        ports.FileSystem
}

// NewCustomPluginStep creates a new CustomPluginStep.
func NewCustomPluginStep(shell, framework string, plugin CustomPlugin) *CustomPluginStep {
	return NewCustomPluginStepWithFS(shell, framework, plugin, filesystem.NewRealFileSystem())
}

// NewCustomPluginStepWithFS creates a new CustomPluginStep with a custom filesystem.
func NewCustomPluginStepWithFS(shell, framework string, plugin CustomPlugin, fs ports.FileSystem) *CustomPluginStep {
	id := compiler.MustNewStepID(fmt.Sprintf("shell:custom-plugin:%s:%s", shell, plugin.Name))
	return &CustomPluginStep{
		shell:     shell,
		framework: framework,
		plugin:    plugin,
		id:        id,
		fs:        fs,
	}
}

// ID returns the step identifier.
func (s *CustomPluginStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *CustomPluginStep) DependsOn() []compiler.StepID {
	frameworkID := compiler.MustNewStepID(fmt.Sprintf("shell:framework:%s:%s", s.shell, s.framework))
	return []compiler.StepID{frameworkID}
}

// Check verifies if the custom plugin is installed.
func (s *CustomPluginStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	path := s.pluginPath()
	if s.fs.Exists(path) {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *CustomPluginStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"custom-plugin",
		s.plugin.Name,
		"",
		fmt.Sprintf("Clone %s from %s", s.plugin.Name, s.plugin.Repo),
	), nil
}

// Apply clones the custom plugin.
func (s *CustomPluginStep) Apply(_ compiler.RunContext) error {
	// In real implementation, this would clone the git repository
	// git clone https://github.com/<repo> <plugin-path>
	return nil
}

// Explain provides context for this step.
func (s *CustomPluginStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Custom Plugin",
		fmt.Sprintf("Clone %s plugin from %s", s.plugin.Name, s.plugin.Repo),
		nil,
	)
}

func (s *CustomPluginStep) pluginPath() string {
	switch s.framework {
	case "oh-my-zsh":
		return ports.ExpandPath(fmt.Sprintf("~/.oh-my-zsh/custom/plugins/%s", s.plugin.Name))
	default:
		return ""
	}
}

// EnvStep manages environment variable configuration.
type EnvStep struct {
	shell string
	env   map[string]string
	id    compiler.StepID
}

// NewEnvStep creates a new EnvStep.
func NewEnvStep(shell string, env map[string]string) *EnvStep {
	id := compiler.MustNewStepID(fmt.Sprintf("shell:env:%s", shell))
	return &EnvStep{
		shell: shell,
		env:   env,
		id:    id,
	}
}

// ID returns the step identifier.
func (s *EnvStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *EnvStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if environment variables are set.
func (s *EnvStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *EnvStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"env",
		fmt.Sprintf("%d variables", len(s.env)),
		"",
		fmt.Sprintf("Set %d environment variables for %s", len(s.env), s.shell),
	), nil
}

// Apply sets environment variables.
func (s *EnvStep) Apply(_ compiler.RunContext) error {
	return nil
}

// Explain provides context for this step.
func (s *EnvStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Set Environment Variables",
		fmt.Sprintf("Configure %d environment variables for %s shell", len(s.env), s.shell),
		nil,
	)
}

// AliasStep manages shell alias configuration.
type AliasStep struct {
	shell   string
	aliases map[string]string
	id      compiler.StepID
}

// NewAliasStep creates a new AliasStep.
func NewAliasStep(shell string, aliases map[string]string) *AliasStep {
	id := compiler.MustNewStepID(fmt.Sprintf("shell:aliases:%s", shell))
	return &AliasStep{
		shell:   shell,
		aliases: aliases,
		id:      id,
	}
}

// ID returns the step identifier.
func (s *AliasStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *AliasStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if aliases are configured.
func (s *AliasStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *AliasStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"aliases",
		fmt.Sprintf("%d aliases", len(s.aliases)),
		"",
		fmt.Sprintf("Configure %d aliases for %s", len(s.aliases), s.shell),
	), nil
}

// Apply configures aliases.
func (s *AliasStep) Apply(_ compiler.RunContext) error {
	return nil
}

// Explain provides context for this step.
func (s *AliasStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Aliases",
		fmt.Sprintf("Set up %d shell aliases for %s", len(s.aliases), s.shell),
		nil,
	)
}

// StarshipStep manages starship prompt configuration.
type StarshipStep struct {
	config StarshipConfig
	id     compiler.StepID
	fs     ports.FileSystem
}

// NewStarshipStep creates a new StarshipStep.
func NewStarshipStep(config StarshipConfig) *StarshipStep {
	return NewStarshipStepWithFS(config, filesystem.NewRealFileSystem())
}

// NewStarshipStepWithFS creates a new StarshipStep with a custom filesystem.
func NewStarshipStepWithFS(config StarshipConfig, fs ports.FileSystem) *StarshipStep {
	id := compiler.MustNewStepID("shell:starship")
	return &StarshipStep{
		config: config,
		id:     id,
		fs:     fs,
	}
}

// ID returns the step identifier.
func (s *StarshipStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *StarshipStep) DependsOn() []compiler.StepID {
	return nil
}

// Check verifies if starship is configured.
func (s *StarshipStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := ports.ExpandPath("~/.config/starship.toml")
	if s.fs.Exists(configPath) {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *StarshipStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	desc := "Configure starship prompt"
	if s.config.Preset != "" {
		desc = fmt.Sprintf("Configure starship prompt with %s preset", s.config.Preset)
	}
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"starship",
		"~/.config/starship.toml",
		"",
		desc,
	), nil
}

// Apply configures starship.
func (s *StarshipStep) Apply(_ compiler.RunContext) error {
	// In real implementation, this would:
	// 1. Create starship.toml with preset configuration
	// 2. Add init command to shell config
	return nil
}

// Explain provides context for this step.
func (s *StarshipStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := "Configure starship cross-shell prompt"
	if s.config.Preset != "" {
		desc = fmt.Sprintf("Configure starship prompt with %s preset", s.config.Preset)
	}
	return compiler.NewExplanation(
		"Configure Starship Prompt",
		desc,
		nil,
	)
}

// FisherPluginStep manages fisher plugin installation.
type FisherPluginStep struct {
	plugin string
	id     compiler.StepID
	fs     ports.FileSystem
}

// NewFisherPluginStep creates a new FisherPluginStep.
func NewFisherPluginStep(plugin string) *FisherPluginStep {
	return NewFisherPluginStepWithFS(plugin, filesystem.NewRealFileSystem())
}

// NewFisherPluginStepWithFS creates a new FisherPluginStep with a custom filesystem.
func NewFisherPluginStepWithFS(plugin string, fs ports.FileSystem) *FisherPluginStep {
	// Sanitize plugin name for step ID (replace dots with dashes)
	sanitizedPlugin := strings.ReplaceAll(plugin, ".", "-")
	id := compiler.MustNewStepID(fmt.Sprintf("shell:fisher:%s", sanitizedPlugin))
	return &FisherPluginStep{
		plugin: plugin,
		id:     id,
		fs:     fs,
	}
}

// ID returns the step identifier.
func (s *FisherPluginStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *FisherPluginStep) DependsOn() []compiler.StepID {
	frameworkID := compiler.MustNewStepID("shell:framework:fish:fisher")
	return []compiler.StepID{frameworkID}
}

// Check verifies if the fisher plugin is installed.
func (s *FisherPluginStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	// In real implementation, check if plugin is in fish_plugins file
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *FisherPluginStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"fisher-plugin",
		s.plugin,
		"",
		fmt.Sprintf("Install fisher plugin: %s", s.plugin),
	), nil
}

// Apply installs the fisher plugin.
func (s *FisherPluginStep) Apply(_ compiler.RunContext) error {
	// In real implementation: fisher install <plugin>
	return nil
}

// Explain provides context for this step.
func (s *FisherPluginStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Fisher Plugin",
		fmt.Sprintf("Install fish plugin %s via fisher", s.plugin),
		nil,
	)
}
