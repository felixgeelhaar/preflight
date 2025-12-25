// Package plugin provides plugin discovery, loading, and management.
package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/semver"
)

// UpgradeInfo contains information about an available upgrade.
type UpgradeInfo struct {
	// Name is the plugin name
	Name string
	// CurrentVersion is the currently installed version
	CurrentVersion string
	// LatestVersion is the latest available version
	LatestVersion string
	// UpgradeAvailable indicates if an upgrade is available
	UpgradeAvailable bool
	// Source is the plugin source (git URL from .git/config)
	Source string
	// ChangelogURL is a link to the changelog (if available)
	ChangelogURL string
}

// UpgradeChecker checks for available plugin upgrades.
type UpgradeChecker struct {
	registry *Registry
	cloner   *GitCloner
}

// NewUpgradeChecker creates a new upgrade checker.
func NewUpgradeChecker(registry *Registry) *UpgradeChecker {
	return &UpgradeChecker{
		registry: registry,
		cloner:   NewGitCloner(),
	}
}

// CheckUpgrade checks if an upgrade is available for a specific plugin.
func (c *UpgradeChecker) CheckUpgrade(ctx context.Context, name string) (*UpgradeInfo, error) {
	if c.registry == nil {
		return nil, fmt.Errorf("registry not initialized")
	}

	plugin, ok := c.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("plugin %q not found", name)
	}

	info := &UpgradeInfo{
		Name:           name,
		CurrentVersion: plugin.Manifest.Version,
	}

	// Try to get git remote URL from plugin path
	source := getGitRemoteURL(plugin.Path)
	info.Source = source

	// Skip non-git sources
	if source == "" {
		info.LatestVersion = plugin.Manifest.Version
		info.UpgradeAvailable = false
		return info, nil
	}

	// Check for latest version from git
	latestVersion, err := c.cloner.LatestVersion(ctx, source)
	if err != nil {
		// Can't check for updates, return current version (not an error condition)
		info.LatestVersion = plugin.Manifest.Version
		info.UpgradeAvailable = false
		return info, nil //nolint:nilerr // Intentional: graceful fallback when remote unavailable
	}

	info.LatestVersion = latestVersion

	// Compare versions
	current := normalizeVersion(plugin.Manifest.Version)
	latest := normalizeVersion(latestVersion)

	if semver.IsValid(current) && semver.IsValid(latest) {
		info.UpgradeAvailable = semver.Compare(latest, current) > 0
	}

	// Construct changelog URL if it's a GitHub repo
	if info.UpgradeAvailable && isGitHubURL(source) {
		info.ChangelogURL = constructChangelogURL(source, latestVersion)
	}

	return info, nil
}

// CheckAllUpgrades checks for upgrades for all registered plugins.
func (c *UpgradeChecker) CheckAllUpgrades(ctx context.Context) ([]UpgradeInfo, error) {
	if c.registry == nil {
		return nil, fmt.Errorf("registry not initialized")
	}

	plugins := c.registry.List()
	results := make([]UpgradeInfo, 0, len(plugins))

	for _, plugin := range plugins {
		info, err := c.CheckUpgrade(ctx, plugin.Manifest.Name)
		if err != nil {
			// Skip plugins that can't be checked
			continue
		}
		results = append(results, *info)
	}

	return results, nil
}

// Upgrade upgrades a plugin to the latest version.
func (c *UpgradeChecker) Upgrade(ctx context.Context, name string, dryRun bool) (*UpgradeInfo, error) {
	info, err := c.CheckUpgrade(ctx, name)
	if err != nil {
		return nil, err
	}

	if !info.UpgradeAvailable {
		return info, nil
	}

	if dryRun {
		return info, nil
	}

	// Get the plugin
	plugin, ok := c.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("plugin %q not found", name)
	}

	// Update using git
	if err := c.cloner.Update(ctx, plugin.Path); err != nil {
		return nil, fmt.Errorf("failed to update plugin: %w", err)
	}

	// Reload the plugin to update manifest
	loader := NewLoader()
	updatedPlugin, err := loader.LoadFromPath(plugin.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to reload plugin after update: %w", err)
	}

	// Update registry
	if !c.registry.Remove(name) {
		return nil, fmt.Errorf("failed to remove old plugin from registry")
	}

	if err := c.registry.Register(updatedPlugin); err != nil {
		return nil, fmt.Errorf("failed to register updated plugin: %w", err)
	}

	// Update info with new version
	info.CurrentVersion = updatedPlugin.Manifest.Version
	info.UpgradeAvailable = false

	return info, nil
}

