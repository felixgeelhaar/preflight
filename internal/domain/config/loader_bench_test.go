package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
)

// createBenchmarkConfig creates a temporary config for benchmarking.
func createBenchmarkConfig(b *testing.B, numLayers int) (string, func()) {
	b.Helper()

	tempDir, err := os.MkdirTemp("", "preflight-bench-*")
	if err != nil {
		b.Fatal(err)
	}

	layersDir := filepath.Join(tempDir, "layers")
	if err := os.MkdirAll(layersDir, 0o755); err != nil {
		b.Fatal(err)
	}

	// Create manifest
	manifest := "targets:\n  default:\n"
	for i := 0; i < numLayers; i++ {
		manifest += "    - layer" + string(rune('a'+i)) + "\n"
	}
	manifestPath := filepath.Join(tempDir, "preflight.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		b.Fatal(err)
	}

	// Create layers with varying complexity
	for i := 0; i < numLayers; i++ {
		name := "layer" + string(rune('a'+i))
		layer := createBenchmarkLayer(name, i+1)
		layerPath := filepath.Join(layersDir, name+".yaml")
		if err := os.WriteFile(layerPath, []byte(layer), 0o644); err != nil {
			b.Fatal(err)
		}
	}

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	return manifestPath, cleanup
}

// createBenchmarkLayer creates a layer with varying content.
func createBenchmarkLayer(name string, multiplier int) string {
	layer := "name: " + name + "\n"

	// Add packages (lists allow duplicates when merging)
	layer += "packages:\n  brew:\n    formulae:\n"
	for i := 0; i < 10*multiplier; i++ {
		layer += fmt.Sprintf("      - pkg%d\n", i)
	}
	layer += "    casks:\n"
	for i := 0; i < 5*multiplier; i++ {
		layer += fmt.Sprintf("      - cask%d\n", i)
	}

	// Add git config
	layer += "git:\n"
	layer += "  user:\n"
	layer += "    name: \"Test User\"\n"
	layer += "    email: \"test@example.com\"\n"
	layer += "  alias:\n"
	// Use unique keys
	for i := 0; i < 5*multiplier; i++ {
		layer += fmt.Sprintf("    alias%d: \"command%d\"\n", i, i)
	}

	// Add shell config
	layer += "shell:\n"
	layer += "  default: zsh\n"
	layer += "  env:\n"
	for i := 0; i < 5*multiplier; i++ {
		layer += fmt.Sprintf("    VAR%d: \"value%d\"\n", i, i)
	}
	layer += "  aliases:\n"
	for i := 0; i < 5*multiplier; i++ {
		layer += fmt.Sprintf("    sh%d: \"command%d\"\n", i, i)
	}

	return layer
}

