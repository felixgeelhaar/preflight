package files

import (
	"testing"
)

func TestParseConfig_Empty(t *testing.T) {
	raw := map[string]interface{}{}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Links) != 0 {
		t.Errorf("Links len = %d, want 0", len(cfg.Links))
	}
	if len(cfg.Templates) != 0 {
		t.Errorf("Templates len = %d, want 0", len(cfg.Templates))
	}
	if len(cfg.Copies) != 0 {
		t.Errorf("Copies len = %d, want 0", len(cfg.Copies))
	}
}

func TestParseConfig_Links_Simple(t *testing.T) {
	raw := map[string]interface{}{
		"links": []interface{}{
			map[string]interface{}{
				"src":  "dotfiles/.zshrc",
				"dest": "~/.zshrc",
			},
		},
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Links) != 1 {
		t.Fatalf("Links len = %d, want 1", len(cfg.Links))
	}
	if cfg.Links[0].Src != "dotfiles/.zshrc" {
		t.Errorf("Links[0].Src = %q, want %q", cfg.Links[0].Src, "dotfiles/.zshrc")
	}
	if cfg.Links[0].Dest != "~/.zshrc" {
		t.Errorf("Links[0].Dest = %q, want %q", cfg.Links[0].Dest, "~/.zshrc")
	}
}

func TestParseConfig_Links_WithOptions(t *testing.T) {
	raw := map[string]interface{}{
		"links": []interface{}{
			map[string]interface{}{
				"src":    "dotfiles/.gitconfig",
				"dest":   "~/.gitconfig",
				"force":  true,
				"backup": true,
			},
		},
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if !cfg.Links[0].Force {
		t.Error("Links[0].Force should be true")
	}
	if !cfg.Links[0].Backup {
		t.Error("Links[0].Backup should be true")
	}
}

func TestParseConfig_Templates(t *testing.T) {
	raw := map[string]interface{}{
		"templates": []interface{}{
			map[string]interface{}{
				"src":  "templates/gitconfig.tmpl",
				"dest": "~/.gitconfig",
			},
		},
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Templates) != 1 {
		t.Fatalf("Templates len = %d, want 1", len(cfg.Templates))
	}
	if cfg.Templates[0].Src != "templates/gitconfig.tmpl" {
		t.Errorf("Templates[0].Src = %q, want %q", cfg.Templates[0].Src, "templates/gitconfig.tmpl")
	}
}

func TestParseConfig_Templates_WithVars(t *testing.T) {
	raw := map[string]interface{}{
		"templates": []interface{}{
			map[string]interface{}{
				"src":  "templates/gitconfig.tmpl",
				"dest": "~/.gitconfig",
				"vars": map[string]interface{}{
					"name":  "John Doe",
					"email": "john@example.com",
				},
			},
		},
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.Templates[0].Vars["name"] != "John Doe" {
		t.Errorf("Templates[0].Vars[name] = %q, want %q", cfg.Templates[0].Vars["name"], "John Doe")
	}
}

func TestParseConfig_Copies(t *testing.T) {
	raw := map[string]interface{}{
		"copies": []interface{}{
			map[string]interface{}{
				"src":  "files/id_rsa.pub",
				"dest": "~/.ssh/id_rsa.pub",
				"mode": "0644",
			},
		},
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Copies) != 1 {
		t.Fatalf("Copies len = %d, want 1", len(cfg.Copies))
	}
	if cfg.Copies[0].Mode != "0644" {
		t.Errorf("Copies[0].Mode = %q, want %q", cfg.Copies[0].Mode, "0644")
	}
}

func TestParseConfig_Full(t *testing.T) {
	raw := map[string]interface{}{
		"links": []interface{}{
			map[string]interface{}{
				"src":  "dotfiles/.zshrc",
				"dest": "~/.zshrc",
			},
		},
		"templates": []interface{}{
			map[string]interface{}{
				"src":  "templates/gitconfig.tmpl",
				"dest": "~/.gitconfig",
			},
		},
		"copies": []interface{}{
			map[string]interface{}{
				"src":  "files/script.sh",
				"dest": "~/.local/bin/script.sh",
			},
		},
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Links) != 1 {
		t.Errorf("Links len = %d, want 1", len(cfg.Links))
	}
	if len(cfg.Templates) != 1 {
		t.Errorf("Templates len = %d, want 1", len(cfg.Templates))
	}
	if len(cfg.Copies) != 1 {
		t.Errorf("Copies len = %d, want 1", len(cfg.Copies))
	}
}

func TestParseConfig_InvalidLinks(t *testing.T) {
	raw := map[string]interface{}{
		"links": "not-a-list",
	}
	_, err := ParseConfig(raw)
	if err == nil {
		t.Error("ParseConfig() should return error for invalid links")
	}
}

func TestParseConfig_LinkMissingSrc(t *testing.T) {
	raw := map[string]interface{}{
		"links": []interface{}{
			map[string]interface{}{
				"dest": "~/.zshrc",
			},
		},
	}
	_, err := ParseConfig(raw)
	if err == nil {
		t.Error("ParseConfig() should return error for link missing src")
	}
}

func TestParseConfig_LinkMissingDest(t *testing.T) {
	raw := map[string]interface{}{
		"links": []interface{}{
			map[string]interface{}{
				"src": "dotfiles/.zshrc",
			},
		},
	}
	_, err := ParseConfig(raw)
	if err == nil {
		t.Error("ParseConfig() should return error for link missing dest")
	}
}

func TestLink_ID(t *testing.T) {
	link := Link{Src: "dotfiles/.zshrc", Dest: "~/.zshrc"}
	id := link.ID()
	if id == "" {
		t.Error("Link.ID() should not be empty")
	}
}

func TestTemplate_ID(t *testing.T) {
	tmpl := Template{Src: "templates/config.tmpl", Dest: "~/.config/app/config"}
	id := tmpl.ID()
	if id == "" {
		t.Error("Template.ID() should not be empty")
	}
}

func TestCopy_ID(t *testing.T) {
	cp := Copy{Src: "files/script.sh", Dest: "~/.local/bin/script.sh"}
	id := cp.ID()
	if id == "" {
		t.Error("Copy.ID() should not be empty")
	}
}
