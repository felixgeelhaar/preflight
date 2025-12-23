package config

import (
	"os"
	"testing"
)

func TestMergedConfig_Raw(t *testing.T) {
	merged := &MergedConfig{
		Packages: PackageSet{
			Brew: BrewPackages{
				Taps:     []string{"homebrew/cask"},
				Formulae: []string{"git", "ripgrep"},
				Casks:    []string{"iterm2"},
			},
		},
		Files: []FileDeclaration{
			{Path: "~/.zshrc", Mode: FileModeGenerated},
		},
	}

	raw := merged.Raw()

	// Check brew section
	brew, ok := raw["brew"].(map[string]interface{})
	if !ok {
		t.Fatal("expected brew section to be map")
	}

	taps, ok := brew["taps"].([]interface{})
	if !ok {
		t.Fatal("expected taps to be []interface{}")
	}
	if len(taps) != 1 || taps[0].(string) != "homebrew/cask" {
		t.Errorf("taps = %v, want [homebrew/cask]", taps)
	}

	formulae, ok := brew["formulae"].([]interface{})
	if !ok {
		t.Fatal("expected formulae to be []interface{}")
	}
	if len(formulae) != 2 {
		t.Errorf("formulae len = %d, want 2", len(formulae))
	}

	casks, ok := brew["casks"].([]interface{})
	if !ok {
		t.Fatal("expected casks to be []interface{}")
	}
	if len(casks) != 1 {
		t.Errorf("casks len = %d, want 1", len(casks))
	}

	// Check files section exists
	files, ok := raw["files"].(map[string]interface{})
	if !ok {
		t.Fatal("expected files section to be map")
	}
	_ = files // Will add more checks when files format is finalized
}

func TestMergedConfig_Raw_Empty(t *testing.T) {
	merged := &MergedConfig{}
	raw := merged.Raw()

	if len(raw) == 0 {
		t.Error("expected raw to have sections even if empty")
	}
}

func TestMergedConfig_Raw_GitConfig(t *testing.T) {
	merged := &MergedConfig{
		Git: GitConfig{
			User: GitUserConfig{
				Name:       "John Doe",
				Email:      "john@example.com",
				SigningKey: "ABCD1234",
			},
			Core: GitCoreConfig{
				Editor:       "nvim",
				AutoCRLF:     "input",
				ExcludesFile: "~/.gitignore_global",
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
				"st": "status",
			},
			Includes: []GitInclude{
				{Path: "~/.gitconfig.work", IfConfig: "gitdir:~/work/"},
			},
		},
	}

	raw := merged.Raw()

	// Check git section
	git, ok := raw["git"].(map[string]interface{})
	if !ok {
		t.Fatal("expected git section to be map")
	}

	// Check user section
	user, ok := git["user"].(map[string]interface{})
	if !ok {
		t.Fatal("expected user section to be map")
	}
	if user["name"] != "John Doe" {
		t.Errorf("user.name = %v, want John Doe", user["name"])
	}
	if user["email"] != "john@example.com" {
		t.Errorf("user.email = %v, want john@example.com", user["email"])
	}
	if user["signingkey"] != "ABCD1234" {
		t.Errorf("user.signingkey = %v, want ABCD1234", user["signingkey"])
	}

	// Check core section
	core, ok := git["core"].(map[string]interface{})
	if !ok {
		t.Fatal("expected core section to be map")
	}
	if core["editor"] != "nvim" {
		t.Errorf("core.editor = %v, want nvim", core["editor"])
	}

	// Check commit section
	commit, ok := git["commit"].(map[string]interface{})
	if !ok {
		t.Fatal("expected commit section to be map")
	}
	if commit["gpgsign"] != true {
		t.Errorf("commit.gpgsign = %v, want true", commit["gpgsign"])
	}

	// Check gpg section
	gpg, ok := git["gpg"].(map[string]interface{})
	if !ok {
		t.Fatal("expected gpg section to be map")
	}
	if gpg["format"] != "openpgp" {
		t.Errorf("gpg.format = %v, want openpgp", gpg["format"])
	}

	// Check alias section
	alias, ok := git["alias"].(map[string]interface{})
	if !ok {
		t.Fatal("expected alias section to be map")
	}
	if alias["co"] != "checkout" {
		t.Errorf("alias.co = %v, want checkout", alias["co"])
	}

	// Check includes section
	includes, ok := git["includes"].([]interface{})
	if !ok {
		t.Fatal("expected includes to be []interface{}")
	}
	if len(includes) != 1 {
		t.Errorf("includes len = %d, want 1", len(includes))
	}
}

