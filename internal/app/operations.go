package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/felixgeelhaar/preflight/internal/adapters/command"
	"github.com/felixgeelhaar/preflight/internal/adapters/github"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/nvim"
	"github.com/felixgeelhaar/preflight/internal/templates"
	"github.com/felixgeelhaar/preflight/internal/validation"
)

// Capture discovers current machine configuration.
func (p *Preflight) Capture(ctx context.Context, opts CaptureOptions) (*CaptureFindings, error) {
	findings := &CaptureFindings{
		CapturedAt: time.Now(),
		HomeDir:    opts.HomeDir,
		Items:      make([]CapturedItem, 0),
		Providers:  make([]string, 0),
	}

	if findings.HomeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		findings.HomeDir = home
	}

	// Determine which providers to capture
	providers := opts.Providers
	if len(providers) == 0 {
		providers = defaultCaptureProviders()
	}

	for _, provider := range providers {
		findings.Providers = append(findings.Providers, provider)

		items, err := p.captureProvider(ctx, provider, findings.HomeDir, opts.IncludeSecrets)
		if err != nil {
			findings.Warnings = append(findings.Warnings, fmt.Sprintf("%s: %v", provider, err))
			continue
		}

		findings.Items = append(findings.Items, items...)
	}

	return findings, nil
}

func defaultCaptureProviders() []string {
	common := []string{
		"git",
		"ssh",
		"shell",
		"nvim",
		"vscode",
		"runtime",
		"npm",
		"go",
		"pip",
		"gem",
		"cargo",
	}

	plat, err := platform.Detect()
	if err != nil || plat == nil {
		return append([]string{"brew", "apt", "winget", "chocolatey", "scoop"}, common...)
	}

	switch plat.OS() {
	case platform.OSDarwin:
		return append([]string{"brew"}, common...)
	case platform.OSLinux:
		return append([]string{"apt"}, common...)
	case platform.OSWindows:
		return append([]string{"winget", "chocolatey", "scoop"}, common...)
	default:
		return append([]string{"brew", "apt", "winget", "chocolatey", "scoop"}, common...)
	}
}

