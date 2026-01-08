package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// DiscoveredConfig represents a discovered configuration file or directory.
type DiscoveredConfig struct {
	// Path is the absolute path to the config file/directory
	Path string
	// HomeRelPath is the path relative to home directory
	HomeRelPath string
	// Provider is the matched provider (if any)
	Provider string
	// IsDirectory indicates if this is a directory
	IsDirectory bool
	// Size is the file size (0 for directories)
	Size int64
	// Description is a human-readable description
	Description string
}

// DiscoveryResult holds the results of config discovery.
type DiscoveryResult struct {
	// Configs are the discovered configuration files/directories
	Configs []DiscoveredConfig
	// SkippedPaths are paths that were skipped (sensitive files, etc.)
	SkippedPaths []string
}

// ConfigDiscoverer discovers configuration files in the filesystem.
type ConfigDiscoverer struct {
	fs      ports.FileSystem
	homeDir string
}

// NewConfigDiscoverer creates a new config discoverer.
func NewConfigDiscoverer(fs ports.FileSystem, homeDir string) *ConfigDiscoverer {
	return &ConfigDiscoverer{
		fs:      fs,
		homeDir: homeDir,
	}
}

// discoveryPattern defines a pattern for discovering configs.
type discoveryPattern struct {
	// Pattern is a glob pattern relative to home directory
	Pattern string
	// Provider is the provider this config belongs to
	Provider string
	// Description is a human-readable description
	Description string
	// IsDirectory indicates if this pattern matches directories
	IsDirectory bool
}

// getDiscoveryPatterns returns patterns for discovering config files.
func getDiscoveryPatterns() []discoveryPattern {
	return []discoveryPattern{
		// Shell configurations
		{Pattern: ".zshrc", Provider: "shell", Description: "Zsh configuration"},
		{Pattern: ".zshenv", Provider: "shell", Description: "Zsh environment"},
		{Pattern: ".zprofile", Provider: "shell", Description: "Zsh profile"},
		{Pattern: ".bashrc", Provider: "shell", Description: "Bash configuration"},
		{Pattern: ".bash_profile", Provider: "shell", Description: "Bash profile"},
		{Pattern: ".profile", Provider: "shell", Description: "Shell profile"},
		{Pattern: ".config/zsh", Provider: "shell", Description: "Zsh config directory", IsDirectory: true},

		// Git configurations
		{Pattern: ".gitconfig", Provider: "git", Description: "Git configuration"},
		{Pattern: ".gitconfig.d", Provider: "git", Description: "Git config directory", IsDirectory: true},
		{Pattern: ".config/git", Provider: "git", Description: "Git XDG config", IsDirectory: true},

		// SSH configurations
		{Pattern: ".ssh/config", Provider: "ssh", Description: "SSH configuration"},

		// Terminal emulators
		{Pattern: ".wezterm.lua", Provider: "terminal", Description: "WezTerm configuration"},
		{Pattern: ".config/wezterm", Provider: "terminal", Description: "WezTerm XDG config", IsDirectory: true},
		{Pattern: ".config/alacritty", Provider: "terminal", Description: "Alacritty configuration", IsDirectory: true},
		{Pattern: ".alacritty.toml", Provider: "terminal", Description: "Alacritty TOML config"},
		{Pattern: ".alacritty.yml", Provider: "terminal", Description: "Alacritty YAML config"},
		{Pattern: ".config/kitty", Provider: "terminal", Description: "Kitty configuration", IsDirectory: true},
		{Pattern: ".config/ghostty", Provider: "terminal", Description: "Ghostty configuration", IsDirectory: true},
		{Pattern: ".hyper.js", Provider: "terminal", Description: "Hyper configuration"},
		{Pattern: "Library/Preferences/com.googlecode.iterm2.plist", Provider: "terminal", Description: "iTerm2 preferences (macOS)"},
		{Pattern: "Library/Application Support/iTerm2/DynamicProfiles", Provider: "terminal", Description: "iTerm2 dynamic profiles", IsDirectory: true},
		{Pattern: "AppData/Local/Packages/Microsoft.WindowsTerminal_8wekyb3d8bbwe/LocalState/settings.json", Provider: "terminal", Description: "Windows Terminal settings"},

		// Editors
		{Pattern: ".config/nvim", Provider: "nvim", Description: "Neovim configuration", IsDirectory: true},
		{Pattern: ".vimrc", Provider: "nvim", Description: "Vim configuration"},
		{Pattern: ".vim", Provider: "nvim", Description: "Vim directory", IsDirectory: true},

		// VS Code
		{Pattern: ".config/Code/User/settings.json", Provider: "vscode", Description: "VS Code settings"},
		{Pattern: ".config/Code/User/keybindings.json", Provider: "vscode", Description: "VS Code keybindings"},
		{Pattern: "Library/Application Support/Code/User/settings.json", Provider: "vscode", Description: "VS Code settings (macOS)"},
		{Pattern: "Library/Application Support/Code/User/keybindings.json", Provider: "vscode", Description: "VS Code keybindings (macOS)"},

		// Cursor (VS Code fork)
		{Pattern: ".config/Cursor/User/settings.json", Provider: "cursor", Description: "Cursor settings"},
		{Pattern: "Library/Application Support/Cursor/User/settings.json", Provider: "cursor", Description: "Cursor settings (macOS)"},

		// Zed
		{Pattern: ".config/zed", Provider: "zed", Description: "Zed configuration", IsDirectory: true},

		// Tmux
		{Pattern: ".tmux.conf", Provider: "tmux", Description: "Tmux configuration"},
		{Pattern: ".config/tmux", Provider: "tmux", Description: "Tmux XDG config", IsDirectory: true},

		// Starship
		{Pattern: ".config/starship.toml", Provider: "starship", Description: "Starship prompt config"},

		// Docker
		{Pattern: ".docker/config.json", Provider: "docker", Description: "Docker configuration"},

		// Kubernetes
		{Pattern: ".kube/config", Provider: "kubernetes", Description: "Kubernetes configuration"},

		// AWS
		{Pattern: ".aws/config", Provider: "aws", Description: "AWS configuration"},

		// Other common configs
		{Pattern: ".editorconfig", Provider: "files", Description: "EditorConfig"},
		{Pattern: ".prettierrc", Provider: "files", Description: "Prettier configuration"},
		{Pattern: ".prettierrc.json", Provider: "files", Description: "Prettier JSON config"},
		{Pattern: ".prettierrc.yaml", Provider: "files", Description: "Prettier YAML config"},
		{Pattern: ".eslintrc", Provider: "files", Description: "ESLint configuration"},
		{Pattern: ".eslintrc.json", Provider: "files", Description: "ESLint JSON config"},

		// XDG config directories to scan
		{Pattern: ".config/bat", Provider: "files", Description: "Bat configuration", IsDirectory: true},
		{Pattern: ".config/htop", Provider: "files", Description: "htop configuration", IsDirectory: true},
		{Pattern: ".config/ripgrep", Provider: "files", Description: "ripgrep configuration", IsDirectory: true},
		{Pattern: ".config/fd", Provider: "files", Description: "fd configuration", IsDirectory: true},
		{Pattern: ".config/lazygit", Provider: "files", Description: "LazyGit configuration", IsDirectory: true},
		{Pattern: ".config/lsd", Provider: "files", Description: "lsd configuration", IsDirectory: true},
		{Pattern: ".config/bottom", Provider: "files", Description: "bottom configuration", IsDirectory: true},
	}
}

