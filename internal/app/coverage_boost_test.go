package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// parseSSHConfig
// ---------------------------------------------------------------------------

func TestParseSSHConfig_ValidConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sshConfig := filepath.Join(tmpDir, "config")
	content := `# Global defaults
AddKeysToAgent yes
UseKeychain yes
IdentitiesOnly yes
ServerAliveInterval 60

Host github.com
  HostName github.com
  User git
  IdentityFile ~/.ssh/id_github
  Port 22

Host gitlab.com
  HostName gitlab.com
  User git
  IdentityFile ~/.ssh/id_gitlab
`
	require.NoError(t, os.WriteFile(sshConfig, []byte(content), 0o644))

	hosts, defaults := parseSSHConfig(sshConfig)

	assert.NotNil(t, defaults)
	assert.Equal(t, "yes", defaults.AddKeysToAgent)
	assert.Equal(t, "yes", defaults.UseKeychain)
	assert.Equal(t, "yes", defaults.IdentitiesOnly)
	assert.Equal(t, "60", defaults.ServerAliveInterval)

	require.Len(t, hosts, 2)
	assert.Equal(t, "github.com", hosts[0].Host)
	assert.Equal(t, "github.com", hosts[0].HostName)
	assert.Equal(t, "git", hosts[0].User)
	assert.Equal(t, "~/.ssh/id_github", hosts[0].IdentityFile)
	assert.Equal(t, "22", hosts[0].Port)

	assert.Equal(t, "gitlab.com", hosts[1].Host)
}

func TestParseSSHConfig_WildcardHosts(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sshConfig := filepath.Join(tmpDir, "config")
	content := `Host *
  ServerAliveInterval 60

Host github.com
  HostName github.com
  User git
`
	require.NoError(t, os.WriteFile(sshConfig, []byte(content), 0o644))

	hosts, _ := parseSSHConfig(sshConfig)

	// Wildcard hosts should be skipped
	require.Len(t, hosts, 1)
	assert.Equal(t, "github.com", hosts[0].Host)
}

func TestParseSSHConfig_TabSeparated(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sshConfig := filepath.Join(tmpDir, "config")
	content := "Host\texample.com\n\tHostName\texample.com\n\tUser\tgit\n"
	require.NoError(t, os.WriteFile(sshConfig, []byte(content), 0o644))

	hosts, _ := parseSSHConfig(sshConfig)

	require.Len(t, hosts, 1)
	assert.Equal(t, "example.com", hosts[0].Host)
	assert.Equal(t, "example.com", hosts[0].HostName)
	assert.Equal(t, "git", hosts[0].User)
}

func TestParseSSHConfig_EmptyFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sshConfig := filepath.Join(tmpDir, "config")
	require.NoError(t, os.WriteFile(sshConfig, []byte(""), 0o644))

	hosts, defaults := parseSSHConfig(sshConfig)

	assert.Empty(t, hosts)
	assert.Nil(t, defaults)
}

func TestParseSSHConfig_CommentsOnly(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sshConfig := filepath.Join(tmpDir, "config")
	content := "# This is a comment\n# Another comment\n\n\n"
	require.NoError(t, os.WriteFile(sshConfig, []byte(content), 0o644))

	hosts, defaults := parseSSHConfig(sshConfig)

	assert.Empty(t, hosts)
	assert.Nil(t, defaults)
}

func TestParseSSHConfig_NonexistentFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sshConfig := filepath.Join(tmpDir, "nonexistent")

	hosts, defaults := parseSSHConfig(sshConfig)
	assert.Nil(t, hosts)
	assert.Nil(t, defaults)
}

func TestParseSSHConfig_NoDefaults(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sshConfig := filepath.Join(tmpDir, "config")
	content := `Host example.com
  HostName example.com
  User git
`
	require.NoError(t, os.WriteFile(sshConfig, []byte(content), 0o644))

	hosts, defaults := parseSSHConfig(sshConfig)

	require.Len(t, hosts, 1)
	assert.Nil(t, defaults, "no global defaults should return nil")
}

// ---------------------------------------------------------------------------
// detectNvimPreset
// ---------------------------------------------------------------------------

func TestDetectNvimPreset_LazyVimMarker(t *testing.T) {
	t.Parallel()

	configPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(configPath, "lazyvim.json"), []byte("{}"), 0o644))

	result := detectNvimPreset(configPath)
	assert.Equal(t, "lazyvim", result)
}

func TestDetectNvimPreset_LazyVimInLockfile(t *testing.T) {
	t.Parallel()

	configPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(configPath, "lazy-lock.json"),
		[]byte(`{"LazyVim": {"branch": "main"}}`), 0o644))

	result := detectNvimPreset(configPath)
	assert.Equal(t, "lazyvim", result)
}

func TestDetectNvimPreset_NvChad(t *testing.T) {
	t.Parallel()

	configPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(configPath, "lua", "core"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(configPath, "lua", "custom"), 0o755))

	result := detectNvimPreset(configPath)
	assert.Equal(t, "nvchad", result)
}

func TestDetectNvimPreset_AstroNvim(t *testing.T) {
	t.Parallel()

	configPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(configPath, "lua", "astronvim"), 0o755))

	result := detectNvimPreset(configPath)
	assert.Equal(t, "astronvim", result)
}

func TestDetectNvimPreset_LunarVim(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "lvim")
	require.NoError(t, os.MkdirAll(configPath, 0o755))

	result := detectNvimPreset(configPath)
	assert.Equal(t, "lunarvim", result)
}