// UpgradeAll upgrades all plugins with available updates.
func (c *UpgradeChecker) UpgradeAll(ctx context.Context, dryRun bool) ([]UpgradeInfo, error) {
	upgrades, err := c.CheckAllUpgrades(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]UpgradeInfo, 0)
	for _, info := range upgrades {
		if !info.UpgradeAvailable {
			continue
		}

		result, err := c.Upgrade(ctx, info.Name, dryRun)
		if err != nil {
			// Record the failed upgrade but continue
			info.UpgradeAvailable = false
			results = append(results, info)
			continue
		}

		results = append(results, *result)
	}

	return results, nil
}

// getGitRemoteURL reads the git remote URL from a plugin path.
func getGitRemoteURL(path string) string {
	gitConfigPath := filepath.Join(path, ".git", "config")
	content, err := os.ReadFile(gitConfigPath)
	if err != nil {
		return ""
	}

	// Simple parsing - look for url = line after [remote "origin"]
	lines := splitLines(string(content))
	inOrigin := false
	for _, line := range lines {
		line = trimSpace(line)
		if line == `[remote "origin"]` {
			inOrigin = true
			continue
		}
		if inOrigin {
			if len(line) > 0 && line[0] == '[' {
				inOrigin = false
				continue
			}
			if hasPrefix(line, "url = ") {
				return line[6:]
			}
		}
	}
	return ""
}

// splitLines splits a string into lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// trimSpace removes leading and trailing whitespace.
func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// hasPrefix checks if s starts with prefix.
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// normalizeVersion ensures version has "v" prefix for semver comparison.
func normalizeVersion(v string) string {
	if v == "" {
		return "v0.0.0"
	}
	if v[0] != 'v' {
		return "v" + v
	}
	return v
}

// isGitHubURL checks if a URL is a GitHub repository URL.
func isGitHubURL(url string) bool {
	return len(url) > 19 && url[:19] == "https://github.com/" ||
		len(url) > 15 && url[:15] == "git@github.com:"
}

// constructChangelogURL builds a changelog URL for a GitHub repository.
func constructChangelogURL(repoURL, version string) string {
	// Convert git URL to web URL
	webURL := repoURL
	if len(webURL) > 15 && webURL[:15] == "git@github.com:" {
		webURL = "https://github.com/" + webURL[15:]
	}
	if len(webURL) > 4 && webURL[len(webURL)-4:] == ".git" {
		webURL = webURL[:len(webURL)-4]
	}

	return webURL + "/releases/tag/" + version
}

// FormatUpgradeInfo formats upgrade information for display.
func FormatUpgradeInfo(info *UpgradeInfo) string {
	if info == nil {
		return ""
	}

	if !info.UpgradeAvailable {
		return fmt.Sprintf("%s: %s (up to date)", info.Name, info.CurrentVersion)
	}

	return fmt.Sprintf("%s: %s → %s", info.Name, info.CurrentVersion, info.LatestVersion)
}

// FormatUpgradeList formats a list of upgrade info for display.
func FormatUpgradeList(infos []UpgradeInfo) string {
	if len(infos) == 0 {
		return "No plugins installed."
	}

	var hasUpgrades bool
	for _, info := range infos {
		if info.UpgradeAvailable {
			hasUpgrades = true
			break
		}
	}

	if !hasUpgrades {
		return "All plugins are up to date."
	}

	result := "Available upgrades:\n\n"
	for _, info := range infos {
		if info.UpgradeAvailable {
			result += fmt.Sprintf("  %s\n", FormatUpgradeInfo(&info))
			if info.ChangelogURL != "" {
				result += fmt.Sprintf("    → %s\n", info.ChangelogURL)
			}
		}
	}

	return result
}
