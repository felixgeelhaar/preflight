package deps

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveToolDependency_ExplicitBrew(t *testing.T) {
	cfg := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"node"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	dep, ok := ResolveToolDependency(ctx, plat, ToolNode)
	require.True(t, ok)
	require.True(t, dep.Explicit)
	require.Equal(t, "brew", dep.Manager)
	require.Equal(t, "brew:formula:node", dep.StepID.String())
}

func TestResolveToolBootstrap_ImplicitBrew(t *testing.T) {
	cfg := map[string]interface{}{
		"npm": map[string]interface{}{
			"packages": []interface{}{"eslint"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	boot, ok := ResolveToolBootstrap(ctx, plat, ToolNode)
	require.True(t, ok)
	require.Equal(t, ToolNode, boot.Tool)
	require.Equal(t, "brew", boot.Manager)
	require.Equal(t, "node", boot.PackageName)
	require.Equal(t, "bootstrap:tool:node", boot.StepID.String())
}

// --- ResolveToolDeps ---

func TestResolveToolDeps_NilConfig(t *testing.T) {
	t.Parallel()
	ctx := compiler.NewCompileContext(nil)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	result := ResolveToolDeps(ctx, plat, ToolNode)
	assert.Nil(t, result)
}

func TestResolveToolDeps_ReturnsStepID(t *testing.T) {
	t.Parallel()
	cfg := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"python@3.12"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	result := ResolveToolDeps(ctx, plat, ToolPython)
	require.Len(t, result, 1)
	assert.Equal(t, "brew:formula:python@3.12", result[0].String())
}

func TestResolveToolDeps_UnknownTool(t *testing.T) {
	t.Parallel()
	cfg := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"node"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	result := ResolveToolDeps(ctx, plat, Tool("unknown"))
	assert.Nil(t, result)
}

// --- ResolveToolDependency ---

func TestResolveToolDependency_NilConfig(t *testing.T) {
	t.Parallel()
	ctx := compiler.NewCompileContext(nil)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	_, ok := ResolveToolDependency(ctx, plat, ToolNode)
	assert.False(t, ok)
}

func TestResolveToolDependency_AllToolsBrew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tool     Tool
		formula  string
		expected string
	}{
		{ToolNode, "node", "brew:formula:node"},
		{ToolPython, "python", "brew:formula:python"},
		{ToolRuby, "ruby", "brew:formula:ruby"},
		{ToolGo, "go", "brew:formula:go"},
		{ToolRust, "rust", "brew:formula:rust"},
	}

	for _, tt := range tests {
		t.Run(string(tt.tool), func(t *testing.T) {
			t.Parallel()
			cfg := map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{tt.formula},
				},
			}
			ctx := compiler.NewCompileContext(cfg)
			plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

			dep, ok := ResolveToolDependency(ctx, plat, tt.tool)
			require.True(t, ok)
			assert.True(t, dep.Explicit)
			assert.Equal(t, "brew", dep.Manager)
			assert.Equal(t, tt.expected, dep.StepID.String())
		})
	}
}

func TestResolveToolDependency_BrewMapEntry(t *testing.T) {
	t.Parallel()
	cfg := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{
				map[string]interface{}{"name": "python", "tap": "homebrew/core"},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	dep, ok := ResolveToolDependency(ctx, plat, ToolPython)
	require.True(t, ok)
	assert.Equal(t, "brew:formula:homebrew/core/python", dep.StepID.String())
}

func TestResolveToolDependency_BrewPrefixMatch(t *testing.T) {
	t.Parallel()
	cfg := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"python@3.12"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	dep, ok := ResolveToolDependency(ctx, plat, ToolPython)
	require.True(t, ok)
	assert.Equal(t, "brew:formula:python@3.12", dep.StepID.String())
}

// --- ResolveToolBootstrap ---

func TestResolveToolBootstrap_NilConfig(t *testing.T) {
	t.Parallel()
	ctx := compiler.NewCompileContext(nil)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	_, ok := ResolveToolBootstrap(ctx, plat, ToolNode)
	assert.False(t, ok)
}

func TestResolveToolBootstrap_ExplicitReturnsNothing(t *testing.T) {
	t.Parallel()
	cfg := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"node"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)

	_, ok := ResolveToolBootstrap(ctx, plat, ToolNode)
	assert.False(t, ok)
}

// --- findAptTool ---

func TestFindAptTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		section  map[string]interface{}
		tool     Tool
		expected string
	}{
		{
			name:     "nil section",
			section:  nil,
			tool:     ToolNode,
			expected: "",
		},
		{
			name:     "no packages key",
			section:  map[string]interface{}{},
			tool:     ToolNode,
			expected: "",
		},
		{
			name: "string entry match",
			section: map[string]interface{}{
				"packages": []interface{}{"nodejs"},
			},
			tool:     ToolNode,
			expected: "apt:package:nodejs",
		},
		{
			name: "map entry match",
			section: map[string]interface{}{
				"packages": []interface{}{
					map[string]interface{}{"name": "python3"},
				},
			},
			tool:     ToolPython,
			expected: "apt:package:python3",
		},
		{
			name: "no match",
			section: map[string]interface{}{
				"packages": []interface{}{"curl"},
			},
			tool:     ToolNode,
			expected: "",
		},
		{
			name: "all tools",
			section: map[string]interface{}{
				"packages": []interface{}{"golang-go"},
			},
			tool:     ToolGo,
			expected: "apt:package:golang-go",
		},
		{
			name: "rust via cargo",
			section: map[string]interface{}{
				"packages": []interface{}{"cargo"},
			},
			tool:     ToolRust,
			expected: "apt:package:cargo",
		},
		{
			name: "ruby",
			section: map[string]interface{}{
				"packages": []interface{}{"ruby-full"},
			},
			tool:     ToolRuby,
			expected: "apt:package:ruby-full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := findAptTool(tt.section, tt.tool)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- findChocoTool ---

func TestFindChocoTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		section  map[string]interface{}
		tool     Tool
		expected string
	}{
		{
			name:     "nil section",
			section:  nil,
			tool:     ToolNode,
			expected: "",
		},
		{
			name:     "no packages key",
			section:  map[string]interface{}{},
			tool:     ToolNode,
			expected: "",
		},
		{
			name: "string entry match",
			section: map[string]interface{}{
				"packages": []interface{}{"nodejs-lts"},
			},
			tool:     ToolNode,
			expected: "chocolatey:package:nodejs-lts",
		},
		{
			name: "map entry match",
			section: map[string]interface{}{
				"packages": []interface{}{
					map[string]interface{}{"name": "golang"},
				},
			},
			tool:     ToolGo,
			expected: "chocolatey:package:golang",
		},
		{
			name: "python",
			section: map[string]interface{}{
				"packages": []interface{}{"python"},
			},
			tool:     ToolPython,
			expected: "chocolatey:package:python",
		},
		{
			name: "ruby",
			section: map[string]interface{}{
				"packages": []interface{}{"ruby"},
			},
			tool:     ToolRuby,
			expected: "chocolatey:package:ruby",
		},
		{
			name: "rust",
			section: map[string]interface{}{
				"packages": []interface{}{"rustup"},
			},
			tool:     ToolRust,
			expected: "chocolatey:package:rustup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := findChocoTool(tt.section, tt.tool)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- findWingetTool ---

func TestFindWingetTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		section  map[string]interface{}
		tool     Tool
		expected string
	}{
		{
			name:     "nil section",
			section:  nil,
			tool:     ToolNode,
			expected: "",
		},
		{
			name:     "no packages key",
			section:  map[string]interface{}{},
			tool:     ToolNode,
			expected: "",
		},
		{
			name: "node by id string",
			section: map[string]interface{}{
				"packages": []interface{}{"OpenJS.NodeJS.LTS"},
			},
			tool:     ToolNode,
			expected: "winget:package:OpenJS.NodeJS.LTS",
		},
		{
			name: "node by id map",
			section: map[string]interface{}{
				"packages": []interface{}{
					map[string]interface{}{"id": "OpenJS.NodeJS.LTS"},
				},
			},
			tool:     ToolNode,
			expected: "winget:package:OpenJS.NodeJS.LTS",
		},
		{
			name: "python prefix match",
			section: map[string]interface{}{
				"packages": []interface{}{"Python.Python.3.12"},
			},
			tool:     ToolPython,
			expected: "winget:package:Python.Python.3.12",
		},
		{
			name: "go prefix match",
			section: map[string]interface{}{
				"packages": []interface{}{"GoLang.Go.1.22"},
			},
			tool:     ToolGo,
			expected: "winget:package:GoLang.Go.1.22",
		},
		{
			name: "ruby",
			section: map[string]interface{}{
				"packages": []interface{}{"RubyInstallerTeam.Ruby"},
			},
			tool:     ToolRuby,
			expected: "winget:package:RubyInstallerTeam.Ruby",
		},
		{
			name: "rust",
			section: map[string]interface{}{
				"packages": []interface{}{"Rustlang.Rustup"},
			},
			tool:     ToolRust,
			expected: "winget:package:Rustlang.Rustup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := findWingetTool(tt.section, tt.tool)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- findScoopTool ---

func TestFindScoopTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		section  map[string]interface{}
		tool     Tool
		expected string
	}{
		{
			name:     "nil section",
			section:  nil,
			tool:     ToolNode,
			expected: "",
		},
		{
			name:     "no packages key",
			section:  map[string]interface{}{},
			tool:     ToolNode,
			expected: "",
		},
		{
			name: "string entry match",
			section: map[string]interface{}{
				"packages": []interface{}{"nodejs-lts"},
			},
			tool:     ToolNode,
			expected: "scoop:package:nodejs-lts",
		},
		{
			name: "map entry with bucket",
			section: map[string]interface{}{
				"packages": []interface{}{
					map[string]interface{}{"name": "python", "bucket": "main"},
				},
			},
			tool:     ToolPython,
			expected: "scoop:package:main/python",
		},
		{
			name: "go",
			section: map[string]interface{}{
				"packages": []interface{}{"go"},
			},
			tool:     ToolGo,
			expected: "scoop:package:go",
		},
		{
			name: "rust",
			section: map[string]interface{}{
				"packages": []interface{}{"rustup"},
			},
			tool:     ToolRust,
			expected: "scoop:package:rustup",
		},
		{
			name: "ruby",
			section: map[string]interface{}{
				"packages": []interface{}{"ruby"},
			},
			tool:     ToolRuby,
			expected: "scoop:package:ruby",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := findScoopTool(tt.section, tt.tool)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- parseNamedEntry ---

func TestParseNamedEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entry    interface{}
		key      string
		expected string
	}{
		{name: "string entry", entry: "nodejs", key: "name", expected: "nodejs"},
		{name: "map with key", entry: map[string]interface{}{"name": "python3"}, key: "name", expected: "python3"},
		{name: "map with id key", entry: map[string]interface{}{"id": "Git.Git"}, key: "id", expected: "Git.Git"},
		{name: "map missing key", entry: map[string]interface{}{"other": "value"}, key: "name", expected: ""},
		{name: "nil entry", entry: nil, key: "name", expected: ""},
		{name: "int entry", entry: 42, key: "name", expected: ""},
		{name: "map with non-string value", entry: map[string]interface{}{"name": 42}, key: "name", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseNamedEntry(tt.entry, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- parseBrewFormula ---

func TestParseBrewFormula(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		entry        interface{}
		expectedName string
		expectedFull string
	}{
		{name: "simple string", entry: "node", expectedName: "node", expectedFull: "node"},
		{name: "with tap prefix", entry: "homebrew/core/python", expectedName: "python", expectedFull: "homebrew/core/python"},
		{name: "map with name", entry: map[string]interface{}{"name": "go"}, expectedName: "go", expectedFull: "go"},
		{name: "map with name and tap", entry: map[string]interface{}{"name": "rust", "tap": "homebrew/core"}, expectedName: "rust", expectedFull: "homebrew/core/rust"},
		{name: "map empty name", entry: map[string]interface{}{"name": ""}, expectedName: "", expectedFull: ""},
		{name: "map no name key", entry: map[string]interface{}{"other": "value"}, expectedName: "", expectedFull: ""},
		{name: "nil entry", entry: nil, expectedName: "", expectedFull: ""},
		{name: "int entry", entry: 42, expectedName: "", expectedFull: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name, full := parseBrewFormula(tt.entry)
			assert.Equal(t, tt.expectedName, name)
			assert.Equal(t, tt.expectedFull, full)
		})
	}
}

// --- parseScoopPackage ---

func TestParseScoopPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		entry        interface{}
		expectedName string
		expectedFull string
	}{
		{name: "simple string", entry: "python", expectedName: "python", expectedFull: "python"},
		{name: "map with name", entry: map[string]interface{}{"name": "go"}, expectedName: "go", expectedFull: "go"},
		{name: "map with bucket", entry: map[string]interface{}{"name": "ruby", "bucket": "extras"}, expectedName: "ruby", expectedFull: "extras/ruby"},
		{name: "map empty name", entry: map[string]interface{}{"name": ""}, expectedName: "", expectedFull: ""},
		{name: "map no name key", entry: map[string]interface{}{"other": "value"}, expectedName: "", expectedFull: ""},
		{name: "nil entry", entry: nil, expectedName: "", expectedFull: ""},
		{name: "int entry", entry: 42, expectedName: "", expectedFull: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name, full := parseScoopPackage(tt.entry)
			assert.Equal(t, tt.expectedName, name)
			assert.Equal(t, tt.expectedFull, full)
		})
	}
}

// --- matchName ---

func TestMatchName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		match    toolMatch
		expected bool
	}{
		{name: "exact match", value: "node", match: toolMatch{names: []string{"node"}}, expected: true},
		{name: "case insensitive", value: "Node", match: toolMatch{names: []string{"node"}}, expected: true},
		{name: "prefix match", value: "python@3.12", match: toolMatch{prefixes: []string{"python@"}}, expected: true},
		{name: "no match", value: "curl", match: toolMatch{names: []string{"node"}}, expected: false},
		{name: "empty match", value: "node", match: toolMatch{}, expected: false},
		{name: "multiple names", value: "rustup", match: toolMatch{names: []string{"rust", "rustup"}}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := matchName(tt.value, tt.match)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- preferredManagers ---

func TestPreferredManagers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plat     *platform.Platform
		expected []string
	}{
		{name: "nil platform", plat: nil, expected: []string{"brew", "apt", "winget", "chocolatey", "scoop"}},
		{name: "darwin", plat: platform.New(platform.OSDarwin, "amd64", platform.EnvNative), expected: []string{"brew"}},
		{name: "linux", plat: platform.New(platform.OSLinux, "amd64", platform.EnvNative), expected: []string{"apt"}},
		{name: "windows", plat: platform.New(platform.OSWindows, "amd64", platform.EnvNative), expected: []string{"winget", "chocolatey", "scoop"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := preferredManagers(tt.plat)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- implicitToolPackage ---

func TestImplicitToolPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manager  string
		tool     Tool
		expected string
		ok       bool
	}{
		{name: "brew node", manager: "brew", tool: ToolNode, expected: "node", ok: true},
		{name: "brew python", manager: "brew", tool: ToolPython, expected: "python", ok: true},
		{name: "apt nodejs", manager: "apt", tool: ToolNode, expected: "nodejs", ok: true},
		{name: "apt python3", manager: "apt", tool: ToolPython, expected: "python3", ok: true},
		{name: "apt go", manager: "apt", tool: ToolGo, expected: "golang-go", ok: true},
		{name: "winget node", manager: "winget", tool: ToolNode, expected: "OpenJS.NodeJS.LTS", ok: true},
		{name: "winget rust", manager: "winget", tool: ToolRust, expected: "Rustlang.Rustup", ok: true},
		{name: "choco node", manager: "chocolatey", tool: ToolNode, expected: "nodejs-lts", ok: true},
		{name: "choco go", manager: "chocolatey", tool: ToolGo, expected: "golang", ok: true},
		{name: "scoop node", manager: "scoop", tool: ToolNode, expected: "nodejs-lts", ok: true},
		{name: "scoop rust", manager: "scoop", tool: ToolRust, expected: "rustup", ok: true},
		{name: "unknown manager", manager: "unknown", tool: ToolNode, expected: "", ok: false},
		{name: "unknown tool", manager: "brew", tool: Tool("unknown"), expected: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pkg, ok := implicitToolPackage(tt.manager, tt.tool)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, pkg)
		})
	}
}

// --- bootstrapToolStepID ---

func TestBootstrapToolStepID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "bootstrap:tool:node", bootstrapToolStepID(ToolNode))
	assert.Equal(t, "bootstrap:tool:python", bootstrapToolStepID(ToolPython))
	assert.Equal(t, "bootstrap:tool:go", bootstrapToolStepID(ToolGo))
}

