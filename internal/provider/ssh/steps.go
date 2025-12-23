package ssh

import (
	"bytes"
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// ConfigStep generates the ~/.ssh/config file.
type ConfigStep struct {
	cfg *Config
	id  compiler.StepID
	fs  ports.FileSystem
}

// NewConfigStep creates a new ConfigStep.
func NewConfigStep(cfg *Config, fs ports.FileSystem) *ConfigStep {
	id := compiler.MustNewStepID("ssh:config")
	return &ConfigStep{
		cfg: cfg,
		id:  id,
		fs:  fs,
	}
}

// ID returns the step identifier.
func (s *ConfigStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *ConfigStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the config needs to be updated.
func (s *ConfigStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	path := ports.ExpandPath(s.cfg.ConfigPath())

	if !s.fs.Exists(path) {
		return compiler.StatusNeedsApply, nil
	}

	// Compare content
	existing, err := s.fs.ReadFile(path)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	expected := s.generateConfig()
	if bytes.Equal(existing, expected) {
		return compiler.StatusSatisfied, nil
	}

	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *ConfigStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"sshconfig",
		s.cfg.ConfigPath(),
		"",
		"generated",
	), nil
}

// Apply generates and writes the ~/.ssh/config file.
func (s *ConfigStep) Apply(_ compiler.RunContext) error {
	// Validate all SSH config values before writing to prevent injection attacks
	if err := s.validateConfig(); err != nil {
		return fmt.Errorf("SSH config validation failed: %w", err)
	}

	path := ports.ExpandPath(s.cfg.ConfigPath())
	content := s.generateConfig()

	// Ensure ~/.ssh directory exists
	sshDir := ports.ExpandPath("~/.ssh")
	if !s.fs.Exists(sshDir) {
		if err := s.fs.MkdirAll(sshDir, 0700); err != nil {
			return fmt.Errorf("failed to create .ssh directory: %w", err)
		}
	}

	// Write with restrictive permissions (SSH requires 0600 or more restrictive)
	if err := s.fs.WriteFile(path, content, 0600); err != nil {
		return fmt.Errorf("failed to write ssh config: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *ConfigStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Generate SSH Config",
		fmt.Sprintf("Generates %s with host configurations and settings.", s.cfg.ConfigPath()),
		nil,
	)
}

// generateConfig generates the ~/.ssh/config content.
func (s *ConfigStep) generateConfig() []byte {
	var buf bytes.Buffer

	// Write Include directive first (if present)
	if s.cfg.Include != "" {
		fmt.Fprintf(&buf, "Include %s\n\n", s.cfg.Include)
	}

	// Write defaults (Host *)
	if s.hasDefaults() {
		buf.WriteString("Host *\n")
		if s.cfg.Defaults.AddKeysToAgent {
			buf.WriteString("    AddKeysToAgent yes\n")
		}
		if s.cfg.Defaults.IdentitiesOnly {
			buf.WriteString("    IdentitiesOnly yes\n")
		}
		if s.cfg.Defaults.ForwardAgent {
			buf.WriteString("    ForwardAgent yes\n")
		}
		if s.cfg.Defaults.ServerAliveInterval > 0 {
			fmt.Fprintf(&buf, "    ServerAliveInterval %d\n", s.cfg.Defaults.ServerAliveInterval)
		}
		if s.cfg.Defaults.ServerAliveCountMax > 0 {
			fmt.Fprintf(&buf, "    ServerAliveCountMax %d\n", s.cfg.Defaults.ServerAliveCountMax)
		}
		buf.WriteString("\n")
	}

	// Write host blocks
	for _, host := range s.cfg.Hosts {
		fmt.Fprintf(&buf, "Host %s\n", host.Host)

		if host.HostName != "" {
			fmt.Fprintf(&buf, "    HostName %s\n", host.HostName)
		}
		if host.User != "" {
			fmt.Fprintf(&buf, "    User %s\n", host.User)
		}
		if host.Port > 0 {
			fmt.Fprintf(&buf, "    Port %d\n", host.Port)
		}
		if host.IdentityFile != "" {
			fmt.Fprintf(&buf, "    IdentityFile %s\n", host.IdentityFile)
		}
		if host.IdentitiesOnly {
			buf.WriteString("    IdentitiesOnly yes\n")
		}
		if host.ForwardAgent {
			buf.WriteString("    ForwardAgent yes\n")
		}
		if host.ProxyCommand != "" {
			fmt.Fprintf(&buf, "    ProxyCommand %s\n", host.ProxyCommand)
		}
		if host.ProxyJump != "" {
			fmt.Fprintf(&buf, "    ProxyJump %s\n", host.ProxyJump)
		}
		if host.LocalForward != "" {
			fmt.Fprintf(&buf, "    LocalForward %s\n", host.LocalForward)
		}
		if host.RemoteForward != "" {
			fmt.Fprintf(&buf, "    RemoteForward %s\n", host.RemoteForward)
		}
		if host.RequestTTY != "" {
			fmt.Fprintf(&buf, "    RequestTTY %s\n", host.RequestTTY)
		}
		if host.AddKeysToAgent {
			buf.WriteString("    AddKeysToAgent yes\n")
		}
		if host.UseKeychain {
			buf.WriteString("    UseKeychain yes\n")
		}
		if host.IgnoreUnknown != "" {
			fmt.Fprintf(&buf, "    IgnoreUnknown %s\n", host.IgnoreUnknown)
		}

		buf.WriteString("\n")
	}

	// Write match blocks
	for _, match := range s.cfg.Matches {
		fmt.Fprintf(&buf, "Match %s\n", match.Match)

		if match.HostName != "" {
			fmt.Fprintf(&buf, "    HostName %s\n", match.HostName)
		}
		if match.User != "" {
			fmt.Fprintf(&buf, "    User %s\n", match.User)
		}
		if match.IdentityFile != "" {
			fmt.Fprintf(&buf, "    IdentityFile %s\n", match.IdentityFile)
		}
		if match.ProxyCommand != "" {
			fmt.Fprintf(&buf, "    ProxyCommand %s\n", match.ProxyCommand)
		}
		if match.ProxyJump != "" {
			fmt.Fprintf(&buf, "    ProxyJump %s\n", match.ProxyJump)
		}

		buf.WriteString("\n")
	}

	return buf.Bytes()
}

// hasDefaults returns true if any default options are set.
func (s *ConfigStep) hasDefaults() bool {
	d := s.cfg.Defaults
	return d.AddKeysToAgent || d.IdentitiesOnly || d.ForwardAgent ||
		d.ServerAliveInterval > 0 || d.ServerAliveCountMax > 0
}

// validateConfig validates all SSH config values to prevent injection attacks.
func (s *ConfigStep) validateConfig() error {
	// Validate Include directive
	if s.cfg.Include != "" {
		if err := validation.ValidateSSHParameter(s.cfg.Include); err != nil {
			return fmt.Errorf("invalid Include directive: %w", err)
		}
	}

	// Validate host blocks
	for i, host := range s.cfg.Hosts {
		if host.Host != "" {
			if err := validation.ValidateHostname(host.Host); err != nil {
				return fmt.Errorf("host[%d] invalid Host pattern: %w", i, err)
			}
		}
		if host.HostName != "" {
			if err := validation.ValidateHostname(host.HostName); err != nil {
				return fmt.Errorf("host[%d] invalid HostName: %w", i, err)
			}
		}
		if host.User != "" {
			if err := validation.ValidateSSHParameter(host.User); err != nil {
				return fmt.Errorf("host[%d] invalid User: %w", i, err)
			}
		}
		if host.IdentityFile != "" {
			if err := validation.ValidatePath(host.IdentityFile); err != nil {
				return fmt.Errorf("host[%d] invalid IdentityFile: %w", i, err)
			}
		}
		// ProxyCommand is particularly security-sensitive
		if host.ProxyCommand != "" {
			if err := validation.ValidateSSHProxyCommand(host.ProxyCommand); err != nil {
				return fmt.Errorf("host[%d] invalid ProxyCommand: %w", i, err)
			}
		}
		if host.ProxyJump != "" {
			if err := validation.ValidateSSHParameter(host.ProxyJump); err != nil {
				return fmt.Errorf("host[%d] invalid ProxyJump: %w", i, err)
			}
		}
		if host.LocalForward != "" {
			if err := validation.ValidateSSHParameter(host.LocalForward); err != nil {
				return fmt.Errorf("host[%d] invalid LocalForward: %w", i, err)
			}
		}
		if host.RemoteForward != "" {
			if err := validation.ValidateSSHParameter(host.RemoteForward); err != nil {
				return fmt.Errorf("host[%d] invalid RemoteForward: %w", i, err)
			}
		}
		if host.RequestTTY != "" {
			if err := validation.ValidateSSHParameter(host.RequestTTY); err != nil {
				return fmt.Errorf("host[%d] invalid RequestTTY: %w", i, err)
			}
		}
		if host.IgnoreUnknown != "" {
			if err := validation.ValidateSSHParameter(host.IgnoreUnknown); err != nil {
				return fmt.Errorf("host[%d] invalid IgnoreUnknown: %w", i, err)
			}
		}
	}

	// Validate match blocks
	for i, match := range s.cfg.Matches {
		if match.Match != "" {
			if err := validation.ValidateSSHParameter(match.Match); err != nil {
				return fmt.Errorf("match[%d] invalid Match pattern: %w", i, err)
			}
		}
		if match.HostName != "" {
			if err := validation.ValidateHostname(match.HostName); err != nil {
				return fmt.Errorf("match[%d] invalid HostName: %w", i, err)
			}
		}
		if match.User != "" {
			if err := validation.ValidateSSHParameter(match.User); err != nil {
				return fmt.Errorf("match[%d] invalid User: %w", i, err)
			}
		}
		if match.IdentityFile != "" {
			if err := validation.ValidatePath(match.IdentityFile); err != nil {
				return fmt.Errorf("match[%d] invalid IdentityFile: %w", i, err)
			}
		}
		// ProxyCommand is particularly security-sensitive
		if match.ProxyCommand != "" {
			if err := validation.ValidateSSHProxyCommand(match.ProxyCommand); err != nil {
				return fmt.Errorf("match[%d] invalid ProxyCommand: %w", i, err)
			}
		}
		if match.ProxyJump != "" {
			if err := validation.ValidateSSHParameter(match.ProxyJump); err != nil {
				return fmt.Errorf("match[%d] invalid ProxyJump: %w", i, err)
			}
		}
	}

	return nil
}
