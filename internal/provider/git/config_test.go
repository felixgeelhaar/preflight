package git

import (
	"testing"
)

func TestParseConfig_Empty(t *testing.T) {
	cfg, err := ParseConfig(map[string]interface{}{})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("ParseConfig() returned nil")
	}
}

func TestParseConfig_UserConfig(t *testing.T) {
	raw := map[string]interface{}{
		"user": map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.User.Name != "John Doe" {
		t.Errorf("User.Name = %q, want %q", cfg.User.Name, "John Doe")
	}
	if cfg.User.Email != "john@example.com" {
		t.Errorf("User.Email = %q, want %q", cfg.User.Email, "john@example.com")
	}
}

func TestParseConfig_SigningConfig(t *testing.T) {
	raw := map[string]interface{}{
		"user": map[string]interface{}{
			"name":       "John Doe",
			"email":      "john@example.com",
			"signingkey": "ABCD1234",
		},
		"commit": map[string]interface{}{
			"gpgsign": true,
		},
		"gpg": map[string]interface{}{
			"format": "openpgp",
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.User.SigningKey != "ABCD1234" {
		t.Errorf("User.SigningKey = %q, want %q", cfg.User.SigningKey, "ABCD1234")
	}
	if !cfg.Commit.GPGSign {
		t.Error("Commit.GPGSign = false, want true")
	}
	if cfg.GPG.Format != "openpgp" {
		t.Errorf("GPG.Format = %q, want %q", cfg.GPG.Format, "openpgp")
	}
}

func TestParseConfig_CoreConfig(t *testing.T) {
	raw := map[string]interface{}{
		"core": map[string]interface{}{
			"editor":       "nvim",
			"autocrlf":     "input",
			"excludesfile": "~/.gitignore_global",
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.Core.Editor != "nvim" {
		t.Errorf("Core.Editor = %q, want %q", cfg.Core.Editor, "nvim")
	}
	if cfg.Core.AutoCRLF != "input" {
		t.Errorf("Core.AutoCRLF = %q, want %q", cfg.Core.AutoCRLF, "input")
	}
	if cfg.Core.ExcludesFile != "~/.gitignore_global" {
		t.Errorf("Core.ExcludesFile = %q, want %q", cfg.Core.ExcludesFile, "~/.gitignore_global")
	}
}

func TestParseConfig_Aliases(t *testing.T) {
	raw := map[string]interface{}{
		"alias": map[string]interface{}{
			"co": "checkout",
			"br": "branch",
			"st": "status",
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(cfg.Aliases) != 3 {
		t.Errorf("len(Aliases) = %d, want 3", len(cfg.Aliases))
	}
	if cfg.Aliases["co"] != "checkout" {
		t.Errorf("Aliases[co] = %q, want %q", cfg.Aliases["co"], "checkout")
	}
}

func TestParseConfig_Includes(t *testing.T) {
	raw := map[string]interface{}{
		"includes": []interface{}{
			map[string]interface{}{
				"path":     "~/.gitconfig.work",
				"ifconfig": "gitdir:~/work/",
			},
			map[string]interface{}{
				"path": "~/.gitconfig.personal",
			},
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(cfg.Includes) != 2 {
		t.Fatalf("len(Includes) = %d, want 2", len(cfg.Includes))
	}

	if cfg.Includes[0].Path != "~/.gitconfig.work" {
		t.Errorf("Includes[0].Path = %q, want %q", cfg.Includes[0].Path, "~/.gitconfig.work")
	}
	if cfg.Includes[0].IfConfig != "gitdir:~/work/" {
		t.Errorf("Includes[0].IfConfig = %q, want %q", cfg.Includes[0].IfConfig, "gitdir:~/work/")
	}
}

func TestParseConfig_InvalidIncludes(t *testing.T) {
	raw := map[string]interface{}{
		"includes": "not-a-list",
	}

	_, err := ParseConfig(raw)
	if err == nil {
		t.Error("ParseConfig() should return error for invalid includes")
	}
}

func TestInclude_ID(t *testing.T) {
	inc := Include{Path: "~/.gitconfig.work", IfConfig: "gitdir:~/work/"}
	id := inc.ID()

	if id == "" {
		t.Error("Include.ID() returned empty string")
	}
}

func TestConfig_ConfigPath(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		wantPath string
	}{
		{
			name:     "default path",
			cfg:      Config{},
			wantPath: "~/.gitconfig",
		},
		{
			name:     "custom path",
			cfg:      Config{Path: "~/.config/git/config"},
			wantPath: "~/.config/git/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.ConfigPath(); got != tt.wantPath {
				t.Errorf("ConfigPath() = %q, want %q", got, tt.wantPath)
			}
		})
	}
}
