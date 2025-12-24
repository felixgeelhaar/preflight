package vscode

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// RemoteWSLExtensionID is the extension ID for VS Code Remote-WSL.
const RemoteWSLExtensionID = "ms-vscode-remote.remote-wsl"

// RemoteWSLSetupStep installs the Remote-WSL extension on Windows.
type RemoteWSLSetupStep struct {
	id       compiler.StepID
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewRemoteWSLSetupStep creates a new RemoteWSLSetupStep.
func NewRemoteWSLSetupStep(runner ports.CommandRunner, plat *platform.Platform) *RemoteWSLSetupStep {
	id := compiler.MustNewStepID("vscode:wsl:setup")
	return &RemoteWSLSetupStep{
		id:       id,
		runner:   runner,
		platform: plat,
	}
}

// ID returns the step identifier.
func (s *RemoteWSLSetupStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *RemoteWSLSetupStep) DependsOn() []compiler.StepID {
	return nil
}

// codeCommand returns the appropriate VS Code CLI command for the platform.
func (s *RemoteWSLSetupStep) codeCommand() string {
	if s.platform != nil && s.platform.IsWSL() {
		return "code.exe"
	}
	return "code"
}

// Check verifies if the Remote-WSL extension is installed.
func (s *RemoteWSLSetupStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), s.codeCommand(), "--list-extensions")
	if err != nil {
		return compiler.StatusUnknown, err
	}

	extensions := strings.Split(result.Stdout, "\n")
	for _, ext := range extensions {
		if strings.TrimSpace(ext) == RemoteWSLExtensionID {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *RemoteWSLSetupStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"extension",
		RemoteWSLExtensionID,
		"",
		"Install VS Code Remote-WSL extension",
	), nil
}

// Apply installs the Remote-WSL extension.
func (s *RemoteWSLSetupStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), s.codeCommand(),
		"--install-extension", RemoteWSLExtensionID, "--force")
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("failed to install Remote-WSL extension: %s", result.Stderr)
	}
	return nil
}

// Explain provides context for this step.
func (s *RemoteWSLSetupStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Remote-WSL Extension",
		"Install the VS Code Remote-WSL extension to enable development inside WSL from Windows",
		[]string{
			"https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-wsl",
			"https://code.visualstudio.com/docs/remote/wsl",
		},
	).WithTradeoffs([]string{
		"+ Develop in Linux environment with Windows VS Code",
		"+ Access to WSL filesystem and tools",
		"+ Seamless integration between Windows and WSL",
		"- Requires WSL to be installed and configured",
	})
}

// RemoteWSLExtensionStep installs an extension in the WSL remote context.
type RemoteWSLExtensionStep struct {
	extension string
	distro    string
	id        compiler.StepID
	runner    ports.CommandRunner
	platform  *platform.Platform
}

// NewRemoteWSLExtensionStep creates a new RemoteWSLExtensionStep.
func NewRemoteWSLExtensionStep(extension, distro string, runner ports.CommandRunner, plat *platform.Platform) *RemoteWSLExtensionStep {
	// Replace dots with underscores for valid step ID
	safeExt := strings.ReplaceAll(extension, ".", "_")
	id := compiler.MustNewStepID(fmt.Sprintf("vscode:wsl:extension:%s", safeExt))
	return &RemoteWSLExtensionStep{
		extension: extension,
		distro:    distro,
		id:        id,
		runner:    runner,
		platform:  plat,
	}
}

// ID returns the step identifier.
func (s *RemoteWSLExtensionStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *RemoteWSLExtensionStep) DependsOn() []compiler.StepID {
	// WSL extensions depend on the Remote-WSL setup
	return []compiler.StepID{compiler.MustNewStepID("vscode:wsl:setup")}
}

// codeCommand returns the appropriate VS Code CLI command for the platform.
func (s *RemoteWSLExtensionStep) codeCommand() string {
	if s.platform != nil && s.platform.IsWSL() {
		return "code.exe"
	}
	return "code"
}

// remoteTarget returns the remote target string for WSL.
func (s *RemoteWSLExtensionStep) remoteTarget() string {
	if s.distro != "" {
		return fmt.Sprintf("wsl+%s", s.distro)
	}
	// When running in WSL, use the current distro
	if s.platform != nil && s.platform.IsWSL() && s.platform.WSLDistro() != "" {
		return fmt.Sprintf("wsl+%s", s.platform.WSLDistro())
	}
	// Default to generic WSL
	return "wsl"
}