func TestLoader_Load(t *testing.T) {
	// This test requires setting up temp files
	// We'll test with minimal setup
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
}

func TestLoader_Load_Integration(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create manifest
	manifest := `
targets:
  default:
    - base
`
	if err := writeFile(t, tmpDir+"/preflight.yaml", manifest); err != nil {
		t.Fatal(err)
	}

	// Create layers directory
	if err := mkdir(t, tmpDir+"/layers"); err != nil {
		t.Fatal(err)
	}

	// Create base layer
	baseLayer := `
name: base
packages:
  brew:
    formulae:
      - git
      - ripgrep
`
	if err := writeFile(t, tmpDir+"/layers/base.yaml", baseLayer); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	target, err := NewTargetName("default")
	if err != nil {
		t.Fatal(err)
	}

	merged, err := loader.Load(tmpDir+"/preflight.yaml", target)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify the merged config
	if len(merged.Packages.Brew.Formulae) != 2 {
		t.Errorf("formulae len = %d, want 2", len(merged.Packages.Brew.Formulae))
	}
}

func TestMergedConfig_Raw_SSHConfig(t *testing.T) {
	t.Parallel()

	merged := &MergedConfig{
		SSH: SSHConfig{
			Include: "~/.ssh/config.d/*",
			Defaults: SSHDefaultsConfig{
				AddKeysToAgent:      true,
				IdentitiesOnly:      true,
				ForwardAgent:        true,
				ServerAliveInterval: 60,
				ServerAliveCountMax: 3,
			},
			Hosts: []SSHHostConfig{
				{
					Host:           "github.com",
					HostName:       "github.com",
					User:           "git",
					IdentityFile:   "~/.ssh/id_github",
					IdentitiesOnly: true,
					ForwardAgent:   true,
					AddKeysToAgent: true,
					UseKeychain:    true,
				},
				{
					Host:         "bastion",
					HostName:     "bastion.example.com",
					User:         "admin",
					Port:         2222,
					ProxyCommand: "nc -X 5 -x proxy:1080 %h %p",
					LocalForward: "8080:localhost:80",
				},
				{
					Host:          "internal",
					ProxyJump:     "bastion",
					RemoteForward: "9090:localhost:8080",
				},
			},
			Matches: []SSHMatchConfig{
				{
					Match:        "host *.example.com",
					HostName:     "example.com",
					User:         "deploy",
					IdentityFile: "~/.ssh/id_deploy",
					ProxyCommand: "ssh -W %h:%p bastion",
				},
				{
					Match:     "host internal",
					ProxyJump: "jump-host",
				},
			},
		},
	}

	raw := merged.Raw()

	// Check SSH section
	ssh, ok := raw["ssh"].(map[string]interface{})
	if !ok {
		t.Fatal("expected ssh section to be map")
	}

	// Check include
	if ssh["include"] != "~/.ssh/config.d/*" {
		t.Errorf("ssh.include = %v, want ~/.ssh/config.d/*", ssh["include"])
	}

	// Check defaults
	defaults, ok := ssh["defaults"].(map[string]interface{})
	if !ok {
		t.Fatal("expected defaults section to be map")
	}
	if defaults["addkeystoagent"] != true {
		t.Errorf("defaults.addkeystoagent = %v, want true", defaults["addkeystoagent"])
	}
	if defaults["identitiesonly"] != true {
		t.Errorf("defaults.identitiesonly = %v, want true", defaults["identitiesonly"])
	}
	if defaults["forwardagent"] != true {
		t.Errorf("defaults.forwardagent = %v, want true", defaults["forwardagent"])
	}
	if defaults["serveraliveinterval"] != 60 {
		t.Errorf("defaults.serveraliveinterval = %v, want 60", defaults["serveraliveinterval"])
	}
	if defaults["serveralivecountmax"] != 3 {
		t.Errorf("defaults.serveralivecountmax = %v, want 3", defaults["serveralivecountmax"])
	}

	// Check hosts
	hosts, ok := ssh["hosts"].([]interface{})
	if !ok {
		t.Fatal("expected hosts to be []interface{}")
	}
	if len(hosts) != 3 {
		t.Errorf("hosts len = %d, want 3", len(hosts))
	}

	// Check first host details
	host0 := hosts[0].(map[string]interface{})
	if host0["host"] != "github.com" {
		t.Errorf("host0.host = %v, want github.com", host0["host"])
	}
	if host0["hostname"] != "github.com" {
		t.Errorf("host0.hostname = %v, want github.com", host0["hostname"])
	}
	if host0["user"] != "git" {
		t.Errorf("host0.user = %v, want git", host0["user"])
	}
	if host0["identityfile"] != "~/.ssh/id_github" {
		t.Errorf("host0.identityfile = %v, want ~/.ssh/id_github", host0["identityfile"])
	}
	if host0["identitiesonly"] != true {
		t.Errorf("host0.identitiesonly = %v, want true", host0["identitiesonly"])
	}
	if host0["forwardagent"] != true {
		t.Errorf("host0.forwardagent = %v, want true", host0["forwardagent"])
	}
	if host0["addkeystoagent"] != true {
		t.Errorf("host0.addkeystoagent = %v, want true", host0["addkeystoagent"])
	}
	if host0["usekeychain"] != true {
		t.Errorf("host0.usekeychain = %v, want true", host0["usekeychain"])
	}

	// Check second host (port and proxy)
	host1 := hosts[1].(map[string]interface{})
	if host1["port"] != 2222 {
		t.Errorf("host1.port = %v, want 2222", host1["port"])
	}
	if host1["proxycommand"] != "nc -X 5 -x proxy:1080 %h %p" {
		t.Errorf("host1.proxycommand = %v", host1["proxycommand"])
	}
	if host1["localforward"] != "8080:localhost:80" {
		t.Errorf("host1.localforward = %v", host1["localforward"])
	}

	// Check third host (proxyjump and remoteforward)
	host2 := hosts[2].(map[string]interface{})
	if host2["proxyjump"] != "bastion" {
		t.Errorf("host2.proxyjump = %v, want bastion", host2["proxyjump"])
	}
	if host2["remoteforward"] != "9090:localhost:8080" {
		t.Errorf("host2.remoteforward = %v", host2["remoteforward"])
	}

	// Check matches
	matches, ok := ssh["matches"].([]interface{})
	if !ok {
		t.Fatal("expected matches to be []interface{}")
	}
	if len(matches) != 2 {
		t.Errorf("matches len = %d, want 2", len(matches))
	}

	match0 := matches[0].(map[string]interface{})
	if match0["match"] != "host *.example.com" {
		t.Errorf("match0.match = %v", match0["match"])
	}
	if match0["hostname"] != "example.com" {
		t.Errorf("match0.hostname = %v", match0["hostname"])
	}
	if match0["user"] != "deploy" {
		t.Errorf("match0.user = %v", match0["user"])
	}
	if match0["identityfile"] != "~/.ssh/id_deploy" {
		t.Errorf("match0.identityfile = %v", match0["identityfile"])
	}
	if match0["proxycommand"] != "ssh -W %h:%p bastion" {
		t.Errorf("match0.proxycommand = %v", match0["proxycommand"])
	}

	match1 := matches[1].(map[string]interface{})
	if match1["proxyjump"] != "jump-host" {
		t.Errorf("match1.proxyjump = %v", match1["proxyjump"])
	}
}