func BenchmarkLoader_LoadManifest(b *testing.B) {
	manifestPath, cleanup := createBenchmarkConfig(b, 5)
	defer cleanup()

	loader := config.NewLoader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.LoadManifest(manifestPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoader_LoadLayer(b *testing.B) {
	manifestPath, cleanup := createBenchmarkConfig(b, 5)
	defer cleanup()

	layerPath := filepath.Join(filepath.Dir(manifestPath), "layers", "layera.yaml")
	loader := config.NewLoader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.LoadLayer(layerPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoader_Load_5Layers(b *testing.B) {
	manifestPath, cleanup := createBenchmarkConfig(b, 5)
	defer cleanup()

	loader := config.NewLoader()
	targetName, _ := config.NewTargetName("default")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.Load(manifestPath, targetName)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoader_Load_10Layers(b *testing.B) {
	manifestPath, cleanup := createBenchmarkConfig(b, 10)
	defer cleanup()

	loader := config.NewLoader()
	targetName, _ := config.NewTargetName("default")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.Load(manifestPath, targetName)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoader_Load_20Layers(b *testing.B) {
	manifestPath, cleanup := createBenchmarkConfig(b, 20)
	defer cleanup()

	loader := config.NewLoader()
	targetName, _ := config.NewTargetName("default")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.Load(manifestPath, targetName)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseManifest(b *testing.B) {
	manifestYAML := []byte(`
targets:
  default:
    - base
    - identity.work
    - role.dev
  personal:
    - base
    - identity.personal
    - role.dev
  minimal:
    - base
defaults:
  mode: locked
  editor: nvim
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := config.ParseManifest(manifestYAML)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseLayer_Small(b *testing.B) {
	layerYAML := []byte(`
name: base
git:
  user:
    name: "Test User"
    email: "test@example.com"
  core:
    editor: nvim
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := config.ParseLayer(layerYAML)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseLayer_Medium(b *testing.B) {
	layerYAML := []byte(`
name: base
packages:
  brew:
    taps:
      - homebrew/cask-fonts
    formulae:
      - ripgrep
      - fzf
      - bat
      - fd
      - jq
      - yq
      - git
      - gh
      - neovim
      - tmux
    casks:
      - docker
      - visual-studio-code
      - wezterm
      - 1password
git:
  user:
    name: "Test User"
    email: "test@example.com"
  core:
    editor: nvim
  alias:
    co: checkout
    br: branch
    ci: commit
    st: status
shell:
  default: zsh
  env:
    EDITOR: nvim
    VISUAL: nvim
  aliases:
    ll: "ls -la"
    la: "ls -A"
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := config.ParseLayer(layerYAML)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseLayer_Large(b *testing.B) {
	layerYAML := []byte(`
name: full
packages:
  brew:
    taps:
      - homebrew/cask-fonts
      - homebrew/services
    formulae:
      - ripgrep
      - fzf
      - bat
      - fd
      - jq
      - yq
      - git
      - gh
      - neovim
      - tmux
      - zsh
      - starship
      - kubectl
      - helm
      - terraform
      - awscli
      - gcloud
      - azure-cli
      - docker
      - docker-compose
    casks:
      - docker
      - visual-studio-code
      - wezterm
      - 1password
      - slack
      - zoom
      - notion
      - figma
git:
  user:
    name: "Test User"
    email: "test@example.com"
    signingkey: "ABCD1234"
  core:
    editor: nvim
    autocrlf: input
    excludesfile: ~/.gitignore_global
  commit:
    gpgsign: true
  gpg:
    format: ssh
  alias:
    co: checkout
    br: branch
    ci: commit
    st: status
    lg: log --oneline --graph
    unstage: reset HEAD --
    last: log -1 HEAD
    amend: commit --amend
  includes:
    - path: ~/.gitconfig.work
      ifconfig: user.email:work@company.com
shell:
  default: zsh
  shells:
    - name: zsh
      framework: oh-my-zsh
      theme: robbyrussell
      plugins:
        - git
        - docker
        - kubectl
        - aws
        - terraform
  starship:
    enabled: true
    preset: plain-text
  env:
    EDITOR: nvim
    VISUAL: nvim
    GOPATH: $HOME/go
    PATH: $GOPATH/bin:$PATH
  aliases:
    ll: "ls -la"
    la: "ls -A"
    k: kubectl
    tf: terraform
ssh:
  defaults:
    addkeystoagent: true
    identitiesonly: true
  hosts:
    - host: github.com
      hostname: github.com
      user: git
      identityfile: ~/.ssh/id_github
    - host: gitlab.com
      hostname: gitlab.com
      user: git
      identityfile: ~/.ssh/id_gitlab
nvim:
  preset: kickstart
  plugin_manager: lazy
  ensure_install: true
vscode:
  extensions:
    - golang.go
    - rust-lang.rust-analyzer
    - ms-python.python
    - esbenp.prettier-vscode
    - dbaeumer.vscode-eslint
  settings:
    editor.fontSize: 14
    editor.tabSize: 2
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := config.ParseLayer(layerYAML)
		if err != nil {
			b.Fatal(err)
		}
	}
}
