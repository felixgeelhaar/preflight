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
	}
}

// Capture captures dotfiles for all configured providers.
func (c *DotfilesCapturer) Capture() (*DotfilesCaptureResult, error) {
	result := &DotfilesCaptureResult{
		TargetDir: c.getDotfilesDir(),
		Target:    c.target,
	}

	for _, cfg := range c.configs {
		dotfiles, warnings, err := c.captureProvider(cfg)
		if err != nil {
			return nil, err
		}
		result.Dotfiles = append(result.Dotfiles, dotfiles...)
		result.Warnings = append(result.Warnings, warnings...)
	}

	return result, nil
}

// CaptureProvider captures dotfiles for a specific provider.
func (c *DotfilesCapturer) CaptureProvider(provider string) (*DotfilesCaptureResult, error) {
	for _, cfg := range c.configs {
		if cfg.Provider == provider {
			dotfiles, warnings, err := c.captureProvider(cfg)
			if err != nil {
				return nil, err
			}
			return &DotfilesCaptureResult{
				Dotfiles:  dotfiles,
				TargetDir: c.getDotfilesDir(),
				Target:    c.target,
				Warnings:  warnings,
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

// captureProvider captures dotfiles for a single provider configuration.
func (c *DotfilesCapturer) captureProvider(cfg DotfilesCaptureConfig) ([]CapturedDotfile, []string, error) {
	var dotfiles []CapturedDotfile
	var warnings []string

	destDir := filepath.Join(c.getDotfilesDir(), cfg.TargetDir)

	for _, sourcePath := range cfg.SourcePaths {
		// Expand ~ to home directory
		expandedPath := c.expandPath(sourcePath)

		// Check if source exists
		info, err := os.Stat(expandedPath)
		if os.IsNotExist(err) {
			continue // Skip non-existent paths
		}
		if err != nil {
			warnings = append(warnings, "failed to stat "+sourcePath+": "+err.Error())
			continue
		}

		if info.IsDir() {
			// Capture directory recursively
			captured, err := c.captureDirectory(cfg.Provider, expandedPath, destDir, cfg.ExcludePaths)
			if err != nil {
				return nil, nil, err
			}
			dotfiles = append(dotfiles, captured...)
		} else {
			// Capture single file
			captured, err := c.captureFile(cfg.Provider, expandedPath, destDir, filepath.Base(expandedPath))
			if err != nil {
				return nil, nil, err
			}
			if captured != nil {
				dotfiles = append(dotfiles, *captured)
			}
		}
	}

	return dotfiles, warnings, nil
}

// captureDirectory recursively captures a directory.
func (c *DotfilesCapturer) captureDirectory(provider, sourceDir, destDir string, excludes []string) ([]CapturedDotfile, error) {
	var dotfiles []CapturedDotfile

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, err
	}

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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

	return dotfiles, err
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