func TestMergedConfig_Raw_RuntimeConfig(t *testing.T) {
	t.Parallel()

	merged := &MergedConfig{
		Runtime: RuntimeConfig{
			Backend: "rtx",
			Scope:   "global",
			Tools: []RuntimeToolConfig{
				{Name: "go", Version: "1.23"},
				{Name: "node", Version: "20.10.0"},
				{Name: "python", Version: "3.12"},
			},
		},
	}

	raw := merged.Raw()

	runtime, ok := raw["runtime"].(map[string]interface{})
	if !ok {
		t.Fatal("expected runtime section to be map")
	}

	if runtime["backend"] != "rtx" {
		t.Errorf("runtime.backend = %v, want rtx", runtime["backend"])
	}
	if runtime["scope"] != "global" {
		t.Errorf("runtime.scope = %v, want global", runtime["scope"])
	}

	tools, ok := runtime["tools"].([]interface{})
	if !ok {
		t.Fatal("expected tools to be []interface{}")
	}
	if len(tools) != 3 {
		t.Errorf("tools len = %d, want 3", len(tools))
	}

	tool0 := tools[0].(map[string]interface{})
	if tool0["name"] != "go" {
		t.Errorf("tool0.name = %v, want go", tool0["name"])
	}
	if tool0["version"] != "1.23" {
		t.Errorf("tool0.version = %v, want 1.23", tool0["version"])
	}
}

