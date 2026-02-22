package config

import (
	"errors"
	"fmt"
	"strings"
)

// Error codes for categorization.
const (
	ErrCodeConfigNotFound    = "CONFIG_NOT_FOUND"
	ErrCodeConfigInvalid     = "CONFIG_INVALID"
	ErrCodeConfigParse       = "CONFIG_PARSE"
	ErrCodeLayerNotFound     = "LAYER_NOT_FOUND"
	ErrCodeLayerInvalid      = "LAYER_INVALID"
	ErrCodeTargetNotFound    = "TARGET_NOT_FOUND"
	ErrCodeTargetInvalid     = "TARGET_INVALID"
	ErrCodeMergeConflict     = "MERGE_CONFLICT"
	ErrCodeValidationFailed  = "VALIDATION_FAILED"
	ErrCodeFileNotFound      = "FILE_NOT_FOUND"
	ErrCodeFilePermission    = "FILE_PERMISSION"
	ErrCodeTemplateInvalid   = "TEMPLATE_INVALID"
	ErrCodeProviderNotFound  = "PROVIDER_NOT_FOUND"
	ErrCodeCircularReference = "CIRCULAR_REFERENCE"
)

// UserError represents a user-friendly error with actionable suggestions.
type UserError struct {
	Code       string // Error code for categorization (e.g., "CONFIG_NOT_FOUND")
	Message    string // User-friendly error message
	Context    string // File path, line number, or other location context
	Suggestion string // Actionable suggestion to fix the error
	Underlying error  // Wrapped error for error chain
}

// Error returns the formatted error message.
func (e *UserError) Error() string {
	var b strings.Builder

	// Write main message
	b.WriteString(e.Message)

	// Add context if present
	if e.Context != "" {
		fmt.Fprintf(&b, " (at %s)", e.Context)
	}

	return b.String()
}

// Unwrap returns the underlying error for error chain support.
func (e *UserError) Unwrap() error {
	return e.Underlying
}

// Is supports errors.Is() for comparing error codes.
func (e *UserError) Is(target error) bool {
	if t, ok := target.(*UserError); ok {
		return e.Code == t.Code
	}
	return false
}

// Format returns a fully formatted error with all details.
func (e *UserError) Format() string {
	var b strings.Builder

	// Error code and message
	fmt.Fprintf(&b, "[%s] %s", e.Code, e.Message)

	// Context
	if e.Context != "" {
		fmt.Fprintf(&b, "\n  Location: %s", e.Context)
	}

	// Suggestion
	if e.Suggestion != "" {
		fmt.Fprintf(&b, "\n  Suggestion: %s", e.Suggestion)
	}

	return b.String()
}

// NewUserError creates a new UserError with the given code and message.
func NewUserError(code, message string) *UserError {
	return &UserError{
		Code:    code,
		Message: message,
	}
}

// WithContext returns a new UserError with context set.
func (e *UserError) WithContext(ctx string) *UserError {
	return &UserError{
		Code:       e.Code,
		Message:    e.Message,
		Context:    ctx,
		Suggestion: e.Suggestion,
		Underlying: e.Underlying,
	}
}

// WithSuggestion returns a new UserError with suggestion set.
func (e *UserError) WithSuggestion(suggestion string) *UserError {
	return &UserError{
		Code:       e.Code,
		Message:    e.Message,
		Context:    e.Context,
		Suggestion: suggestion,
		Underlying: e.Underlying,
	}
}

// WithUnderlying returns a new UserError wrapping another error.
func (e *UserError) WithUnderlying(err error) *UserError {
	return &UserError{
		Code:       e.Code,
		Message:    e.Message,
		Context:    e.Context,
		Suggestion: e.Suggestion,
		Underlying: err,
	}
}

// ErrorList accumulates multiple errors for comprehensive reporting.
type ErrorList struct {
	errors []*UserError
}

// NewErrorList creates an empty ErrorList.
func NewErrorList() *ErrorList {
	return &ErrorList{
		errors: make([]*UserError, 0),
	}
}

// Add adds an error to the list.
func (l *ErrorList) Add(err *UserError) {
	if err != nil {
		l.errors = append(l.errors, err)
	}
}

// AddValidation adds a validation error to the list.
func (l *ErrorList) AddValidation(field, message, suggestion string) {
	l.Add(&UserError{
		Code:       ErrCodeValidationFailed,
		Message:    fmt.Sprintf("%s: %s", field, message),
		Context:    field,
		Suggestion: suggestion,
	})
}

// HasErrors returns true if there are any errors.
func (l *ErrorList) HasErrors() bool {
	return len(l.errors) > 0
}

// Len returns the number of errors.
func (l *ErrorList) Len() int {
	return len(l.errors)
}

// Errors returns the list of errors.
func (l *ErrorList) Errors() []*UserError {
	result := make([]*UserError, len(l.errors))
	copy(result, l.errors)
	return result
}

// Error implements the error interface for ErrorList.
func (l *ErrorList) Error() string {
	if len(l.errors) == 0 {
		return ""
	}
	if len(l.errors) == 1 {
		return l.errors[0].Error()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d errors occurred:\n", len(l.errors))
	for i, err := range l.errors {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, err.Error())
	}
	return b.String()
}

// Format returns a detailed formatted output of all errors.
func (l *ErrorList) Format() string {
	if len(l.errors) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d error(s):\n", len(l.errors))
	for i, err := range l.errors {
		fmt.Fprintf(&b, "\n--- Error %d ---\n", i+1)
		b.WriteString(err.Format())
		b.WriteString("\n")
	}
	return b.String()
}

