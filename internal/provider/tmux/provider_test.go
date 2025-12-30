package tmux_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/tmux"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := tmux.NewProvider(runner)

	assert.Equal(t, "tmux", p.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := tmux.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptySection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := tmux.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"tmux": map[string]interface{}{},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithPlugins(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := tmux.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"tmux": map[string]interface{}{
			"plugins": []interface{}{"tmux-plugins/tpm", "tmux-plugins/tmux-sensible"},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// TPM step + 2 plugin steps
	require.Len(t, steps, 3)
	assert.Equal(t, "tmux:tpm", steps[0].ID().String())
}

func TestProvider_Compile_WithSettings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := tmux.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"tmux": map[string]interface{}{
			"settings": map[string]interface{}{
				"prefix": "C-a",
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "tmux:config", steps[0].ID().String())
}

func TestProvider_Compile_WithConfigFile(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := tmux.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"tmux": map[string]interface{}{
			"config_file": "~/.config/tmux/tmux.conf",
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "tmux:config", steps[0].ID().String())
}

func TestProvider_Compile_Complete(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := tmux.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"tmux": map[string]interface{}{
			"plugins": []interface{}{"tmux-plugins/tmux-sensible"},
			"settings": map[string]interface{}{
				"prefix": "C-a",
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// TPM + plugin + config
	require.Len(t, steps, 3)
}
