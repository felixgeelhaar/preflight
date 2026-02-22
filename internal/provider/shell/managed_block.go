package shell

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	blockStartFmt = "# >>> preflight %s >>>"
	blockEndFmt   = "# <<< preflight %s <<<"
)

// ReadManagedBlock extracts the content between preflight managed block markers.
// Returns empty string if the block is not found.
func ReadManagedBlock(content, section string) string {
	start := fmt.Sprintf(blockStartFmt, section)
	end := fmt.Sprintf(blockEndFmt, section)

	startIdx := strings.Index(content, start)
	if startIdx == -1 {
		return ""
	}

	endIdx := strings.Index(content, end)
	if endIdx == -1 {
		return ""
	}

	// Extract content between markers (after the start line, before the end line)
	blockStart := startIdx + len(start)
	// Skip the newline after the start marker
	if blockStart < len(content) && content[blockStart] == '\n' {
		blockStart++
	}

	if blockStart >= endIdx {
		return ""
	}

	return content[blockStart:endIdx]
}

// WriteManagedBlock replaces (or appends) a managed block in the content.
// If the block already exists, it is replaced. Otherwise, it is appended.
func WriteManagedBlock(content, section, block string) string {
	start := fmt.Sprintf(blockStartFmt, section)
	end := fmt.Sprintf(blockEndFmt, section)

	managedBlock := start + "\n" + block + end + "\n"

	startIdx := strings.Index(content, start)
	if startIdx == -1 {
		// Block doesn't exist, append it
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + "\n" + managedBlock
	}

	endIdx := strings.Index(content, end)
	if endIdx == -1 {
		// Malformed block: start exists but no end. Replace from start to EOF.
		return content[:startIdx] + managedBlock
	}

	// Replace existing block (including end marker and trailing newline)
	afterEnd := endIdx + len(end)
	if afterEnd < len(content) && content[afterEnd] == '\n' {
		afterEnd++
	}

	return content[:startIdx] + managedBlock + content[afterEnd:]
}

// generateEnvBlock produces the content for a managed env block.
func generateEnvBlock(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "export %s=%q\n", k, env[k])
	}
	return b.String()
}

// generateAliasBlock produces the content for a managed aliases block.
func generateAliasBlock(aliases map[string]string) string {
	if len(aliases) == 0 {
		return ""
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(aliases))
	for k := range aliases {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "alias %s=%q\n", k, aliases[k])
	}
	return b.String()
}

// pluginsLineRe matches the plugins=(...) line in .zshrc.
var pluginsLineRe = regexp.MustCompile(`(?m)^plugins=\(([^)]*)\)\s*$`)

// containsPlugin checks if a plugin is listed in the shell config content.
// For oh-my-zsh, it checks the plugins=(...) line.
func containsPlugin(content, plugin string) bool {
	matches := pluginsLineRe.FindStringSubmatch(content)
	if len(matches) < 2 {
		return false
	}

	pluginList := strings.Fields(matches[1])
	for _, p := range pluginList {
		if p == plugin {
			return true
		}
	}
	return false
}

// addPluginToConfig adds a plugin to the plugins=(...) line in shell config.
// If no plugins line exists, one is created.
func addPluginToConfig(content, plugin string) string {
	if containsPlugin(content, plugin) {
		return content
	}

	if pluginsLineRe.MatchString(content) {
		// Add to existing plugins line
		return pluginsLineRe.ReplaceAllStringFunc(content, func(match string) string {
			sub := pluginsLineRe.FindStringSubmatch(match)
			if len(sub) < 2 {
				return match
			}
			existing := strings.TrimSpace(sub[1])
			if existing == "" {
				return fmt.Sprintf("plugins=(%s)", plugin)
			}
			return fmt.Sprintf("plugins=(%s %s)", existing, plugin)
		})
	}

	// No plugins line found, append one
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + fmt.Sprintf("plugins=(%s)\n", plugin)
}

// shellConfigPath returns the config file path for a given shell name.
func shellConfigPath(shellName string) string {
	switch shellName {
	case "zsh":
		return "~/.zshrc"
	case "bash":
		return "~/.bashrc"
	case "fish":
		return "~/.config/fish/config.fish"
	default:
		return ""
	}
}
