package main

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseModeValue_Valid(t *testing.T) {
	for _, mode := range []config.ReproducibilityMode{
		config.ModeIntent,
		config.ModeLocked,
		config.ModeFrozen,
	} {
		got, err := parseModeValue(string(mode))
		require.NoError(t, err)
		assert.Equal(t, mode, got)
	}
}

func TestParseModeValue_Invalid(t *testing.T) {
	_, err := parseModeValue("invalid-mode")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")
}

func TestResolveModeOverride_ReturnsMode(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("mode", "", "mode override")
	require.NoError(t, cmd.Flags().Set("mode", string(config.ModeLocked)))

	mode, err := resolveModeOverride(cmd)
	require.NoError(t, err)
	require.NotNil(t, mode)
	assert.Equal(t, config.ModeLocked, *mode)
}

func TestResolveModeOverride_NoFlagChanged(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("mode", "", "mode override")

	mode, err := resolveModeOverride(cmd)
	require.NoError(t, err)
	assert.Nil(t, mode)
}