// --- toolMatchForManager ---

func TestToolMatchForManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		manager string
		tool    Tool
		empty   bool
	}{
		{name: "brew node", manager: "brew", tool: ToolNode, empty: false},
		{name: "apt python", manager: "apt", tool: ToolPython, empty: false},
		{name: "choco go", manager: "chocolatey", tool: ToolGo, empty: false},
		{name: "winget rust", manager: "winget", tool: ToolRust, empty: false},
		{name: "scoop ruby", manager: "scoop", tool: ToolRuby, empty: false},
		{name: "unknown manager", manager: "unknown", tool: ToolNode, empty: true},
		{name: "brew unknown tool", manager: "brew", tool: Tool("unknown"), empty: true},
		{name: "apt unknown tool", manager: "apt", tool: Tool("unknown"), empty: true},
		{name: "choco unknown tool", manager: "chocolatey", tool: Tool("unknown"), empty: true},
		{name: "winget unknown tool", manager: "winget", tool: Tool("unknown"), empty: true},
		{name: "scoop unknown tool", manager: "scoop", tool: Tool("unknown"), empty: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			match := toolMatchForManager(tt.tool, tt.manager)
			if tt.empty {
				assert.Empty(t, match.names)
				assert.Empty(t, match.prefixes)
			} else {
				assert.True(t, len(match.names) > 0 || len(match.prefixes) > 0)
			}
		})
	}
}

