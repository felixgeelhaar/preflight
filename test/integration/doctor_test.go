package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctor_DetectsNoIssues(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	manifest := `
targets:
  default:
    - base
`
	layer := `
name: base
`

	configPath := h.CreateConfig(manifest, layer)

	// Run doctor
	report, err := h.Doctor(configPath, "default")
	require.NoError(t, err)
	require.NotNil(t, report)

	// Empty config should have no issues
	assert.False(t, report.HasIssues())
}

func TestDoctor_ReportsConfigPath(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	manifest := `
targets:
  default:
    - base
`
	layer := `
name: base
`

	configPath := h.CreateConfig(manifest, layer)

	// Run doctor
	report, err := h.Doctor(configPath, "default")
	require.NoError(t, err)

	// Should report correct config path
	assert.Equal(t, configPath, report.ConfigPath)
	assert.Equal(t, "default", report.Target)
}

func TestDoctor_TracksCheckTime(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	manifest := `
targets:
  default:
    - base
`
	layer := `
name: base
`

	configPath := h.CreateConfig(manifest, layer)

	// Run doctor
	report, err := h.Doctor(configPath, "default")
	require.NoError(t, err)

	// Should have check time
	assert.False(t, report.CheckedAt.IsZero())
	assert.Positive(t, report.Duration)
}
