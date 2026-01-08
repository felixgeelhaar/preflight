package jetbrains_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/jetbrains"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := jetbrains.NewProvider(runner)

	assert.Equal(t, "jetbrains", provider.Name())
}

func TestProvider_Compile_NoConfig(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := jetbrains.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_EmptySection(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := jetbrains.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"jetbrains": map[string]interface{}{},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Empty(t, steps)
}

func TestProvider_Compile_WithSingleIDE(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := jetbrains.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"jetbrains": map[string]interface{}{
			"ides": []interface{}{
				map[string]interface{}{
					"name": "GoLand",
					"plugins": []interface{}{
						"com.intellij.go",
						"org.jetbrains.plugins.go-template",
					},
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "jetbrains:goland:plugins", steps[0].ID().String())
}

func TestProvider_Compile_WithMultipleIDEs(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := jetbrains.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"jetbrains": map[string]interface{}{
			"ides": []interface{}{
				map[string]interface{}{
					"name":    "GoLand",
					"plugins": []interface{}{"plugin1"},
				},
				map[string]interface{}{
					"name":    "PyCharm",
					"plugins": []interface{}{"plugin2"},
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 2)
}

func TestProvider_Compile_WithSharedPlugins(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := jetbrains.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"jetbrains": map[string]interface{}{
			"shared_plugins": []interface{}{
				"com.wakatime.intellij",
				"izhangzhihao.rainbow.brackets",
			},
			"ides": []interface{}{
				map[string]interface{}{
					"name":    "GoLand",
					"plugins": []interface{}{"go-plugin"},
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	// The step should include both shared and IDE-specific plugins
	assert.Equal(t, "jetbrains:goland:plugins", steps[0].ID().String())
}

func TestProvider_Compile_WithSettings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := jetbrains.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"jetbrains": map[string]interface{}{
			"ides": []interface{}{
				map[string]interface{}{
					"name":   "IntelliJIdea",
					"keymap": "VSCode",
					"settings": map[string]interface{}{
						"editor.fontSize": 14,
					},
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "jetbrains:intellijidea:settings", steps[0].ID().String())
}

func TestProvider_Compile_WithSettingsSync(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := jetbrains.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"jetbrains": map[string]interface{}{
			"settings_sync": map[string]interface{}{
				"enabled":          true,
				"sync_plugins":     true,
				"sync_code_styles": true,
			},
			"ides": []interface{}{
				map[string]interface{}{
					"name": "WebStorm",
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "jetbrains:webstorm:settingssync", steps[0].ID().String())
}

func TestProvider_Compile_Complete(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := jetbrains.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"jetbrains": map[string]interface{}{
			"shared_plugins": []interface{}{
				"com.wakatime.intellij",
			},
			"settings_sync": map[string]interface{}{
				"enabled":      true,
				"sync_plugins": true,
			},
			"ides": []interface{}{
				map[string]interface{}{
					"name":    "GoLand",
					"plugins": []interface{}{"go-plugin"},
					"keymap":  "VSCode",
					"settings": map[string]interface{}{
						"fontSize": 14,
					},
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	assert.Len(t, steps, 3) // plugins + settings + settingssync
}

func TestProvider_Compile_DisabledIDE(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := jetbrains.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"jetbrains": map[string]interface{}{
			"ides": []interface{}{
				map[string]interface{}{
					"name":     "GoLand",
					"plugins":  []interface{}{"plugin1"},
					"disabled": true,
				},
				map[string]interface{}{
					"name":    "PyCharm",
					"plugins": []interface{}{"plugin2"},
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)

	require.NoError(t, err)
	require.Len(t, steps, 1)
	// Only PyCharm should be configured
	assert.Equal(t, "jetbrains:pycharm:plugins", steps[0].ID().String())
}

func TestProvider_Compile_InvalidIDE(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	provider := jetbrains.NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"jetbrains": map[string]interface{}{
			"ides": []interface{}{
				map[string]interface{}{
					"name":    "InvalidIDE",
					"plugins": []interface{}{"plugin1"},
				},
			},
		},
	})
	_, err := provider.Compile(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown JetBrains IDE")
}
