package app

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// DotfilesCaptureConfig defines what to capture for each provider.
type DotfilesCaptureConfig struct {
	// Provider is the name of the provider (nvim, shell, etc.)
	Provider string
	// SourcePaths are the paths to capture (supports globs, ~ expanded)
	SourcePaths []string
	// ExcludePaths are patterns to exclude (supports globs)
	ExcludePaths []string
	// TargetDir is the subdirectory under dotfiles/
	TargetDir string
}

// CapturedDotfile represents a captured dotfile entry.
type CapturedDotfile struct {
	// Provider is the provider that owns this dotfile
	Provider string
	// SourcePath is the original path on the source machine
	SourcePath string
	// RelativePath is the path relative to the provider's target dir
	RelativePath string
	// DestPath is the full path under dotfiles/
	DestPath string
	// IsDirectory indicates if this is a directory
	IsDirectory bool
	// Size is the file size in bytes (0 for directories)
	Size int64
}

// BrokenSymlink represents a broken symlink encountered during capture.
type BrokenSymlink struct {
	Path   string
	Target string
}

// DotfilesCaptureResult holds the results of a dotfiles capture.
type DotfilesCaptureResult struct {
	// Dotfiles are the captured dotfile entries
	Dotfiles []CapturedDotfile
	// TargetDir is the root directory where dotfiles were captured
	TargetDir string
	// Target is the target name (for per-target directories)
	Target string
	// Warnings are any warnings encountered during capture
	Warnings []string
	// BrokenSymlinks are symlinks that point to non-existent targets
	BrokenSymlinks []BrokenSymlink
}

// TotalSize returns the total size of all captured dotfiles.
func (r DotfilesCaptureResult) TotalSize() int64 {
	var total int64
	for _, d := range r.Dotfiles {
		total += d.Size
	}
	return total
}

// FileCount returns the number of files captured (excluding directories).
func (r DotfilesCaptureResult) FileCount() int {
	count := 0
	for _, d := range r.Dotfiles {
		if !d.IsDirectory {
			count++
		}
	}
	return count
}

// DotfilesByProvider returns dotfiles grouped by provider.
func (r DotfilesCaptureResult) DotfilesByProvider() map[string][]CapturedDotfile {
	result := make(map[string][]CapturedDotfile)
	for _, d := range r.Dotfiles {
		result[d.Provider] = append(result[d.Provider], d)
	}
	return result
}

// DotfilesCapturer handles copying config files to dotfiles/.
type DotfilesCapturer struct {
	fs        ports.FileSystem
	homeDir   string
	targetDir string
	target    string
	configs   []DotfilesCaptureConfig
}

// NewDotfilesCapturer creates a new dotfiles capturer.
func NewDotfilesCapturer(fs ports.FileSystem, homeDir, targetDir string) *DotfilesCapturer {
	return &DotfilesCapturer{
		fs:        fs,
		homeDir:   homeDir,
		targetDir: targetDir,
		configs:   DefaultCaptureConfigs(),
	}
}

// WithTarget sets the target for per-target dotfiles.
func (c *DotfilesCapturer) WithTarget(target string) *DotfilesCapturer {
	c.target = target
	return c
}

// WithConfigs sets custom capture configurations.
func (c *DotfilesCapturer) WithConfigs(configs []DotfilesCaptureConfig) *DotfilesCapturer {
	c.configs = configs
	return c
}

// DefaultCaptureConfigs returns the default capture configurations for each provider.
func DefaultCaptureConfigs() []DotfilesCaptureConfig {
	return []DotfilesCaptureConfig{
		{
			Provider: "nvim",
			SourcePaths: []string{
				"~/.config/nvim",
			},
			ExcludePaths: []string{
				"lazy",
				"lazy-lock.json", // Will be regenerated
				"pack",
				".git",
				"*.swp",
				"*.swo",
				"*~",
			},
			TargetDir: "nvim",
		},
		{
			Provider: "shell",
			SourcePaths: []string{
				"~/.zshrc",
				"~/.zshenv",
				"~/.zprofile",
				"~/.zlogin",
				"~/.zlogout",
				"~/.zshrc.d",
				"~/.config/zsh",
				"~/.zsh",
			},
			ExcludePaths: []string{
				".zcompdump*",
				"*.zwc",
				".zsh_history",
				".zsh_sessions",
			},
			TargetDir: "shell",
		},
		{
			Provider: "starship",
			SourcePaths: []string{
				"~/.config/starship.toml",
			},
			TargetDir: "starship",
		},
		{
			Provider: "tmux",
			SourcePaths: []string{
				"~/.tmux.conf",
				"~/.config/tmux",
			},
			ExcludePaths: []string{
				"plugins",
				"resurrect",
			},
			TargetDir: "tmux",
		},
		{
			Provider: "vscode",
			SourcePaths: []string{
				"~/Library/Application Support/Code/User/settings.json",
				"~/Library/Application Support/Code/User/keybindings.json",
				"~/.config/Code/User/settings.json",
				"~/.config/Code/User/keybindings.json",
			},
			TargetDir: "vscode",
		},
		{
			Provider: "ssh",
			SourcePaths: []string{
				"~/.ssh/config",
			},
			ExcludePaths: []string{
				"id_*",
				"*.pem",
				"*.key",
				"known_hosts",
				"authorized_keys",
			},
			TargetDir: "ssh",
		},
		{
			Provider: "git",
			SourcePaths: []string{
				"~/.gitconfig.d",
				"~/.config/git",
			},
			ExcludePaths: []string{
				"credentials",
			},
			TargetDir: "git",
		},
		{
			Provider: "terminal",
			SourcePaths: []string{
				// WezTerm
				"~/.wezterm.lua",
				"~/.config/wezterm",
				// Alacritty
				"~/.alacritty.toml",
				"~/.alacritty.yml",
				"~/.config/alacritty",
				// Kitty
				"~/.config/kitty",
				// Hyper
				"~/.hyper.js",
				"~/.config/hyper",
				// iTerm2 (macOS)
				"~/Library/Preferences/com.googlecode.iterm2.plist",
				// Ghostty
				"~/.config/ghostty",
			},
			ExcludePaths: []string{
				"*.log",
				"cache",
			},
			TargetDir: "terminal",
		},
	}
}

