package docker

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)
	assert.Equal(t, "docker", provider.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Nil(t, steps)
}

func TestProvider_Compile_BasicConfig(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"docker": map[string]interface{}{
			"install": true,
		},
	})

	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 2) // install + buildkit (default)

	// Verify step IDs
	ids := make([]string, len(steps))
	for i, step := range steps {
		ids[i] = step.ID().String()
	}
	assert.Contains(t, ids, "docker:install")
	assert.Contains(t, ids, "docker:buildkit")
}

func TestProvider_Compile_WithKubernetes(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"docker": map[string]interface{}{
			"install":    true,
			"kubernetes": true,
		},
	})

	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 3) // install + buildkit + kubernetes

	ids := make([]string, len(steps))
	for i, step := range steps {
		ids[i] = step.ID().String()
	}
	assert.Contains(t, ids, "docker:install")
	assert.Contains(t, ids, "docker:buildkit")
	assert.Contains(t, ids, "docker:kubernetes")
}

func TestProvider_Compile_WithContexts(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"docker": map[string]interface{}{
			"install": true,
			"contexts": []interface{}{
				map[string]interface{}{
					"name": "production",
					"host": "ssh://user@prod.example.com",
				},
				map[string]interface{}{
					"name": "staging",
					"host": "ssh://user@staging.example.com",
				},
			},
		},
	})

	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 4) // install + buildkit + 2 contexts

	ids := make([]string, len(steps))
	for i, step := range steps {
		ids[i] = step.ID().String()
	}
	assert.Contains(t, ids, "docker:install")
	assert.Contains(t, ids, "docker:context:production")
	assert.Contains(t, ids, "docker:context:staging")
}

func TestProvider_Compile_DisabledInstall(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"docker": map[string]interface{}{
			"install": false,
		},
	})

	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1) // only buildkit (install disabled)

	assert.Equal(t, "docker:buildkit", steps[0].ID().String())
}

func TestProvider_Compile_DisabledBuildKit(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"docker": map[string]interface{}{
			"install":  true,
			"buildkit": false,
		},
	})

	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 1) // only install (buildkit disabled)

	assert.Equal(t, "docker:install", steps[0].ID().String())
}

func TestProvider_Compile_InvalidConfig(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"docker": map[string]interface{}{
			"registries": "not-a-list",
		},
	})

	_, err := provider.Compile(ctx)
	assert.Error(t, err)
}

func TestProvider_ImplementsInterface(_ *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	var _ compiler.Provider = provider
}
