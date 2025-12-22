package config_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLayerName_EmptyString_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := config.NewLayerName("")

	require.Error(t, err)
	require.ErrorIs(t, err, config.ErrEmptyLayerName)
}

func TestNewLayerName_ValidName_ReturnsLayerName(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"base",
		"identity.work",
		"role.go-developer",
		"device.macbook-pro",
	}

	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ln, err := config.NewLayerName(name)

			require.NoError(t, err)
			assert.Equal(t, name, ln.String())
		})
	}
}

func TestLayerName_String_ReturnsOriginalValue(t *testing.T) {
	t.Parallel()

	expected := "identity.personal"
	ln, err := config.NewLayerName(expected)
	require.NoError(t, err)

	actual := ln.String()

	assert.Equal(t, expected, actual)
}

func TestNewLayerName_WhitespaceOnly_ReturnsError(t *testing.T) {
	t.Parallel()

	_, err := config.NewLayerName("   ")

	require.Error(t, err)
	require.ErrorIs(t, err, config.ErrEmptyLayerName)
}

func TestNewLayerName_InvalidCharacters_ReturnsError(t *testing.T) {
	t.Parallel()

	invalidNames := []string{
		"layer/name",
		"layer\\name",
		"layer:name",
		"layer name",
		"layer\tname",
		"layer\nname",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := config.NewLayerName(name)

			require.Error(t, err)
			require.ErrorIs(t, err, config.ErrInvalidLayerName)
		})
	}
}

func TestLayerName_IsZero_ReturnsTrueForZeroValue(t *testing.T) {
	t.Parallel()

	var ln config.LayerName

	assert.True(t, ln.IsZero())
}

func TestLayerName_IsZero_ReturnsFalseForValidValue(t *testing.T) {
	t.Parallel()

	ln, err := config.NewLayerName("base")
	require.NoError(t, err)

	assert.False(t, ln.IsZero())
}