// Capture captures dotfiles for all configured providers.
func (c *DotfilesCapturer) Capture() (*DotfilesCaptureResult, error) {
	result := &DotfilesCaptureResult{
		TargetDir: c.getDotfilesDir(),
		Target:    c.target,
	}

	for _, cfg := range c.configs {
		providerResult, err := c.captureProvider(cfg)
		if err != nil {
			return nil, err
		}
		result.Dotfiles = append(result.Dotfiles, providerResult.dotfiles...)
		result.Warnings = append(result.Warnings, providerResult.warnings...)
		result.BrokenSymlinks = append(result.BrokenSymlinks, providerResult.brokenSymlinks...)
	}

	return result, nil
}

// CaptureProvider captures dotfiles for a specific provider.
func (c *DotfilesCapturer) CaptureProvider(provider string) (*DotfilesCaptureResult, error) {
	for _, cfg := range c.configs {
		if cfg.Provider == provider {
			providerResult, err := c.captureProvider(cfg)
			if err != nil {
				return nil, err
			}
			return &DotfilesCaptureResult{
				Dotfiles:       providerResult.dotfiles,
				TargetDir:      c.getDotfilesDir(),
				Target:         c.target,
				Warnings:       providerResult.warnings,
				BrokenSymlinks: providerResult.brokenSymlinks,
			}, nil
		}
	}
	return &DotfilesCaptureResult{
		TargetDir: c.getDotfilesDir(),
		Target:    c.target,
	}, nil
}

// getDotfilesDir returns the dotfiles directory, considering per-target support.
func (c *DotfilesCapturer) getDotfilesDir() string {
	if c.target != "" {
		return filepath.Join(c.targetDir, "dotfiles."+c.target)
	}
	return filepath.Join(c.targetDir, "dotfiles")
}

// captureProviderResult holds results from capturing a single provider.
type captureProviderResult struct {
	dotfiles       []CapturedDotfile
	warnings       []string
	brokenSymlinks []BrokenSymlink
}

// captureProvider captures dotfiles for a single provider configuration.
func (c *DotfilesCapturer) captureProvider(cfg DotfilesCaptureConfig) (*captureProviderResult, error) {
	result := &captureProviderResult{}

	destDir := filepath.Join(c.getDotfilesDir(), cfg.TargetDir)

	for _, sourcePath := range cfg.SourcePaths {
		// Expand ~ to home directory
		expandedPath := c.expandPath(sourcePath)

		// Use Lstat to not follow symlinks - detect broken symlinks at source level
		info, err := os.Lstat(expandedPath)
		if os.IsNotExist(err) {
			continue // Skip non-existent paths
		}
		if err != nil {
			result.warnings = append(result.warnings, "failed to stat "+sourcePath+": "+err.Error())
			continue
		}

		// Check if source path itself is a symlink
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(expandedPath)
			if err != nil {
				result.warnings = append(result.warnings, "failed to read symlink "+sourcePath+": "+err.Error())
				continue
			}
			// Check if target exists and resolve the symlink
			resolvedPath, err := filepath.EvalSymlinks(expandedPath)
			if err != nil {
				result.brokenSymlinks = append(result.brokenSymlinks, BrokenSymlink{
					Path:   expandedPath,
					Target: target,
				})
				continue
			}
			// Valid symlink - use resolved path and get info of target
			expandedPath = resolvedPath
			info, err = os.Stat(expandedPath)
			if err != nil {
				result.warnings = append(result.warnings, "failed to stat symlink target "+sourcePath+": "+err.Error())
				continue
			}
		}

		if info.IsDir() {
			// Capture directory recursively
			captured, brokenLinks, err := c.captureDirectory(cfg.Provider, expandedPath, destDir, cfg.ExcludePaths)
			if err != nil {
				return nil, err
			}
			result.dotfiles = append(result.dotfiles, captured...)
			result.brokenSymlinks = append(result.brokenSymlinks, brokenLinks...)
		} else {
			// Capture single file
			captured, err := c.captureFile(cfg.Provider, expandedPath, destDir, filepath.Base(expandedPath))
			if err != nil {
				return nil, err
			}
			if captured != nil {
				result.dotfiles = append(result.dotfiles, *captured)
			}
		}
	}

	return result, nil
}