// AsError returns the ErrorList as an error, or nil if empty.
func (l *ErrorList) AsError() error {
	if !l.HasErrors() {
		return nil
	}
	return l
}

// Common user-friendly error constructors.

// NewConfigNotFoundError creates an error for missing config file.
func NewConfigNotFoundError(path string) *UserError {
	return &UserError{
		Code:       ErrCodeConfigNotFound,
		Message:    fmt.Sprintf("configuration file not found: %s", path),
		Context:    path,
		Suggestion: "Run 'preflight init' to create a new configuration, or check the file path.",
	}
}

// NewConfigParseError creates an error for YAML parsing failures.
func NewConfigParseError(path string, err error) *UserError {
	return &UserError{
		Code:       ErrCodeConfigParse,
		Message:    "failed to parse configuration file",
		Context:    path,
		Suggestion: "Check your YAML syntax. Common issues: incorrect indentation, missing colons, or unquoted special characters.",
		Underlying: err,
	}
}

// NewLayerNotFoundError creates an error for missing layer file.
func NewLayerNotFoundError(name, path string) *UserError {
	return &UserError{
		Code:       ErrCodeLayerNotFound,
		Message:    fmt.Sprintf("layer '%s' not found", name),
		Context:    path,
		Suggestion: fmt.Sprintf("Create the layer file at '%s' or remove it from your target's layer list.", path),
	}
}

// NewTargetNotFoundError creates an error for missing target.
func NewTargetNotFoundError(name string, available []string) *UserError {
	suggestion := "Check your preflight.yaml for available targets."
	if len(available) > 0 {
		suggestion = fmt.Sprintf("Available targets: %s", strings.Join(available, ", "))
	}
	return &UserError{
		Code:       ErrCodeTargetNotFound,
		Message:    fmt.Sprintf("target '%s' not found in manifest", name),
		Suggestion: suggestion,
	}
}

// NewValidationFailedError creates a validation error.
func NewValidationFailedError(field, message string) *UserError {
	return &UserError{
		Code:    ErrCodeValidationFailed,
		Message: fmt.Sprintf("validation failed for '%s': %s", field, message),
		Context: field,
	}
}

// NewCircularReferenceError creates an error for circular layer references.
func NewCircularReferenceError(chain []string) *UserError {
	return &UserError{
		Code:       ErrCodeCircularReference,
		Message:    fmt.Sprintf("circular reference detected: %s", strings.Join(chain, " → ")),
		Suggestion: "Review your layer includes to break the circular dependency.",
	}
}

// NewTemplateInvalidError creates an error for invalid template.
func NewTemplateInvalidError(path string, err error) *UserError {
	return &UserError{
		Code:       ErrCodeTemplateInvalid,
		Message:    "template parsing failed",
		Context:    path,
		Suggestion: "Check your template syntax. Ensure all variables are properly formatted: {{ .Variable }}",
		Underlying: err,
	}
}

// IsUserError checks if an error is a UserError with a specific code.
func IsUserError(err error, code string) bool {
	var ue *UserError
	if errors.As(err, &ue) {
		return ue.Code == code
	}
	return false
}

// GetUserError extracts a UserError from an error chain, if present.
func GetUserError(err error) *UserError {
	var ue *UserError
	if errors.As(err, &ue) {
		return ue
	}
	return nil
}

// NewYAMLParseError translates technical YAML errors into user-friendly messages.
func NewYAMLParseError(path string, err error) *UserError {
	errStr := err.Error()
	var message, suggestion string

	switch {
	case strings.Contains(errStr, "cannot unmarshal !!map into []string"):
		message = "invalid targets format"
		suggestion = `Targets should be a list of layer names, not a nested object.

Correct format:
  targets:
    default:
      - base
      - identity.work

Incorrect format:
  targets:
    default:
      layers:
        - base`

	case strings.Contains(errStr, "cannot unmarshal !!seq into map"):
		message = "expected an object but found a list"
		suggestion = "Check that you're using 'key: value' format instead of '- item' list format."

	case strings.Contains(errStr, "cannot unmarshal !!str into"):
		message = "unexpected string value"
		suggestion = "Check that nested values are properly structured with correct indentation."

	case strings.Contains(errStr, "did not find expected key"):
		message = "missing required field or incorrect indentation"
		suggestion = "YAML is sensitive to indentation. Use 2 spaces (not tabs) for each level."

	case strings.Contains(errStr, "mapping values are not allowed"):
		message = "invalid YAML structure"
		suggestion = "Check for missing colons after keys, or incorrect indentation."

	case strings.Contains(errStr, "found character that cannot start"):
		message = "invalid character in YAML"
		suggestion = "Quote string values that contain special characters like ':', '#', or '{'."

	default:
		message = "invalid YAML syntax"
		suggestion = "Check your YAML syntax. Common issues: incorrect indentation, missing colons, or unquoted special characters."
	}

	// Extract line number if present
	context := path
	if strings.Contains(errStr, "line ") {
		// Try to extract line number from error
		parts := strings.Split(errStr, "line ")
		if len(parts) > 1 {
			lineInfo := strings.Split(parts[1], ":")[0]
			context = fmt.Sprintf("%s (line %s)", path, lineInfo)
		}
	}

	return &UserError{
		Code:       ErrCodeConfigParse,
		Message:    message,
		Context:    context,
		Suggestion: suggestion,
		Underlying: err,
	}
}