func (p *Preflight) captureProvider(ctx context.Context, provider, homeDir string, includeSecrets bool) ([]CapturedItem, error) {
	now := time.Now()
	var items []CapturedItem

	switch provider {
	case "brew":
		items = p.captureBrewFormulae(ctx, now)
	case "git":
		items = p.captureGitConfig(homeDir, now)
	case "ssh":
		items = p.captureSSHConfig(homeDir, now, includeSecrets)
	case "shell":
		items = p.captureShellConfig(homeDir, now)
	case "nvim":
		items = p.captureNvimConfig(homeDir, now)
	case "vscode":
		items = p.captureVSCodeExtensions(ctx, now)
	case "runtime":
		items = p.captureRuntimeVersions(ctx, now)
	// Windows package managers
	case "chocolatey":
		items = p.captureChocolateyPackages(ctx, now)
	case "scoop":
		items = p.captureScoopPackages(ctx, now)
	case "winget":
		items = p.captureWingetPackages(ctx, now)
	// Linux package managers
	case "apt":
		items = p.captureAPTPackages(ctx, now)
	// Language package managers
	case "npm":
		items = p.captureNpmGlobals(ctx, now)
	case "go":
		items = p.captureGoTools(ctx, now)
	case "pip":
		items = p.capturePipPackages(ctx, now)
	case "gem":
		items = p.captureGemPackages(ctx, now)
	case "cargo":
		items = p.captureCargoCrates(ctx, now)
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	return items, nil
}

func (p *Preflight) captureBrewFormulae(_ context.Context, capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	// Capture formulae
	cmd := exec.Command("brew", "list", "--formula", "-1")
	output, err := cmd.Output()
	if err == nil {
		formulae := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, f := range formulae {
			if f == "" {
				continue
			}
			items = append(items, CapturedItem{
				Provider:   "brew",
				Name:       f,
				Value:      f,
				Source:     "brew list --formula",
				CapturedAt: capturedAt,
			})
		}
	}

	// Capture casks
	cmd = exec.Command("brew", "list", "--cask", "-1")
	output, err = cmd.Output()
	if err == nil {
		casks := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, c := range casks {
			if c == "" {
				continue
			}
			items = append(items, CapturedItem{
				Provider:   "brew-cask",
				Name:       c,
				Value:      c,
				Source:     "brew list --cask",
				CapturedAt: capturedAt,
			})
		}
	}

	return items
}

func (p *Preflight) captureGitConfig(homeDir string, capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	gitconfigPath := filepath.Join(homeDir, ".gitconfig")
	if _, err := os.Stat(gitconfigPath); err == nil {
		// Read key config values
		keys := []string{"user.name", "user.email", "core.editor", "init.defaultBranch"}
		for _, key := range keys {
			// #nosec G204 -- key is from a fixed allowlist.
			cmd := exec.Command("git", "config", "--global", key)
			output, err := cmd.Output()
			if err == nil {
				items = append(items, CapturedItem{
					Provider:   "git",
					Name:       key,
					Value:      strings.TrimSpace(string(output)),
					Source:     gitconfigPath,
					CapturedAt: capturedAt,
				})
			}
		}
	}

	return items
}

func (p *Preflight) captureSSHConfig(homeDir string, capturedAt time.Time, _ bool) []CapturedItem {
	var items []CapturedItem

	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
	if _, err := os.Stat(sshConfigPath); err == nil {
		items = append(items, CapturedItem{
			Provider:   "ssh",
			Name:       "config",
			Value:      sshConfigPath,
			Source:     sshConfigPath,
			CapturedAt: capturedAt,
		})
	}

	return items
}

func (p *Preflight) captureShellConfig(homeDir string, capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	shellFiles := []string{".zshrc", ".bashrc", ".bash_profile"}
	for _, file := range shellFiles {
		path := filepath.Join(homeDir, file)
		if _, err := os.Stat(path); err == nil {
			items = append(items, CapturedItem{
				Provider:   "shell",
				Name:       file,
				Value:      path,
				Source:     path,
				CapturedAt: capturedAt,
			})
		}
	}

	return items
}

func (p *Preflight) captureNvimConfig(homeDir string, capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	if version := captureCommandVersion("nvim", "--version"); version != "" {
		items = append(items, CapturedItem{
			Provider:   "nvim",
			Name:       "version",
			Value:      version,
			Source:     "nvim --version",
			CapturedAt: capturedAt,
		})
	}

	// Check for Neovim configuration directory
	nvimConfigDir := filepath.Join(homeDir, ".config", "nvim")
	if info, err := os.Stat(nvimConfigDir); err == nil && info.IsDir() {
		items = append(items, CapturedItem{
			Provider:   "nvim",
			Name:       "config",
			Value:      nvimConfigDir,
			Source:     nvimConfigDir,
			CapturedAt: capturedAt,
		})

		// Check for lazy-lock.json (Lazy.nvim plugin manager)
		lazyLockPath := filepath.Join(nvimConfigDir, "lazy-lock.json")
		if _, err := os.Stat(lazyLockPath); err == nil {
			items = append(items, CapturedItem{
				Provider:   "nvim",
				Name:       "lazy-lock.json",
				Value:      lazyLockPath,
				Source:     lazyLockPath,
				CapturedAt: capturedAt,
			})
		}

		// Check for packer compiled (Packer plugin manager)
		packerPath := filepath.Join(nvimConfigDir, "plugin", "packer_compiled.lua")
		if _, err := os.Stat(packerPath); err == nil {
			items = append(items, CapturedItem{
				Provider:   "nvim",
				Name:       "packer_compiled.lua",
				Value:      packerPath,
				Source:     packerPath,
				CapturedAt: capturedAt,
			})
		}
	}

	// Also check for init.vim in legacy location
	legacyInitVim := filepath.Join(homeDir, ".vimrc")
	if _, err := os.Stat(legacyInitVim); err == nil {
		items = append(items, CapturedItem{
			Provider:   "nvim",
			Name:       ".vimrc",
			Value:      legacyInitVim,
			Source:     legacyInitVim,
			CapturedAt: capturedAt,
		})
	}

	return items
}

func (p *Preflight) captureVSCodeExtensions(_ context.Context, capturedAt time.Time) []CapturedItem {
	items := make([]CapturedItem, 0, 8) // Pre-allocate for version + extensions

	if version := captureCommandVersion("code", "--version"); version != "" {
		items = append(items, CapturedItem{
			Provider:   "vscode",
			Name:       "version",
			Value:      version,
			Source:     "code --version",
			CapturedAt: capturedAt,
		})
	}

	// Try to list installed extensions
	cmd := exec.Command("code", "--list-extensions")
	output, err := cmd.Output()
	if err != nil {
		// VS Code not installed or command failed
		return items
	}

	extensions := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, ext := range extensions {
		if ext == "" {
			continue
		}
		items = append(items, CapturedItem{
			Provider:   "vscode",
			Name:       ext,
			Value:      ext,
			Source:     "code --list-extensions",
			CapturedAt: capturedAt,
		})
	}

	return items
}

func (p *Preflight) captureRuntimeVersions(_ context.Context, capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	items = append(items, p.captureRuntimeManagerVersions(capturedAt)...)

	// Try mise (formerly rtx) first
	if miseItems := p.captureMiseVersions("mise", capturedAt); len(miseItems) > 0 {
		items = append(items, miseItems...)
		return items
	}

	// Try rtx for older setups
	if rtxItems := p.captureMiseVersions("rtx", capturedAt); len(rtxItems) > 0 {
		items = append(items, rtxItems...)
		return items
	}

	// Fall back to asdf
	if asdfItems := p.captureAsdfVersions(capturedAt); len(asdfItems) > 0 {
		items = append(items, asdfItems...)
	}

	return items
}

func (p *Preflight) captureRuntimeManagerVersions(capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	if version := captureCommandVersion("mise", "--version"); version != "" {
		items = append(items, CapturedItem{
			Provider:   "runtime",
			Name:       "mise",
			Value:      version,
			Source:     "mise --version",
			CapturedAt: capturedAt,
		})
	}

	if version := captureCommandVersion("rtx", "--version"); version != "" {
		items = append(items, CapturedItem{
			Provider:   "runtime",
			Name:       "rtx",
			Value:      version,
			Source:     "rtx --version",
			CapturedAt: capturedAt,
		})
	}

	if version := captureCommandVersion("asdf", "--version"); version != "" {
		items = append(items, CapturedItem{
			Provider:   "runtime",
			Name:       "asdf",
			Value:      version,
			Source:     "asdf --version",
			CapturedAt: capturedAt,
		})
	}

	return items
}

func (p *Preflight) captureMiseVersions(commandName string, capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	// Try 'mise list' command
	cmd := exec.Command(commandName, "list", "--current")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Parse mise output format: tool@version
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			items = append(items, CapturedItem{
				Provider:   "runtime",
				Name:       fields[0],
				Value:      fields[1],
				Source:     fmt.Sprintf("%s list", commandName),
				CapturedAt: capturedAt,
			})
		}
	}

	return items
}

func (p *Preflight) captureAsdfVersions(capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	// Try 'asdf current' command
	cmd := exec.Command("asdf", "current")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Parse asdf output format: tool  version  (source)
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			items = append(items, CapturedItem{
				Provider:   "runtime",
				Name:       fields[0],
				Value:      fields[1],
				Source:     "asdf current",
				CapturedAt: capturedAt,
			})
		}
	}

	return items
}

func captureCommandVersion(command string, args ...string) string {
	output, err := exec.Command(command, args...).Output()
	if err != nil {
		return ""
	}
	return parseVersionOutput(output)
}

func parseVersionOutput(output []byte) string {
	line := strings.TrimSpace(string(output))
	if line == "" {
		return ""
	}

	firstLine := strings.SplitN(line, "\n", 2)[0]
	for _, field := range strings.Fields(firstLine) {
		cleaned := strings.Trim(field, ",;()")
		cleaned = strings.TrimPrefix(cleaned, "v")
		cleaned = strings.TrimPrefix(cleaned, "V")
		if containsDigit(cleaned) {
			return cleaned
		}
	}

	return ""
}