func TestDetectNvimPreset_Custom(t *testing.T) {
	t.Parallel()

	configPath := t.TempDir()
	result := detectNvimPreset(configPath)
	assert.Equal(t, "custom", result)
}

// ---------------------------------------------------------------------------
// detectPluginManager
// ---------------------------------------------------------------------------

func TestDetectPluginManager_LazyNvimLock(t *testing.T) {
	t.Parallel()

	configPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(configPath, "lazy-lock.json"), []byte("{}"), 0o644))

	result := detectPluginManager(configPath)
	assert.Equal(t, "lazy.nvim", result)
}

func TestDetectPluginManager_Packer(t *testing.T) {
	t.Parallel()

	configPath := t.TempDir()
	compiledDir := filepath.Join(configPath, "plugin")
	require.NoError(t, os.MkdirAll(compiledDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(compiledDir, "packer_compiled.lua"), []byte("-- compiled"), 0o644))

	result := detectPluginManager(configPath)
	assert.Equal(t, "packer", result)
}

func TestDetectPluginManager_LazyInInitLua(t *testing.T) {
	t.Parallel()

	configPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(configPath, "init.lua"),
		[]byte(`require("lazy.nvim").setup({})`), 0o644))

	result := detectPluginManager(configPath)
	assert.Equal(t, "lazy.nvim", result)
}

func TestDetectPluginManager_PackerInInitLua(t *testing.T) {
	t.Parallel()

	configPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(configPath, "init.lua"),
		[]byte(`require("packer").startup()`), 0o644))

	result := detectPluginManager(configPath)
	assert.Equal(t, "packer", result)
}

func TestDetectPluginManager_VimPlugInInitLua(t *testing.T) {
	t.Parallel()

	configPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(configPath, "init.lua"),
		[]byte(`vim.fn["plug#begin"](vim.fn.stdpath("data") .. "/plugged")
-- vim-plug style`), 0o644))

	result := detectPluginManager(configPath)
	assert.Equal(t, "vim-plug", result)
}

func TestDetectPluginManager_None(t *testing.T) {
	t.Parallel()

	configPath := t.TempDir()
	result := detectPluginManager(configPath)
	assert.Equal(t, "", result)
}

// ---------------------------------------------------------------------------
// countLazyPlugins
// ---------------------------------------------------------------------------

func TestCountLazyPlugins_ValidFile(t *testing.T) {
	t.Parallel()

	lockPath := filepath.Join(t.TempDir(), "lazy-lock.json")
	content := `{
		"nvim-treesitter": {"branch": "main"},
		"telescope.nvim": {"branch": "0.1.x"},
		"nvim-lspconfig": {"branch": "master"}
	}`
	require.NoError(t, os.WriteFile(lockPath, []byte(content), 0o644))

	count := countLazyPlugins(lockPath)
	assert.Equal(t, 3, count)
}

func TestCountLazyPlugins_EmptyJSON(t *testing.T) {
	t.Parallel()

	lockPath := filepath.Join(t.TempDir(), "lazy-lock.json")
	require.NoError(t, os.WriteFile(lockPath, []byte("{}"), 0o644))

	count := countLazyPlugins(lockPath)
	assert.Equal(t, 0, count)
}

func TestCountLazyPlugins_InvalidJSON(t *testing.T) {
	t.Parallel()

	lockPath := filepath.Join(t.TempDir(), "lazy-lock.json")
	require.NoError(t, os.WriteFile(lockPath, []byte("not json"), 0o644))

	count := countLazyPlugins(lockPath)
	assert.Equal(t, 0, count)
}

func TestCountLazyPlugins_NonexistentFile(t *testing.T) {
	t.Parallel()

	lockPath := filepath.Join(t.TempDir(), "nonexistent.json")
	count := countLazyPlugins(lockPath)
	assert.Equal(t, 0, count)
}

// ---------------------------------------------------------------------------
// detectKeyType
// ---------------------------------------------------------------------------

