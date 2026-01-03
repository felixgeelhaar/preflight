package macos

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
	if len(cfg.Defaults) != 0 {
		t.Errorf("Defaults len = %d, want 0", len(cfg.Defaults))
	}
}

func TestParseConfig_Defaults(t *testing.T) {
	raw := map[string]interface{}{
		"defaults": []interface{}{
			map[string]interface{}{
				"domain": "com.apple.dock",
				"key":    "autohide",
				"type":   "bool",
				"value":  true,
			},
			map[string]interface{}{
				"domain": "NSGlobalDomain",
				"key":    "AppleShowScrollBars",
				"value":  "Always",
			},
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(cfg.Defaults) != 2 {
		t.Fatalf("Defaults len = %d, want 2", len(cfg.Defaults))
	}

	// First default
	def := cfg.Defaults[0]
	if def.Domain != "com.apple.dock" {
		t.Errorf("Domain = %q, want %q", def.Domain, "com.apple.dock")
	}
	if def.Key != "autohide" {
		t.Errorf("Key = %q, want %q", def.Key, "autohide")
	}
	if def.Type != "bool" {
		t.Errorf("Type = %q, want %q", def.Type, "bool")
	}
	if def.Value != true {
		t.Errorf("Value = %v, want true", def.Value)
	}

	// Second default (no type specified)
	def = cfg.Defaults[1]
	if def.Type != "string" {
		t.Errorf("Default type = %q, want %q", def.Type, "string")
	}
}

func TestParseConfig_Defaults_InvalidFormat(t *testing.T) {
	raw := map[string]interface{}{
		"defaults": "not a list",
	}

	_, err := ParseConfig(raw)
	if err == nil {
		t.Fatal("expected error for invalid defaults format")
	}
}

func TestParseConfig_Defaults_MissingDomain(t *testing.T) {
	raw := map[string]interface{}{
		"defaults": []interface{}{
			map[string]interface{}{
				"key":   "autohide",
				"value": true,
			},
		},
	}

	_, err := ParseConfig(raw)
	if err == nil {
		t.Fatal("expected error for missing domain")
	}
}

func TestParseConfig_Defaults_MissingKey(t *testing.T) {
	raw := map[string]interface{}{
		"defaults": []interface{}{
			map[string]interface{}{
				"domain": "com.apple.dock",
				"value":  true,
			},
		},
	}

	_, err := ParseConfig(raw)
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestParseConfig_Defaults_InvalidItem(t *testing.T) {
	raw := map[string]interface{}{
		"defaults": []interface{}{
			"not an object",
		},
	}

	_, err := ParseConfig(raw)
	if err == nil {
		t.Fatal("expected error for invalid default item")
	}
}

func TestParseConfig_Dock(t *testing.T) {
	raw := map[string]interface{}{
		"dock": map[string]interface{}{
			"add":    []interface{}{"Safari", "Terminal", "Finder"},
			"remove": []interface{}{"FaceTime", "Maps"},
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(cfg.Dock.Add) != 3 {
		t.Errorf("Dock.Add len = %d, want 3", len(cfg.Dock.Add))
	}
	if cfg.Dock.Add[0] != "Safari" {
		t.Errorf("Dock.Add[0] = %q, want %q", cfg.Dock.Add[0], "Safari")
	}

	if len(cfg.Dock.Remove) != 2 {
		t.Errorf("Dock.Remove len = %d, want 2", len(cfg.Dock.Remove))
	}
	if cfg.Dock.Remove[0] != "FaceTime" {
		t.Errorf("Dock.Remove[0] = %q, want %q", cfg.Dock.Remove[0], "FaceTime")
	}
}

func TestParseConfig_Finder(t *testing.T) {
	raw := map[string]interface{}{
		"finder": map[string]interface{}{
			"show_hidden":     true,
			"show_extensions": true,
			"show_path_bar":   false,
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.Finder.ShowHidden == nil || *cfg.Finder.ShowHidden != true {
		t.Errorf("ShowHidden = %v, want true", cfg.Finder.ShowHidden)
	}
	if cfg.Finder.ShowExtensions == nil || *cfg.Finder.ShowExtensions != true {
		t.Errorf("ShowExtensions = %v, want true", cfg.Finder.ShowExtensions)
	}
	if cfg.Finder.ShowPathBar == nil || *cfg.Finder.ShowPathBar != false {
		t.Errorf("ShowPathBar = %v, want false", cfg.Finder.ShowPathBar)
	}
}

func TestParseConfig_Finder_Partial(t *testing.T) {
	raw := map[string]interface{}{
		"finder": map[string]interface{}{
			"show_hidden": true,
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.Finder.ShowHidden == nil || *cfg.Finder.ShowHidden != true {
		t.Errorf("ShowHidden = %v, want true", cfg.Finder.ShowHidden)
	}
	if cfg.Finder.ShowExtensions != nil {
		t.Error("ShowExtensions should be nil when not specified")
	}
	if cfg.Finder.ShowPathBar != nil {
		t.Error("ShowPathBar should be nil when not specified")
	}
}

func TestParseConfig_Keyboard(t *testing.T) {
	raw := map[string]interface{}{
		"keyboard": map[string]interface{}{
			"key_repeat":         2,
			"initial_key_repeat": 15,
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.Keyboard.KeyRepeat == nil || *cfg.Keyboard.KeyRepeat != 2 {
		t.Errorf("KeyRepeat = %v, want 2", cfg.Keyboard.KeyRepeat)
	}
	if cfg.Keyboard.InitialKeyRepeat == nil || *cfg.Keyboard.InitialKeyRepeat != 15 {
		t.Errorf("InitialKeyRepeat = %v, want 15", cfg.Keyboard.InitialKeyRepeat)
	}
}

func TestParseConfig_Keyboard_Partial(t *testing.T) {
	raw := map[string]interface{}{
		"keyboard": map[string]interface{}{
			"key_repeat": 2,
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.Keyboard.KeyRepeat == nil || *cfg.Keyboard.KeyRepeat != 2 {
		t.Errorf("KeyRepeat = %v, want 2", cfg.Keyboard.KeyRepeat)
	}
	if cfg.Keyboard.InitialKeyRepeat != nil {
		t.Error("InitialKeyRepeat should be nil when not specified")
	}
}