func containsDigit(value string) bool {
	for _, r := range value {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// captureChocolateyPackages captures installed Chocolatey packages (Windows).
func (p *Preflight) captureChocolateyPackages(_ context.Context, capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	// Check if choco is available
	cmd := exec.Command("choco", "list", "--local-only", "--limit-output")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Chocolatey output format: PackageName|Version
		parts := strings.SplitN(line, "|", 2)
		if len(parts) >= 1 {
			name := strings.TrimSpace(parts[0])
			if name == "" {
				continue
			}
			items = append(items, CapturedItem{
				Provider:   "chocolatey",
				Name:       name,
				Source:     "choco list",
				CapturedAt: capturedAt,
			})
		}
	}

	return items
}

// captureScoopPackages captures installed Scoop packages (Windows).
func (p *Preflight) captureScoopPackages(_ context.Context, capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	// Scoop list command
	cmd := exec.Command("scoop", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		// Skip header line
		if i == 0 || line == "" {
			continue
		}
		// Scoop output format: Name  Version  Source  Updated
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			name := fields[0]
			if name == "" || name == "Name" {
				continue
			}
			items = append(items, CapturedItem{
				Provider:   "scoop",
				Name:       name,
				Source:     "scoop list",
				CapturedAt: capturedAt,
			})
		}
	}

	return items
}

// captureWingetPackages captures installed WinGet packages (Windows).
func (p *Preflight) captureWingetPackages(_ context.Context, capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	// WinGet list command with machine-readable output
	cmd := exec.Command("winget", "list", "--disable-interactivity")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	headerPassed := false
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Skip until we pass the header separator
		if strings.HasPrefix(line, "-") {
			headerPassed = true
			continue
		}
		if !headerPassed {
			continue
		}
		// WinGet output is space-delimited: Name  Id  Version  Available  Source
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			// Use the package ID (second field) as it's more reliable
			id := fields[1]
			if id == "" {
				continue
			}
			items = append(items, CapturedItem{
				Provider:   "winget",
				Name:       id,
				Source:     "winget list",
				CapturedAt: capturedAt,
			})
		}
	}

	return items
}

// captureAPTPackages captures installed APT packages (Linux/Debian).
func (p *Preflight) captureAPTPackages(_ context.Context, capturedAt time.Time) []CapturedItem {
	// Use dpkg-query for reliable package listing
	cmd := exec.Command("dpkg-query", "-W", "-f=${Package}\n")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	items := make([]CapturedItem, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		items = append(items, CapturedItem{
			Provider:   "apt",
			Name:       strings.TrimSpace(line),
			Source:     "dpkg-query",
			CapturedAt: capturedAt,
		})
	}

	return items
}

// captureNpmGlobals captures globally installed npm packages.
func (p *Preflight) captureNpmGlobals(_ context.Context, capturedAt time.Time) []CapturedItem {
	// Run npm list -g --depth=0 --json
	cmd := exec.Command("npm", "list", "-g", "--depth=0", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Parse JSON output
	var result struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil
	}

	items := make([]CapturedItem, 0, len(result.Dependencies))
	for name, info := range result.Dependencies {
		// Skip npm itself
		if name == "npm" {
			continue
		}
		// Format as name@version
		value := name
		if info.Version != "" {
			value = fmt.Sprintf("%s@%s", name, info.Version)
		}
		items = append(items, CapturedItem{
			Provider:   "npm",
			Name:       name,
			Value:      value,
			Source:     "npm list -g",
			CapturedAt: capturedAt,
		})
	}

	return items
}

// captureGoTools captures installed Go tools from GOBIN or GOPATH/bin.
// Only captures tools that have valid Go module paths (installed via go install).
func (p *Preflight) captureGoTools(_ context.Context, capturedAt time.Time) []CapturedItem {
	// Determine the Go bin directory
	gobin := os.Getenv("GOBIN")
	if gobin == "" {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil
			}
			gopath = filepath.Join(home, "go")
		}
		gobin = filepath.Join(gopath, "bin")
	}

	// List files in the directory
	entries, err := os.ReadDir(gobin)
	if err != nil {
		return nil
	}

	items := make([]CapturedItem, 0, len(entries))
	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		binaryPath := filepath.Join(gobin, entry.Name())

		// Get module path using go version -m
		modulePath := getGoToolModulePath(binaryPath)
		if modulePath == "" {
			// Skip tools without valid module paths (e.g., copied from Homebrew)
			continue
		}

		items = append(items, CapturedItem{
			Provider:   "go",
			Name:       entry.Name(),
			Value:      modulePath,
			Source:     gobin,
			CapturedAt: capturedAt,
		})
	}

	return items
}

// getGoToolModulePath extracts the Go module path from a binary using go version -m.
// Returns empty string if the binary is not a Go binary or doesn't have module info.
func getGoToolModulePath(binaryPath string) string {
	cmd := exec.Command("go", "version", "-m", binaryPath)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse output to find the module path
	// Format:
	// /path/to/binary: go1.21.0
	//         path    github.com/user/tool
	//         mod     github.com/user/tool    v1.2.3  h1:...
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for "path" line which contains the main module path
		if strings.HasPrefix(line, "path\t") || strings.HasPrefix(line, "path ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				modulePath := fields[1]
				// Validate it looks like a Go module path (contains a domain)
				if isValidGoModulePath(modulePath) {
					return modulePath
				}
			}
		}
	}
	return ""
}

// isValidGoModulePath checks if a string looks like a valid Go module path.
// Valid module paths start with a domain (contain at least one dot and a slash).
func isValidGoModulePath(path string) bool {
	// Must contain a dot (for the domain) and a slash (for the path)
	// Examples: github.com/user/tool, golang.org/x/tools
	if !strings.Contains(path, ".") {
		return false
	}
	if !strings.Contains(path, "/") {
		return false
	}
	// Shouldn't start with . or /
	if strings.HasPrefix(path, ".") || strings.HasPrefix(path, "/") {
		return false
	}
	return true
}

// capturePipPackages captures user-installed pip packages.
func (p *Preflight) capturePipPackages(_ context.Context, capturedAt time.Time) []CapturedItem {
	// Run pip list --format=json --user
	cmd := exec.Command("pip", "list", "--format=json", "--user")
	output, err := cmd.Output()
	if err != nil {
		// Try pip3 as fallback
		cmd = exec.Command("pip3", "list", "--format=json", "--user")
		output, err = cmd.Output()
		if err != nil {
			return nil
		}
	}

	// Parse JSON output - array of objects with name and version fields
	var packages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(output, &packages); err != nil {
		return nil
	}

	items := make([]CapturedItem, 0, len(packages))
	for _, pkg := range packages {
		// Format as name==version (pip convention)
		value := pkg.Name
		if pkg.Version != "" {
			value = fmt.Sprintf("%s==%s", pkg.Name, pkg.Version)
		}
		items = append(items, CapturedItem{
			Provider:   "pip",
			Name:       pkg.Name,
			Value:      value,
			Source:     "pip list --user",
			CapturedAt: capturedAt,
		})
	}

	return items
}

