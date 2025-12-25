package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *UserError
		expected string
	}{
		{
			name: "simple message",
			err: &UserError{
				Code:    ErrCodeConfigNotFound,
				Message: "config file not found",
			},
			expected: "config file not found",
		},
		{
			name: "message with context",
			err: &UserError{
				Code:    ErrCodeConfigNotFound,
				Message: "config file not found",
				Context: "preflight.yaml",
			},
			expected: "config file not found (at preflight.yaml)",
		},
		{
			name: "message with all fields",
			err: &UserError{
				Code:       ErrCodeConfigNotFound,
				Message:    "config file not found",
				Context:    "preflight.yaml",
				Suggestion: "run init",
			},
			expected: "config file not found (at preflight.yaml)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestUserError_Format(t *testing.T) {
	t.Parallel()

	err := &UserError{
		Code:       ErrCodeConfigNotFound,
		Message:    "config file not found",
		Context:    "preflight.yaml",
		Suggestion: "Run 'preflight init' to create a new configuration",
	}

	formatted := err.Format()

	assert.Contains(t, formatted, "[CONFIG_NOT_FOUND]")
	assert.Contains(t, formatted, "config file not found")
	assert.Contains(t, formatted, "Location: preflight.yaml")
	assert.Contains(t, formatted, "Suggestion: Run 'preflight init'")
}

func TestUserError_Unwrap(t *testing.T) {
	t.Parallel()

	underlying := errors.New("underlying error")
	err := &UserError{
		Code:       ErrCodeConfigParse,
		Message:    "parse failed",
		Underlying: underlying,
	}

	assert.Equal(t, underlying, err.Unwrap())
	assert.ErrorIs(t, err, underlying)
}

func TestUserError_Is(t *testing.T) {
	t.Parallel()

	err1 := &UserError{Code: ErrCodeConfigNotFound, Message: "not found 1"}
	err2 := &UserError{Code: ErrCodeConfigNotFound, Message: "not found 2"}
	err3 := &UserError{Code: ErrCodeConfigParse, Message: "parse error"}

	assert.ErrorIs(t, err1, err2)
	assert.NotErrorIs(t, err1, err3)
}

func TestUserError_WithContext(t *testing.T) {
	t.Parallel()

	original := &UserError{
		Code:       ErrCodeConfigNotFound,
		Message:    "not found",
		Suggestion: "fix it",
	}

	withContext := original.WithContext("path/to/file.yaml")

	assert.Equal(t, original.Code, withContext.Code)
	assert.Equal(t, original.Message, withContext.Message)
	assert.Equal(t, original.Suggestion, withContext.Suggestion)
	assert.Equal(t, "path/to/file.yaml", withContext.Context)
	assert.Empty(t, original.Context) // Original unchanged
}

func TestUserError_WithSuggestion(t *testing.T) {
	t.Parallel()

	original := &UserError{
		Code:    ErrCodeConfigNotFound,
		Message: "not found",
		Context: "file.yaml",
	}

	withSuggestion := original.WithSuggestion("Run preflight init")

	assert.Equal(t, original.Code, withSuggestion.Code)
	assert.Equal(t, original.Message, withSuggestion.Message)
	assert.Equal(t, original.Context, withSuggestion.Context)
	assert.Equal(t, "Run preflight init", withSuggestion.Suggestion)
	assert.Empty(t, original.Suggestion) // Original unchanged
}

func TestUserError_WithUnderlying(t *testing.T) {
	t.Parallel()

	underlying := errors.New("root cause")
	original := &UserError{
		Code:    ErrCodeConfigParse,
		Message: "parse failed",
	}

	withUnderlying := original.WithUnderlying(underlying)

	assert.Equal(t, underlying, withUnderlying.Underlying)
	assert.NoError(t, original.Underlying) // Original unchanged
}

func TestNewUserError(t *testing.T) {
	t.Parallel()

	err := NewUserError(ErrCodeValidationFailed, "validation error")

	assert.Equal(t, ErrCodeValidationFailed, err.Code)
	assert.Equal(t, "validation error", err.Message)
	assert.Empty(t, err.Context)
	assert.Empty(t, err.Suggestion)
}

func TestErrorList_Add(t *testing.T) {
	t.Parallel()

	list := NewErrorList()
	assert.False(t, list.HasErrors())
	assert.Equal(t, 0, list.Len())

	list.Add(&UserError{Code: ErrCodeConfigNotFound, Message: "err1"})
	list.Add(&UserError{Code: ErrCodeConfigParse, Message: "err2"})
	list.Add(nil) // Should be ignored

	assert.True(t, list.HasErrors())
	assert.Equal(t, 2, list.Len())
}

func TestErrorList_AddValidation(t *testing.T) {
	t.Parallel()

	list := NewErrorList()
	list.AddValidation("files[0].path", "path is required", "Add a path to the file")

	require.Equal(t, 1, list.Len())
	err := list.Errors()[0]
	assert.Equal(t, ErrCodeValidationFailed, err.Code)
	assert.Contains(t, err.Message, "files[0].path")
	assert.Contains(t, err.Message, "path is required")
	assert.Equal(t, "Add a path to the file", err.Suggestion)
}

func TestErrorList_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*ErrorList)
		contains []string
	}{
		{
			name:     "empty list",
			setup:    func(_ *ErrorList) {},
			contains: nil,
		},
		{
			name: "single error",
			setup: func(l *ErrorList) {
				l.Add(&UserError{Code: "ERR1", Message: "first error"})
			},
			contains: []string{"first error"},
		},
		{
			name: "multiple errors",
			setup: func(l *ErrorList) {
				l.Add(&UserError{Code: "ERR1", Message: "first error"})
				l.Add(&UserError{Code: "ERR2", Message: "second error"})
			},
			contains: []string{"2 errors occurred", "first error", "second error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			list := NewErrorList()
			tt.setup(list)
			errStr := list.Error()

			for _, s := range tt.contains {
				assert.Contains(t, errStr, s)
			}
		})
	}
}

