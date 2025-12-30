package kubernetes_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/kubernetes"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := kubernetes.NewProvider(runner)

	assert.Equal(t, "kubernetes", p.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := kubernetes.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptySection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := kubernetes.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"kubernetes": map[string]interface{}{},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithPlugins(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := kubernetes.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"kubernetes": map[string]interface{}{
			"plugins": []interface{}{"ctx", "ns"},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 2)
	assert.Equal(t, "kubernetes:plugin:ctx", steps[0].ID().String())
	assert.Equal(t, "kubernetes:plugin:ns", steps[1].ID().String())
}

func TestProvider_Compile_WithContexts(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := kubernetes.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"kubernetes": map[string]interface{}{
			"contexts": []interface{}{
				map[string]interface{}{
					"name":    "dev",
					"cluster": "dev-cluster",
				},
			},
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "kubernetes:context:dev", steps[0].ID().String())
}

func TestProvider_Compile_WithDefaultNamespace(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := kubernetes.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"kubernetes": map[string]interface{}{
			"default_namespace": "my-namespace",
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "kubernetes:namespace:my-namespace", steps[0].ID().String())
}

func TestProvider_Compile_Complete(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	p := kubernetes.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"kubernetes": map[string]interface{}{
			"plugins": []interface{}{"ctx"},
			"contexts": []interface{}{
				map[string]interface{}{
					"name":    "dev",
					"cluster": "dev-cluster",
				},
			},
			"default_namespace": "my-namespace",
		},
	})
	steps, err := p.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 3)
}