// captureGemPackages captures installed Ruby gems.
func (p *Preflight) captureGemPackages(_ context.Context, capturedAt time.Time) []CapturedItem {
	// Run gem list --no-versions for cleaner output
	cmd := exec.Command("gem", "list", "--local")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Parse output - each line is "gemname (version, version, ...)"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	items := make([]CapturedItem, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Parse "gemname (version)" format
		parts := strings.SplitN(line, " ", 2)
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}

		value := name
		if len(parts) > 1 {
			// Extract first version from "(version, version, ...)"
			versionPart := strings.Trim(parts[1], "()")
			versions := strings.Split(versionPart, ", ")
			if len(versions) > 0 && versions[0] != "" {
				value = fmt.Sprintf("%s@%s", name, strings.TrimSpace(versions[0]))
			}
		}

		items = append(items, CapturedItem{
			Provider:   "gem",
			Name:       name,
			Value:      value,
			Source:     "gem list",
			CapturedAt: capturedAt,
		})
	}

	return items
}

// captureCargoCrates captures installed Cargo crates.
func (p *Preflight) captureCargoCrates(_ context.Context, capturedAt time.Time) []CapturedItem {
	// Run cargo install --list
	cmd := exec.Command("cargo", "install", "--list")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Parse output - format is:
	// crate-name v1.2.3:
	//     binary1
	//     binary2
	// another-crate v2.0.0:
	//     binary
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	items := make([]CapturedItem, 0)
	for _, line := range lines {
		// Only process lines that start with a crate name (not indented binary names)
		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		// Parse "crate-name v1.2.3:" format
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		version := strings.TrimPrefix(parts[1], "v")
		version = strings.TrimSuffix(version, ":")

		value := name
		if version != "" {
			value = fmt.Sprintf("%s@%s", name, version)
		}

		items = append(items, CapturedItem{
			Provider:   "cargo",
			Name:       name,
			Value:      value,
			Source:     "cargo install --list",
			CapturedAt: capturedAt,
		})
	}

	return items
}