// Check verifies if the extension is installed in WSL.
func (s *RemoteWSLExtensionStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	// List extensions in the WSL remote context
	result, err := s.runner.Run(ctx.Context(), s.codeCommand(),
		"--remote", s.remoteTarget(), "--list-extensions")
	if err != nil {
		// If code fails with remote, it might not be set up yet
		return compiler.StatusNeedsApply, nil //nolint:nilerr // expected behavior
	}

	extensions := strings.Split(result.Stdout, "\n")
	for _, ext := range extensions {
		if strings.TrimSpace(ext) == s.extension {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *RemoteWSLExtensionStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	target := s.remoteTarget()
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"wsl-extension",
		s.extension,
		"",
		fmt.Sprintf("Install extension %s in %s", s.extension, target),
	), nil
}

// Apply installs the extension in the WSL remote context.
func (s *RemoteWSLExtensionStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), s.codeCommand(),
		"--remote", s.remoteTarget(),
		"--install-extension", s.extension, "--force")
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("failed to install extension %s in WSL: %s", s.extension, result.Stderr)
	}
	return nil
}

// Explain provides context for this step.
func (s *RemoteWSLExtensionStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	target := s.remoteTarget()
	return compiler.NewExplanation(
		"Install WSL Remote Extension",
		fmt.Sprintf("Install extension %s in the WSL remote context (%s)", s.extension, target),
		[]string{
			fmt.Sprintf("https://marketplace.visualstudio.com/items?itemName=%s", s.extension),
			"https://code.visualstudio.com/docs/remote/wsl",
		},
	).WithTradeoffs([]string{
		"+ Extension runs in WSL for better Linux compatibility",
		"+ Access to Linux-native tools and libraries",
		"- Extension must support remote development",
	})
}

// RemoteWSLSettingsStep manages VS Code settings specific to WSL remote.
type RemoteWSLSettingsStep struct {
	settings map[string]interface{}
	distro   string
	id       compiler.StepID
	fs       ports.FileSystem
	platform *platform.Platform
}

// NewRemoteWSLSettingsStep creates a new RemoteWSLSettingsStep.
func NewRemoteWSLSettingsStep(settings map[string]interface{}, distro string, fs ports.FileSystem, plat *platform.Platform) *RemoteWSLSettingsStep {
	id := compiler.MustNewStepID("vscode:wsl:settings")
	return &RemoteWSLSettingsStep{
		settings: settings,
		distro:   distro,
		id:       id,
		fs:       fs,
		platform: plat,
	}
}

// ID returns the step identifier.
func (s *RemoteWSLSettingsStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns dependencies for this step.
func (s *RemoteWSLSettingsStep) DependsOn() []compiler.StepID {
	return []compiler.StepID{compiler.MustNewStepID("vscode:wsl:setup")}
}

// Check verifies if settings need to be applied.
func (s *RemoteWSLSettingsStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	settingsPath := s.getSettingsPath()
	if !s.fs.Exists(settingsPath) {
		return compiler.StatusNeedsApply, nil
	}
	// Future: deep compare settings
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *RemoteWSLSettingsStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeModify,
		"wsl-settings",
		"settings.json",
		"",
		"Configure VS Code settings for WSL remote",
	), nil
}

// Apply writes the settings file.
func (s *RemoteWSLSettingsStep) Apply(_ compiler.RunContext) error {
	settingsPath := s.getSettingsPath()

	// Merge with existing settings
	existingSettings := make(map[string]interface{})
	if s.fs.Exists(settingsPath) {
		content, err := s.fs.ReadFile(settingsPath)
		if err == nil {
			_ = json.Unmarshal(content, &existingSettings)
		}
	}

	// Merge WSL-specific settings into existing settings
	for k, v := range s.settings {
		existingSettings[k] = v
	}

	// Write merged settings
	data, err := json.MarshalIndent(existingSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := s.fs.WriteFile(settingsPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	return nil
}

// Explain provides context for this step.
func (s *RemoteWSLSettingsStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure WSL Remote Settings",
		"Apply VS Code settings specific to WSL remote development, such as default extensions and terminal configuration",
		[]string{
			"https://code.visualstudio.com/docs/remote/wsl",
			"https://code.visualstudio.com/docs/getstarted/settings",
		},
	).WithTradeoffs([]string{
		"+ WSL-specific settings don't affect Windows development",
		"+ Can customize terminal, extensions, and paths for Linux",
		"- Settings require VS Code restart to take effect",
	})
}

func (s *RemoteWSLSettingsStep) getSettingsPath() string {
	// VS Code settings are stored in Windows AppData when using Remote-WSL
	// but can also be configured via the remote's settings
	return ports.ExpandPath("~/.config/Code/User/settings.json")
}
