package brew

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// TapStep represents a Homebrew tap installation step.
type TapStep struct {
	tap    string
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewTapStep creates a new TapStep.
func NewTapStep(tap string, runner ports.CommandRunner) *TapStep {
	id, _ := compiler.NewStepID("brew:tap:" + tap)
	return &TapStep{
		tap:    tap,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *TapStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *TapStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the tap is already installed.
func (s *TapStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "brew", "tap")
	if err != nil {
		return compiler.StatusUnknown, err
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("brew tap failed: %s", result.Stderr)
	}

	taps := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, t := range taps {
		if t == s.tap {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *TapStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "tap", s.tap, "", s.tap), nil
}

// Apply executes the tap installation.
func (s *TapStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "brew", "tap", s.tap)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("brew tap %s failed: %s", s.tap, result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *TapStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Add Homebrew Tap",
		fmt.Sprintf("Adds the %s tap to Homebrew, enabling installation of formulae and casks from this repository.", s.tap),
		[]string{fmt.Sprintf("https://github.com/%s", s.tap)},
	)
}

// FormulaStep represents a Homebrew formula installation step.
type FormulaStep struct {
	formula Formula
	id      compiler.StepID
	runner  ports.CommandRunner
}

// NewFormulaStep creates a new FormulaStep.
func NewFormulaStep(formula Formula, runner ports.CommandRunner) *FormulaStep {
	id, _ := compiler.NewStepID("brew:formula:" + formula.FullName())
	return &FormulaStep{
		formula: formula,
		id:      id,
		runner:  runner,
	}
}

// ID returns the step identifier.
func (s *FormulaStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *FormulaStep) DependsOn() []compiler.StepID {
	if s.formula.Tap != "" {
		tapID, _ := compiler.NewStepID("brew:tap:" + s.formula.Tap)
		return []compiler.StepID{tapID}
	}
	return nil
}

// Check determines if the formula is already installed.
func (s *FormulaStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "brew", "list", "--formula")
	if err != nil {
		return compiler.StatusUnknown, err
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("brew list failed: %s", result.Stderr)
	}

	formulae := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, f := range formulae {
		if f == s.formula.Name {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *FormulaStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "formula", s.formula.FullName(), "", "latest"), nil
}

// Apply executes the formula installation.
func (s *FormulaStep) Apply(ctx compiler.RunContext) error {
	args := []string{"install", s.formula.Name}
	args = append(args, s.formula.Args...)

	result, err := s.runner.Run(ctx.Context(), "brew", args...)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("brew install %s failed: %s", s.formula.Name, result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *FormulaStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	desc := fmt.Sprintf("Installs the %s formula via Homebrew.", s.formula.Name)
	if len(s.formula.Args) > 0 {
		desc += fmt.Sprintf(" With args: %s", strings.Join(s.formula.Args, " "))
	}
	return compiler.NewExplanation(
		"Install Homebrew Formula",
		desc,
		[]string{fmt.Sprintf("https://formulae.brew.sh/formula/%s", s.formula.Name)},
	)
}

// CaskStep represents a Homebrew cask installation step.
type CaskStep struct {
	cask   Cask
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewCaskStep creates a new CaskStep.
func NewCaskStep(cask Cask, runner ports.CommandRunner) *CaskStep {
	id, _ := compiler.NewStepID("brew:cask:" + cask.FullName())
	return &CaskStep{
		cask:   cask,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *CaskStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *CaskStep) DependsOn() []compiler.StepID {
	if s.cask.Tap != "" {
		tapID, _ := compiler.NewStepID("brew:tap:" + s.cask.Tap)
		return []compiler.StepID{tapID}
	}
	return nil
}

// Check determines if the cask is already installed.
func (s *CaskStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "brew", "list", "--cask")
	if err != nil {
		return compiler.StatusUnknown, err
	}
	if !result.Success() {
		return compiler.StatusUnknown, fmt.Errorf("brew list --cask failed: %s", result.Stderr)
	}

	casks := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, c := range casks {
		if c == s.cask.Name {
			return compiler.StatusSatisfied, nil
		}
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *CaskStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "cask", s.cask.FullName(), "", "latest"), nil
}

// Apply executes the cask installation.
func (s *CaskStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "brew", "install", "--cask", s.cask.Name)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("brew install --cask %s failed: %s", s.cask.Name, result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *CaskStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Homebrew Cask",
		fmt.Sprintf("Installs the %s application via Homebrew Cask.", s.cask.Name),
		[]string{fmt.Sprintf("https://formulae.brew.sh/cask/%s", s.cask.Name)},
	)
}