func TestErrorList_Format(t *testing.T) {
	t.Parallel()

	list := NewErrorList()
	list.Add(&UserError{
		Code:       ErrCodeConfigNotFound,
		Message:    "config not found",
		Context:    "preflight.yaml",
		Suggestion: "Run init",
	})
	list.Add(&UserError{
		Code:    ErrCodeValidationFailed,
		Message: "validation error",
	})

	formatted := list.Format()

	assert.Contains(t, formatted, "Found 2 error(s)")
	assert.Contains(t, formatted, "[CONFIG_NOT_FOUND]")
	assert.Contains(t, formatted, "[VALIDATION_FAILED]")
	assert.Contains(t, formatted, "Location: preflight.yaml")
	assert.Contains(t, formatted, "Suggestion: Run init")
}

func TestErrorList_Errors(t *testing.T) {
	t.Parallel()

	list := NewErrorList()
	err1 := &UserError{Code: "ERR1", Message: "error 1"}
	err2 := &UserError{Code: "ERR2", Message: "error 2"}
	list.Add(err1)
	list.Add(err2)

	errors := list.Errors()

	assert.Len(t, errors, 2)
	assert.Equal(t, err1, errors[0])
	assert.Equal(t, err2, errors[1])

	// Verify it's a copy
	errors[0] = nil
	assert.NotNil(t, list.Errors()[0])
}

func TestErrorList_AsError(t *testing.T) {
	t.Parallel()

	t.Run("empty list returns nil", func(t *testing.T) {
		t.Parallel()
		list := NewErrorList()
		assert.NoError(t, list.AsError())
	})

	t.Run("non-empty list returns error", func(t *testing.T) {
		t.Parallel()
		list := NewErrorList()
		list.Add(&UserError{Code: "ERR", Message: "error"})
		assert.Error(t, list.AsError())
	})
}

func TestNewConfigNotFoundError(t *testing.T) {
	t.Parallel()

	err := NewConfigNotFoundError("/path/to/config.yaml")

	assert.Equal(t, ErrCodeConfigNotFound, err.Code)
	assert.Contains(t, err.Message, "/path/to/config.yaml")
	assert.Equal(t, "/path/to/config.yaml", err.Context)
	assert.Contains(t, err.Suggestion, "preflight init")
}

func TestNewConfigParseError(t *testing.T) {
	t.Parallel()

	underlying := errors.New("yaml: unmarshal error")
	err := NewConfigParseError("/path/to/config.yaml", underlying)

	assert.Equal(t, ErrCodeConfigParse, err.Code)
	assert.Equal(t, "/path/to/config.yaml", err.Context)
	assert.Contains(t, err.Suggestion, "YAML syntax")
	assert.ErrorIs(t, err, underlying)
}

func TestNewLayerNotFoundError(t *testing.T) {
	t.Parallel()

	err := NewLayerNotFoundError("identity.work", "layers/identity.work.yaml")

	assert.Equal(t, ErrCodeLayerNotFound, err.Code)
	assert.Contains(t, err.Message, "identity.work")
	assert.Equal(t, "layers/identity.work.yaml", err.Context)
	assert.Contains(t, err.Suggestion, "layers/identity.work.yaml")
}

func TestNewTargetNotFoundError(t *testing.T) {
	t.Parallel()

	t.Run("with available targets", func(t *testing.T) {
		t.Parallel()
		err := NewTargetNotFoundError("development", []string{"default", "work", "personal"})

		assert.Equal(t, ErrCodeTargetNotFound, err.Code)
		assert.Contains(t, err.Message, "development")
		assert.Contains(t, err.Suggestion, "default")
		assert.Contains(t, err.Suggestion, "work")
	})

	t.Run("without available targets", func(t *testing.T) {
		t.Parallel()
		err := NewTargetNotFoundError("development", nil)

		assert.Equal(t, ErrCodeTargetNotFound, err.Code)
		assert.Contains(t, err.Suggestion, "preflight.yaml")
	})
}

func TestNewValidationFailedError(t *testing.T) {
	t.Parallel()

	err := NewValidationFailedError("files[0].path", "cannot be empty")

	assert.Equal(t, ErrCodeValidationFailed, err.Code)
	assert.Contains(t, err.Message, "files[0].path")
	assert.Contains(t, err.Message, "cannot be empty")
	assert.Equal(t, "files[0].path", err.Context)
}

