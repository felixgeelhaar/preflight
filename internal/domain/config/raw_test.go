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

func writeFile(t *testing.T, path, content string) error {
	t.Helper()
	return os.WriteFile(path, []byte(content), 0644)
}

func mkdir(t *testing.T, path string) error {
	t.Helper()
	return os.MkdirAll(path, 0755)
}
