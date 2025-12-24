package compiler

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *StepError
		expected string
	}{
		{
			name: "message only",
			err: &StepError{
				Code:    ErrCodeCompileFailed,
				Message: "compilation failed",
			},
			expected: "compilation failed",
		},
		{
			name: "with provider",
			err: &StepError{
				Code:     ErrCodeProviderFailed,
				Message:  "provider error",
				Provider: "brew",
			},
			expected: `provider "brew": provider error`,
		},
		{
			name: "with step ID",
			err: &StepError{
				Code:    ErrCodeApplyFailed,
				Message: "apply failed",
				StepID:  "brew:install:git",
			},
			expected: `step "brew:install:git": apply failed`,
		},
		{
			name: "with provider and step ID",
			err: &StepError{
				Code:     ErrCodeApplyFailed,
				Message:  "apply failed",
				Provider: "brew",
				StepID:   "brew:install:git",
			},
			expected: `provider "brew", step "brew:install:git": apply failed`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestStepError_Format(t *testing.T) {
	t.Parallel()

	underlying := errors.New("command not found: brew")
	err := &StepError{
		Code:       ErrCodeProviderFailed,
		Message:    "provider failed to compile steps",
		Provider:   "brew",
		StepID:     "brew:install:git",
		Suggestion: "Check if Homebrew is installed",
		Underlying: underlying,
	}

	formatted := err.Format()

	assert.Contains(t, formatted, "[PROVIDER_FAILED]")
	assert.Contains(t, formatted, "provider failed to compile steps")
	assert.Contains(t, formatted, "Provider: brew")
	assert.Contains(t, formatted, "Step: brew:install:git")
	assert.Contains(t, formatted, "Suggestion: Check if Homebrew is installed")
	assert.Contains(t, formatted, "Cause: command not found: brew")
}

func TestStepError_Unwrap(t *testing.T) {
	t.Parallel()

	underlying := errors.New("root cause")
	err := &StepError{
		Code:       ErrCodeProviderFailed,
		Message:    "provider failed",
		Underlying: underlying,
	}

	assert.Equal(t, underlying, err.Unwrap())
	assert.ErrorIs(t, err, underlying)
}

func TestNewStepError(t *testing.T) {
	t.Parallel()

	err := NewStepError(ErrCodeCompileFailed, "compilation error")

	assert.Equal(t, ErrCodeCompileFailed, err.Code)
	assert.Equal(t, "compilation error", err.Message)
	assert.Empty(t, err.Provider)
	assert.Empty(t, err.StepID)
}

func TestStepError_WithProvider(t *testing.T) {
	t.Parallel()

	original := &StepError{
		Code:    ErrCodeProviderFailed,
		Message: "error",
		StepID:  "step-1",
	}

	withProvider := original.WithProvider("brew")

	assert.Equal(t, "brew", withProvider.Provider)
	assert.Equal(t, "step-1", withProvider.StepID)
	assert.Empty(t, original.Provider) // Original unchanged
}

func TestStepError_WithStepID(t *testing.T) {
	t.Parallel()

	original := &StepError{
		Code:     ErrCodeApplyFailed,
		Message:  "error",
		Provider: "brew",
	}

	withStepID := original.WithStepID("brew:install:git")

	assert.Equal(t, "brew:install:git", withStepID.StepID)
	assert.Equal(t, "brew", withStepID.Provider)
	assert.Empty(t, original.StepID) // Original unchanged
}

func TestStepError_WithSuggestion(t *testing.T) {
	t.Parallel()

	original := &StepError{
		Code:    ErrCodeCheckFailed,
		Message: "check failed",
	}

	withSuggestion := original.WithSuggestion("Try again later")

	assert.Equal(t, "Try again later", withSuggestion.Suggestion)
	assert.Empty(t, original.Suggestion) // Original unchanged
}

func TestStepError_WithUnderlying(t *testing.T) {
	t.Parallel()

	underlying := errors.New("root cause")
	original := &StepError{
		Code:    ErrCodeApplyFailed,
		Message: "apply failed",
	}

	withUnderlying := original.WithUnderlying(underlying)

	assert.Equal(t, underlying, withUnderlying.Underlying)
	assert.NoError(t, original.Underlying) // Original unchanged
}

func TestNewProviderFailedError(t *testing.T) {
	t.Parallel()

	underlying := errors.New("config invalid")
	err := NewProviderFailedError("brew", underlying)

	assert.Equal(t, ErrCodeProviderFailed, err.Code)
	assert.Equal(t, "brew", err.Provider)
	assert.Contains(t, err.Suggestion, "brew")
	assert.ErrorIs(t, err, underlying)
}

func TestNewStepDuplicateError(t *testing.T) {
	t.Parallel()

	err := NewStepDuplicateError("brew:install:git")

	assert.Equal(t, ErrCodeStepDuplicate, err.Code)
	assert.Equal(t, "brew:install:git", err.StepID)
	assert.Contains(t, err.Message, "already exists")
	assert.Contains(t, err.Suggestion, "unique ID")
}

func TestNewDependencyMissingError(t *testing.T) {
	t.Parallel()

	err := NewDependencyMissingError("app:install:nvim", "brew:tap:neovim")

	assert.Equal(t, ErrCodeDependencyMissing, err.Code)
	assert.Equal(t, "app:install:nvim", err.StepID)
	assert.Contains(t, err.Message, "brew:tap:neovim")
	assert.Contains(t, err.Suggestion, "dependencies")
}

func TestNewCyclicDependencyError(t *testing.T) {
	t.Parallel()

	cycle := []string{"step-a", "step-b", "step-c", "step-a"}
	err := NewCyclicDependencyError(cycle)

	assert.Equal(t, ErrCodeCyclicDependency, err.Code)
	assert.Contains(t, err.Message, "step-a → step-b → step-c → step-a")
	assert.Contains(t, err.Suggestion, "circular")
}

func TestNewApplyFailedError(t *testing.T) {
	t.Parallel()

	underlying := errors.New("permission denied")
	err := NewApplyFailedError("files:link:gitconfig", underlying)

	assert.Equal(t, ErrCodeApplyFailed, err.Code)
	assert.Equal(t, "files:link:gitconfig", err.StepID)
	assert.Contains(t, err.Suggestion, "doctor")
	assert.ErrorIs(t, err, underlying)
}

func TestNewCheckFailedError(t *testing.T) {
	t.Parallel()

	underlying := errors.New("timeout")
	err := NewCheckFailedError("brew:install:git", underlying)

	assert.Equal(t, ErrCodeCheckFailed, err.Code)
	assert.Equal(t, "brew:install:git", err.StepID)
	assert.Contains(t, err.Suggestion, "transient")
	assert.ErrorIs(t, err, underlying)
}
