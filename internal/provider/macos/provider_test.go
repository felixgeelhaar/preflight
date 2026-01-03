package macos

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestProvider_Name(t *testing.T) {
	provider := NewProvider(nil)
	if got := provider.Name(); got != "macos" {
		t.Errorf("Name() = %q, want %q", got, "macos")
	}
}

func TestProvider_Compile_Empty(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestProvider_Compile_NoMacOSSection(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"brew": map[string]interface{}{},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestProvider_Compile_Defaults(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"macos": map[string]interface{}{
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
					"type":   "string",
					"value":  "Always",
				},
			},
		},
	})

	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("Compile() len = %d, want 2", len(steps))
	}

	// Verify step IDs
	ids := make(map[string]bool)
	for _, s := range steps {
		ids[s.ID().String()] = true
	}
	if !ids["macos:defaults:com.apple.dock:autohide"] {
		t.Error("Missing macos:defaults:com.apple.dock:autohide step")
	}
	if !ids["macos:defaults:NSGlobalDomain:AppleShowScrollBars"] {
		t.Error("Missing macos:defaults:NSGlobalDomain:AppleShowScrollBars step")
	}
}

func TestProvider_Compile_Dock(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"macos": map[string]interface{}{
			"dock": map[string]interface{}{
				"add":    []interface{}{"Safari", "Terminal"},
				"remove": []interface{}{"FaceTime", "Maps"},
			},
		},
	})

	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 4 {
		t.Errorf("Compile() len = %d, want 4", len(steps))
	}

	ids := make(map[string]bool)
	for _, s := range steps {
		ids[s.ID().String()] = true
	}
	if !ids["macos:dock:add:Safari"] {
		t.Error("Missing dock add Safari step")
	}
	if !ids["macos:dock:add:Terminal"] {
		t.Error("Missing dock add Terminal step")
	}
	if !ids["macos:dock:remove:FaceTime"] {
		t.Error("Missing dock remove FaceTime step")
	}
	if !ids["macos:dock:remove:Maps"] {
		t.Error("Missing dock remove Maps step")
	}
}

func TestProvider_Compile_Finder(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	showHidden := true
	showExtensions := true
	showPathBar := false

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"macos": map[string]interface{}{
			"finder": map[string]interface{}{
				"show_hidden":     showHidden,
				"show_extensions": showExtensions,
				"show_path_bar":   showPathBar,
			},
		},
	})

	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 3 {
		t.Errorf("Compile() len = %d, want 3", len(steps))
	}

	ids := make(map[string]bool)
	for _, s := range steps {
		ids[s.ID().String()] = true
	}
	if !ids["macos:finder:AppleShowAllFiles"] {
		t.Error("Missing finder AppleShowAllFiles step")
	}
	if !ids["macos:finder:AppleShowAllExtensions"] {
		t.Error("Missing finder AppleShowAllExtensions step")
	}
	if !ids["macos:finder:ShowPathbar"] {
		t.Error("Missing finder ShowPathbar step")
	}
}

func TestProvider_Compile_Keyboard(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	keyRepeat := 2
	initialKeyRepeat := 15

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"macos": map[string]interface{}{
			"keyboard": map[string]interface{}{
				"key_repeat":         keyRepeat,
				"initial_key_repeat": initialKeyRepeat,
			},
		},
	})

	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("Compile() len = %d, want 2", len(steps))
	}

	ids := make(map[string]bool)
	for _, s := range steps {
		ids[s.ID().String()] = true
	}
	if !ids["macos:keyboard:KeyRepeat"] {
		t.Error("Missing keyboard KeyRepeat step")
	}
	if !ids["macos:keyboard:InitialKeyRepeat"] {
		t.Error("Missing keyboard InitialKeyRepeat step")
	}
}

func TestProvider_Compile_Mixed(t *testing.T) {
	runner := mocks.NewCommandRunner()
	provider := NewProvider(runner)

	showHidden := true

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"macos": map[string]interface{}{
			"defaults": []interface{}{
				map[string]interface{}{
					"domain": "com.apple.dock",
					"key":    "autohide",
					"type":   "bool",
					"value":  true,
				},
			},
			"dock": map[string]interface{}{
				"add": []interface{}{"Terminal"},
			},
			"finder": map[string]interface{}{
				"show_hidden": showHidden,
			},
		},
	})

	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 3 {
		t.Errorf("Compile() len = %d, want 3", len(steps))
	}
}