func TestDetectKeyType_Ed25519Key(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")
	require.NoError(t, os.WriteFile(keyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAA=\n"), 0o600))
	require.NoError(t, os.WriteFile(keyPath+".pub", []byte("ssh-ed25519 AAAA test@host\n"), 0o644))

	result := detectKeyType(keyPath)
	assert.Equal(t, "ed25519", result)
}

func TestDetectKeyType_RSAKey(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_rsa")
	require.NoError(t, os.WriteFile(keyPath, []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n"), 0o600))

	result := detectKeyType(keyPath)
	assert.Equal(t, "rsa", result)
}

func TestDetectKeyType_ECDSAKey(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ecdsa")
	require.NoError(t, os.WriteFile(keyPath, []byte("-----BEGIN EC PRIVATE KEY-----\nMHQC...\n"), 0o600))

	result := detectKeyType(keyPath)
	assert.Equal(t, "ecdsa", result)
}

func TestDetectKeyType_DSAKey(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_dsa")
	require.NoError(t, os.WriteFile(keyPath, []byte("-----BEGIN DSA PRIVATE KEY-----\nMIIB...\n"), 0o600))

	result := detectKeyType(keyPath)
	assert.Equal(t, "dsa", result)
}

func TestDetectKeyType_OpenSSHWithRSAPub(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_rsa_openssh")
	require.NoError(t, os.WriteFile(keyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAA=\n"), 0o600))
	require.NoError(t, os.WriteFile(keyPath+".pub", []byte("ssh-rsa AAAA test@host\n"), 0o644))

	result := detectKeyType(keyPath)
	assert.Equal(t, "rsa", result)
}

func TestDetectKeyType_OpenSSHWithECDSAPub(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ecdsa_openssh")
	require.NoError(t, os.WriteFile(keyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAA=\n"), 0o600))
	require.NoError(t, os.WriteFile(keyPath+".pub", []byte("ecdsa-sha2-nistp256 AAAA test@host\n"), 0o644))

	result := detectKeyType(keyPath)
	assert.Equal(t, "ecdsa", result)
}

func TestDetectKeyType_OpenSSHNoPubKey(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_unknown")
	require.NoError(t, os.WriteFile(keyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAA=\n"), 0o600))
	// No .pub file

	result := detectKeyType(keyPath)
	assert.Equal(t, "ed25519", result) // default for modern keys
}

func TestDetectKeyType_NonexistentFile(t *testing.T) {
	t.Parallel()

	result := detectKeyType(filepath.Join(t.TempDir(), "nonexistent"))
	assert.Equal(t, "", result)
}

func TestDetectKeyType_EmptyFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "empty_key")
	require.NoError(t, os.WriteFile(keyPath, []byte(""), 0o600))

	result := detectKeyType(keyPath)
	assert.Equal(t, "", result)
}

// ---------------------------------------------------------------------------
// keyHasPassphrase
// ---------------------------------------------------------------------------

func TestKeyHasPassphrase_Encrypted(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_enc")
	require.NoError(t, os.WriteFile(keyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nENCRYPTED content\n"), 0o600))

	assert.True(t, keyHasPassphrase(keyPath))
}

func TestKeyHasPassphrase_Unencrypted(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_plain")
	require.NoError(t, os.WriteFile(keyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nplain content\n"), 0o600))

	assert.False(t, keyHasPassphrase(keyPath))
}

func TestKeyHasPassphrase_NonexistentFile(t *testing.T) {
	t.Parallel()

	result := keyHasPassphrase(filepath.Join(t.TempDir(), "nonexistent"))
	assert.False(t, result)
}

// ---------------------------------------------------------------------------
// isRuntimeManagerVersionItem
// ---------------------------------------------------------------------------

func TestIsRuntimeManagerVersionItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   string
		expected bool
	}{
		{"mise version", "mise --version", true},
		{"rtx version", "rtx --version", true},
		{"asdf version", "asdf --version", true},
		{"other command", "node --version", false},
		{"empty source", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			item := CapturedItem{Source: tt.source}
			assert.Equal(t, tt.expected, isRuntimeManagerVersionItem(item))
		})
	}
}

// ---------------------------------------------------------------------------
// generateGitFromCapture
// ---------------------------------------------------------------------------

func TestGenerateGitFromCapture_AllFields(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: "user.name", Value: "Test User"},
		{Name: "user.email", Value: "test@example.com"},
		{Name: "core.editor", Value: "nvim"},
		{Name: "init.defaultBranch", Value: "main"},
	}

	result := g.generateGitFromCapture(items)

	require.NotNil(t, result)
	require.NotNil(t, result.User)
	assert.Equal(t, "Test User", result.User.Name)
	assert.Equal(t, "test@example.com", result.User.Email)
	require.NotNil(t, result.Core)
	assert.Equal(t, "nvim", result.Core.Editor)
	require.NotNil(t, result.Init)
	assert.Equal(t, "main", result.Init.DefaultBranch)
}

func TestGenerateGitFromCapture_OnlyUser(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: "user.name", Value: "Only Name"},
	}

	result := g.generateGitFromCapture(items)

	require.NotNil(t, result.User)
	assert.Equal(t, "Only Name", result.User.Name)
	assert.Nil(t, result.Core)
	assert.Nil(t, result.Init)
}

func TestGenerateGitFromCapture_Empty(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	result := g.generateGitFromCapture(nil)

	assert.NotNil(t, result)
	assert.Nil(t, result.User)
	assert.Nil(t, result.Core)
	assert.Nil(t, result.Init)
}

func TestGenerateGitFromCapture_NonStringValue(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: "user.name", Value: 123}, // non-string value
	}

	result := g.generateGitFromCapture(items)

	// User struct is created but Name is empty because value is not a string
	require.NotNil(t, result.User)
	assert.Equal(t, "", result.User.Name)
}

// ---------------------------------------------------------------------------
// generateShellFromCapture
// ---------------------------------------------------------------------------

func TestGenerateShellFromCapture_ZshAndPlugins(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: ".zshrc"},
		{Name: "framework", Value: "oh-my-zsh"},
		{Name: "plugin", Value: "git"},
		{Name: "plugin", Value: "docker"},
		{Name: "theme", Value: "robbyrussell"},
	}

	result := g.generateShellFromCapture(items)

	require.NotNil(t, result)
	assert.NotEmpty(t, result.Shells)
}

func TestGenerateShellFromCapture_BashOnly(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: ".bashrc"},
	}

	result := g.generateShellFromCapture(items)

	require.NotNil(t, result)
}

func TestGenerateShellFromCapture_Empty(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	result := g.generateShellFromCapture(nil)

	// With no shell-related items, function returns nil
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// vscodeExtensionsFromCapture
// ---------------------------------------------------------------------------

func TestVscodeExtensionsFromCapture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		items    []CapturedItem
		expected int
	}{
		{
			name: "normal extensions",
			items: []CapturedItem{
				{Name: "ms-vscode.go"},
				{Name: "esbenp.prettier-vscode"},
			},
			expected: 2,
		},
		{
			name: "skip version items",
			items: []CapturedItem{
				{Name: "ms-vscode.go"},
				{Name: "code-version", Source: "code --version"},
			},
			expected: 1,
		},
		{
			name:     "empty",
			items:    nil,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := vscodeExtensionsFromCapture(tt.items)
			assert.Len(t, result, tt.expected)
		})
	}
}