// --- managerCommands ---

func TestManagerCommands(t *testing.T) {
	t.Parallel()

	nativePlat := platform.New(platform.OSDarwin, "amd64", platform.EnvNative)
	wslPlat := platform.New(platform.OSLinux, "amd64", platform.EnvWSL2)

	tests := []struct {
		name     string
		plat     *platform.Platform
		manager  string
		expected []string
	}{
		{name: "brew", plat: nativePlat, manager: "brew", expected: []string{"brew"}},
		{name: "apt", plat: nativePlat, manager: "apt", expected: []string{"apt-get"}},
		{name: "winget native", plat: nativePlat, manager: "winget", expected: []string{"winget"}},
		{name: "winget wsl", plat: wslPlat, manager: "winget", expected: []string{"winget.exe", "winget"}},
		{name: "choco native", plat: nativePlat, manager: "chocolatey", expected: []string{"choco"}},
		{name: "choco wsl", plat: wslPlat, manager: "chocolatey", expected: []string{"choco.exe", "choco"}},
		{name: "scoop native", plat: nativePlat, manager: "scoop", expected: []string{"scoop"}},
		{name: "scoop wsl", plat: wslPlat, manager: "scoop", expected: []string{"scoop.cmd", "scoop"}},
		{name: "unknown", plat: nativePlat, manager: "unknown", expected: nil},
		{name: "nil plat winget", plat: nil, manager: "winget", expected: []string{"winget"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := managerCommands(tt.plat, tt.manager)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- findToolInstaller (dispatcher) ---

func TestFindToolInstaller_UnknownManager(t *testing.T) {
	t.Parallel()
	cfg := map[string]interface{}{}
	ctx := compiler.NewCompileContext(cfg)

	result := findToolInstaller(ctx, "unknown", ToolNode)
	assert.Equal(t, "", result)
}

// --- Integration: apt explicit resolution ---

func TestResolveToolDependency_ExplicitApt(t *testing.T) {
	t.Parallel()
	cfg := map[string]interface{}{
		"apt": map[string]interface{}{
			"packages": []interface{}{"nodejs"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSLinux, "amd64", platform.EnvNative)

	dep, ok := ResolveToolDependency(ctx, plat, ToolNode)
	require.True(t, ok)
	assert.True(t, dep.Explicit)
	assert.Equal(t, "apt", dep.Manager)
	assert.Equal(t, "apt:package:nodejs", dep.StepID.String())
}

// --- Integration: winget explicit resolution ---

func TestResolveToolDependency_ExplicitWinget(t *testing.T) {
	t.Parallel()
	cfg := map[string]interface{}{
		"winget": map[string]interface{}{
			"packages": []interface{}{
				map[string]interface{}{"id": "OpenJS.NodeJS.LTS"},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)

	dep, ok := ResolveToolDependency(ctx, plat, ToolNode)
	require.True(t, ok)
	assert.True(t, dep.Explicit)
	assert.Equal(t, "winget", dep.Manager)
	assert.Equal(t, "winget:package:OpenJS.NodeJS.LTS", dep.StepID.String())
}

// --- Integration: chocolatey explicit resolution ---

func TestResolveToolDependency_ExplicitChocolatey(t *testing.T) {
	t.Parallel()
	cfg := map[string]interface{}{
		"chocolatey": map[string]interface{}{
			"packages": []interface{}{"golang"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)

	dep, ok := ResolveToolDependency(ctx, plat, ToolGo)
	require.True(t, ok)
	assert.True(t, dep.Explicit)
	assert.Equal(t, "chocolatey", dep.Manager)
}

// --- Integration: scoop explicit resolution ---

func TestResolveToolDependency_ExplicitScoop(t *testing.T) {
	t.Parallel()
	cfg := map[string]interface{}{
		"scoop": map[string]interface{}{
			"packages": []interface{}{"rustup"},
		},
	}
	ctx := compiler.NewCompileContext(cfg)
	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)

	dep, ok := ResolveToolDependency(ctx, plat, ToolRust)
	require.True(t, ok)
	assert.True(t, dep.Explicit)
	assert.Equal(t, "scoop", dep.Manager)
	assert.Equal(t, "scoop:package:rustup", dep.StepID.String())
}
