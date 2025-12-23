package runtime

import (
	"testing"
)

func TestParseConfig_EmptyMap_ReturnsEmptyConfig(t *testing.T) {
	raw := map[string]interface{}{}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(cfg.Tools) != 0 {
		t.Errorf("Tools len = %d, want 0", len(cfg.Tools))
	}
}

func TestParseConfig_WithTools_ParsesToolVersions(t *testing.T) {
	raw := map[string]interface{}{
		"tools": []interface{}{
			map[string]interface{}{
				"name":    "node",
				"version": "20.10.0",
			},
			map[string]interface{}{
				"name":    "python",
				"version": "3.12.0",
			},
			map[string]interface{}{
				"name":    "golang",
				"version": "1.21.5",
			},
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(cfg.Tools) != 3 {
		t.Fatalf("Tools len = %d, want 3", len(cfg.Tools))
	}

	// Check node
	if cfg.Tools[0].Name != "node" {
		t.Errorf("Tools[0].Name = %q, want %q", cfg.Tools[0].Name, "node")
	}
	if cfg.Tools[0].Version != "20.10.0" {
		t.Errorf("Tools[0].Version = %q, want %q", cfg.Tools[0].Version, "20.10.0")
	}

	// Check python
	if cfg.Tools[1].Name != "python" {
		t.Errorf("Tools[1].Name = %q, want %q", cfg.Tools[1].Name, "python")
	}
	if cfg.Tools[1].Version != "3.12.0" {
		t.Errorf("Tools[1].Version = %q, want %q", cfg.Tools[1].Version, "3.12.0")
	}

	// Check golang
	if cfg.Tools[2].Name != "golang" {
		t.Errorf("Tools[2].Name = %q, want %q", cfg.Tools[2].Name, "golang")
	}
}

func TestParseConfig_WithBackend_ParsesBackendType(t *testing.T) {
	raw := map[string]interface{}{
		"backend": "rtx",
		"tools": []interface{}{
			map[string]interface{}{
				"name":    "node",
				"version": "20",
			},
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.Backend != "rtx" {
		t.Errorf("Backend = %q, want %q", cfg.Backend, "rtx")
	}
}

func TestParseConfig_WithGlobalScope_ParsesScope(t *testing.T) {
	raw := map[string]interface{}{
		"scope": "global",
		"tools": []interface{}{
			map[string]interface{}{
				"name":    "node",
				"version": "20",
			},
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.Scope != "global" {
		t.Errorf("Scope = %q, want %q", cfg.Scope, "global")
	}
}

func TestParseConfig_WithPlugins_ParsesPluginSources(t *testing.T) {
	raw := map[string]interface{}{
		"plugins": []interface{}{
			map[string]interface{}{
				"name": "golang",
				"url":  "https://github.com/asdf-community/asdf-golang.git",
			},
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(cfg.Plugins) != 1 {
		t.Fatalf("Plugins len = %d, want 1", len(cfg.Plugins))
	}

	if cfg.Plugins[0].Name != "golang" {
		t.Errorf("Plugins[0].Name = %q, want %q", cfg.Plugins[0].Name, "golang")
	}
	if cfg.Plugins[0].URL != "https://github.com/asdf-community/asdf-golang.git" {
		t.Errorf("Plugins[0].URL = %q, want custom URL", cfg.Plugins[0].URL)
	}
}

func TestConfig_ToolVersionsPath_GlobalScope(t *testing.T) {
	cfg := &Config{Scope: "global"}

	path := cfg.ToolVersionsPath()
	if path != "~/.tool-versions" {
		t.Errorf("ToolVersionsPath() = %q, want %q", path, "~/.tool-versions")
	}
}

func TestConfig_ToolVersionsPath_ProjectScope(t *testing.T) {
	cfg := &Config{Scope: "project"}

	path := cfg.ToolVersionsPath()
	if path != ".tool-versions" {
		t.Errorf("ToolVersionsPath() = %q, want %q", path, ".tool-versions")
	}
}

func TestConfig_ToolVersionsPath_DefaultsToGlobal(t *testing.T) {
	cfg := &Config{}

	path := cfg.ToolVersionsPath()
	if path != "~/.tool-versions" {
		t.Errorf("ToolVersionsPath() = %q, want %q", path, "~/.tool-versions")
	}
}