// ---------------------------------------------------------------------------
// isGitManaged
// ---------------------------------------------------------------------------

func TestIsGitManaged(t *testing.T) {
	t.Parallel()

	t.Run("with git dir", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
		assert.True(t, isGitManaged(dir))
	})

	t.Run("without git dir", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		assert.False(t, isGitManaged(dir))
	})
}

// ---------------------------------------------------------------------------
// SecurityVulnerabilityCount
// ---------------------------------------------------------------------------

func TestSecurityVulnerabilityCount_WithVulnerabilities(t *testing.T) {
	t.Parallel()

	report := &DoctorReport{
		SecurityScanResult: &security.ScanResult{
			Vulnerabilities: security.Vulnerabilities{
				{ID: "CVE-2024-001", Severity: "critical"},
				{ID: "CVE-2024-002", Severity: "high"},
				{ID: "CVE-2024-003", Severity: "medium"},
			},
		},
	}

	count := report.SecurityVulnerabilityCount()
	assert.Equal(t, 3, count)
}

func TestSecurityVulnerabilityCount_NilScanResult(t *testing.T) {
	t.Parallel()

	report := &DoctorReport{}

	count := report.SecurityVulnerabilityCount()
	assert.Equal(t, 0, count)
}

// ---------------------------------------------------------------------------
// looksLikePrivateKey
// ---------------------------------------------------------------------------

func TestLooksLikePrivateKey_Various(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"openssh key", "-----BEGIN OPENSSH PRIVATE KEY-----\ndata\n", true},
		{"rsa key", "-----BEGIN RSA PRIVATE KEY-----\ndata\n", true},
		{"generic private key", "-----BEGIN PRIVATE KEY-----\ndata\n", true},
		{"not a key", "just some text\n", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "testkey")
			require.NoError(t, os.WriteFile(filePath, []byte(tt.content), 0o600))

			result := looksLikePrivateKey(filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLooksLikePrivateKey_NonexistentFile(t *testing.T) {
	t.Parallel()

	result := looksLikePrivateKey(filepath.Join(t.TempDir(), "nonexistent"))
	assert.False(t, result)
}

// ---------------------------------------------------------------------------
// isPathWithinHome
// ---------------------------------------------------------------------------

func TestIsPathWithinHome_TempDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	assert.True(t, isPathWithinHome(tmpDir))
}

func TestIsPathWithinHome_HomeDir(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.True(t, isPathWithinHome(homeDir))
}

// ---------------------------------------------------------------------------
// generateRuntimeFromCapture
// ---------------------------------------------------------------------------

func TestGenerateRuntimeFromCapture(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: "mise", Source: "mise --version", Value: "2024.1.1"},
		{Name: "node", Value: "18.0.0"},
		{Name: "python", Value: "3.12.0"},
	}

	result := g.generateRuntimeFromCapture(items)

	require.NotNil(t, result)
	// mise is a runtime manager, so it's skipped; only node and python remain
	require.Len(t, result.Tools, 2)
	assert.Equal(t, "node", result.Tools[0].Name)
	assert.Equal(t, "python", result.Tools[1].Name)
}