// Doctor checks system state against configuration and reports issues.
func (p *Preflight) Doctor(ctx context.Context, opts DoctorOptions) (*DoctorReport, error) {
	startTime := time.Now()

	report := &DoctorReport{
		ConfigPath:   opts.ConfigPath,
		Target:       opts.Target,
		Issues:       make([]DoctorIssue, 0),
		BinaryChecks: make([]BinaryCheckResult, 0),
		CheckedAt:    startTime,
	}

	// Load and compile configuration
	plan, err := p.Plan(ctx, opts.ConfigPath, opts.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Check each step for drift
	for _, entry := range plan.Entries() {
		status := entry.Status()
		step := entry.Step()

		switch status {
		case compiler.StatusNeedsApply:
			diff := entry.Diff()
			report.Issues = append(report.Issues, DoctorIssue{
				Provider:   step.ID().Provider(),
				StepID:     step.ID().String(),
				Severity:   SeverityWarning,
				Message:    "Configuration drift detected",
				Expected:   diff.Summary(),
				Actual:     "current state differs",
				Fixable:    true,
				FixCommand: "preflight apply",
			})

		case compiler.StatusFailed:
			report.Issues = append(report.Issues, DoctorIssue{
				Provider: step.ID().Provider(),
				StepID:   step.ID().String(),
				Severity: SeverityError,
				Message:  "Step check failed",
				Fixable:  false,
			})

		case compiler.StatusUnknown:
			report.Issues = append(report.Issues, DoctorIssue{
				Provider: step.ID().Provider(),
				StepID:   step.ID().String(),
				Severity: SeverityInfo,
				Message:  "Unable to determine step status",
				Fixable:  false,
			})

		case compiler.StatusSatisfied, compiler.StatusSkipped:
			// No issues for satisfied or skipped steps
		}
	}

	// Run provider-specific doctor checks
	p.runProviderDoctorChecks(ctx, plan, report)

	// Generate config patches if UpdateConfig is enabled
	if opts.UpdateConfig && len(report.Issues) > 0 {
		configDir := filepath.Dir(opts.ConfigPath)
		driftService, err := DefaultDriftService()
		if err == nil {
			generator := NewPatchGenerator(driftService)
			patches := generator.GenerateFromIssues(report.Issues, configDir)
			report.SuggestedPatches = patches
		}
	}

	report.Duration = time.Since(startTime)
	return report, nil
}

// runProviderDoctorChecks runs health checks for providers used in the plan.
func (p *Preflight) runProviderDoctorChecks(ctx context.Context, plan *execution.Plan, report *DoctorReport) {
	// Check which providers are used in the plan
	providersUsed := make(map[string]bool)
	for _, entry := range plan.Entries() {
		providersUsed[entry.Step().ID().Provider()] = true
	}

	// Run nvim doctor checks if nvim is used
	if providersUsed["nvim"] {
		runner := command.NewRealRunner()
		doctor := nvim.NewDoctorCheck(runner)
		binaryResults := doctor.CheckBinaries(ctx)

		for _, br := range binaryResults {
			report.BinaryChecks = append(report.BinaryChecks, BinaryCheckResult{
				Name:       br.Name,
				Found:      br.Found,
				Version:    br.Version,
				Path:       br.Path,
				MeetsMin:   br.MeetsMin,
				MinVersion: "", // Will be filled from check
				Required:   false,
				Purpose:    br.Purpose,
			})
		}

		// Get required info for each binary
		for _, check := range doctor.RequiredBinaries() {
			for i := range report.BinaryChecks {
				if report.BinaryChecks[i].Name == check.Name {
					report.BinaryChecks[i].MinVersion = check.MinVersion
					report.BinaryChecks[i].Required = check.Required
					break
				}
			}
		}

		// Add binary issues to main issues list
		if doctor.HasIssues(binaryResults) {
			for _, br := range binaryResults {
				// Find the check info for this binary
				var check nvim.BinaryCheck
				for _, c := range doctor.RequiredBinaries() {
					if c.Name == br.Name {
						check = c
						break
					}
				}

				if check.Required && !br.Found {
					report.Issues = append(report.Issues, DoctorIssue{
						Provider:   "nvim",
						StepID:     "nvim:binary:" + br.Name,
						Severity:   SeverityError,
						Message:    fmt.Sprintf("Required binary '%s' not found", br.Name),
						Expected:   fmt.Sprintf("%s installed", br.Name),
						Actual:     "not found",
						Fixable:    true,
						FixCommand: fmt.Sprintf("brew install %s", br.Name),
					})
				} else if check.Required && !br.MeetsMin {
					report.Issues = append(report.Issues, DoctorIssue{
						Provider:   "nvim",
						StepID:     "nvim:binary:" + br.Name,
						Severity:   SeverityWarning,
						Message:    fmt.Sprintf("Binary '%s' version too low", br.Name),
						Expected:   fmt.Sprintf(">= %s", check.MinVersion),
						Actual:     br.Version,
						Fixable:    true,
						FixCommand: fmt.Sprintf("brew upgrade %s", br.Name),
					})
				}
			}
		}
	}
}

// Fix applies fixes for issues found by Doctor and verifies the result.
func (p *Preflight) Fix(ctx context.Context, report *DoctorReport) (*FixResult, error) {
	if report == nil || !report.HasIssues() {
		return &FixResult{}, nil
	}

	// Collect fixable issues
	var fixableIssues []DoctorIssue
	for _, issue := range report.Issues {
		if issue.Fixable {
			fixableIssues = append(fixableIssues, issue)
		}
	}

	if len(fixableIssues) == 0 {
		return &FixResult{
			RemainingIssues: report.Issues,
		}, nil
	}

	// Re-run plan and apply
	plan, err := p.Plan(ctx, report.ConfigPath, report.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to create fix plan: %w", err)
	}

	_, err = p.Apply(ctx, plan, false)
	if err != nil {
		return nil, fmt.Errorf("failed to apply fixes: %w", err)
	}

	// Verify by re-running doctor
	verifyOpts := NewDoctorOptions(report.ConfigPath, report.Target)
	verifyReport, err := p.Doctor(ctx, verifyOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to verify fixes: %w", err)
	}

	// Determine which issues were fixed
	remainingStepIDs := make(map[string]bool)
	for _, issue := range verifyReport.Issues {
		remainingStepIDs[issue.StepID] = true
	}

	var fixed []DoctorIssue
	var remaining []DoctorIssue
	for _, issue := range fixableIssues {
		if remainingStepIDs[issue.StepID] {
			remaining = append(remaining, issue)
		} else {
			fixed = append(fixed, issue)
		}
	}

	return &FixResult{
		FixedIssues:        fixed,
		RemainingIssues:    remaining,
		VerificationReport: verifyReport,
	}, nil
}

// Diff shows differences between configuration and current system state.
func (p *Preflight) Diff(ctx context.Context, configPath, target string) (*DiffResult, error) {
	result := &DiffResult{
		ConfigPath: configPath,
		Target:     target,
		Entries:    make([]DiffEntry, 0),
		DiffedAt:   time.Now(),
	}

	// Create plan to see differences
	plan, err := p.Plan(ctx, configPath, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Convert plan entries to diff entries
	for _, entry := range plan.Entries() {
		if entry.Status() != compiler.StatusNeedsApply {
			continue
		}

		step := entry.Step()
		diff := entry.Diff()

		result.Entries = append(result.Entries, DiffEntry{
			Provider: step.ID().Provider(),
			Path:     step.ID().String(),
			Type:     DiffTypeModified,
			Expected: diff.Summary(),
			Actual:   "current state",
		})
	}

	return result, nil
}

// LockUpdate updates the lockfile with current versions.
func (p *Preflight) LockUpdate(ctx context.Context, configPath string) error {
	lockPath := strings.TrimSuffix(configPath, filepath.Ext(configPath)) + ".lock"

	// Check if lockfile repository is configured
	if p.lockRepo == nil {
		return fmt.Errorf("lockfile repository not configured")
	}

	target, err := selectLockTarget(configPath)
	if err != nil {
		return err
	}

	cfg, err := p.loadConfig(configPath, target)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	resolver := versionResolverAdapter{
		resolver: lock.NewResolver(lock.NewLockfile(config.ModeIntent, lock.MachineInfoFromSystem())),
	}

	configRoot := filepath.Dir(configPath)
	compileCtx := compiler.NewCompileContext(cfg).
		WithResolver(resolver).
		WithConfigRoot(configRoot).
		WithTarget(target)

	graph, err := p.compiler.CompileWithContext(compileCtx)
	if err != nil {
		return fmt.Errorf("failed to compile: %w", err)
	}

	plan, err := p.planner.Plan(ctx, graph)
	if err != nil {
		return fmt.Errorf("failed to plan: %w", err)
	}

	if err := p.UpdateLockFromPlan(ctx, configPath, plan); err != nil {
		return fmt.Errorf("failed to update lockfile: %w", err)
	}

	p.printf("Lockfile updated: %s\n", lockPath)
	return nil
}

func selectLockTarget(configPath string) (string, error) {
	loader := config.NewLoader()
	manifest, err := loader.LoadManifest(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to load manifest: %w", err)
	}

	if _, ok := manifest.Targets["default"]; ok {
		return "default", nil
	}

	targets := make([]string, 0, len(manifest.Targets))
	for name := range manifest.Targets {
		targets = append(targets, name)
	}
	sort.Strings(targets)
	if len(targets) == 0 {
		return "", fmt.Errorf("no targets defined in manifest")
	}

	return targets[0], nil
}

// LockFreeze freezes the lockfile to prevent version changes.
func (p *Preflight) LockFreeze(ctx context.Context, configPath string) error {
	lockPath := strings.TrimSuffix(configPath, filepath.Ext(configPath)) + ".lock"

	// Check if lockfile repository is configured
	if p.lockRepo == nil {
		return fmt.Errorf("lockfile repository not configured")
	}

	lockfile, err := p.lockRepo.Load(ctx, lockPath)
	if err != nil {
		return fmt.Errorf("lockfile not found: %w", err)
	}

	// Change mode to frozen
	lockfile = lockfile.WithMode(config.ModeFrozen)

	// Save the frozen lockfile
	if err := p.lockRepo.Save(ctx, lockPath, lockfile); err != nil {
		return fmt.Errorf("failed to save lockfile: %w", err)
	}

	p.printf("Lockfile frozen: %s\n", lockPath)
	return nil
}

// RepoInit initializes a configuration repository.
func (p *Preflight) RepoInit(ctx context.Context, opts RepoOptions) error {
	// Validate inputs to prevent command injection
	if err := validation.ValidateGitPath(opts.Path); err != nil {
		return fmt.Errorf("invalid repository path: %w", err)
	}
	if opts.Remote != "" {
		if err := validation.ValidateGitRemoteURL(opts.Remote); err != nil {
			return fmt.Errorf("invalid remote URL: %w", err)
		}
	}
	if opts.Branch != "" {
		if err := validation.ValidateGitBranch(opts.Branch); err != nil {
			return fmt.Errorf("invalid branch name: %w", err)
		}
	}

	// Check if already initialized
	gitDir := filepath.Join(opts.Path, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return fmt.Errorf("repository already initialized at %s", opts.Path)
	}

	// Initialize git repository
	// #nosec G204 -- opts.Path validated by validation.ValidateGitPath.
	cmd := exec.CommandContext(ctx, "git", "init", opts.Path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Set up remote if provided
	if opts.Remote != "" {
		// #nosec G204 -- opts.Remote validated by validation.ValidateGitRemoteURL.
		cmd = exec.CommandContext(ctx, "git", "-C", opts.Path, "remote", "add", "origin", opts.Remote)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
	}

	// Create initial branch
	// #nosec G204 -- opts.Branch validated by validation.ValidateGitBranch.
	cmd = exec.CommandContext(ctx, "git", "-C", opts.Path, "checkout", "-b", opts.Branch)
	_ = cmd.Run() // Ignore error if branch already exists

	p.printf("Repository initialized: %s\n", opts.Path)
	return nil
}

// RepoInitGitHub initializes a configuration repository and creates a GitHub remote.
func (p *Preflight) RepoInitGitHub(ctx context.Context, opts GitHubRepoOptions) error {
	// Validate inputs to prevent command injection
	if err := validation.ValidateGitPath(opts.Path); err != nil {
		return fmt.Errorf("invalid repository path: %w", err)
	}
	if err := validation.ValidateGitRepoName(opts.Name); err != nil {
		return fmt.Errorf("invalid repository name: %w", err)
	}
	if opts.Branch != "" {
		if err := validation.ValidateGitBranch(opts.Branch); err != nil {
			return fmt.Errorf("invalid branch name: %w", err)
		}
	}

	runner := command.NewRealRunner()
	ghClient := github.NewClient(runner)

	// Step 1: Check GitHub authentication
	p.printf("Checking GitHub authentication...\n")
	authed, err := ghClient.IsAuthenticated(ctx)
	if err != nil {
		return fmt.Errorf("failed to check GitHub auth: %w", err)
	}
	if !authed {
		return fmt.Errorf("not authenticated with GitHub. Run 'gh auth login' first")
	}

	// Get authenticated user for README template
	owner, err := ghClient.GetAuthenticatedUser(ctx)
	if err != nil {
		p.printf("Warning: could not get GitHub username: %v\n", err)
		owner = ""
	}

	// Step 2: Create .gitignore if it doesn't exist
	gitignorePath := filepath.Join(opts.Path, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		p.printf("Creating .gitignore...\n")
		// #nosec G306 -- .gitignore is intended to be world-readable.
		if err := os.WriteFile(gitignorePath, []byte(templates.GitignoreTemplate), 0o644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
	}

	// Step 3: Create README.md if it doesn't exist
	readmePath := filepath.Join(opts.Path, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		p.printf("Creating README.md...\n")
		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName:    opts.Name,
			Description: opts.Description,
			Owner:       owner,
		})
		if err != nil {
			return fmt.Errorf("failed to generate README: %w", err)
		}
		// #nosec G306 -- README is intended to be world-readable.
		if err := os.WriteFile(readmePath, []byte(readme), 0o644); err != nil {
			return fmt.Errorf("failed to create README.md: %w", err)
		}
	}

	// Step 4: Initialize git repository if not already
	gitDir := filepath.Join(opts.Path, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		p.printf("Initializing git repository...\n")
		// #nosec G204 -- opts.Path validated by validation.ValidateGitPath.
		cmd := exec.CommandContext(ctx, "git", "init", opts.Path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to initialize git: %w", err)
		}

		// Create initial branch
		// #nosec G204 -- opts.Branch validated by validation.ValidateGitBranch.
		cmd = exec.CommandContext(ctx, "git", "-C", opts.Path, "checkout", "-b", opts.Branch)
		_ = cmd.Run()
	}

	// Step 5: Stage and commit
	p.printf("Creating initial commit...\n")
	// #nosec G204 -- opts.Path validated by validation.ValidateGitPath.
	cmd := exec.CommandContext(ctx, "git", "-C", opts.Path, "add", "-A")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}

	// #nosec G204 -- opts.Path validated by validation.ValidateGitPath.
	cmd = exec.CommandContext(ctx, "git", "-C", opts.Path, "commit", "-m", "Initial preflight configuration")
	_ = cmd.Run() // Ignore if nothing to commit

	// Step 6: Create GitHub repository
	p.printf("Creating GitHub repository '%s'...\n", opts.Name)
	repoInfo, err := ghClient.CreateRepository(ctx, ports.GitHubCreateOptions{
		Name:        opts.Name,
		Description: opts.Description,
		Private:     opts.Private,
	})
	if err != nil {
		return fmt.Errorf("failed to create GitHub repository: %w", err)
	}

	// Step 7: Set remote and push
	p.printf("Setting up remote...\n")
	if err := ghClient.SetRemote(ctx, opts.Path, repoInfo.SSHURL); err != nil {
		return fmt.Errorf("failed to set remote: %w", err)
	}

	// Push to remote
	p.printf("Pushing to GitHub...\n")
	// #nosec G204 -- opts.Path and opts.Branch validated by validation.ValidateGitPath/ValidateGitBranch.
	cmd = exec.CommandContext(ctx, "git", "-C", opts.Path, "push", "-u", "origin", opts.Branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push to GitHub: %s", output)
	}

	p.printf("\n✓ Repository created successfully!\n")
	p.printf("  URL: %s\n", repoInfo.URL)
	p.printf("  SSH: %s\n", repoInfo.SSHURL)
	if opts.Private {
		p.printf("  Visibility: private\n")
	} else {
		p.printf("  Visibility: public\n")
	}

	return nil
}

// RepoStatus returns the status of a configuration repository.
func (p *Preflight) RepoStatus(ctx context.Context, path string) (*RepoStatus, error) {
	// Validate path to prevent command injection
	if err := validation.ValidateGitPath(path); err != nil {
		return nil, fmt.Errorf("invalid repository path: %w", err)
	}

	status := &RepoStatus{
		Path: path,
	}

	// Check if git repo exists
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		status.Initialized = false
		return status, nil
	}
	status.Initialized = true

	// Get current branch
	cmd := exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	if output, err := cmd.Output(); err == nil {
		status.Branch = strings.TrimSpace(string(output))
	}

	// Get remote
	cmd = exec.CommandContext(ctx, "git", "-C", path, "remote", "get-url", "origin")
	if output, err := cmd.Output(); err == nil {
		status.Remote = strings.TrimSpace(string(output))
	}

	// Check for uncommitted changes
	cmd = exec.CommandContext(ctx, "git", "-C", path, "status", "--porcelain")
	if output, err := cmd.Output(); err == nil {
		status.HasChanges = len(strings.TrimSpace(string(output))) > 0
	}

	// Get ahead/behind counts
	cmd = exec.CommandContext(ctx, "git", "-C", path, "rev-list", "--count", "--left-right", "@{upstream}...HEAD")
	if output, err := cmd.Output(); err == nil {
		parts := strings.Fields(strings.TrimSpace(string(output)))
		if len(parts) == 2 {
			_, _ = fmt.Sscanf(parts[0], "%d", &status.Behind)
			_, _ = fmt.Sscanf(parts[1], "%d", &status.Ahead)
		}
	}

	// Get last commit
	cmd = exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "--short", "HEAD")
	if output, err := cmd.Output(); err == nil {
		status.LastCommit = strings.TrimSpace(string(output))
	}

	// Get last commit time
	cmd = exec.CommandContext(ctx, "git", "-C", path, "log", "-1", "--format=%ct")
	if output, err := cmd.Output(); err == nil {
		var timestamp int64
		if n, _ := fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &timestamp); n == 1 {
			status.LastCommitAt = time.Unix(timestamp, 0)
		}
	}

	return status, nil
}

