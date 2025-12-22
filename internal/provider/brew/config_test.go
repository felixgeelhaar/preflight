package brew

import (
	"testing"
)

func TestParseConfig_Empty(t *testing.T) {
	raw := map[string]interface{}{}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Taps) != 0 {
		t.Errorf("Taps len = %d, want 0", len(cfg.Taps))
	}
	if len(cfg.Formulae) != 0 {
		t.Errorf("Formulae len = %d, want 0", len(cfg.Formulae))
	}
	if len(cfg.Casks) != 0 {
		t.Errorf("Casks len = %d, want 0", len(cfg.Casks))
	}
}

func TestParseConfig_Taps(t *testing.T) {
	raw := map[string]interface{}{
		"taps": []interface{}{"homebrew/cask", "homebrew/core"},
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Taps) != 2 {
		t.Fatalf("Taps len = %d, want 2", len(cfg.Taps))
	}
	if cfg.Taps[0] != "homebrew/cask" {
		t.Errorf("Taps[0] = %q, want %q", cfg.Taps[0], "homebrew/cask")
	}
}

func TestParseConfig_Formulae_Simple(t *testing.T) {
	raw := map[string]interface{}{
		"formulae": []interface{}{"git", "curl", "wget"},
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Formulae) != 3 {
		t.Fatalf("Formulae len = %d, want 3", len(cfg.Formulae))
	}
	if cfg.Formulae[0].Name != "git" {
		t.Errorf("Formulae[0].Name = %q, want %q", cfg.Formulae[0].Name, "git")
	}
}

func TestParseConfig_Formulae_WithOptions(t *testing.T) {
	raw := map[string]interface{}{
		"formulae": []interface{}{
			map[string]interface{}{
				"name": "neovim",
				"args": []interface{}{"--HEAD"},
			},
		},
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Formulae) != 1 {
		t.Fatalf("Formulae len = %d, want 1", len(cfg.Formulae))
	}
	if cfg.Formulae[0].Name != "neovim" {
		t.Errorf("Formulae[0].Name = %q, want %q", cfg.Formulae[0].Name, "neovim")
	}
	if len(cfg.Formulae[0].Args) != 1 || cfg.Formulae[0].Args[0] != "--HEAD" {
		t.Errorf("Formulae[0].Args = %v, want [--HEAD]", cfg.Formulae[0].Args)
	}
}

func TestParseConfig_Casks(t *testing.T) {
	raw := map[string]interface{}{
		"casks": []interface{}{"visual-studio-code", "docker", "slack"},
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Casks) != 3 {
		t.Fatalf("Casks len = %d, want 3", len(cfg.Casks))
	}
	if cfg.Casks[0].Name != "visual-studio-code" {
		t.Errorf("Casks[0].Name = %q, want %q", cfg.Casks[0].Name, "visual-studio-code")
	}
}

func TestParseConfig_Full(t *testing.T) {
	raw := map[string]interface{}{
		"taps": []interface{}{"homebrew/cask"},
		"formulae": []interface{}{
			"git",
			map[string]interface{}{
				"name": "neovim",
				"args": []interface{}{"--HEAD"},
			},
		},
		"casks": []interface{}{"docker"},
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Taps) != 1 {
		t.Errorf("Taps len = %d, want 1", len(cfg.Taps))
	}
	if len(cfg.Formulae) != 2 {
		t.Errorf("Formulae len = %d, want 2", len(cfg.Formulae))
	}
	if len(cfg.Casks) != 1 {
		t.Errorf("Casks len = %d, want 1", len(cfg.Casks))
	}
}

func TestFormula_FullName(t *testing.T) {
	tests := []struct {
		formula  Formula
		expected string
	}{
		{Formula{Name: "git"}, "git"},
		{Formula{Name: "neovim", Tap: "homebrew/core"}, "homebrew/core/neovim"},
	}

	for _, tt := range tests {
		if got := tt.formula.FullName(); got != tt.expected {
			t.Errorf("Formula{%q, %q}.FullName() = %q, want %q",
				tt.formula.Name, tt.formula.Tap, got, tt.expected)
		}
	}
}

func TestCask_FullName(t *testing.T) {
	tests := []struct {
		cask     Cask
		expected string
	}{
		{Cask{Name: "docker"}, "docker"},
		{Cask{Name: "font-fira-code", Tap: "homebrew/cask-fonts"}, "homebrew/cask-fonts/font-fira-code"},
	}

	for _, tt := range tests {
		if got := tt.cask.FullName(); got != tt.expected {
			t.Errorf("Cask{%q, %q}.FullName() = %q, want %q",
				tt.cask.Name, tt.cask.Tap, got, tt.expected)
		}
	}
}
