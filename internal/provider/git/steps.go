package git

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// ConfigStep generates the .gitconfig file.
type ConfigStep struct {
	cfg *Config
	id  compiler.StepID
	fs  ports.FileSystem
}

// NewConfigStep creates a new ConfigStep.
func NewConfigStep(cfg *Config, fs ports.FileSystem) *ConfigStep {
	id := compiler.MustNewStepID("git:config")
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
		"gitconfig",
		s.cfg.ConfigPath(),
		"",
		"generated",
	), nil
}

// Apply generates and writes the .gitconfig file.
func (s *ConfigStep) Apply(_ compiler.RunContext) error {
	path := ports.ExpandPath(s.cfg.ConfigPath())
	content := s.generateConfig()

	if err := s.fs.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write gitconfig: %w", err)
	}

	return nil
}

// Explain provides a human-readable explanation.
func (s *ConfigStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Generate Git Config",
		fmt.Sprintf("Generates %s with user, core, and alias settings.", s.cfg.ConfigPath()),
		nil,
	)
}

// generateConfig generates the .gitconfig content.
func (s *ConfigStep) generateConfig() []byte {
	var buf bytes.Buffer

	// Write user section
	if s.cfg.User.Name != "" || s.cfg.User.Email != "" || s.cfg.User.SigningKey != "" {
		buf.WriteString("[user]\n")
		if s.cfg.User.Name != "" {
			fmt.Fprintf(&buf, "\tname = %s\n", s.cfg.User.Name)
		}
		if s.cfg.User.Email != "" {
			fmt.Fprintf(&buf, "\temail = %s\n", s.cfg.User.Email)
		}
		if s.cfg.User.SigningKey != "" {
			fmt.Fprintf(&buf, "\tsigningkey = %s\n", s.cfg.User.SigningKey)
		}
	}

	// Write core section
	if s.cfg.Core.Editor != "" || s.cfg.Core.AutoCRLF != "" || s.cfg.Core.ExcludesFile != "" {
		buf.WriteString("[core]\n")
		if s.cfg.Core.Editor != "" {
			fmt.Fprintf(&buf, "\teditor = %s\n", s.cfg.Core.Editor)
		}
		if s.cfg.Core.AutoCRLF != "" {
			fmt.Fprintf(&buf, "\tautocrlf = %s\n", s.cfg.Core.AutoCRLF)
		}
		if s.cfg.Core.ExcludesFile != "" {
			fmt.Fprintf(&buf, "\texcludesfile = %s\n", s.cfg.Core.ExcludesFile)
		}
	}

	// Write commit section
	if s.cfg.Commit.GPGSign {
		buf.WriteString("[commit]\n")
		buf.WriteString("\tgpgsign = true\n")
	}

	// Write gpg section
	if s.cfg.GPG.Format != "" || s.cfg.GPG.Program != "" {
		buf.WriteString("[gpg]\n")
		if s.cfg.GPG.Format != "" {
			fmt.Fprintf(&buf, "\tformat = %s\n", s.cfg.GPG.Format)
		}
		if s.cfg.GPG.Program != "" {
			fmt.Fprintf(&buf, "\tprogram = %s\n", s.cfg.GPG.Program)
		}
	}

	// Write aliases section (sorted for deterministic output)
	if len(s.cfg.Aliases) > 0 {
		buf.WriteString("[alias]\n")
		keys := make([]string, 0, len(s.cfg.Aliases))
		for k := range s.cfg.Aliases {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&buf, "\t%s = %s\n", k, s.cfg.Aliases[k])
		}
	}

	// Write includeIf sections
	for _, inc := range s.cfg.Includes {
		if inc.IfConfig != "" {
			fmt.Fprintf(&buf, "[includeIf \"%s\"]\n", inc.IfConfig)
		} else {
			buf.WriteString("[include]\n")
		}
		fmt.Fprintf(&buf, "\tpath = %s\n", inc.Path)
	}

	return buf.Bytes()
}