// captureDirectory recursively captures a directory.
func (c *DotfilesCapturer) captureDirectory(provider, sourceDir, destDir string, excludes []string) ([]CapturedDotfile, []BrokenSymlink, error) {
	var dotfiles []CapturedDotfile
	var brokenSymlinks []BrokenSymlink

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, nil, err
	}

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		// Handle walk errors - often caused by broken symlinks
		if walkErr != nil {
			// Check if this is a broken symlink
			linfo, lstatErr := os.Lstat(path)
			if lstatErr == nil && linfo.Mode()&os.ModeSymlink != 0 {
				// It's a symlink that couldn't be followed - broken symlink
				target, _ := os.Readlink(path)
				brokenSymlinks = append(brokenSymlinks, BrokenSymlink{
					Path:   path,
					Target: target,
				})
				return nil // Skip and continue
			}
			// Not a symlink issue, propagate the error
			return walkErr
		}

		// Get relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Check if path matches any exclude pattern
		if c.shouldExclude(relPath, excludes) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if this is a symlink
		linfo, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if linfo.Mode()&os.ModeSymlink != 0 {
			// It's a symlink - check if it's valid
			target, readErr := os.Readlink(path)
			if readErr != nil {
				brokenSymlinks = append(brokenSymlinks, BrokenSymlink{Path: path, Target: ""})
				return nil //nolint:nilerr // intentionally skipping broken symlinks
			}
			_, evalErr := filepath.EvalSymlinks(path)
			if evalErr != nil {
				// Broken symlink - skip it
				brokenSymlinks = append(brokenSymlinks, BrokenSymlink{
					Path:   path,
					Target: target,
				})
				return nil //nolint:nilerr // intentionally skipping broken symlinks
			}
			// Valid symlink - get the actual info
			var statErr error
			info, statErr = os.Stat(path)
			if statErr != nil {
				brokenSymlinks = append(brokenSymlinks, BrokenSymlink{
					Path:   path,
					Target: target,
				})
				return nil //nolint:nilerr // intentionally skipping broken symlinks
			}
		}

		destPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			// Create directory
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
			dotfiles = append(dotfiles, CapturedDotfile{
				Provider:     provider,
				SourcePath:   path,
				RelativePath: relPath,
				DestPath:     destPath,
				IsDirectory:  true,
			})
		} else {
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			// Copy file
			if err := c.copyFile(path, destPath); err != nil {
				return err
			}
			dotfiles = append(dotfiles, CapturedDotfile{
				Provider:     provider,
				SourcePath:   path,
				RelativePath: relPath,
				DestPath:     destPath,
				IsDirectory:  false,
				Size:         info.Size(),
			})
		}

		return nil
	})

	return dotfiles, brokenSymlinks, err
}

// captureFile captures a single file.
func (c *DotfilesCapturer) captureFile(provider, sourcePath, destDir, fileName string) (*CapturedDotfile, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}

	destPath := filepath.Join(destDir, fileName)

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, err
	}

	// Copy file
	if err := c.copyFile(sourcePath, destPath); err != nil {
		return nil, err
	}

	return &CapturedDotfile{
		Provider:     provider,
		SourcePath:   sourcePath,
		RelativePath: fileName,
		DestPath:     destPath,
		IsDirectory:  false,
		Size:         info.Size(),
	}, nil
}

// shouldExclude checks if a path matches any exclude pattern.
func (c *DotfilesCapturer) shouldExclude(path string, excludes []string) bool {
	for _, pattern := range excludes {
		// Check if the pattern matches the full path or any component
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}
		// Also check against full relative path
		matched, _ = filepath.Match(pattern, path)
		if matched {
			return true
		}
		// Check if any path component matches
		parts := strings.Split(path, string(filepath.Separator))
		for _, part := range parts {
			matched, _ = filepath.Match(pattern, part)
			if matched {
				return true
			}
		}
	}
	return false
}

// copyFile copies a file from src to dst.
func (c *DotfilesCapturer) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	// Get source file info for permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	destFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	// Copy contents
	_, err = io.Copy(destFile, sourceFile)
	return err
}

// expandPath expands ~ to the home directory.
func (c *DotfilesCapturer) expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(c.homeDir, path[2:])
	}
	if path == "~" {
		return c.homeDir
	}
	return path
}
