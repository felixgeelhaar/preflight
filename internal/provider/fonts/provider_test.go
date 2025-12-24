package fonts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	p := NewProvider(mocks.NewCommandRunner())
	assert.Equal(t, "fonts", p.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	p := NewProvider(mocks.NewCommandRunner())
	ctx := compiler.NewCompileContext(map[string]interface{}{})

	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Nil(t, steps)
}

func TestProvider_Compile_EmptyConfig(t *testing.T) {
	t.Parallel()

	p := NewProvider(mocks.NewCommandRunner())
	ctx := compiler.NewCompileContext(map[string]interface{}{
		"fonts": map[string]interface{}{},
	})

	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithNerdFonts(t *testing.T) {
	t.Parallel()

	p := NewProvider(mocks.NewCommandRunner())
	ctx := compiler.NewCompileContext(map[string]interface{}{
		"fonts": map[string]interface{}{
			"nerd_fonts": []interface{}{"JetBrainsMono", "FiraCode", "Hack"},
		},
	})

	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 3)

	// Verify step IDs
	assert.Equal(t, compiler.MustNewStepID("fonts:nerd:JetBrainsMono"), steps[0].ID())
	assert.Equal(t, compiler.MustNewStepID("fonts:nerd:FiraCode"), steps[1].ID())
	assert.Equal(t, compiler.MustNewStepID("fonts:nerd:Hack"), steps[2].ID())
}

func TestProvider_Compile_InvalidConfig(t *testing.T) {
	t.Parallel()

	p := NewProvider(mocks.NewCommandRunner())
	ctx := compiler.NewCompileContext(map[string]interface{}{
		"fonts": map[string]interface{}{
			"nerd_fonts": "not-a-list",
		},
	})

	steps, err := p.Compile(ctx)

	require.Error(t, err)
	assert.Nil(t, steps)
}

func TestProvider_Compile_StepDependencies(t *testing.T) {
	t.Parallel()

	p := NewProvider(mocks.NewCommandRunner())
	ctx := compiler.NewCompileContext(map[string]interface{}{
		"fonts": map[string]interface{}{
			"nerd_fonts": []interface{}{"JetBrainsMono"},
		},
	})

	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)

	// Verify the step depends on the cask-fonts tap
	deps := steps[0].DependsOn()
	require.Len(t, deps, 1)
	assert.Equal(t, compiler.MustNewStepID("brew:tap:homebrew/cask-fonts"), deps[0])
}

// TestProvider_ImplementsInterface verifies the provider implements the interface.
func TestProvider_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ compiler.Provider = (*Provider)(nil)
}