// getSensitivePatterns returns patterns for sensitive files that should be skipped.
func getSensitivePatterns() []string {
	return []string{
		// Private keys
		".ssh/id_*",
		".ssh/*.pem",
		".ssh/*.key",
		".gnupg/private-keys*",
		".gnupg/secring*",

		// Credentials and tokens
		".netrc",
		".npmrc",
		".pypirc",
		".gem/credentials",
		".docker/config.json", // May contain auth tokens
		".aws/credentials",
		".kube/config", // May contain tokens

		// History files
		".*_history",
		".zsh_history",
		".bash_history",
		".node_repl_history",
		".python_history",

		// Environment files with secrets
		".env",
		".env.*",
		"*.env.local",

		// Session/state files
		".zsh_sessions",
		".bash_sessions",
	}
}

// Discover scans the home directory for configuration files.
func (d *ConfigDiscoverer) Discover() (*DiscoveryResult, error) {
	result := &DiscoveryResult{}
	patterns := getDiscoveryPatterns()
	sensitivePatterns := getSensitivePatterns()

	for _, pattern := range patterns {
		fullPath := filepath.Join(d.homeDir, pattern.Pattern)

		// Check if path exists
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			continue // Skip on error
		}

		// Check if it's a sensitive file
		if d.isSensitive(pattern.Pattern, sensitivePatterns) {
			result.SkippedPaths = append(result.SkippedPaths, pattern.Pattern)
			continue
		}

		config := DiscoveredConfig{
			Path:        fullPath,
			HomeRelPath: pattern.Pattern,
			Provider:    pattern.Provider,
			IsDirectory: info.IsDir(),
			Description: pattern.Description,
		}

		if !info.IsDir() {
			config.Size = info.Size()
		} else {
			// Calculate directory size
			size, _ := d.calculateDirSize(fullPath)
			config.Size = size
		}

		result.Configs = append(result.Configs, config)
	}

	return result, nil
}