func TestNewCircularReferenceError(t *testing.T) {
	t.Parallel()

	err := NewCircularReferenceError([]string{"base", "layer1", "layer2", "base"})

	assert.Equal(t, ErrCodeCircularReference, err.Code)
	assert.Contains(t, err.Message, "base → layer1 → layer2 → base")
	assert.Contains(t, err.Suggestion, "circular")
}

func TestNewTemplateInvalidError(t *testing.T) {
	t.Parallel()

	underlying := errors.New("template: parse error")
	err := NewTemplateInvalidError("templates/gitconfig.tmpl", underlying)

	assert.Equal(t, ErrCodeTemplateInvalid, err.Code)
	assert.Equal(t, "templates/gitconfig.tmpl", err.Context)
	assert.Contains(t, err.Suggestion, "template syntax")
	assert.ErrorIs(t, err, underlying)
}

func TestIsUserError(t *testing.T) {
	t.Parallel()

	t.Run("is UserError with matching code", func(t *testing.T) {
		t.Parallel()
		err := NewConfigNotFoundError("path")
		assert.True(t, IsUserError(err, ErrCodeConfigNotFound))
	})

	t.Run("is UserError with different code", func(t *testing.T) {
		t.Parallel()
		err := NewConfigNotFoundError("path")
		assert.False(t, IsUserError(err, ErrCodeConfigParse))
	})

	t.Run("not a UserError", func(t *testing.T) {
		t.Parallel()
		err := errors.New("regular error")
		assert.False(t, IsUserError(err, ErrCodeConfigNotFound))
	})

	t.Run("wrapped UserError", func(t *testing.T) {
		t.Parallel()
		ue := NewConfigNotFoundError("path")
		wrapped := errors.New("wrapped: " + ue.Error())
		// Not wrapped with %w, so this should be false
		assert.False(t, IsUserError(wrapped, ErrCodeConfigNotFound))
	})
}

func TestGetUserError(t *testing.T) {
	t.Parallel()

	t.Run("returns UserError", func(t *testing.T) {
		t.Parallel()
		err := NewConfigNotFoundError("path")
		ue := GetUserError(err)
		require.NotNil(t, ue)
		assert.Equal(t, ErrCodeConfigNotFound, ue.Code)
	})

	t.Run("returns nil for non-UserError", func(t *testing.T) {
		t.Parallel()
		err := errors.New("regular error")
		ue := GetUserError(err)
		assert.Nil(t, ue)
	})
}

func TestNewYAMLParseError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		err             error
		expectedMessage string
		expectedContext string
		suggestionHas   string
	}{
		{
			name:            "map into slice error",
			path:            "preflight.yaml",
			err:             errors.New("yaml: unmarshal errors:\n  line 5: cannot unmarshal !!map into []string"),
			expectedMessage: "invalid targets format",
			expectedContext: "preflight.yaml (line 5)",
			suggestionHas:   "Targets should be a list",
		},
		{
			name:            "seq into map error",
			path:            "layer.yaml",
			err:             errors.New("yaml: unmarshal errors:\n  line 3: cannot unmarshal !!seq into map"),
			expectedMessage: "expected an object but found a list",
			expectedContext: "layer.yaml (line 3)",
			suggestionHas:   "key: value",
		},
		{
			name:            "str into type error",
			path:            "config.yaml",
			err:             errors.New("yaml: cannot unmarshal !!str into int"),
			expectedMessage: "unexpected string value",
			expectedContext: "config.yaml",
			suggestionHas:   "indentation",
		},
		{
			name:            "missing key error",
			path:            "config.yaml",
			err:             errors.New("yaml: did not find expected key"),
			expectedMessage: "missing required field or incorrect indentation",
			expectedContext: "config.yaml",
			suggestionHas:   "2 spaces",
		},
		{
			name:            "mapping values error",
			path:            "config.yaml",
			err:             errors.New("yaml: mapping values are not allowed here"),
			expectedMessage: "invalid YAML structure",
			expectedContext: "config.yaml",
			suggestionHas:   "colons",
		},
		{
			name:            "invalid character error",
			path:            "config.yaml",
			err:             errors.New("yaml: found character that cannot start any token"),
			expectedMessage: "invalid character in YAML",
			expectedContext: "config.yaml",
			suggestionHas:   "special characters",
		},
		{
			name:            "generic yaml error",
			path:            "config.yaml",
			err:             errors.New("yaml: some other error"),
			expectedMessage: "invalid YAML syntax",
			expectedContext: "config.yaml",
			suggestionHas:   "YAML syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userErr := NewYAMLParseError(tt.path, tt.err)

			assert.Equal(t, ErrCodeConfigParse, userErr.Code)
			assert.Equal(t, tt.expectedMessage, userErr.Message)
			assert.Equal(t, tt.expectedContext, userErr.Context)
			assert.Contains(t, userErr.Suggestion, tt.suggestionHas)
			assert.ErrorIs(t, userErr, tt.err)
		})
	}
}
