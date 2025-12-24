package compiler

import (
	"fmt"
	"strings"
)

// Error codes for compiler operations.
const (
	ErrCodeProviderFailed    = "PROVIDER_FAILED"
	ErrCodeStepDuplicate     = "STEP_DUPLICATE"
	ErrCodeStepNotFound      = "STEP_NOT_FOUND"
	ErrCodeDependencyMissing = "DEPENDENCY_MISSING"
	ErrCodeCyclicDependency  = "CYCLIC_DEPENDENCY"
	ErrCodeCompileFailed     = "COMPILE_FAILED"
	ErrCodePlanFailed        = "PLAN_FAILED"
	ErrCodeApplyFailed       = "APPLY_FAILED"
	ErrCodeCheckFailed       = "CHECK_FAILED"
)

// StepError represents a user-friendly compiler error with actionable suggestions.
type StepError struct {
	Code       string // Error code for categorization
	Message    string // User-friendly error message
	Provider   string // Provider that caused the error
	StepID     string // Step ID if applicable
	Suggestion string // Actionable suggestion to fix the error
	Underlying error  // Wrapped error for error chain
}

// Error returns the formatted error message.
func (e *StepError) Error() string {
	var parts []string

	if e.Provider != "" {
		parts = append(parts, fmt.Sprintf("provider %q", e.Provider))
	}
	if e.StepID != "" {
		parts = append(parts, fmt.Sprintf("step %q", e.StepID))
	}

	if len(parts) > 0 {
		return fmt.Sprintf("%s: %s", strings.Join(parts, ", "), e.Message)
	}
	return e.Message
}

// Unwrap returns the underlying error for error chain support.
func (e *StepError) Unwrap() error {
	return e.Underlying
}

// Format returns a fully formatted error with all details.
func (e *StepError) Format() string {
	var b strings.Builder

	// Error code and message
	b.WriteString(fmt.Sprintf("[%s] %s", e.Code, e.Message))

	// Provider context
	if e.Provider != "" {
		b.WriteString(fmt.Sprintf("\n  Provider: %s", e.Provider))
	}

	// Step context
	if e.StepID != "" {
		b.WriteString(fmt.Sprintf("\n  Step: %s", e.StepID))
	}

	// Suggestion
	if e.Suggestion != "" {
		b.WriteString(fmt.Sprintf("\n  Suggestion: %s", e.Suggestion))
	}

	// Underlying error
	if e.Underlying != nil {
		b.WriteString(fmt.Sprintf("\n  Cause: %s", e.Underlying.Error()))
	}

	return b.String()
}

// NewStepError creates a new StepError with the given code and message.
func NewStepError(code, message string) *StepError {
	return &StepError{
		Code:    code,
		Message: message,
	}
}

// WithProvider returns a new StepError with provider set.
func (e *StepError) WithProvider(provider string) *StepError {
	return &StepError{
		Code:       e.Code,
		Message:    e.Message,
		Provider:   provider,
		StepID:     e.StepID,
		Suggestion: e.Suggestion,
		Underlying: e.Underlying,
	}
}

// WithStepID returns a new StepError with step ID set.
func (e *StepError) WithStepID(stepID string) *StepError {
	return &StepError{
		Code:       e.Code,
		Message:    e.Message,
		Provider:   e.Provider,
		StepID:     stepID,
		Suggestion: e.Suggestion,
		Underlying: e.Underlying,
	}
}

// WithSuggestion returns a new StepError with suggestion set.
func (e *StepError) WithSuggestion(suggestion string) *StepError {
	return &StepError{
		Code:       e.Code,
		Message:    e.Message,
		Provider:   e.Provider,
		StepID:     e.StepID,
		Suggestion: suggestion,
		Underlying: e.Underlying,
	}
}

// WithUnderlying returns a new StepError wrapping another error.
func (e *StepError) WithUnderlying(err error) *StepError {
	return &StepError{
		Code:       e.Code,
		Message:    e.Message,
		Provider:   e.Provider,
		StepID:     e.StepID,
		Suggestion: e.Suggestion,
		Underlying: err,
	}
}

// Common compiler error constructors.

// NewProviderFailedError creates an error for provider compilation failure.
func NewProviderFailedError(provider string, err error) *StepError {
	return &StepError{
		Code:       ErrCodeProviderFailed,
		Message:    "provider failed to compile steps",
		Provider:   provider,
		Suggestion: fmt.Sprintf("Check your %s configuration for syntax errors or missing required fields.", provider),
		Underlying: err,
	}
}

// NewStepDuplicateError creates an error for duplicate step ID.
func NewStepDuplicateError(stepID string) *StepError {
	return &StepError{
		Code:       ErrCodeStepDuplicate,
		Message:    "step with this ID already exists in the graph",
		StepID:     stepID,
		Suggestion: "Each step must have a unique ID. Check for duplicate tool installations or conflicting configurations.",
	}
}

// NewDependencyMissingError creates an error for missing step dependency.
func NewDependencyMissingError(stepID, dependsOn string) *StepError {
	return &StepError{
		Code:       ErrCodeDependencyMissing,
		Message:    fmt.Sprintf("step depends on '%s' which does not exist", dependsOn),
		StepID:     stepID,
		Suggestion: "Ensure all dependencies are defined. This may indicate a missing provider or configuration.",
	}
}

// NewCyclicDependencyError creates an error for cyclic dependencies.
func NewCyclicDependencyError(cycle []string) *StepError {
	return &StepError{
		Code:       ErrCodeCyclicDependency,
		Message:    fmt.Sprintf("cyclic dependency detected: %s", strings.Join(cycle, " → ")),
		Suggestion: "Review your step dependencies to break the circular chain.",
	}
}

// NewApplyFailedError creates an error for step apply failure.
func NewApplyFailedError(stepID string, err error) *StepError {
	return &StepError{
		Code:       ErrCodeApplyFailed,
		Message:    "step failed to apply",
		StepID:     stepID,
		Suggestion: "Check the error details and run 'preflight doctor' for more information.",
		Underlying: err,
	}
}

// NewCheckFailedError creates an error for step check failure.
func NewCheckFailedError(stepID string, err error) *StepError {
	return &StepError{
		Code:       ErrCodeCheckFailed,
		Message:    "step status check failed",
		StepID:     stepID,
		Suggestion: "The step could not determine its current status. This may be a transient error.",
		Underlying: err,
	}
}
