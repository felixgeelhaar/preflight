package fonts

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// NerdFontStep represents a Nerd Font installation step.
type NerdFontStep struct {
	fontName string
	caskName string
	id       compiler.StepID
	runner   ports.CommandRunner
}

// NewNerdFontStep creates a new NerdFontStep.
func NewNerdFontStep(fontName string, runner ports.CommandRunner) *NerdFontStep {
	id := compiler.MustNewStepID("fonts:nerd:" + fontName)
	caskName := NerdFontCaskName(fontName)

	return &NerdFontStep{
		fontName: fontName,
		caskName: caskName,
		id:       id,
		runner:   runner,
	}
}

// ID returns the step identifier.
func (s *NerdFontStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
// All Nerd Font steps depend on the cask-fonts tap.
func (s *NerdFontStep) DependsOn() []compiler.StepID {
	return []compiler.StepID{
		compiler.MustNewStepID("brew:tap:" + CaskFontsTap),
	}
}

// CaskName returns the Homebrew cask name for this font.
func (s *NerdFontStep) CaskName() string {
	return s.caskName
}

// Check determines if the font cask is already installed.
func (s *NerdFontStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "brew", "list", "--cask")
	if err != nil {
		return compiler.StatusUnknown, err
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("brew list --cask failed: %s", result.Stderr)
	}

	casks := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, cask := range casks {
		if cask == s.caskName {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *NerdFontStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(
		compiler.DiffTypeAdd,
		"nerd_font",
		s.fontName,
		"",
		s.caskName,
	), nil
}

// Apply installs the font cask.
func (s *NerdFontStep) Apply(ctx compiler.RunContext) error {
	// Validate cask name before execution to prevent command injection
	if err := validation.ValidateCaskName(s.caskName); err != nil {
		return fmt.Errorf("invalid cask name: %w", err)
	}

	result, err := s.runner.Run(ctx.Context(), "brew", "install", "--cask", s.caskName)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("brew install --cask %s failed: %s", s.caskName, result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *NerdFontStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Nerd Font",
		fmt.Sprintf("Installs %s Nerd Font via Homebrew cask. Nerd Fonts are patched fonts that include programming ligatures and icons for terminal/editor use.", s.fontName),
		[]string{
			"https://www.nerdfonts.com/",
			"https://github.com/ryanoasis/nerd-fonts",
			fmt.Sprintf("https://formulae.brew.sh/cask/%s", s.caskName),
		},
	).WithTradeoffs([]string{
		"+ Consistent font rendering across terminal and editors",
		"+ Includes icons for file types, git status, etc.",
		"+ Required for powerline-style prompts (Starship, oh-my-posh)",
		"- Large download size (~50-100MB per font family)",
	})
}

// Ensure NerdFontStep implements compiler.Step.
var _ compiler.Step = (*NerdFontStep)(nil)
