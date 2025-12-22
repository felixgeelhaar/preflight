package config

import "fmt"

// ValidationError represents a validation failure.
type ValidationError struct {
	Field   string
	Message string
}

// Error returns a formatted error message.
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validator validates configuration.
type Validator struct{}

// NewValidator creates a new Validator.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates a MergedConfig and returns any validation errors.
func (v *Validator) Validate(config *MergedConfig) []ValidationError {
	var errors []ValidationError

	errors = append(errors, v.validateFiles(config.Files)...)

	return errors
}

func (v *Validator) validateFiles(files []FileDeclaration) []ValidationError {
	var errors []ValidationError

	for i, file := range files {
		// Check path is not empty
		if file.Path == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("files[%d].path", i),
				Message: "path is required",
			})
		}

		// Check mode is valid
		if !v.isValidFileMode(file.Mode) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("files[%d].mode", i),
				Message: fmt.Sprintf("invalid file mode: %s", file.Mode),
			})
		}

		// Check template is provided for template mode
		if file.Mode == FileModeTemplate && file.Template == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("files[%d].template", i),
				Message: "template path required for template mode",
			})
		}
	}

	return errors
}

func (v *Validator) isValidFileMode(mode FileMode) bool {
	switch mode {
	case FileModeGenerated, FileModeTemplate, FileModeBYO, "":
		return true
	default:
		return false
	}
}