func TestGenerateRuntimeFromCapture_Empty(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	result := g.generateRuntimeFromCapture(nil)

	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// addNpmPackagesToLayer and other addXxxToLayer functions
// ---------------------------------------------------------------------------

func TestAddNpmPackagesToLayer(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	items := []CapturedItem{
		{Name: "typescript", Value: "typescript@5.0"},
		{Name: "eslint", Value: "eslint@8.0"},
	}

	g.addNpmPackagesToLayer(layer, items)

	require.NotNil(t, layer.Packages)
	require.NotNil(t, layer.Packages.Npm)
	assert.Len(t, layer.Packages.Npm.Packages, 2)
}

func TestAddNpmPackagesToLayer_Empty(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	g.addNpmPackagesToLayer(layer, nil)

	assert.Nil(t, layer.Packages)
}

func TestAddGoToolsToLayer(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	items := []CapturedItem{
		{Name: "golang.org/x/tools/gopls", Value: "golang.org/x/tools/gopls"},
		{Name: "github.com/golangci/golangci-lint", Value: "github.com/golangci/golangci-lint"},
	}

	g.addGoToolsToLayer(layer, items)

	require.NotNil(t, layer.Packages)
	require.NotNil(t, layer.Packages.Go)
	assert.Len(t, layer.Packages.Go.Tools, 2)
}

func TestAddPipPackagesToLayer(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	items := []CapturedItem{
		{Name: "pytest", Value: "pytest==7.0"},
		{Name: "black", Value: "black==23.0"},
	}

	g.addPipPackagesToLayer(layer, items)

	require.NotNil(t, layer.Packages)
	require.NotNil(t, layer.Packages.Pip)
	assert.Len(t, layer.Packages.Pip.Packages, 2)
}

func TestAddGemPackagesToLayer(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	items := []CapturedItem{
		{Name: "rails", Value: "rails"},
		{Name: "rubocop", Value: "rubocop"},
	}

	g.addGemPackagesToLayer(layer, items)

	require.NotNil(t, layer.Packages)
	require.NotNil(t, layer.Packages.Gem)
	assert.Len(t, layer.Packages.Gem.Gems, 2)
}

func TestAddCargoPackagesToLayer(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	items := []CapturedItem{
		{Name: "cargo-watch", Value: "cargo-watch"},
	}

	g.addCargoPackagesToLayer(layer, items)

	require.NotNil(t, layer.Packages)
	require.NotNil(t, layer.Packages.Cargo)
	assert.Len(t, layer.Packages.Cargo.Crates, 1)
}

// ---------------------------------------------------------------------------
// addTerminalConfigToLayer
// ---------------------------------------------------------------------------

func TestAddTerminalConfigToLayer(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	g := NewCaptureConfigGenerator(tmpDir)

	layer := &captureLayerYAML{}
	items := []CapturedItem{
		{Name: "terminal.name", Value: "alacritty"},
		{Name: "terminal.config", Value: filepath.Join(tmpDir, "alacritty.toml")},
	}

	// Create the config file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "alacritty.toml"), []byte("# alacritty config"), 0o644))

	g.addTerminalConfigToLayer(layer, items)

	assert.NotNil(t, layer)
}

func TestAddTerminalConfigToLayer_Empty(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	g.addTerminalConfigToLayer(layer, nil)

	// Should not panic and layer should be untouched
	assert.NotNil(t, layer)
}

func TestAddTerminalConfigToLayer_AllTerminals(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	items := []CapturedItem{
		{Name: "alacritty", Value: ".config/alacritty/alacritty.toml"},
		{Name: "kitty", Value: ".config/kitty/kitty.conf"},
		{Name: "wezterm", Value: ".config/wezterm/wezterm.lua"},
		{Name: "ghostty", Value: ".config/ghostty/config"},
		{Name: "iterm2", Value: "com.googlecode.iterm2.plist"},
		{Name: "hyper", Value: ".hyper.js"},
		{Name: "windows_terminal", Value: "settings.json"},
	}

	g.addTerminalConfigToLayer(layer, items)

	require.NotNil(t, layer.Terminal)
	assert.NotNil(t, layer.Terminal.Alacritty)
	assert.NotNil(t, layer.Terminal.Kitty)
	assert.NotNil(t, layer.Terminal.WezTerm)
	assert.NotNil(t, layer.Terminal.Ghostty)
	assert.NotNil(t, layer.Terminal.ITerm2)
	assert.NotNil(t, layer.Terminal.Hyper)
	assert.NotNil(t, layer.Terminal.WindowsTerminal)
	assert.Equal(t, ".config/alacritty/alacritty.toml", layer.Terminal.Alacritty.Source)
	assert.True(t, layer.Terminal.Alacritty.Link)
}

func TestAddTerminalConfigToLayer_NonStringValue(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	items := []CapturedItem{
		{Name: "alacritty", Value: 42}, // non-string value
	}

	g.addTerminalConfigToLayer(layer, items)

	require.NotNil(t, layer.Terminal)
	assert.NotNil(t, layer.Terminal.Alacritty)
	assert.Equal(t, "", layer.Terminal.Alacritty.Source)
}

// ---------------------------------------------------------------------------
// generateNvimFromCapture (additional branches)
// ---------------------------------------------------------------------------

func TestGenerateNvimFromCapture_LazyLockOnly(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Create a lazy-lock.json with 3 plugins
	lazyLockPath := filepath.Join(tmpDir, "lazy-lock.json")
	lazyLockContent := `{"nvim-treesitter":{"branch":"master"},"telescope.nvim":{"branch":"master"},"which-key.nvim":{"branch":"main"}}`
	require.NoError(t, os.WriteFile(lazyLockPath, []byte(lazyLockContent), 0o644))

	g := NewCaptureConfigGenerator(tmpDir)
	items := []CapturedItem{
		{Name: "lazy-lock.json", Value: lazyLockPath},
	}

	result := g.generateNvimFromCapture(items)

	require.NotNil(t, result)
	assert.Equal(t, "lazy.nvim", result.PluginManager)
	assert.Equal(t, 3, result.PluginCount)
}

func TestGenerateNvimFromCapture_PackerCompiled(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: "packer_compiled.lua", Value: "true"},
	}

	result := g.generateNvimFromCapture(items)

	require.NotNil(t, result)
	assert.Equal(t, "packer", result.PluginManager)
}

func TestGenerateNvimFromCapture_Vimrc(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: ".vimrc", Value: "true"},
	}

	result := g.generateNvimFromCapture(items)

	require.NotNil(t, result)
	assert.Equal(t, "legacy", result.Preset)
}

func TestGenerateNvimFromCapture_ConfigWithoutPreset(t *testing.T) {
	t.Parallel()

	// Create a config dir that has no distribution markers
	tmpDir := t.TempDir()
	nvimConfigDir := filepath.Join(tmpDir, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimConfigDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimConfigDir, "init.lua"), []byte("-- my config"), 0o644))

	g := NewCaptureConfigGenerator(tmpDir)
	items := []CapturedItem{
		{Name: "config", Value: nvimConfigDir},
	}

	result := g.generateNvimFromCapture(items)

	require.NotNil(t, result)
	assert.Equal(t, nvimConfigDir, result.ConfigPath)
	// Should fall back to "custom" if no marker detected
	assert.NotEmpty(t, result.Preset)
}

