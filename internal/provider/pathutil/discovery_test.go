package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigFinder_GetCandidatePaths(t *testing.T) {
	// Clear XDG_CONFIG_HOME to use default
	t.Setenv("XDG_CONFIG_HOME", "")

	finder := NewConfigFinderWithHome("/home/testuser", "linux")

	opts := ConfigSearchOpts{
		EnvVar:         "TEST_CONFIG_DIR",
		ConfigFileName: "config.toml",
		XDGSubpath:     "testapp/config.toml",
		LinuxPaths:     []string{"~/.testapp.conf"},
		LegacyPaths:    []string{"~/.testapprc"},
	}

	// Without env var set
	paths := finder.GetCandidatePaths(opts)

	// Should include XDG path and legacy paths
	if len(paths) < 2 {
		t.Errorf("Expected at least 2 paths, got %d", len(paths))
	}

	// First path should be XDG based
	expectedXDG := "/home/testuser/.config/testapp/config.toml"
	if paths[0] != expectedXDG {
		t.Errorf("Expected first path %s, got %s", expectedXDG, paths[0])
	}
}

func TestConfigFinder_GetCandidatePaths_WithEnvVar(t *testing.T) {
	finder := NewConfigFinderWithHome("/home/testuser", "linux")

	// Set env var
	t.Setenv("TEST_CONFIG_DIR", "/custom/config")

	opts := ConfigSearchOpts{
		EnvVar:         "TEST_CONFIG_DIR",
		ConfigFileName: "config.toml",
		XDGSubpath:     "testapp/config.toml",
	}

	paths := finder.GetCandidatePaths(opts)

	// First path should be from env var
	expected := "/custom/config/config.toml"
	if len(paths) == 0 || paths[0] != expected {
		t.Errorf("Expected first path %s, got %v", expected, paths)
	}
}

func TestConfigFinder_GetCandidatePaths_XDGConfigHome(t *testing.T) {
	finder := NewConfigFinderWithHome("/home/testuser", "linux")

	// Set custom XDG_CONFIG_HOME
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")

	opts := ConfigSearchOpts{
		XDGSubpath: "testapp/config.toml",
	}

	paths := finder.GetCandidatePaths(opts)

	expected := "/custom/xdg/testapp/config.toml"
	if len(paths) == 0 || paths[0] != expected {
		t.Errorf("Expected path %s, got %v", expected, paths)
	}
}

func TestConfigFinder_BestPracticePath(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		opts     ConfigSearchOpts
		expected string
	}{
		{
			name: "XDG path preferred on Linux",
			goos: "linux",
			opts: ConfigSearchOpts{
				XDGSubpath: "alacritty/alacritty.toml",
				LinuxPaths: []string{"~/.alacritty.toml"},
			},
			expected: "/home/testuser/.config/alacritty/alacritty.toml",
		},
		{
			name: "XDG path preferred on macOS",
			goos: "darwin",
			opts: ConfigSearchOpts{
				XDGSubpath: "alacritty/alacritty.toml",
				MacOSPaths: []string{"~/.alacritty.toml"},
			},
			expected: "/home/testuser/.config/alacritty/alacritty.toml",
		},
		{
			name: "Platform path when no XDG",
			goos: "darwin",
			opts: ConfigSearchOpts{
				MacOSPaths: []string{"~/Library/Preferences/com.app.plist"},
			},
			expected: "/home/testuser/Library/Preferences/com.app.plist",
		},
		{
			name: "Legacy fallback",
			goos: "linux",
			opts: ConfigSearchOpts{
				LegacyPaths: []string{"~/.hyper.js"},
			},
			expected: "/home/testuser/.hyper.js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finder := NewConfigFinderWithHome("/home/testuser", tt.goos)
			result := finder.BestPracticePath(tt.opts)
			if result != tt.expected {
				t.Errorf("BestPracticePath() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestConfigFinder_FindConfig(t *testing.T) {
	// Clear XDG_CONFIG_HOME to use default (home/.config)
	t.Setenv("XDG_CONFIG_HOME", "")

	// Create temp directory with test files
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "testapp")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	finder := NewConfigFinderWithHome(tmpDir, "linux")

	opts := ConfigSearchOpts{
		XDGSubpath:  "testapp/config.toml",
		LegacyPaths: []string{"~/.testapp.conf"},
	}

	result := finder.FindConfig(opts)

	if result != configFile {
		t.Errorf("FindConfig() = %s, want %s", result, configFile)
	}
}

func TestConfigFinder_FindConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	finder := NewConfigFinderWithHome(tmpDir, "linux")

	opts := ConfigSearchOpts{
		XDGSubpath:  "nonexistent/config.toml",
		LegacyPaths: []string{"~/.nonexistent.conf"},
	}

	result := finder.FindConfig(opts)

	if result != "" {
		t.Errorf("FindConfig() = %s, want empty string", result)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/.config/test", filepath.Join(home, ".config/test")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		result := ExpandPath(tt.input)
		if result != tt.expected {
			t.Errorf("ExpandPath(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	tmpFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{tmpDir, true},
		{tmpFile, false},
		{"/nonexistent/path", false},
	}

	for _, tt := range tests {
		result := DirExists(tt.path)
		if result != tt.expected {
			t.Errorf("DirExists(%s) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}
