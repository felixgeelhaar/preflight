package config_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator_Validate_ValidConfig_ReturnsNoErrors(t *testing.T) {
	t.Parallel()

	merged := &config.MergedConfig{
		Packages: config.PackageSet{
			Brew: config.BrewPackages{
				Formulae: []string{"git", "ripgrep"},
			},
		},
		Files: []config.FileDeclaration{
			{Path: "~/.gitconfig", Mode: config.FileModeGenerated},
		},
	}

	validator := config.NewValidator()
	errors := validator.Validate(merged)

	assert.Empty(t, errors)
}

func TestValidator_Validate_EmptyFilePath_ReturnsError(t *testing.T) {
	t.Parallel()

	merged := &config.MergedConfig{
		Files: []config.FileDeclaration{
			{Path: "", Mode: config.FileModeGenerated},
		},
	}

	validator := config.NewValidator()
	errors := validator.Validate(merged)

	require.Len(t, errors, 1)
	assert.Equal(t, "files[0].path", errors[0].Field)
	assert.Equal(t, "path is required", errors[0].Message)
}

func TestValidator_Validate_InvalidFileMode_ReturnsError(t *testing.T) {
	t.Parallel()

	merged := &config.MergedConfig{
		Files: []config.FileDeclaration{
			{Path: "~/.gitconfig", Mode: config.FileMode("invalid")},
		},
	}

	validator := config.NewValidator()
	errors := validator.Validate(merged)

	require.Len(t, errors, 1)
	assert.Equal(t, "files[0].mode", errors[0].Field)
	assert.Contains(t, errors[0].Message, "invalid file mode")
}

func TestValidator_Validate_TemplateModeMissingTemplate_ReturnsError(t *testing.T) {
	t.Parallel()

	merged := &config.MergedConfig{
		Files: []config.FileDeclaration{
			{Path: "~/.gitconfig", Mode: config.FileModeTemplate, Template: ""},
		},
	}

	validator := config.NewValidator()
	errors := validator.Validate(merged)

	require.Len(t, errors, 1)
	assert.Equal(t, "files[0].template", errors[0].Field)
	assert.Contains(t, errors[0].Message, "template path required")
}

func TestValidator_Validate_MultipleErrors_ReturnsAllErrors(t *testing.T) {
	t.Parallel()

	merged := &config.MergedConfig{
		Files: []config.FileDeclaration{
			{Path: "", Mode: config.FileModeGenerated},
			{Path: "~/.ssh/config", Mode: config.FileModeTemplate, Template: ""},
		},
	}

	validator := config.NewValidator()
	errors := validator.Validate(merged)

	assert.Len(t, errors, 2)
}

func TestValidator_Validate_EmptyConfig_ReturnsNoErrors(t *testing.T) {
	t.Parallel()

	merged := &config.MergedConfig{}

	validator := config.NewValidator()
	errors := validator.Validate(merged)

	assert.Empty(t, errors)
}

func TestValidationError_Error_ReturnsFormattedMessage(t *testing.T) {
	t.Parallel()

	err := config.ValidationError{
		Field:   "files[0].path",
		Message: "path is required",
	}

	assert.Equal(t, "files[0].path: path is required", err.Error())
}