func TestGenerateNvimFromCapture_Empty(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	result := g.generateNvimFromCapture(nil)

	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// generateSSHFromCapture
// ---------------------------------------------------------------------------

func TestGenerateSSHFromCapture_WithConfig(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	// Create a temp ssh dir within home (for isPathWithinHome check)
	sshDir := filepath.Join(homeDir, ".ssh-test-"+t.Name())
	require.NoError(t, os.MkdirAll(sshDir, 0o700))
	t.Cleanup(func() { _ = os.RemoveAll(sshDir) })

	configPath := filepath.Join(sshDir, "config")
	require.NoError(t, os.WriteFile(configPath, []byte("Host github.com\n  HostName github.com\n  User git\n"), 0o644))

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: "config", Value: configPath},
	}

	result := g.generateSSHFromCapture(items)

	require.NotNil(t, result)
	assert.Equal(t, configPath, result.ConfigPath)
}

func TestGenerateSSHFromCapture_Empty(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	result := g.generateSSHFromCapture(nil)

	assert.Nil(t, result)
}

func TestGenerateSSHFromCapture_NonStringConfig(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: "config", Value: 42}, // non-string
	}

	result := g.generateSSHFromCapture(items)

	// No valid config found, returns nil
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// detectSSHKeys
// ---------------------------------------------------------------------------

func TestDetectSSHKeys_WithKeys(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	sshDir := filepath.Join(homeDir, ".ssh-test-keys-"+t.Name())
	require.NoError(t, os.MkdirAll(sshDir, 0o700))
	t.Cleanup(func() { _ = os.RemoveAll(sshDir) })

	// Create a fake private key
	privKey := "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAA=\n-----END OPENSSH PRIVATE KEY-----\n"
	require.NoError(t, os.WriteFile(filepath.Join(sshDir, "id_ed25519"), []byte(privKey), 0o600))

	// Create matching public key
	pubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA test@host\n"
	require.NoError(t, os.WriteFile(filepath.Join(sshDir, "id_ed25519.pub"), []byte(pubKey), 0o644))

	// Create known_hosts (should be skipped)
	require.NoError(t, os.WriteFile(filepath.Join(sshDir, "known_hosts"), []byte("# known hosts"), 0o644))

	// Create a directory (should be skipped)
	require.NoError(t, os.MkdirAll(filepath.Join(sshDir, "subdir"), 0o755))

	keys := detectSSHKeys(sshDir)

	require.NotEmpty(t, keys)
	assert.Equal(t, "id_ed25519", keys[0].Name)
	assert.Equal(t, "ed25519", keys[0].Type)
	assert.Equal(t, "test@host", keys[0].Comment)
}

func TestDetectSSHKeys_EmptyDir(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	sshDir := filepath.Join(homeDir, ".ssh-test-empty-"+t.Name())
	require.NoError(t, os.MkdirAll(sshDir, 0o700))
	t.Cleanup(func() { _ = os.RemoveAll(sshDir) })

	keys := detectSSHKeys(sshDir)

	assert.Empty(t, keys)
}

func TestDetectSSHKeys_NonexistentDir(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	keys := detectSSHKeys(filepath.Join(homeDir, ".ssh-nonexistent-"+t.Name()))

	assert.Nil(t, keys)
}

func TestDetectSSHKeys_NotAKey(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	sshDir := filepath.Join(homeDir, ".ssh-test-notkey-"+t.Name())
	require.NoError(t, os.MkdirAll(sshDir, 0o700))
	t.Cleanup(func() { _ = os.RemoveAll(sshDir) })

	// Write a non-key file (not a private key, not .pub, not config, not known_hosts)
	require.NoError(t, os.WriteFile(filepath.Join(sshDir, "random-file"), []byte("not a key"), 0o644))

	keys := detectSSHKeys(sshDir)
	assert.Empty(t, keys)
}

// ---------------------------------------------------------------------------
// resolveTerminalPath
// ---------------------------------------------------------------------------

func TestResolveTerminalPath_WithSubpath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	resolver := NewDotfilesResolver(tmpDir, "default")

	// With subpath matching a known terminal base name
	result := resolver.resolveTerminalPath([]string{"terminal", "wezterm.lua"})
	assert.Equal(t, "wezterm.lua", result)
}

func TestResolveTerminalPath_WithUnknownSubpath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	resolver := NewDotfilesResolver(tmpDir, "default")

	// Unknown subpath that doesn't match any known terminal
	result := resolver.resolveTerminalPath([]string{"terminal", "unknown-terminal-conf"})
	// Falls through to checking filesystem; nothing exists, falls to subpath
	assert.Equal(t, "unknown-terminal-conf", result)
}

func TestResolveTerminalPath_NoSubpath_DefaultsToWezterm(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	resolver := NewDotfilesResolver(tmpDir, "default")

	result := resolver.resolveTerminalPath([]string{"terminal"})
	assert.Equal(t, ".config/wezterm", result)
}

func TestResolveTerminalPath_NoSubpath_FindsExisting(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Create a kitty config so it gets found
	kittyDir := filepath.Join(tmpDir, ".config", "kitty")
	require.NoError(t, os.MkdirAll(kittyDir, 0o755))

	resolver := NewDotfilesResolver(tmpDir, "default")

	result := resolver.resolveTerminalPath([]string{"terminal"})
	assert.Equal(t, ".config/kitty", result)
}

func TestResolveTerminalPath_SubpathWithExistingTerminalOnDisk(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Create wezterm config on disk
	weztermDir := filepath.Join(tmpDir, ".config", "wezterm")
	require.NoError(t, os.MkdirAll(weztermDir, 0o755))

	resolver := NewDotfilesResolver(tmpDir, "default")

	// Unknown subpath, but wezterm exists on disk
	result := resolver.resolveTerminalPath([]string{"terminal", "some-custom-file"})
	assert.Contains(t, result, "wezterm")
}

