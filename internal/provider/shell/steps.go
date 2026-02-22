package shell

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/adapters/command"
	"github.com/felixgeelhaar/preflight/internal/adapters/filesystem"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// FrameworkStep manages shell framework installation (oh-my-zsh, fisher, etc.).
type FrameworkStep struct {
	config Entry
	id     compiler.StepID
	fs     ports.FileSystem
	runner ports.CommandRunner
}

// NewFrameworkStep creates a new FrameworkStep with real dependencies.
func NewFrameworkStep(config Entry) *FrameworkStep {
	return NewFrameworkStepWith(config, filesystem.NewRealFileSystem(), command.NewRealRunner())
}

// NewFrameworkStepWithFS creates a new FrameworkStep with a custom filesystem and no runner.
//
// Deprecated: Use NewFrameworkStepWith for full functionality.
func NewFrameworkStepWithFS(config Entry, fs ports.FileSystem) *FrameworkStep {
	return NewFrameworkStepWith(config, fs, nil)
}

// NewFrameworkStepWith creates a new FrameworkStep with custom dependencies.
func NewFrameworkStepWith(config Entry, fs ports.FileSystem, runner ports.CommandRunner) *FrameworkStep {
	id := compiler.MustNewStepID(fmt.Sprintf("shell:framework:%s:%s", config.Name, config.Framework))
	return &FrameworkStep{
		config: config,
		id:     id,
		fs:     fs,
		runner: runner,
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
func (s *FrameworkStep) Apply(ctx compiler.RunContext) error {
	if s.runner == nil {
		return fmt.Errorf("command runner not configured for framework step")
	}

	switch s.config.Framework {
	case "oh-my-zsh":
		installScript := `RUNZSH=no KEEP_ZSHRC=yes sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"`
		result, err := s.runner.Run(ctx.Context(), "/bin/bash", "-c", installScript)
		if err != nil {
			return fmt.Errorf("oh-my-zsh install failed: %w", err)
		}
		if !result.Success() {
			return fmt.Errorf("oh-my-zsh install failed: %s", result.Stderr)
		}
	case "fisher":
		installScript := `curl -sL https://raw.githubusercontent.com/jorgebucaran/fisher/main/functions/fisher.fish | source && fisher install jorgebucaran/fisher`
		result, err := s.runner.Run(ctx.Context(), "fish", "-c", installScript)
		if err != nil {
			return fmt.Errorf("fisher install failed: %w", err)
		}
		if !result.Success() {
			return fmt.Errorf("fisher install failed: %s", result.Stderr)
		}
	case "oh-my-fish":
		result, err := s.runner.Run(ctx.Context(), "/bin/bash", "-c", "curl -L https://get.oh-my.fish | fish")
		if err != nil {
			return fmt.Errorf("oh-my-fish install failed: %w", err)
		}
		if !result.Success() {
			return fmt.Errorf("oh-my-fish install failed: %s", result.Stderr)
		}
	default:
		return fmt.Errorf("unsupported framework: %s", s.config.Framework)
	}
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
	fs        ports.FileSystem
}

// NewPluginStep creates a new PluginStep.
func NewPluginStep(shell, framework, plugin string) *PluginStep {
	return NewPluginStepWithFS(shell, framework, plugin, filesystem.NewRealFileSystem())
}

// NewPluginStepWithFS creates a new PluginStep with a custom filesystem.
func NewPluginStepWithFS(shell, framework, plugin string, fs ports.FileSystem) *PluginStep {
	id := compiler.MustNewStepID(fmt.Sprintf("shell:plugin:%s:%s", shell, plugin))
	return &PluginStep{
		shell:     shell,
		framework: framework,
		plugin:    plugin,
		id:        id,
		fs:        fs,
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

// Check verifies if the plugin is enabled in the shell config.
func (s *PluginStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := ports.ExpandPath(shellConfigPath(s.shell))
	if configPath == "" {
		return compiler.StatusNeedsApply, nil
	}

	content, err := s.fs.ReadFile(configPath)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // missing config means needs apply
	}

	if containsPlugin(string(content), s.plugin) {
		return compiler.StatusSatisfied, nil
	}
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

// Apply enables the plugin in the shell configuration file.
func (s *PluginStep) Apply(_ compiler.RunContext) error {
	configPath := ports.ExpandPath(shellConfigPath(s.shell))
	if configPath == "" {
		return fmt.Errorf("unsupported shell for plugin management: %s", s.shell)
	}

	content, err := s.fs.ReadFile(configPath)
	if err != nil {
		// Config file doesn't exist yet, create with plugin
		newContent := fmt.Sprintf("plugins=(%s)\n", s.plugin)
		return s.fs.WriteFile(configPath, []byte(newContent), 0o644)
	}

	updated := addPluginToConfig(string(content), s.plugin)
	return s.fs.WriteFile(configPath, []byte(updated), 0o644)
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
	runner    ports.CommandRunner
}

// NewCustomPluginStep creates a new CustomPluginStep with real dependencies.
func NewCustomPluginStep(shell, framework string, plugin CustomPlugin) *CustomPluginStep {
	return NewCustomPluginStepWith(shell, framework, plugin, filesystem.NewRealFileSystem(), command.NewRealRunner())
}

// NewCustomPluginStepWithFS creates a new CustomPluginStep with a custom filesystem and no runner.
//
// Deprecated: Use NewCustomPluginStepWith for full functionality.
func NewCustomPluginStepWithFS(shell, framework string, plugin CustomPlugin, fs ports.FileSystem) *CustomPluginStep {
	return NewCustomPluginStepWith(shell, framework, plugin, fs, nil)
}

// NewCustomPluginStepWith creates a new CustomPluginStep with custom dependencies.
func NewCustomPluginStepWith(shell, framework string, plugin CustomPlugin, fs ports.FileSystem, runner ports.CommandRunner) *CustomPluginStep {
	id := compiler.MustNewStepID(fmt.Sprintf("shell:custom-plugin:%s:%s", shell, plugin.Name))
	return &CustomPluginStep{
		shell:     shell,
		framework: framework,
		plugin:    plugin,
		id:        id,
		fs:        fs,
		runner:    runner,
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

// Apply clones the custom plugin from its git repository.
func (s *CustomPluginStep) Apply(ctx compiler.RunContext) error {
	if s.runner == nil {
		return fmt.Errorf("command runner not configured for custom plugin step")
	}

	path := s.pluginPath()
	if path == "" {
		return fmt.Errorf("unsupported framework for custom plugins: %s", s.framework)
	}

	result, err := s.runner.Run(ctx.Context(), "git", "clone", s.plugin.Repo, path)
	if err != nil {
		return fmt.Errorf("git clone failed for %s: %w", s.plugin.Name, err)
	}
	if !result.Success() {
		return fmt.Errorf("git clone failed for %s: %s", s.plugin.Name, result.Stderr)
	}
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
	fs    ports.FileSystem
}

// NewEnvStep creates a new EnvStep.
func NewEnvStep(shell string, env map[string]string) *EnvStep {
	return NewEnvStepWithFS(shell, env, filesystem.NewRealFileSystem())
}

// NewEnvStepWithFS creates a new EnvStep with a custom filesystem.
func NewEnvStepWithFS(shell string, env map[string]string, fs ports.FileSystem) *EnvStep {
	id := compiler.MustNewStepID(fmt.Sprintf("shell:env:%s", shell))
	return &EnvStep{
		shell: shell,
		env:   env,
		id:    id,
		fs:    fs,
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

// Check verifies if environment variables are set in the config file.
func (s *EnvStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := ports.ExpandPath(shellConfigPath(s.shell))
	if configPath == "" {
		return compiler.StatusNeedsApply, nil
	}

	content, err := s.fs.ReadFile(configPath)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // missing config means needs apply
	}

	existing := ReadManagedBlock(string(content), "env")
	desired := generateEnvBlock(s.env)
	if existing == desired {
		return compiler.StatusSatisfied, nil
	}
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

// Apply writes environment variables to the shell config file.
func (s *EnvStep) Apply(_ compiler.RunContext) error {
	configPath := ports.ExpandPath(shellConfigPath(s.shell))
	if configPath == "" {
		return fmt.Errorf("unsupported shell for env management: %s", s.shell)
	}

	content, err := s.fs.ReadFile(configPath)
	if err != nil {
		content = []byte{}
	}

	block := generateEnvBlock(s.env)
	updated := WriteManagedBlock(string(content), "env", block)
	return s.fs.WriteFile(configPath, []byte(updated), 0o644)
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
	fs      ports.FileSystem
}

// NewAliasStep creates a new AliasStep.
func NewAliasStep(shell string, aliases map[string]string) *AliasStep {
	return NewAliasStepWithFS(shell, aliases, filesystem.NewRealFileSystem())
}

// NewAliasStepWithFS creates a new AliasStep with a custom filesystem.
func NewAliasStepWithFS(shell string, aliases map[string]string, fs ports.FileSystem) *AliasStep {
	id := compiler.MustNewStepID(fmt.Sprintf("shell:aliases:%s", shell))
	return &AliasStep{
		shell:   shell,
		aliases: aliases,
		id:      id,
		fs:      fs,
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

// Check verifies if aliases are configured in the shell config file.
func (s *AliasStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	configPath := ports.ExpandPath(shellConfigPath(s.shell))
	if configPath == "" {
		return compiler.StatusNeedsApply, nil
	}

	content, err := s.fs.ReadFile(configPath)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // missing config means needs apply
	}

	existing := ReadManagedBlock(string(content), "aliases")
	desired := generateAliasBlock(s.aliases)
	if existing == desired {
		return compiler.StatusSatisfied, nil
	}
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

// Apply writes aliases to the shell config file.
func (s *AliasStep) Apply(_ compiler.RunContext) error {
	configPath := ports.ExpandPath(shellConfigPath(s.shell))
	if configPath == "" {
		return fmt.Errorf("unsupported shell for alias management: %s", s.shell)
	}

	content, err := s.fs.ReadFile(configPath)
	if err != nil {
		content = []byte{}
	}

	block := generateAliasBlock(s.aliases)
	updated := WriteManagedBlock(string(content), "aliases", block)
	return s.fs.WriteFile(configPath, []byte(updated), 0o644)
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

// Apply creates the starship configuration file.
func (s *StarshipStep) Apply(_ compiler.RunContext) error {
	configPath := ports.ExpandPath("~/.config/starship.toml")
	if err := s.fs.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	content := s.generateConfig()
	return s.fs.WriteFile(configPath, content, 0o644)
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

func (s *StarshipStep) generateConfig() []byte {
	var b strings.Builder
	b.WriteString("# Managed by preflight\n")
	if s.config.Preset != "" {
		fmt.Fprintf(&b, `"$schema" = 'https://starship.rs/config-schema.json'`+"\n")
		fmt.Fprintf(&b, "palette = '%s'\n", s.config.Preset)
	}
	return []byte(b.String())
}

// FisherPluginStep manages fisher plugin installation.
type FisherPluginStep struct {
	plugin string
	id     compiler.StepID
	fs     ports.FileSystem
	runner ports.CommandRunner
}

// NewFisherPluginStep creates a new FisherPluginStep with real dependencies.
func NewFisherPluginStep(plugin string) *FisherPluginStep {
	return NewFisherPluginStepWith(plugin, filesystem.NewRealFileSystem(), command.NewRealRunner())
}

// NewFisherPluginStepWithFS creates a new FisherPluginStep with a custom filesystem and no runner.
//
// Deprecated: Use NewFisherPluginStepWith for full functionality.
func NewFisherPluginStepWithFS(plugin string, fs ports.FileSystem) *FisherPluginStep {
	return NewFisherPluginStepWith(plugin, fs, nil)
}

// NewFisherPluginStepWith creates a new FisherPluginStep with custom dependencies.
func NewFisherPluginStepWith(plugin string, fs ports.FileSystem, runner ports.CommandRunner) *FisherPluginStep {
	// Sanitize plugin name for step ID (replace dots with dashes)
	sanitizedPlugin := strings.ReplaceAll(plugin, ".", "-")
	id := compiler.MustNewStepID(fmt.Sprintf("shell:fisher:%s", sanitizedPlugin))
	return &FisherPluginStep{
		plugin: plugin,
		id:     id,
		fs:     fs,
		runner: runner,
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

// Check verifies if the fisher plugin is installed by reading the fish_plugins file.
func (s *FisherPluginStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	pluginsPath := ports.ExpandPath("~/.config/fish/fish_plugins")
	content, err := s.fs.ReadFile(pluginsPath)
	if err != nil {
		return compiler.StatusNeedsApply, nil //nolint:nilerr // missing file means needs apply
	}
	if strings.Contains(string(content), s.plugin) {
		return compiler.StatusSatisfied, nil
	}
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

// Apply installs the fisher plugin via the fish shell.
func (s *FisherPluginStep) Apply(ctx compiler.RunContext) error {
	if s.runner == nil {
		return fmt.Errorf("command runner not configured for fisher plugin step")
	}

	result, err := s.runner.Run(ctx.Context(), "fish", "-c", fmt.Sprintf("fisher install %s", s.plugin))
	if err != nil {
		return fmt.Errorf("fisher install %s failed: %w", s.plugin, err)
	}
	if !result.Success() {
		return fmt.Errorf("fisher install %s failed: %s", s.plugin, result.Stderr)
	}
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
