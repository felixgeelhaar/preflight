package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to create LayerName without error handling in tests
func mustLayerName(s string) LayerName {
	n, err := NewLayerName(s)
	if err != nil {
		panic(err)
	}
	return n
}

func TestNewLayerResolver(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	require.NotNil(t, r)
	assert.NotNil(t, r.layers)
	assert.NotNil(t, r.resolved)
	assert.NotNil(t, r.resolving)
}

func TestLayerResolver_RegisterLayer(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	layer := &Layer{Name: mustLayerName("base")}

	r.RegisterLayer(layer)

	assert.Len(t, r.layers, 1)
	assert.Equal(t, layer, r.layers["base"])
}

func TestLayerResolver_Resolve_Empty(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	result, err := r.Resolve(nil)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestLayerResolver_Resolve_SingleLayer(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	layer := &Layer{
		Name: mustLayerName("base"),
		Packages: PackageSet{
			Brew: BrewPackages{
				Formulae: []string{"git"},
			},
		},
	}
	r.RegisterLayer(layer)

	result, err := r.Resolve([]LayerName{mustLayerName("base")})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "base", result[0].Name.String())
}

func TestLayerResolver_Resolve_LayerNotFound(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()

	_, err := r.Resolve([]LayerName{mustLayerName("nonexistent")})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "layer not found")
}