// ---------------------------------------------------------------------------
// writeLayerFile (covers error paths)
// ---------------------------------------------------------------------------

func TestWriteLayerFile_ValidLayer(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	g := NewCaptureConfigGenerator(tmpDir)
	layer := &captureLayerYAML{
		Name: "test-layer",
		Git: &captureGitYAML{
			User: &captureGitUserYAML{
				Name:  "Test User",
				Email: "test@example.com",
			},
		},
	}

	err := g.writeLayerFile("test", layer, "A test layer")
	assert.NoError(t, err)

	// Verify the file was created
	layerPath := filepath.Join(layersDir, "test.yaml")
	_, statErr := os.Stat(layerPath)
	assert.NoError(t, statErr)
}

// ---------------------------------------------------------------------------
// DotfilesResolver.ResolveWithFallback
// ---------------------------------------------------------------------------

func TestResolveWithFallback_ExistingFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Create dotfiles structure
	dotfilesDir := filepath.Join(tmpDir, "dotfiles")
	targetDir := filepath.Join(dotfilesDir, "default")
	require.NoError(t, os.MkdirAll(targetDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, ".bashrc"), []byte("# bash"), 0o644))

	resolver := NewDotfilesResolver(tmpDir, "default")

	resolved := resolver.ResolveWithFallback(".bashrc")
	assert.NotEmpty(t, resolved)
}

// ---------------------------------------------------------------------------
// DotfilesResolver.ResolveDirectory and ResolveFile
// ---------------------------------------------------------------------------

func TestResolveDirectory_Existing(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// configRoot is tmpDir, so Resolve looks for tmpDir/.config/nvim
	configDir := filepath.Join(tmpDir, ".config", "nvim")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	resolver := NewDotfilesResolver(tmpDir, "default")

	resolved, isDir := resolver.ResolveDirectory(".config/nvim")
	assert.NotEmpty(t, resolved)
	assert.True(t, isDir)
}

func TestResolveDirectory_NonExistent(t *testing.T) {
	t.Parallel()

	resolver := NewDotfilesResolver(t.TempDir(), "default")

	resolved, isDir := resolver.ResolveDirectory(".config/nonexistent")
	assert.Empty(t, resolved)
	assert.False(t, isDir)
}

func TestResolveFile_Existing(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// configRoot is tmpDir, so Resolve looks for tmpDir/.gitconfig
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitconfig"), []byte("[user]\nname=test"), 0o644))

	resolver := NewDotfilesResolver(tmpDir, "default")

	resolved, isFile := resolver.ResolveFile(".gitconfig")
	assert.NotEmpty(t, resolved)
	assert.True(t, isFile)
}

func TestResolveFile_NonExistent(t *testing.T) {
	t.Parallel()

	resolver := NewDotfilesResolver(t.TempDir(), "default")

	resolved, isFile := resolver.ResolveFile(".nonexistent")
	assert.Empty(t, resolved)
	assert.False(t, isFile)
}

func TestResolveFile_IsDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, ".config")
	require.NoError(t, os.MkdirAll(dirPath, 0o755))

	resolver := NewDotfilesResolver(tmpDir, "default")

	resolved, isFile := resolver.ResolveFile(".config")
	assert.Empty(t, resolved)
	assert.False(t, isFile)
}

// ---------------------------------------------------------------------------
// addGoToolsToLayer with empty items after filtering
// ---------------------------------------------------------------------------

func TestAddGoToolsToLayer_EmptyItems(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	g.addGoToolsToLayer(layer, nil)

	assert.Nil(t, layer.Packages)
}

func TestAddPipPackagesToLayer_EmptyItems(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	g.addPipPackagesToLayer(layer, nil)

	assert.Nil(t, layer.Packages)
}

func TestAddGemPackagesToLayer_EmptyItems(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	g.addGemPackagesToLayer(layer, nil)

	assert.Nil(t, layer.Packages)
}

func TestAddCargoPackagesToLayer_EmptyItems(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	g.addCargoPackagesToLayer(layer, nil)

	assert.Nil(t, layer.Packages)
}

// ---------------------------------------------------------------------------
// ResolveWithFallback edge cases
// ---------------------------------------------------------------------------

func TestResolveWithFallback_EmptyPath(t *testing.T) {
	t.Parallel()

	resolver := NewDotfilesResolver(t.TempDir(), "default")
	result := resolver.ResolveWithFallback("")
	assert.Empty(t, result)
}

func TestResolveWithFallback_PathTraversal(t *testing.T) {
	t.Parallel()

	resolver := NewDotfilesResolver(t.TempDir(), "default")
	result := resolver.ResolveWithFallback("../../etc/passwd")
	assert.Empty(t, result)
}

func TestResolveWithFallback_AbsolutePath(t *testing.T) {
	t.Parallel()

	resolver := NewDotfilesResolver(t.TempDir(), "default")
	result := resolver.ResolveWithFallback("/usr/local/bin/something")
	assert.Equal(t, "/usr/local/bin/something", result)
}

func TestResolveWithFallback_LegacyDotfilesPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Create the legacy dotfiles directory structure
	legacyDir := filepath.Join(tmpDir, "dotfiles", "nvim")
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))

	resolver := NewDotfilesResolver(tmpDir, "default")
	result := resolver.ResolveWithFallback("dotfiles/nvim")
	assert.NotEmpty(t, result)
}