// PrintDoctorReport outputs a human-readable doctor report.
func (p *Preflight) PrintDoctorReport(report *DoctorReport) {
	p.printf("\nDoctor Report\n")
	p.printf("=============\n\n")

	// Print binary checks if any
	if len(report.BinaryChecks) > 0 {
		p.printf("Binary Checks:\n")
		for _, bc := range report.BinaryChecks {
			var status, detail string
			switch {
			case !bc.Found:
				if bc.Required {
					status = "✗"
				} else {
					status = "○"
				}
				detail = "not found"
			case !bc.MeetsMin && bc.MinVersion != "":
				status = "⚠"
				detail = fmt.Sprintf("v%s (need >= %s)", bc.Version, bc.MinVersion)
			case bc.Version != "":
				status = "✓"
				detail = fmt.Sprintf("v%s", bc.Version)
			default:
				status = "✓"
				detail = "found"
			}

			reqTag := ""
			if bc.Required {
				reqTag = " (required)"
			}
			p.printf("  %s %s%s - %s [%s]\n", status, bc.Name, reqTag, bc.Purpose, detail)
		}
		p.printf("\n")
	}

	if !report.HasIssues() {
		p.printf("✓ No issues found. Your system matches the configuration.\n")
		return
	}

	p.printf("Found %d issue(s):\n\n", report.IssueCount())

	bySeverity := report.IssuesBySeverity()

	// Print errors first
	for _, issue := range bySeverity[SeverityError] {
		p.printf("  ✗ [ERROR] %s: %s\n", issue.StepID, issue.Message)
	}

	// Then warnings
	for _, issue := range bySeverity[SeverityWarning] {
		p.printf("  ⚠ [WARNING] %s: %s\n", issue.StepID, issue.Message)
		if issue.Fixable {
			p.printf("      Fix: %s\n", issue.FixCommand)
		}
	}

	// Then info
	for _, issue := range bySeverity[SeverityInfo] {
		p.printf("  ℹ [INFO] %s: %s\n", issue.StepID, issue.Message)
	}

	p.printf("\nSummary: %d errors, %d warnings, %d fixable\n",
		report.ErrorCount(), report.WarningCount(), report.FixableCount())
}