func TestMergedConfig_Raw_ShellConfig(t *testing.T) {
	t.Parallel()

	merged := &MergedConfig{
		Shell: ShellConfig{
			Default: "zsh",
			Shells: []ShellConfigEntry{
				{
					Name:      "zsh",
					Framework: "oh-my-zsh",
					Theme:     "powerlevel10k/powerlevel10k",
					Plugins:   []string{"git", "docker", "kubectl"},
					CustomPlugins: []ShellCustomPlugin{
						{Name: "zsh-autosuggestions", Repo: "zsh-users/zsh-autosuggestions"},
					},
				},
				{
					Name:    "bash",
					Plugins: []string{"bash-completion"},
				},
			},
			Starship: ShellStarshipConfig{
				Enabled: true,
				Preset:  "nerd-font-symbols",
			},
			Env: map[string]string{
				"EDITOR": "nvim",
				"PAGER":  "less",
			},
			Aliases: map[string]string{
				"ll": "ls -la",
				"gs": "git status",
			},
		},
	}

	raw := merged.Raw()

	shell, ok := raw["shell"].(map[string]interface{})
	if !ok {
		t.Fatal("expected shell section to be map")
	}

	if shell["default"] != "zsh" {
		t.Errorf("shell.default = %v, want zsh", shell["default"])
	}

	// Check shells array
	shells, ok := shell["shells"].([]interface{})
	if !ok {
		t.Fatal("expected shells to be []interface{}")
	}
	if len(shells) != 2 {
		t.Errorf("shells len = %d, want 2", len(shells))
	}

	// Check first shell (zsh)
	shell0 := shells[0].(map[string]interface{})
	if shell0["name"] != "zsh" {
		t.Errorf("shell0.name = %v, want zsh", shell0["name"])
	}
	if shell0["framework"] != "oh-my-zsh" {
		t.Errorf("shell0.framework = %v, want oh-my-zsh", shell0["framework"])
	}
	if shell0["theme"] != "powerlevel10k/powerlevel10k" {
		t.Errorf("shell0.theme = %v", shell0["theme"])
	}

	plugins, ok := shell0["plugins"].([]interface{})
	if !ok {
		t.Fatal("expected shell0.plugins to be []interface{}")
	}
	if len(plugins) != 3 {
		t.Errorf("shell0.plugins len = %d, want 3", len(plugins))
	}

	customPlugins, ok := shell0["custom_plugins"].([]interface{})
	if !ok {
		t.Fatal("expected shell0.custom_plugins to be []interface{}")
	}
	if len(customPlugins) != 1 {
		t.Errorf("shell0.custom_plugins len = %d, want 1", len(customPlugins))
	}

	// Check starship config
	starship, ok := shell["starship"].(map[string]interface{})
	if !ok {
		t.Fatal("expected starship section to be map")
	}
	if starship["enabled"] != true {
		t.Errorf("starship.enabled = %v, want true", starship["enabled"])
	}
	if starship["preset"] != "nerd-font-symbols" {
		t.Errorf("starship.preset = %v", starship["preset"])
	}

	// Check env
	env, ok := shell["env"].(map[string]interface{})
	if !ok {
		t.Fatal("expected env section to be map")
	}
	if env["EDITOR"] != "nvim" {
		t.Errorf("env.EDITOR = %v, want nvim", env["EDITOR"])
	}

	// Check aliases
	aliases, ok := shell["aliases"].(map[string]interface{})
	if !ok {
		t.Fatal("expected aliases section to be map")
	}
	if aliases["ll"] != "ls -la" {
		t.Errorf("aliases.ll = %v", aliases["ll"])
	}
}

func TestMergedConfig_Raw_NvimConfig(t *testing.T) {
	t.Parallel()

	merged := &MergedConfig{
		Nvim: NvimConfig{
			Preset:        "lazyvim",
			PluginManager: "lazy",
			ConfigRepo:    "https://github.com/user/nvim-config",
			EnsureInstall: true,
		},
	}

	raw := merged.Raw()

	nvim, ok := raw["nvim"].(map[string]interface{})
	if !ok {
		t.Fatal("expected nvim section to be map")
	}

	if nvim["preset"] != "lazyvim" {
		t.Errorf("nvim.preset = %v, want lazyvim", nvim["preset"])
	}
	if nvim["plugin_manager"] != "lazy" {
		t.Errorf("nvim.plugin_manager = %v, want lazy", nvim["plugin_manager"])
	}
	if nvim["config_repo"] != "https://github.com/user/nvim-config" {
		t.Errorf("nvim.config_repo = %v", nvim["config_repo"])
	}
	if nvim["ensure_install"] != true {
		t.Errorf("nvim.ensure_install = %v, want true", nvim["ensure_install"])
	}
}

