package starship_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/starship"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := starship.NewProvider(runner)

	assert.Equal(t, "starship", p.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := starship.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptySection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := starship.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"starship": map[string]interface{}{},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// Should have install step
	require.Len(t, steps, 1)
	assert.Equal(t, "starship:install", steps[0].ID().String())
}

func TestProvider_Compile_WithSettings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := starship.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"starship": map[string]interface{}{
			"settings": map[string]interface{}{
				"add_newline": false,
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
	assert.Equal(t, "starship:install", steps[0].ID().String())
	assert.Equal(t, "starship:config", steps[1].ID().String())
}

func TestProvider_Compile_WithPreset(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := starship.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"starship": map[string]interface{}{
			"preset": "nerd-font-symbols",
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
	assert.Equal(t, "starship:install", steps[0].ID().String())
	assert.Equal(t, "starship:config", steps[1].ID().String())
}

func TestProvider_Compile_WithShell(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := starship.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"starship": map[string]interface{}{
			"shell": "zsh",
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
	assert.Equal(t, "starship:install", steps[0].ID().String())
	assert.Equal(t, "starship:shell:zsh", steps[1].ID().String())
}

func TestProvider_Compile_Complete(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := starship.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"starship": map[string]interface{}{
			"preset": "tokyo-night",
			"shell":  "zsh",
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 3)
	assert.Equal(t, "starship:install", steps[0].ID().String())
	assert.Equal(t, "starship:config", steps[1].ID().String())
	assert.Equal(t, "starship:shell:zsh", steps[2].ID().String())
}
