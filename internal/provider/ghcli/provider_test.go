package ghcli_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/ghcli"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := ghcli.NewProvider(runner)

	assert.Equal(t, "github-cli", p.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := ghcli.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptySection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := ghcli.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"github-cli": map[string]interface{}{},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithExtensions(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := ghcli.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"github-cli": map[string]interface{}{
			"extensions": []interface{}{"dlvhdr/gh-dash", "github/gh-copilot"},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
	assert.Equal(t, "ghcli:extension:dlvhdr/gh-dash", steps[0].ID().String())
	assert.Equal(t, "ghcli:extension:github/gh-copilot", steps[1].ID().String())
}

func TestProvider_Compile_WithAliases(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := ghcli.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"github-cli": map[string]interface{}{
			"aliases": map[string]interface{}{
				"co": "pr checkout",
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "ghcli:alias:co", steps[0].ID().String())
}

func TestProvider_Compile_WithConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := ghcli.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"github-cli": map[string]interface{}{
			"config": map[string]interface{}{
				"editor": "nvim",
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "ghcli:config:editor", steps[0].ID().String())
}

func TestProvider_Compile_Complete(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := ghcli.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"github-cli": map[string]interface{}{
			"extensions": []interface{}{"dlvhdr/gh-dash"},
			"aliases": map[string]interface{}{
				"co": "pr checkout",
			},
			"config": map[string]interface{}{
				"editor": "nvim",
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 3)
}