func TestLayerResolver_Resolve_MultipleLayers(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	r.RegisterLayer(&Layer{Name: mustLayerName("base")})
	r.RegisterLayer(&Layer{Name: mustLayerName("dev")})

	result, err := r.Resolve([]LayerName{mustLayerName("base"), mustLayerName("dev")})

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestLayerResolver_mergeLayers_Empty(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	result := r.mergeLayers(nil)

	assert.NotNil(t, result)
}

func TestLayerResolver_mergeLayers_Single(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	layer := &Layer{Name: mustLayerName("base")}

	result := r.mergeLayers([]*Layer{layer})

	assert.Equal(t, layer, result)
}

func TestLayerResolver_mergeLayers_Multiple(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	layer1 := &Layer{
		Name: mustLayerName("base"),
		Packages: PackageSet{
			Brew: BrewPackages{
				Formulae: []string{"git"},
			},
		},
	}
	layer2 := &Layer{
		Name: mustLayerName("dev"),
		Packages: PackageSet{
			Brew: BrewPackages{
				Formulae: []string{"vim"},
			},
		},
	}

	result := r.mergeLayers([]*Layer{layer1, layer2})

	assert.Contains(t, result.Packages.Brew.Formulae, "git")
	assert.Contains(t, result.Packages.Brew.Formulae, "vim")
}

func TestLayerResolver_mergeLayer(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	parent := &Layer{
		Name: mustLayerName("parent"),
		Git: GitConfig{
			User: GitUserConfig{Name: "Parent"},
		},
	}
	child := &Layer{
		Name: mustLayerName("child"),
		Git: GitConfig{
			User: GitUserConfig{Email: "child@example.com"},
		},
	}

	result := r.mergeLayer(parent, child)

	assert.Equal(t, "Parent", result.Git.User.Name)
	assert.Equal(t, "child@example.com", result.Git.User.Email)
}

func TestLayerResolver_mergePackages(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	parent := PackageSet{
		Brew: BrewPackages{
			Taps:     []string{"tap1"},
			Formulae: []string{"git"},
			Casks:    []string{"chrome"},
		},
		Apt: AptPackages{
			PPAs:     []string{"ppa1"},
			Packages: []string{"vim"},
		},
	}
	child := PackageSet{
		Brew: BrewPackages{
			Taps:     []string{"tap2"},
			Formulae: []string{"vim"},
			Casks:    []string{"firefox"},
		},
		Apt: AptPackages{
			PPAs:     []string{"ppa2"},
			Packages: []string{"git"},
		},
	}

	result := r.mergePackages(parent, child)

	assert.Contains(t, result.Brew.Taps, "tap1")
	assert.Contains(t, result.Brew.Taps, "tap2")
	assert.Contains(t, result.Brew.Formulae, "git")
	assert.Contains(t, result.Brew.Formulae, "vim")
	assert.Contains(t, result.Brew.Casks, "chrome")
	assert.Contains(t, result.Brew.Casks, "firefox")
	assert.Contains(t, result.Apt.PPAs, "ppa1")
	assert.Contains(t, result.Apt.PPAs, "ppa2")
	assert.Contains(t, result.Apt.Packages, "vim")
	assert.Contains(t, result.Apt.Packages, "git")
}

func TestLayerResolver_mergeFiles(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	parent := []FileDeclaration{
		{Path: "/home/user/.bashrc", Template: "parent bashrc"},
		{Path: "/home/user/.vimrc", Template: "parent vimrc"},
	}
	child := []FileDeclaration{
		{Path: "/home/user/.bashrc", Template: "child bashrc"}, // overrides
		{Path: "/home/user/.zshrc", Template: "child zshrc"},
	}

	result := r.mergeFiles(parent, child)

	assert.Len(t, result, 3)
	// Find the .bashrc file
	var bashrc FileDeclaration
	for _, f := range result {
		if f.Path == "/home/user/.bashrc" {
			bashrc = f
			break
		}
	}
	assert.Equal(t, "child bashrc", bashrc.Template)
}

func TestLayerResolver_mergeGit(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	parent := GitConfig{
		User: GitUserConfig{
			Name:  "Parent Name",
			Email: "parent@example.com",
		},
		Core: GitCoreConfig{
			Editor: "vim",
		},
		Aliases: map[string]string{
			"st": "status",
		},
		Includes: []GitInclude{{Path: "~/.gitconfig.local"}},
	}
	child := GitConfig{
		User: GitUserConfig{
			Email:      "child@example.com",
			SigningKey: "ABC123",
		},
		Core: GitCoreConfig{
			AutoCRLF: "input",
		},
		Commit: GitCommitConfig{
			GPGSign: true,
		},
		GPG: GitGPGConfig{
			Format:  "openpgp",
			Program: "/usr/bin/gpg",
		},
		Aliases: map[string]string{
			"co": "checkout",
		},
		Includes: []GitInclude{{Path: "~/.gitconfig.work"}},
	}

	result := r.mergeGit(parent, child)

	assert.Equal(t, "Parent Name", result.User.Name)
	assert.Equal(t, "child@example.com", result.User.Email)
	assert.Equal(t, "ABC123", result.User.SigningKey)
	assert.Equal(t, "vim", result.Core.Editor)
	assert.Equal(t, "input", result.Core.AutoCRLF)
	assert.True(t, result.Commit.GPGSign)
	assert.Equal(t, "openpgp", result.GPG.Format)
	assert.Equal(t, "/usr/bin/gpg", result.GPG.Program)
	assert.Contains(t, result.Aliases, "st")
	assert.Contains(t, result.Aliases, "co")
	assert.Len(t, result.Includes, 2)
}

func TestLayerResolver_mergeSSH(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	parent := SSHConfig{
		Include: "~/.ssh/config.d/*",
		Defaults: SSHDefaultsConfig{
			AddKeysToAgent:      true,
			ServerAliveInterval: 60,
		},
		Hosts: []SSHHostConfig{
			{Host: "github.com", HostName: "github.com", User: "git"},
		},
		Matches: []SSHMatchConfig{
			{Match: "host *.local", HostName: "local.example.com"},
		},
	}
	child := SSHConfig{
		Include: "~/.ssh/config.d/*.conf",
		Defaults: SSHDefaultsConfig{
			IdentitiesOnly:      true,
			ForwardAgent:        true,
			ServerAliveCountMax: 3,
		},
		Hosts: []SSHHostConfig{
			{Host: "gitlab.com", HostName: "gitlab.com", User: "git"},
		},
		Matches: []SSHMatchConfig{
			{Match: "host *.corp", HostName: "corp.example.com"},
		},
	}

	result := r.mergeSSH(parent, child)

	assert.Equal(t, "~/.ssh/config.d/*.conf", result.Include)
	assert.True(t, result.Defaults.AddKeysToAgent)
	assert.True(t, result.Defaults.IdentitiesOnly)
	assert.True(t, result.Defaults.ForwardAgent)
	assert.Equal(t, 60, result.Defaults.ServerAliveInterval)
	assert.Equal(t, 3, result.Defaults.ServerAliveCountMax)
	assert.Len(t, result.Hosts, 2)
	assert.Len(t, result.Matches, 2)
}

func TestLayerResolver_mergeRuntime(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	parent := RuntimeConfig{
		Backend: "asdf",
		Scope:   "global",
		Tools: []RuntimeToolConfig{
			{Name: "node", Version: "18"},
		},
		Plugins: []RuntimePluginConfig{
			{Name: "plugin1", URL: "https://example.com/repo1"},
		},
	}
	child := RuntimeConfig{
		Backend: "mise",
		Tools: []RuntimeToolConfig{
			{Name: "go", Version: "1.21"},
		},
		Plugins: []RuntimePluginConfig{
			{Name: "plugin2", URL: "https://example.com/repo2"},
		},
	}

	result := r.mergeRuntime(parent, child)

	assert.Equal(t, "mise", result.Backend)
	assert.Equal(t, "global", result.Scope)
	assert.Len(t, result.Tools, 2)
	assert.Len(t, result.Plugins, 2)
}

func TestLayerResolver_mergeShell(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	parent := ShellConfig{
		Default: "bash",
		Shells: []ShellConfigEntry{
			{Name: "bash", Framework: "oh-my-bash"},
		},
		Starship: ShellStarshipConfig{
			Enabled: true,
		},
		Env: map[string]string{
			"EDITOR": "vim",
		},
		Aliases: map[string]string{
			"ll": "ls -la",
		},
	}
	child := ShellConfig{
		Default: "zsh",
		Shells: []ShellConfigEntry{
			{Name: "zsh", Framework: "oh-my-zsh"},
		},
		Starship: ShellStarshipConfig{
			Preset: "nerd-font-symbols",
		},
		Env: map[string]string{
			"PAGER": "less",
		},
		Aliases: map[string]string{
			"gs": "git status",
		},
	}

	result := r.mergeShell(parent, child)

	assert.Equal(t, "zsh", result.Default)
	assert.Len(t, result.Shells, 2)
	assert.True(t, result.Starship.Enabled)
	assert.Equal(t, "nerd-font-symbols", result.Starship.Preset)
	assert.Contains(t, result.Env, "EDITOR")
	assert.Contains(t, result.Env, "PAGER")
	assert.Contains(t, result.Aliases, "ll")
	assert.Contains(t, result.Aliases, "gs")
}

func TestLayerResolver_mergeNvim(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	parent := NvimConfig{
		Preset:        "lazyvim",
		PluginManager: "lazy",
		EnsureInstall: true,
	}
	child := NvimConfig{
		ConfigRepo: "https://github.com/user/nvim-config",
	}

	result := r.mergeNvim(parent, child)

	assert.Equal(t, "lazyvim", result.Preset)
	assert.Equal(t, "lazy", result.PluginManager)
	assert.Equal(t, "https://github.com/user/nvim-config", result.ConfigRepo)
	assert.True(t, result.EnsureInstall)
}

func TestLayerResolver_mergeVSCode(t *testing.T) {
	t.Parallel()

	r := NewLayerResolver()
	parent := VSCodeConfig{
		Extensions: []string{"ext1", "ext2"},
		Settings: map[string]interface{}{
			"editor.fontSize": 14,
		},
		Keybindings: []VSCodeKeybinding{
			{Key: "ctrl+s", Command: "save"},
		},
	}
	child := VSCodeConfig{
		Extensions: []string{"ext3"},
		Settings: map[string]interface{}{
			"editor.tabSize": 2,
		},
		Keybindings: []VSCodeKeybinding{
			{Key: "ctrl+p", Command: "quickOpen"},
		},
	}

	result := r.mergeVSCode(parent, child)

	assert.Contains(t, result.Extensions, "ext1")
	assert.Contains(t, result.Extensions, "ext2")
	assert.Contains(t, result.Extensions, "ext3")
	assert.Contains(t, result.Settings, "editor.fontSize")
	assert.Contains(t, result.Settings, "editor.tabSize")
	assert.Len(t, result.Keybindings, 2)
}

func TestUniqueStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  []string
		expect []string
	}{
		{
			name:   "empty",
			input:  nil,
			expect: []string{},
		},
		{
			name:   "no duplicates",
			input:  []string{"a", "b", "c"},
			expect: []string{"a", "b", "c"},
		},
		{
			name:   "with duplicates",
			input:  []string{"a", "b", "a", "c", "b"},
			expect: []string{"a", "b", "c"},
		},
		{
			name:   "all same",
			input:  []string{"x", "x", "x"},
			expect: []string{"x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := uniqueStrings(tt.input)
			assert.ElementsMatch(t, tt.expect, result)
		})
	}
}

func TestLayerExtends_Fields(t *testing.T) {
	t.Parallel()

	extends := LayerExtends{
		Parent:   "base",
		Parents:  []string{"base", "common"},
		Override: true,
	}

	assert.Equal(t, "base", extends.Parent)
	assert.Len(t, extends.Parents, 2)
	assert.True(t, extends.Override)
}

func TestInheritableLayer_Struct(t *testing.T) {
	t.Parallel()

	il := InheritableLayer{
		Layer: Layer{
			Name: mustLayerName("test"),
		},
		Extends: LayerExtends{
			Parent: "base",
		},
	}

	assert.Equal(t, "test", il.Layer.Name.String())
	assert.Equal(t, "base", il.Extends.Parent)
}
