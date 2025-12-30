package kubernetes_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_Empty(t *testing.T) {
	t.Parallel()

	cfg, err := kubernetes.ParseConfig(map[string]interface{}{})

	require.NoError(t, err)
	assert.Empty(t, cfg.Plugins)
	assert.Empty(t, cfg.Contexts)
	assert.Empty(t, cfg.DefaultNamespace)
}

func TestParseConfig_WithPlugins(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"plugins": []interface{}{"ctx", "ns", "stern"},
	}

	cfg, err := kubernetes.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Plugins, 3)
	assert.Contains(t, cfg.Plugins, "ctx")
	assert.Contains(t, cfg.Plugins, "ns")
	assert.Contains(t, cfg.Plugins, "stern")
}

func TestParseConfig_WithContexts(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"contexts": []interface{}{
			map[string]interface{}{
				"name":      "dev",
				"cluster":   "dev-cluster",
				"user":      "dev-admin",
				"namespace": "development",
			},
			map[string]interface{}{
				"name":    "prod",
				"cluster": "prod-cluster",
			},
		},
	}

	cfg, err := kubernetes.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Contexts, 2)
	assert.Equal(t, "dev", cfg.Contexts[0].Name)
	assert.Equal(t, "dev-cluster", cfg.Contexts[0].Cluster)
	assert.Equal(t, "dev-admin", cfg.Contexts[0].User)
	assert.Equal(t, "development", cfg.Contexts[0].Namespace)
	assert.Equal(t, "prod", cfg.Contexts[1].Name)
	assert.Empty(t, cfg.Contexts[1].User)
}

func TestParseConfig_WithDefaultNamespace(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"default_namespace": "my-namespace",
	}

	cfg, err := kubernetes.ParseConfig(raw)

	require.NoError(t, err)
	assert.Equal(t, "my-namespace", cfg.DefaultNamespace)
}

func TestParseConfig_Complete(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"plugins": []interface{}{"ctx"},
		"contexts": []interface{}{
			map[string]interface{}{
				"name":    "dev",
				"cluster": "dev-cluster",
			},
		},
		"default_namespace": "my-namespace",
	}

	cfg, err := kubernetes.ParseConfig(raw)

	require.NoError(t, err)
	assert.Len(t, cfg.Plugins, 1)
	assert.Len(t, cfg.Contexts, 1)
	assert.Equal(t, "my-namespace", cfg.DefaultNamespace)
}

func TestParseConfig_InvalidPluginsList(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"plugins": "not-a-list",
	}

	cfg, err := kubernetes.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugins must be a list")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidPluginItem(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"plugins": []interface{}{123},
	}

	cfg, err := kubernetes.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin must be a string")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidContext_NotObject(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"contexts": []interface{}{
			"not-an-object",
		},
	}

	cfg, err := kubernetes.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context must be an object")
	assert.Nil(t, cfg)
}

func TestParseConfig_InvalidContext_NoName(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"contexts": []interface{}{
			map[string]interface{}{
				"cluster": "my-cluster",
			},
		},
	}

	cfg, err := kubernetes.ParseConfig(raw)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context must have a name")
	assert.Nil(t, cfg)
}
