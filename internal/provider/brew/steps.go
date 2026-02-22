package brew

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/commandutil"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

const brewInstallStepID = "brew:install"

// InstallStep ensures Homebrew is installed.
type InstallStep struct {
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewInstallStep creates a new InstallStep.
func NewInstallStep(runner ports.CommandRunner) *InstallStep {
	return &InstallStep{
		id:     compiler.MustNewStepID(brewInstallStepID),
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *InstallStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *InstallStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if Homebrew is installed.
func (s *InstallStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	if _, err := exec.LookPath("brew"); err == nil {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *InstallStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeAdd, "brew", "install", "", "latest"), nil
}

// Apply installs Homebrew using the official install script.
func (s *InstallStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "/bin/bash", "-c", "curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh | /bin/bash")
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("homebrew install failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *InstallStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Install Homebrew",
		"Installs Homebrew to enable package management on this system.",
		[]string{"https://brew.sh/"},
	)
}

// TapStep represents a Homebrew tap installation step.
type TapStep struct {
	tap    string
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewTapStep creates a new TapStep.
func NewTapStep(tap string, runner ports.CommandRunner) *TapStep {
	id := compiler.MustNewStepID("brew:tap:" + tap)
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
	return []compiler.StepID{compiler.MustNewStepID(brewInstallStepID)}
}

// Check determines if the tap is already installed.
func (s *TapStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "brew", "tap")
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return compiler.StatusNeedsApply, nil
		}
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
	// Validate tap name before execution to prevent command injection
	if err := validation.ValidateTapName(s.tap); err != nil {
		return fmt.Errorf("invalid tap name: %w", err)
	}

	result, err := s.runner.Run(ctx.Context(), "brew", "tap", s.tap)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return fmt.Errorf("brew not found in PATH; install Homebrew first")
		}
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
		[]string{
			fmt.Sprintf("https://github.com/%s", s.tap),
			"https://docs.brew.sh/Taps",
		},
	).WithTradeoffs([]string{
		"+ Access to additional packages not in core Homebrew",
		"- Third-party taps may have less stability than core formulae",
		"- Requires trust in the tap maintainer",
	})
}

// FormulaStep represents a Homebrew formula installation step.
type FormulaStep struct {
	formula Formula
	id      compiler.StepID
	runner  ports.CommandRunner
}

// NewFormulaStep creates a new FormulaStep.
func NewFormulaStep(formula Formula, runner ports.CommandRunner) *FormulaStep {
	id := compiler.MustNewStepID("brew:formula:" + formula.FullName())
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
	deps := []compiler.StepID{compiler.MustNewStepID(brewInstallStepID)}
	if s.formula.Tap != "" {
		tapID := compiler.MustNewStepID("brew:tap:" + s.formula.Tap)
		deps = append(deps, tapID)
	}
	return deps
}

// Check determines if the formula is already installed.
func (s *FormulaStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "brew", "list", "--formula")
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return compiler.StatusNeedsApply, nil
		}
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
	// Validate formula name before execution to prevent command injection
	if err := validation.ValidatePackageName(s.formula.Name); err != nil {
		return fmt.Errorf("invalid formula name: %w", err)
	}

	// Validate args - each arg should be a valid brew argument (e.g., --HEAD)
	for _, arg := range s.formula.Args {
		if err := validation.ValidateBrewArg(arg); err != nil {
			return fmt.Errorf("invalid formula argument %q: %w", arg, err)
		}
	}

	args := make([]string, 0, 2+len(s.formula.Args))
	args = append(args, "install", s.formula.Name)
	args = append(args, s.formula.Args...)

	result, err := s.runner.Run(ctx.Context(), "brew", args...)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return fmt.Errorf("brew not found in PATH; install Homebrew first")
		}
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
		[]string{
			fmt.Sprintf("https://formulae.brew.sh/formula/%s", s.formula.Name),
			"https://docs.brew.sh/Formula-Cookbook",
		},
	).WithTradeoffs([]string{
		"+ Managed updates via 'brew upgrade'",
		"+ Consistent installation across macOS versions",
		"- May install additional dependencies",
	})
}

// InstalledVersion returns the installed formula version if available.
func (s *FormulaStep) InstalledVersion(ctx compiler.RunContext) (string, bool, error) {
	result, err := s.runner.Run(ctx.Context(), "brew", "list", "--versions", s.formula.Name)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if !result.Success() {
		return "", false, nil
	}
	fields := strings.Fields(result.Stdout)
	if len(fields) < 2 {
		return "", false, nil
	}
	return fields[1], true, nil
}

// CaskStep represents a Homebrew cask installation step.
type CaskStep struct {
	cask   Cask
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewCaskStep creates a new CaskStep.
func NewCaskStep(cask Cask, runner ports.CommandRunner) *CaskStep {
	id := compiler.MustNewStepID("brew:cask:" + cask.FullName())
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
	deps := []compiler.StepID{compiler.MustNewStepID(brewInstallStepID)}
	if s.cask.Tap != "" {
		tapID := compiler.MustNewStepID("brew:tap:" + s.cask.Tap)
		deps = append(deps, tapID)
	}
	return deps
}

// Check determines if the cask is already installed.
func (s *CaskStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "brew", "list", "--cask")
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return compiler.StatusNeedsApply, nil
		}
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
	// Validate cask name before execution to prevent command injection
	if err := validation.ValidatePackageName(s.cask.Name); err != nil {
		return fmt.Errorf("invalid cask name: %w", err)
	}

	result, err := s.runner.Run(ctx.Context(), "brew", "install", "--cask", s.cask.Name)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return fmt.Errorf("brew not found in PATH; install Homebrew first")
		}
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
		[]string{
			fmt.Sprintf("https://formulae.brew.sh/cask/%s", s.cask.Name),
			"https://docs.brew.sh/Cask-Cookbook",
		},
	).WithTradeoffs([]string{
		"+ Managed by Homebrew for easy updates and removal",
		"+ Reproducible installation across machines",
		"- May require admin password for /Applications",
	})
}

// InstalledVersion returns the installed cask version if available.
func (s *CaskStep) InstalledVersion(ctx compiler.RunContext) (string, bool, error) {
	result, err := s.runner.Run(ctx.Context(), "brew", "list", "--cask", "--versions", s.cask.Name)
	if err != nil {
		if commandutil.IsCommandNotFound(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if !result.Success() {
		return "", false, nil
	}
	fields := strings.Fields(result.Stdout)
	if len(fields) < 2 {
		return "", false, nil
	}
	return fields[1], true, nil
}
