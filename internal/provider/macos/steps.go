package macos

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// DefaultsStep represents a macOS defaults write step.
type DefaultsStep struct {
	setting Default
	id      compiler.StepID
	runner  ports.CommandRunner
}

// NewDefaultsStep creates a new DefaultsStep.
func NewDefaultsStep(setting Default, runner ports.CommandRunner) *DefaultsStep {
	id := compiler.MustNewStepID(fmt.Sprintf("macos:defaults:%s:%s", setting.Domain, setting.Key))
	return &DefaultsStep{
		setting: setting,
		id:      id,
		runner:  runner,
	}
}

// ID returns the step identifier.
func (s *DefaultsStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *DefaultsStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the default is already set.
func (s *DefaultsStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "defaults", "read", s.setting.Domain, s.setting.Key)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	if !result.Success() {
		// Key doesn't exist, needs to be set
		return compiler.StatusNeedsApply, nil
	}

	currentValue := strings.TrimSpace(result.Stdout)
	expectedValue := fmt.Sprintf("%v", s.setting.Value)

	if currentValue == expectedValue {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *DefaultsStep) Plan(ctx compiler.RunContext) (compiler.Diff, error) {
	result, _ := s.runner.Run(ctx.Context(), "defaults", "read", s.setting.Domain, s.setting.Key)
	currentValue := strings.TrimSpace(result.Stdout)
	expectedValue := fmt.Sprintf("%v", s.setting.Value)

	return compiler.NewDiff(compiler.DiffTypeModify, "default", s.setting.Key, currentValue, expectedValue), nil
}

// Apply executes the defaults write.
func (s *DefaultsStep) Apply(ctx compiler.RunContext) error {
	args := []string{"write", s.setting.Domain, s.setting.Key}

	switch s.setting.Type {
	case "bool":
		if v, ok := s.setting.Value.(bool); ok {
			if v {
				args = append(args, "-bool", "true")
			} else {
				args = append(args, "-bool", "false")
			}
		}
	case "int":
		args = append(args, "-int", fmt.Sprintf("%v", s.setting.Value))
	case "float":
		args = append(args, "-float", fmt.Sprintf("%v", s.setting.Value))
	case "array":
		args = append(args, "-array")
		if arr, ok := s.setting.Value.([]interface{}); ok {
			for _, item := range arr {
				args = append(args, fmt.Sprintf("%v", item))
			}
		}
	default:
		args = append(args, "-string", fmt.Sprintf("%v", s.setting.Value))
	}

	result, err := s.runner.Run(ctx.Context(), "defaults", args...)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("defaults write failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *DefaultsStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Set macOS Default",
		fmt.Sprintf("Sets %s.%s to %v", s.setting.Domain, s.setting.Key, s.setting.Value),
		[]string{
			"https://macos-defaults.com",
			"https://developer.apple.com/documentation/foundation/userdefaults",
		},
	).WithTradeoffs([]string{
		"+ Customizes macOS behavior to your preferences",
		"- Some settings require logout/restart to take effect",
		"- May be reset by macOS updates",
	})
}

// DockStep represents a Dock modification step.
type DockStep struct {
	app    string
	add    bool
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewDockStep creates a new DockStep.
func NewDockStep(app string, add bool, runner ports.CommandRunner) *DockStep {
	action := "add"
	if !add {
		action = "remove"
	}
	id := compiler.MustNewStepID(fmt.Sprintf("macos:dock:%s:%s", action, app))
	return &DockStep{
		app:    app,
		add:    add,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *DockStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *DockStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the dock item exists.
func (s *DockStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	// Use dockutil if available, otherwise check via defaults
	result, err := s.runner.Run(ctx.Context(), "dockutil", "--find", s.app)
	if err != nil {
		// dockutil not available, assume needs apply
		return compiler.StatusNeedsApply, nil //nolint:nilerr // dockutil not installed means we need to apply
	}

	exists := result.Success()

	if s.add && exists {
		return compiler.StatusSatisfied, nil
	}
	if !s.add && !exists {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *DockStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	if s.add {
		return compiler.NewDiff(compiler.DiffTypeAdd, "dock", s.app, "", s.app), nil
	}
	return compiler.NewDiff(compiler.DiffTypeRemove, "dock", s.app, s.app, ""), nil
}

// Apply modifies the dock.
func (s *DockStep) Apply(ctx compiler.RunContext) error {
	var result ports.CommandResult
	var err error

	if s.add {
		result, err = s.runner.Run(ctx.Context(), "dockutil", "--add", s.app)
	} else {
		result, err = s.runner.Run(ctx.Context(), "dockutil", "--remove", s.app)
	}

	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("dockutil failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *DockStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	action := "Adds"
	if !s.add {
		action = "Removes"
	}
	return compiler.NewExplanation(
		"Modify Dock",
		fmt.Sprintf("%s %s to/from the Dock", action, s.app),
		[]string{
			"https://github.com/kcrawford/dockutil",
		},
	).WithTradeoffs([]string{
		"+ Keeps dock consistent across setups",
		"- Requires dockutil to be installed",
	})
}

// FinderStep represents a Finder preference step.
type FinderStep struct {
	key    string
	value  bool
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewFinderStep creates a new FinderStep.
func NewFinderStep(key string, value bool, runner ports.CommandRunner) *FinderStep {
	id := compiler.MustNewStepID(fmt.Sprintf("macos:finder:%s", key))
	return &FinderStep{
		key:    key,
		value:  value,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *FinderStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *FinderStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the Finder setting is already configured.
func (s *FinderStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "defaults", "read", "com.apple.finder", s.key)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	if !result.Success() {
		return compiler.StatusNeedsApply, nil
	}

	currentValue := strings.TrimSpace(result.Stdout)
	expectedValue := "0"
	if s.value {
		expectedValue = "1"
	}

	if currentValue == expectedValue {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *FinderStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeModify, "finder", s.key, "", strconv.FormatBool(s.value)), nil
}

// Apply executes the Finder preference change.
func (s *FinderStep) Apply(ctx compiler.RunContext) error {
	boolStr := "false"
	if s.value {
		boolStr = "true"
	}
	result, err := s.runner.Run(ctx.Context(), "defaults", "write", "com.apple.finder", s.key, "-bool", boolStr)
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("defaults write failed: %s", result.Stderr)
	}

	// Restart Finder to apply changes
	_, _ = s.runner.Run(ctx.Context(), "killall", "Finder")
	return nil
}

// Explain provides a human-readable explanation.
func (s *FinderStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Finder",
		fmt.Sprintf("Sets Finder %s to %v", s.key, s.value),
		[]string{
			"https://macos-defaults.com/#finder",
		},
	).WithTradeoffs([]string{
		"+ Customizes Finder to your preferences",
		"- Finder will restart to apply changes",
	})
}

// KeyboardStep represents a keyboard preference step.
type KeyboardStep struct {
	key    string
	value  int
	id     compiler.StepID
	runner ports.CommandRunner
}

// NewKeyboardStep creates a new KeyboardStep.
func NewKeyboardStep(key string, value int, runner ports.CommandRunner) *KeyboardStep {
	id := compiler.MustNewStepID(fmt.Sprintf("macos:keyboard:%s", key))
	return &KeyboardStep{
		key:    key,
		value:  value,
		id:     id,
		runner: runner,
	}
}

// ID returns the step identifier.
func (s *KeyboardStep) ID() compiler.StepID {
	return s.id
}

// DependsOn returns the step dependencies.
func (s *KeyboardStep) DependsOn() []compiler.StepID {
	return nil
}

// Check determines if the keyboard setting is already configured.
func (s *KeyboardStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	result, err := s.runner.Run(ctx.Context(), "defaults", "read", "NSGlobalDomain", s.key)
	if err != nil {
		return compiler.StatusUnknown, err
	}

	if !result.Success() {
		return compiler.StatusNeedsApply, nil
	}

	currentValue := strings.TrimSpace(result.Stdout)
	expectedValue := strconv.Itoa(s.value)

	if currentValue == expectedValue {
		return compiler.StatusSatisfied, nil
	}
	return compiler.StatusNeedsApply, nil
}

// Plan returns the diff for this step.
func (s *KeyboardStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.NewDiff(compiler.DiffTypeModify, "keyboard", s.key, "", strconv.Itoa(s.value)), nil
}

// Apply executes the keyboard preference change.
func (s *KeyboardStep) Apply(ctx compiler.RunContext) error {
	result, err := s.runner.Run(ctx.Context(), "defaults", "write", "NSGlobalDomain", s.key, "-int", strconv.Itoa(s.value))
	if err != nil {
		return err
	}
	if !result.Success() {
		return fmt.Errorf("defaults write failed: %s", result.Stderr)
	}
	return nil
}

// Explain provides a human-readable explanation.
func (s *KeyboardStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation(
		"Configure Keyboard",
		fmt.Sprintf("Sets keyboard %s to %d", s.key, s.value),
		[]string{
			"https://macos-defaults.com/#keyboard",
		},
	).WithTradeoffs([]string{
		"+ Customizes keyboard repeat rate",
		"- May require logout to take effect",
	})
}