// PrintCaptureFindings outputs captured configuration.
func (p *Preflight) PrintCaptureFindings(findings *CaptureFindings) {
	p.printf("\nCapture Results\n")
	p.printf("===============\n\n")

	p.printf("Captured %d items from %d providers\n\n",
		findings.ItemCount(), len(findings.Providers))

	byProvider := findings.ItemsByProvider()
	for provider, items := range byProvider {
		p.printf("%s (%d items):\n", provider, len(items))
		for _, item := range items {
			p.printf("  - %s\n", item.Name)
		}
		p.printf("\n")
	}

	if len(findings.Warnings) > 0 {
		p.printf("Warnings:\n")
		for _, w := range findings.Warnings {
			p.printf("  ⚠ %s\n", w)
		}
	}
}

// PrintDiff outputs differences in unified format.
func (p *Preflight) PrintDiff(result *DiffResult) {
	p.printf("\nConfiguration Diff\n")
	p.printf("==================\n\n")

	if !result.HasDifferences() {
		p.printf("No differences. Configuration matches system state.\n")
		return
	}

	p.printf("Found %d difference(s):\n\n", len(result.Entries))

	byProvider := result.EntriesByProvider()
	for provider, entries := range byProvider {
		p.printf("%s:\n", provider)
		for _, entry := range entries {
			symbol := "~"
			switch entry.Type {
			case DiffTypeAdded:
				symbol = "+"
			case DiffTypeRemoved:
				symbol = "-"
			case DiffTypeModified:
				symbol = "~"
			}
			p.printf("  %s %s\n", symbol, entry.Path)
			if entry.Expected != "" {
				p.printf("      expected: %s\n", entry.Expected)
			}
		}
		p.printf("\n")
	}
}

