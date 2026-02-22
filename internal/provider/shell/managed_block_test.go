package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadManagedBlock_Found(t *testing.T) {
	t.Parallel()

	content := `# some config
# >>> preflight env >>>
export EDITOR="nvim"
# <<< preflight env <<<
# more config`

	result := ReadManagedBlock(content, "env")
	assert.Equal(t, "export EDITOR=\"nvim\"\n", result)
}

func TestReadManagedBlock_NotFound(t *testing.T) {
	t.Parallel()

	content := "# some config\n"
	result := ReadManagedBlock(content, "env")
	assert.Empty(t, result)
}

func TestReadManagedBlock_Empty(t *testing.T) {
	t.Parallel()

	content := `# >>> preflight env >>>
# <<< preflight env <<<`

	result := ReadManagedBlock(content, "env")
	assert.Empty(t, result)
}

func TestWriteManagedBlock_NewBlock(t *testing.T) {
	t.Parallel()

	content := "# existing config\n"
	block := "export FOO=\"bar\"\n"

	result := WriteManagedBlock(content, "env", block)

	assert.Contains(t, result, "# >>> preflight env >>>")
	assert.Contains(t, result, "export FOO=\"bar\"")
	assert.Contains(t, result, "# <<< preflight env <<<")
	assert.Contains(t, result, "# existing config")
}

func TestWriteManagedBlock_ReplaceExisting(t *testing.T) {
	t.Parallel()

	content := `# before
# >>> preflight env >>>
export OLD="value"
# <<< preflight env <<<
# after`

	block := "export NEW=\"value\"\n"
	result := WriteManagedBlock(content, "env", block)

	assert.Contains(t, result, "# before")
	assert.Contains(t, result, "# after")
	assert.Contains(t, result, "export NEW=\"value\"")
	assert.NotContains(t, result, "export OLD=\"value\"")
}

func TestWriteManagedBlock_EmptyContent(t *testing.T) {
	t.Parallel()

	block := "export FOO=\"bar\"\n"
	result := WriteManagedBlock("", "env", block)

	assert.Contains(t, result, "# >>> preflight env >>>")
	assert.Contains(t, result, "export FOO=\"bar\"")
}

func TestGenerateEnvBlock(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"EDITOR": "nvim",
		"SHELL":  "/bin/zsh",
	}

	result := generateEnvBlock(env)

	assert.Contains(t, result, `export EDITOR="nvim"`)
	assert.Contains(t, result, `export SHELL="/bin/zsh"`)
}

func TestGenerateEnvBlock_Empty(t *testing.T) {
	t.Parallel()

	result := generateEnvBlock(map[string]string{})
	assert.Empty(t, result)
}

func TestGenerateEnvBlock_Deterministic(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"Z_VAR": "z",
		"A_VAR": "a",
		"M_VAR": "m",
	}

	// Run multiple times to verify deterministic output
	first := generateEnvBlock(env)
	for i := 0; i < 10; i++ {
		assert.Equal(t, first, generateEnvBlock(env))
	}
}

func TestGenerateAliasBlock(t *testing.T) {
	t.Parallel()

	aliases := map[string]string{
		"ll": "ls -la",
		"k":  "kubectl",
	}

	result := generateAliasBlock(aliases)

	assert.Contains(t, result, `alias k="kubectl"`)
	assert.Contains(t, result, `alias ll="ls -la"`)
}

func TestGenerateAliasBlock_Empty(t *testing.T) {
	t.Parallel()

	result := generateAliasBlock(map[string]string{})
	assert.Empty(t, result)
}

func TestContainsPlugin_Found(t *testing.T) {
	t.Parallel()

	content := "plugins=(git docker kubectl)\n"
	assert.True(t, containsPlugin(content, "git"))
	assert.True(t, containsPlugin(content, "docker"))
	assert.True(t, containsPlugin(content, "kubectl"))
}

func TestContainsPlugin_NotFound(t *testing.T) {
	t.Parallel()

	content := "plugins=(git docker)\n"
	assert.False(t, containsPlugin(content, "kubectl"))
}

func TestContainsPlugin_NoPluginsLine(t *testing.T) {
	t.Parallel()

	content := "# just comments\n"
	assert.False(t, containsPlugin(content, "git"))
}

func TestContainsPlugin_EmptyPlugins(t *testing.T) {
	t.Parallel()

	content := "plugins=()\n"
	assert.False(t, containsPlugin(content, "git"))
}

func TestAddPluginToConfig_AddToExisting(t *testing.T) {
	t.Parallel()

	content := "plugins=(git docker)\n"
	result := addPluginToConfig(content, "kubectl")

	assert.Contains(t, result, "kubectl")
	assert.Contains(t, result, "git")
	assert.Contains(t, result, "docker")
}

func TestAddPluginToConfig_AlreadyPresent(t *testing.T) {
	t.Parallel()

	content := "plugins=(git docker)\n"
	result := addPluginToConfig(content, "git")

	assert.Equal(t, content, result)
}

func TestAddPluginToConfig_NoPluginsLine(t *testing.T) {
	t.Parallel()

	content := "# some config\n"
	result := addPluginToConfig(content, "git")

	assert.Contains(t, result, "plugins=(git)")
}

func TestAddPluginToConfig_EmptyPlugins(t *testing.T) {
	t.Parallel()

	content := "plugins=()\n"
	result := addPluginToConfig(content, "git")

	assert.Contains(t, result, "plugins=(git)")
}

func TestReadManagedBlock_NoEndMarker(t *testing.T) {
	t.Parallel()

	content := "# >>> preflight env >>>\nexport EDITOR=\"nvim\"\n# no end marker"
	result := ReadManagedBlock(content, "env")
	assert.Empty(t, result)
}

func TestWriteManagedBlock_MalformedBlock(t *testing.T) {
	t.Parallel()

	// Start marker exists but no end marker
	content := "# before\n# >>> preflight env >>>\nexport OLD=\"value\"\n# after stuff"
	block := "export NEW=\"value\"\n"

	result := WriteManagedBlock(content, "env", block)

	assert.Contains(t, result, "# before")
	assert.Contains(t, result, "export NEW=\"value\"")
	assert.NotContains(t, result, "export OLD=\"value\"")
}

func TestWriteManagedBlock_ContentNoTrailingNewline(t *testing.T) {
	t.Parallel()

	content := "# no trailing newline"
	block := "export FOO=\"bar\"\n"

	result := WriteManagedBlock(content, "env", block)

	assert.Contains(t, result, "# no trailing newline")
	assert.Contains(t, result, "export FOO=\"bar\"")
}

func TestAddPluginToConfig_EmptyContent(t *testing.T) {
	t.Parallel()

	result := addPluginToConfig("", "git")
	assert.Contains(t, result, "plugins=(git)")
}

func TestShellConfigPath(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "~/.zshrc", shellConfigPath("zsh"))
	assert.Equal(t, "~/.bashrc", shellConfigPath("bash"))
	assert.Equal(t, "~/.config/fish/config.fish", shellConfigPath("fish"))
	assert.Empty(t, shellConfigPath("unknown"))
}