// DiscoverUnknown discovers config files not in the standard patterns.
// This scans ~/.config/ for directories and common dotfiles at ~/.
func (d *ConfigDiscoverer) DiscoverUnknown() (*DiscoveryResult, error) {
	result := &DiscoveryResult{}
	knownPatterns := make(map[string]bool)
	sensitivePatterns := getSensitivePatterns()

	// Build set of known patterns
	for _, p := range getDiscoveryPatterns() {
		knownPatterns[p.Pattern] = true
	}

	// Scan ~/.config/ for unknown directories
	configDir := filepath.Join(d.homeDir, ".config")
	if entries, err := os.ReadDir(configDir); err == nil {
		for _, entry := range entries {
			relPath := filepath.Join(".config", entry.Name())
			if knownPatterns[relPath] {
				continue
			}
			if d.isSensitive(relPath, sensitivePatterns) {
				result.SkippedPaths = append(result.SkippedPaths, relPath)
				continue
			}

			fullPath := filepath.Join(d.homeDir, relPath)
			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			config := DiscoveredConfig{
				Path:        fullPath,
				HomeRelPath: relPath,
				Provider:    "files", // Default to files provider
				IsDirectory: info.IsDir(),
				Description: "Unknown config: " + entry.Name(),
			}

			if !info.IsDir() {
				config.Size = info.Size()
			} else {
				size, _ := d.calculateDirSize(fullPath)
				config.Size = size
			}

			result.Configs = append(result.Configs, config)
		}
	}

	// Scan home directory for dotfiles (.*rc, .*config, etc.)
	if entries, err := os.ReadDir(d.homeDir); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			// Only look for dotfiles
			if !strings.HasPrefix(name, ".") {
				continue
			}
			// Skip known patterns
			if knownPatterns[name] {
				continue
			}
			// Skip common non-config directories
			if d.isCommonNonConfig(name) {
				continue
			}
			if d.isSensitive(name, sensitivePatterns) {
				result.SkippedPaths = append(result.SkippedPaths, name)
				continue
			}

			// Only include likely config files
			if !d.looksLikeConfig(name) {
				continue
			}

			fullPath := filepath.Join(d.homeDir, name)
			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			config := DiscoveredConfig{
				Path:        fullPath,
				HomeRelPath: name,
				Provider:    "files",
				IsDirectory: info.IsDir(),
				Description: "Unknown dotfile: " + name,
			}

			if !info.IsDir() {
				config.Size = info.Size()
			}

			result.Configs = append(result.Configs, config)
		}
	}

	return result, nil
}

// isSensitive checks if a path matches sensitive patterns.
func (d *ConfigDiscoverer) isSensitive(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		// Also check base name
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}

// isCommonNonConfig checks if a name is a common non-config directory.
func (d *ConfigDiscoverer) isCommonNonConfig(name string) bool {
	nonConfig := []string{
		".cache",
		".local",
		".Trash",
		".npm",
		".yarn",
		".pnpm",
		".cargo",
		".rustup",
		".go",
		".gradle",
		".m2",
		".ivy2",
		".sbt",
		".coursier",
		".nvm",
		".pyenv",
		".rbenv",
		".asdf",
		".mise",
		".volta",
		".git",
		".hg",
		".svn",
		".DS_Store",
		".Spotlight-V100",
		".fseventsd",
		".TemporaryItems",
		".Trashes",
		".VolumeIcon.icns",
		".localized",
	}

	for _, nc := range nonConfig {
		if name == nc {
			return true
		}
	}
	return false
}

// looksLikeConfig checks if a filename looks like a configuration file.
func (d *ConfigDiscoverer) looksLikeConfig(name string) bool {
	// Common config file patterns
	configPatterns := []string{
		"*rc",
		"*config",
		"*.conf",
		"*.cfg",
		"*.json",
		"*.yaml",
		"*.yml",
		"*.toml",
		"*.ini",
	}

	for _, pattern := range configPatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

// calculateDirSize calculates the total size of a directory.
func (d *ConfigDiscoverer) calculateDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // Intentionally skip walk errors for non-accessible files
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// ToCapture converts discovered configs to capture configurations.
func (r *DiscoveryResult) ToCapture() []DotfilesCaptureConfig {
	byProvider := make(map[string][]string)

	for _, config := range r.Configs {
		provider := config.Provider
		if provider == "" {
			provider = "files"
		}
		// Convert to home-relative path with ~ prefix
		path := "~/" + config.HomeRelPath
		byProvider[provider] = append(byProvider[provider], path)
	}

	configs := make([]DotfilesCaptureConfig, 0, len(byProvider))
	for provider, paths := range byProvider {
		configs = append(configs, DotfilesCaptureConfig{
			Provider:    provider,
			SourcePaths: paths,
		})
	}

	return configs
}
