package apt_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/apt"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := apt.NewProvider(runner)

	assert.Equal(t, "apt", p.Name())
}

func TestProvider_Compile_Empty(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := apt.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithPackages(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := apt.NewProvider(runner)

	raw := map[string]interface{}{
		"apt": map[string]interface{}{
			"packages": []interface{}{"git", "curl"},
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 4)
}

func TestProvider_Compile_WithPPAs(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := apt.NewProvider(runner)

	raw := map[string]interface{}{
		"apt": map[string]interface{}{
			"ppas": []interface{}{"ppa:git-core/ppa"},
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 3)
}

func TestProvider_Compile_Full(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := apt.NewProvider(runner)

	raw := map[string]interface{}{
		"apt": map[string]interface{}{
			"ppas":     []interface{}{"ppa:git-core/ppa"},
			"packages": []interface{}{"git", "curl"},
		},
	}
	ctx := compiler.NewCompileContext(raw)
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	// ready + update + 1 PPA + 2 packages = 5 steps
	assert.Len(t, steps, 5)
}