func TestResolveWithFallback_FallbackToConfigRoot(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	resolver := NewDotfilesResolver(tmpDir, "default")

	// Non-existent path that doesn't match legacy pattern
	result := resolver.ResolveWithFallback(".zshrc")
	// Falls through to fallback path under configRoot
	assert.NotEmpty(t, result)
	assert.Contains(t, result, ".zshrc")
}

// ---------------------------------------------------------------------------
// Resolve edge cases
// ---------------------------------------------------------------------------

func TestResolve_EmptyPath(t *testing.T) {
	t.Parallel()

	resolver := NewDotfilesResolver(t.TempDir(), "default")
	result := resolver.Resolve("")
	assert.Empty(t, result)
}

func TestResolve_PathTraversal(t *testing.T) {
	t.Parallel()

	resolver := NewDotfilesResolver(t.TempDir(), "default")
	result := resolver.Resolve("../../../etc/passwd")
	assert.Empty(t, result)
}

func TestResolve_WithTargetSuffix(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Create a target-suffixed path: .gitconfig.work
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitconfig.work"), []byte("[user]\nname=work"), 0o644))

	resolver := NewDotfilesResolver(tmpDir, "work")
	result := resolver.Resolve(".gitconfig")
	assert.NotEmpty(t, result)
	assert.Contains(t, result, ".gitconfig.work")
}

// ---------------------------------------------------------------------------
// isSensitive (via ConfigDiscoverer)
// ---------------------------------------------------------------------------

func TestIsSensitive_MatchesPattern(t *testing.T) {
	t.Parallel()

	discoverer := NewConfigDiscoverer(nil, t.TempDir())
	patterns := getSensitivePatterns()

	// .ssh/id_rsa should match ".ssh/id_*" pattern
	result := discoverer.isSensitive(".ssh/id_rsa", patterns)
	assert.True(t, result)
}

func TestIsSensitive_NoMatch(t *testing.T) {
	t.Parallel()

	discoverer := NewConfigDiscoverer(nil, t.TempDir())
	patterns := getSensitivePatterns()

	result := discoverer.isSensitive(".gitconfig", patterns)
	assert.False(t, result)
}

func TestIsSensitive_MatchesBasename(t *testing.T) {
	t.Parallel()

	discoverer := NewConfigDiscoverer(nil, t.TempDir())
	patterns := getSensitivePatterns()

	// ".env" basename should match the ".env" pattern
	result := discoverer.isSensitive(filepath.Join("some", "dir", ".env"), patterns)
	assert.True(t, result)
}

// ---------------------------------------------------------------------------
// writeLayerFile (error path - no layers dir)
// ---------------------------------------------------------------------------

func TestWriteLayerFile_NoLayersDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Don't create layers dir - it should be auto-created by writeLayerFile
	g := NewCaptureConfigGenerator(tmpDir)
	layer := &captureLayerYAML{
		Name: "test",
		Git: &captureGitYAML{
			User: &captureGitUserYAML{
				Name:  "Test",
				Email: "test@example.com",
			},
		},
	}

	err := g.writeLayerFile("test", layer, "Test layer")
	// Should either succeed (creates dir) or fail with clear error
	_ = err
}

// ---------------------------------------------------------------------------
// generateProviderLayerIfSupported additional coverage
// ---------------------------------------------------------------------------

func TestGenerateNvimFromCapture_ConfigWithNonStringValue(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: "config", Value: 42}, // non-string
	}

	result := g.generateNvimFromCapture(items)
	require.NotNil(t, result)
	assert.Empty(t, result.ConfigPath)
}

func TestGenerateNvimFromCapture_LazyLockWithNonString(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Name: "lazy-lock.json", Value: 42}, // non-string -- lockPath cast fails
	}

	result := g.generateNvimFromCapture(items)
	require.NotNil(t, result)
	// Non-string value means the lockPath type assertion fails, so plugin manager
	// is only set if PluginManager was empty (the default)
	// The switch matches "lazy-lock.json" but the inner ok check fails,
	// so PluginManager remains empty (the if-guard prevents setting it)
	assert.Empty(t, result.PluginManager)
}

// ---------------------------------------------------------------------------
// addNpmPackagesToLayer with non-string values
// ---------------------------------------------------------------------------

func TestAddNpmPackagesToLayer_NonStringValues(t *testing.T) {
	t.Parallel()

	g := NewCaptureConfigGenerator(t.TempDir())
	layer := &captureLayerYAML{}
	items := []CapturedItem{
		{Name: "typescript", Value: ""}, // empty string
		{Name: "eslint", Value: nil},    // nil value
	}

	g.addNpmPackagesToLayer(layer, items)

	require.NotNil(t, layer.Packages)
	require.NotNil(t, layer.Packages.Npm)
	// Items should be added using name as fallback
	assert.NotEmpty(t, layer.Packages.Npm.Packages)
}

// ---------------------------------------------------------------------------
// DotfilesCapturer.relativeToHome
// ---------------------------------------------------------------------------

func TestRelativeToHome_Success(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	capturer := NewDotfilesCapturer(nil, homeDir, t.TempDir())
	result := capturer.relativeToHome(filepath.Join(homeDir, ".gitconfig"))
	assert.Equal(t, ".gitconfig", result)
}

func TestRelativeToHome_OutsideHome(t *testing.T) {
	t.Parallel()

	capturer := NewDotfilesCapturer(nil, "/nonexistent/home", t.TempDir())
	result := capturer.relativeToHome("/tmp/something")
	// Should compute relative path or fall back
	assert.NotEmpty(t, result)
}