func TestMergedConfig_Raw_VSCodeConfig(t *testing.T) {
	t.Parallel()

	merged := &MergedConfig{
		VSCode: VSCodeConfig{
			Extensions: []string{
				"golang.go",
				"rust-lang.rust-analyzer",
				"ms-python.python",
			},
			Settings: map[string]interface{}{
				"editor.fontSize":      14,
				"editor.tabSize":       4,
				"workbench.colorTheme": "Catppuccin Mocha",
			},
			Keybindings: []VSCodeKeybinding{
				{Key: "ctrl+shift+b", Command: "workbench.action.tasks.build"},
				{Key: "ctrl+shift+t", Command: "workbench.action.tasks.test"},
			},
		},
	}

	raw := merged.Raw()

	vscode, ok := raw["vscode"].(map[string]interface{})
	if !ok {
		t.Fatal("expected vscode section to be map")
	}

	extensions, ok := vscode["extensions"].([]interface{})
	if !ok {
		t.Fatal("expected vscode.extensions to be []interface{}")
	}
	if len(extensions) != 3 {
		t.Errorf("vscode.extensions len = %d, want 3", len(extensions))
	}

	settings, ok := vscode["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("expected vscode.settings to be map")
	}
	if settings["editor.fontSize"] != 14 {
		t.Errorf("vscode.settings.editor.fontSize = %v, want 14", settings["editor.fontSize"])
	}

	keybindings, ok := vscode["keybindings"].([]interface{})
	if !ok {
		t.Fatal("expected vscode.keybindings to be []interface{}")
	}
	if len(keybindings) != 2 {
		t.Errorf("vscode.keybindings len = %d, want 2", len(keybindings))
	}
}

func TestMergedConfig_Raw_APTPackages(t *testing.T) {
	t.Parallel()

	merged := &MergedConfig{
		Packages: PackageSet{
			Apt: AptPackages{
				PPAs:     []string{"ppa:git-core/ppa", "ppa:neovim-ppa/unstable"},
				Packages: []string{"git", "neovim", "build-essential"},
			},
		},
	}

	raw := merged.Raw()

	apt, ok := raw["apt"].(map[string]interface{})
	if !ok {
		t.Fatal("expected apt section to be map")
	}

	ppas, ok := apt["ppas"].([]interface{})
	if !ok {
		t.Fatal("expected ppas to be []interface{}")
	}
	if len(ppas) != 2 {
		t.Errorf("ppas len = %d, want 2", len(ppas))
	}

	packages, ok := apt["packages"].([]interface{})
	if !ok {
		t.Fatal("expected packages to be []interface{}")
	}
	if len(packages) != 3 {
		t.Errorf("packages len = %d, want 3", len(packages))
	}
}

func TestMergedConfig_Raw_FileModeTemplate(t *testing.T) {
	t.Parallel()

	merged := &MergedConfig{
		Files: []FileDeclaration{
			{Path: "~/.zshrc", Mode: FileModeGenerated, Template: "dotfiles/zshrc"},
			{Path: "~/.config/starship.toml", Mode: FileModeTemplate, Template: "dotfiles/starship.toml.tmpl"},
			{Path: "~/.gitignore_global", Mode: FileModeBYO, Template: "dotfiles/gitignore"},
		},
	}

	raw := merged.Raw()

	files, ok := raw["files"].(map[string]interface{})
	if !ok {
		t.Fatal("expected files section to be map")
	}

	links, ok := files["links"].([]interface{})
	if !ok {
		t.Fatal("expected links to be []interface{}")
	}
	// Generated and BYO become links
	if len(links) != 2 {
		t.Errorf("links len = %d, want 2", len(links))
	}

	templates, ok := files["templates"].([]interface{})
	if !ok {
		t.Fatal("expected templates to be []interface{}")
	}
	if len(templates) != 1 {
		t.Errorf("templates len = %d, want 1", len(templates))
	}

	template0 := templates[0].(map[string]interface{})
	if template0["dest"] != "~/.config/starship.toml" {
		t.Errorf("template0.dest = %v", template0["dest"])
	}
	if template0["src"] != "dotfiles/starship.toml.tmpl" {
		t.Errorf("template0.src = %v", template0["src"])
	}
}

func writeFile(t *testing.T, path, content string) error {
	t.Helper()
	return os.WriteFile(path, []byte(content), 0o644)
}

func mkdir(t *testing.T, path string) error {
	t.Helper()
	return os.MkdirAll(path, 0o755)
}
