package config_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTargetName_EmptyString_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := config.NewTargetName("")

	require.Error(t, err)
	require.ErrorIs(t, err, config.ErrEmptyTargetName)
}

func TestNewTargetName_ValidName_ReturnsTargetName(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"work",
		"personal",
		"dev-machine",
		"home_office",
	}

	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tn, err := config.NewTargetName(name)

			require.NoError(t, err)
			assert.Equal(t, name, tn.String())
		})
	}
}

func TestTargetName_String_ReturnsOriginalValue(t *testing.T) {
	t.Parallel()

	expected := "personal"
	tn, err := config.NewTargetName(expected)
	require.NoError(t, err)

	actual := tn.String()

	assert.Equal(t, expected, actual)
}

func TestNewTargetName_WhitespaceOnly_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := config.NewTargetName("   ")

	require.Error(t, err)
	require.ErrorIs(t, err, config.ErrEmptyTargetName)
}

func TestNewTargetName_InvalidCharacters_ReturnsError(t *testing.T) {
	t.Parallel()

	invalidNames := []string{
		"target/name",
		"target\\name",
		"target:name",
		"target name",
		"target.name", // Dots not allowed in target names (unlike layer names)
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := config.NewTargetName(name)

			require.Error(t, err)
			require.ErrorIs(t, err, config.ErrInvalidTargetName)
		})
	}
}

func TestTargetName_IsZero_ReturnsTrueForZeroValue(t *testing.T) {
	t.Parallel()

	var tn config.TargetName

	assert.True(t, tn.IsZero())
}

func TestTargetName_IsZero_ReturnsFalseForValidValue(t *testing.T) {
	t.Parallel()

	tn, err := config.NewTargetName("work")
	require.NoError(t, err)

	assert.False(t, tn.IsZero())
}