// RepoClone clones a configuration repository and optionally applies it.
func (p *Preflight) RepoClone(ctx context.Context, opts CloneOptions) (*CloneResult, error) {
	// Validate URL to prevent command injection
	if err := validation.ValidateGitRemoteURL(opts.URL); err != nil {
		return nil, fmt.Errorf("invalid clone URL: %w", err)
	}
	// Validate path if provided
	if opts.Path != "" {
		if err := validation.ValidateGitPath(opts.Path); err != nil {
			return nil, fmt.Errorf("invalid destination path: %w", err)
		}
	}

	result := &CloneResult{}

	// Determine destination path
	destPath := opts.Path
	if destPath == "" {
		// Extract repo name from URL for default path
		destPath = extractRepoName(opts.URL)
	}

	// Make path absolute
	if !filepath.IsAbs(destPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		destPath = filepath.Join(cwd, destPath)
	}
	result.Path = destPath

	// Check if path already exists
	if _, err := os.Stat(destPath); err == nil {
		return nil, fmt.Errorf("destination path already exists: %s", destPath)
	}

	// Clone the repository
	p.printf("Cloning %s...\n", opts.URL)
	// #nosec G204 -- opts.URL and destPath validated before use.
	cmd := exec.CommandContext(ctx, "git", "clone", opts.URL, destPath)
	cmd.Stdout = p.out
	cmd.Stderr = p.out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git clone failed: %w", err)
	}

	// Check for preflight.yaml
	configPath := filepath.Join(destPath, "preflight.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Also check for preflight.yml
		configPath = filepath.Join(destPath, "preflight.yml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			p.printf("\n⚠ No preflight.yaml found in repository.\n")
			result.ConfigFound = false
			return result, nil
		}
	}
	result.ConfigFound = true

	// Determine whether to apply
	shouldApply := opts.Apply
	if !shouldApply && !opts.AutoConfirm {
		// Prompt user
		p.printf("\nConfiguration file found. Apply now? [y/N]: ")
		var response string
		if _, err := fmt.Scanln(&response); err == nil {
			response = strings.ToLower(strings.TrimSpace(response))
			shouldApply = response == "y" || response == "yes"
		}
	}

	if shouldApply {
		p.printf("\nApplying configuration...\n")

		// Create plan
		target := opts.Target
		if strings.TrimSpace(target) == "" {
			target = "default"
		}
		plan, err := p.Plan(ctx, configPath, target)
		if err != nil {
			return nil, fmt.Errorf("failed to create plan: %w", err)
		}

		// Show plan summary
		summary := plan.Summary()
		p.printf("Plan: %d steps to apply\n", summary.NeedsApply)

		if RequiresBootstrapConfirmation(plan) && !opts.AutoConfirm && !opts.AllowBootstrap {
			steps := BootstrapSteps(plan)
			p.printf("\nBootstrap steps require confirmation:\n")
			for _, step := range steps {
				p.printf("  - %s\n", step)
			}
			p.printf("Proceed with bootstrapping? [y/N]: ")
			var response string
			if _, err := fmt.Scanln(&response); err != nil {
				// User cancelled input (e.g., Ctrl+C or EOF), skip gracefully
				p.printf("\nSkipping apply. Run 'preflight apply' when ready.\n")
				return result, nil //nolint:nilerr // Intentional: EOF/cancel means skip, not error
			}
			response = strings.ToLower(strings.TrimSpace(response))
			if response != "y" && response != "yes" {
				p.printf("\nSkipping apply. Run 'preflight apply' when ready.\n")
				return result, nil
			}
		}

		// Apply configuration
		stepResults, err := p.Apply(ctx, plan, false)
		if err != nil {
			return nil, fmt.Errorf("failed to apply configuration: %w", err)
		}

		// Count results by status
		applied := 0
		skipped := 0
		failed := 0
		for _, sr := range stepResults {
			switch sr.Status() { //nolint:exhaustive // Only counting relevant statuses
			case compiler.StatusSatisfied:
				applied++
			case compiler.StatusSkipped:
				skipped++
			case compiler.StatusFailed:
				failed++
			}
		}

		result.Applied = true
		result.ApplyResult = &ApplyResult{
			Applied: applied,
			Skipped: skipped,
			Failed:  failed,
		}

		p.printf("\n✓ Applied %d steps (%d skipped, %d failed)\n",
			applied, skipped, failed)
	}

	return result, nil
}

// extractRepoName extracts the repository name from a git URL.
func extractRepoName(url string) string {
	// Handle SSH URLs: git@github.com:user/repo.git
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			name := parts[len(parts)-1]
			return strings.TrimSuffix(name, ".git")
		}
	}

	// Handle HTTPS URLs: https://github.com/user/repo.git
	if strings.Contains(url, "://") {
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			name := parts[len(parts)-1]
			return strings.TrimSuffix(name, ".git")
		}
	}

	// Fallback: use the whole URL as name
	return strings.TrimSuffix(filepath.Base(url), ".git")
}

// PrintRepoStatus outputs repository status.
func (p *Preflight) PrintRepoStatus(status *RepoStatus) {
	p.printf("\nRepository Status\n")
	p.printf("=================\n\n")

	if !status.Initialized {
		p.printf("Not a git repository. Run 'preflight repo init' to initialize.\n")
		return
	}

	p.printf("Path:   %s\n", status.Path)
	p.printf("Branch: %s\n", status.Branch)

	if status.Remote != "" {
		p.printf("Remote: %s\n", status.Remote)
	}

	if status.IsSynced() {
		p.printf("Status: ✓ Up to date\n")
	} else {
		if status.HasChanges {
			p.printf("Status: ⚠ Uncommitted changes\n")
		}
		if status.NeedsPush() {
			p.printf("Status: ↑ %d commit(s) ahead\n", status.Ahead)
		}
		if status.NeedsPull() {
			p.printf("Status: ↓ %d commit(s) behind\n", status.Behind)
		}
	}

	if status.LastCommit != "" {
		p.printf("Last commit: %s (%s)\n", status.LastCommit,
			status.LastCommitAt.Format("2006-01-02 15:04"))
	}
}
