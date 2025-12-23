package config

// ProvenanceMap tracks which layer each value came from.
type ProvenanceMap map[string]map[string]string

// MergedConfig is the result of merging multiple layers.
type MergedConfig struct {
	Packages   PackageSet
	Files      []FileDeclaration
	Git        GitConfig
	SSH        SSHConfig
	Runtime    RuntimeConfig
	Shell      ShellConfig
	Nvim       NvimConfig
	VSCode     VSCodeConfig
	provenance ProvenanceMap
}

// GetProvenance returns the source layer for a given path and value.
func (m *MergedConfig) GetProvenance(path, value string) string {
	if m.provenance == nil {
		return ""
	}
	if pathMap, ok := m.provenance[path]; ok {
		return pathMap[value]
	}
	return ""
}

// Merger merges multiple layers into a single MergedConfig.
type Merger struct{}

// NewMerger creates a new Merger.
func NewMerger() *Merger {
	return &Merger{}
}

// Merge combines layers according to merge semantics.
// - Scalars: last-wins
// - Maps: deep merge
// - Lists: set union (deduplicated)
func (m *Merger) Merge(layers []Layer) (*MergedConfig, error) {
	merged := &MergedConfig{
		provenance: make(ProvenanceMap),
	}

	// Track seen items for deduplication
	formulaeSet := make(map[string]bool)
	casksSet := make(map[string]bool)
	tapsSet := make(map[string]bool)
	ppasSet := make(map[string]bool)
	aptPackagesSet := make(map[string]bool)
	filesMap := make(map[string]FileDeclaration)
	aliasesMap := make(map[string]string)
	includesSet := make(map[string]bool)
	sshHostsMap := make(map[string]SSHHostConfig)
	sshMatchesSet := make(map[string]bool)
	runtimeToolsMap := make(map[string]RuntimeToolConfig)
	runtimePluginsMap := make(map[string]RuntimePluginConfig)
	shellsMap := make(map[string]ShellConfigEntry)
	shellEnvMap := make(map[string]string)
	shellAliasesMap := make(map[string]string)

	for _, layer := range layers {
		// Merge brew formulae
		for _, formula := range layer.Packages.Brew.Formulae {
			if !formulaeSet[formula] {
				formulaeSet[formula] = true
				merged.Packages.Brew.Formulae = append(merged.Packages.Brew.Formulae, formula)
			}
			m.trackProvenance(merged, "packages.brew.formulae", formula, layer.Provenance)
		}

		// Merge brew casks
		for _, cask := range layer.Packages.Brew.Casks {
			if !casksSet[cask] {
				casksSet[cask] = true
				merged.Packages.Brew.Casks = append(merged.Packages.Brew.Casks, cask)
			}
			m.trackProvenance(merged, "packages.brew.casks", cask, layer.Provenance)
		}

		// Merge brew taps
		for _, tap := range layer.Packages.Brew.Taps {
			if !tapsSet[tap] {
				tapsSet[tap] = true
				merged.Packages.Brew.Taps = append(merged.Packages.Brew.Taps, tap)
			}
			m.trackProvenance(merged, "packages.brew.taps", tap, layer.Provenance)
		}

		// Merge apt PPAs
		for _, ppa := range layer.Packages.Apt.PPAs {
			if !ppasSet[ppa] {
				ppasSet[ppa] = true
				merged.Packages.Apt.PPAs = append(merged.Packages.Apt.PPAs, ppa)
			}
			m.trackProvenance(merged, "packages.apt.ppas", ppa, layer.Provenance)
		}

		// Merge apt packages
		for _, pkg := range layer.Packages.Apt.Packages {
			if !aptPackagesSet[pkg] {
				aptPackagesSet[pkg] = true
				merged.Packages.Apt.Packages = append(merged.Packages.Apt.Packages, pkg)
			}
			m.trackProvenance(merged, "packages.apt.packages", pkg, layer.Provenance)
		}

		// Merge files (last-wins for same path)
		for _, file := range layer.Files {
			filesMap[file.Path] = file
			m.trackProvenance(merged, "files", file.Path, layer.Provenance)
		}

		// Merge git config (scalars: last-wins)
		if layer.Git.User.Name != "" {
			merged.Git.User.Name = layer.Git.User.Name
			m.trackProvenance(merged, "git.user.name", layer.Git.User.Name, layer.Provenance)
		}
		if layer.Git.User.Email != "" {
			merged.Git.User.Email = layer.Git.User.Email
			m.trackProvenance(merged, "git.user.email", layer.Git.User.Email, layer.Provenance)
		}
		if layer.Git.User.SigningKey != "" {
			merged.Git.User.SigningKey = layer.Git.User.SigningKey
			m.trackProvenance(merged, "git.user.signingkey", layer.Git.User.SigningKey, layer.Provenance)
		}
		if layer.Git.Core.Editor != "" {
			merged.Git.Core.Editor = layer.Git.Core.Editor
			m.trackProvenance(merged, "git.core.editor", layer.Git.Core.Editor, layer.Provenance)
		}
		if layer.Git.Core.AutoCRLF != "" {
			merged.Git.Core.AutoCRLF = layer.Git.Core.AutoCRLF
			m.trackProvenance(merged, "git.core.autocrlf", layer.Git.Core.AutoCRLF, layer.Provenance)
		}
		if layer.Git.Core.ExcludesFile != "" {
			merged.Git.Core.ExcludesFile = layer.Git.Core.ExcludesFile
			m.trackProvenance(merged, "git.core.excludesfile", layer.Git.Core.ExcludesFile, layer.Provenance)
		}
		if layer.Git.Commit.GPGSign {
			merged.Git.Commit.GPGSign = layer.Git.Commit.GPGSign
			m.trackProvenance(merged, "git.commit.gpgsign", "true", layer.Provenance)
		}
		if layer.Git.GPG.Format != "" {
			merged.Git.GPG.Format = layer.Git.GPG.Format
			m.trackProvenance(merged, "git.gpg.format", layer.Git.GPG.Format, layer.Provenance)
		}
		if layer.Git.GPG.Program != "" {
			merged.Git.GPG.Program = layer.Git.GPG.Program
			m.trackProvenance(merged, "git.gpg.program", layer.Git.GPG.Program, layer.Provenance)
		}

		// Merge aliases (deep merge, last-wins per key)
		for key, value := range layer.Git.Aliases {
			aliasesMap[key] = value
			m.trackProvenance(merged, "git.alias", key, layer.Provenance)
		}

		// Merge includes (set union)
		for _, inc := range layer.Git.Includes {
			key := inc.Path + "|" + inc.IfConfig
			if !includesSet[key] {
				includesSet[key] = true
				merged.Git.Includes = append(merged.Git.Includes, inc)
				m.trackProvenance(merged, "git.includes", inc.Path, layer.Provenance)
			}
		}

		// Merge SSH config
		if layer.SSH.Include != "" {
			merged.SSH.Include = layer.SSH.Include
			m.trackProvenance(merged, "ssh.include", layer.SSH.Include, layer.Provenance)
		}

		// Merge SSH defaults (scalars: last-wins)
		if layer.SSH.Defaults.AddKeysToAgent {
			merged.SSH.Defaults.AddKeysToAgent = true
			m.trackProvenance(merged, "ssh.defaults.addkeystoagent", "true", layer.Provenance)
		}
		if layer.SSH.Defaults.IdentitiesOnly {
			merged.SSH.Defaults.IdentitiesOnly = true
			m.trackProvenance(merged, "ssh.defaults.identitiesonly", "true", layer.Provenance)
		}
		if layer.SSH.Defaults.ForwardAgent {
			merged.SSH.Defaults.ForwardAgent = true
			m.trackProvenance(merged, "ssh.defaults.forwardagent", "true", layer.Provenance)
		}
		if layer.SSH.Defaults.ServerAliveInterval > 0 {
			merged.SSH.Defaults.ServerAliveInterval = layer.SSH.Defaults.ServerAliveInterval
			m.trackProvenance(merged, "ssh.defaults.serveraliveinterval", "set", layer.Provenance)
		}
		if layer.SSH.Defaults.ServerAliveCountMax > 0 {
			merged.SSH.Defaults.ServerAliveCountMax = layer.SSH.Defaults.ServerAliveCountMax
			m.trackProvenance(merged, "ssh.defaults.serveralivecountmax", "set", layer.Provenance)
		}

		// Merge SSH hosts (last-wins per host name)
		for _, host := range layer.SSH.Hosts {
			sshHostsMap[host.Host] = host
			m.trackProvenance(merged, "ssh.hosts", host.Host, layer.Provenance)
		}

		// Merge SSH matches (set union by match pattern)
		for _, match := range layer.SSH.Matches {
			if !sshMatchesSet[match.Match] {
				sshMatchesSet[match.Match] = true
				merged.SSH.Matches = append(merged.SSH.Matches, match)
				m.trackProvenance(merged, "ssh.matches", match.Match, layer.Provenance)
			}
		}

		// Merge runtime config (scalars: last-wins)
		if layer.Runtime.Backend != "" {
			merged.Runtime.Backend = layer.Runtime.Backend
			m.trackProvenance(merged, "runtime.backend", layer.Runtime.Backend, layer.Provenance)
		}
		if layer.Runtime.Scope != "" {
			merged.Runtime.Scope = layer.Runtime.Scope
			m.trackProvenance(merged, "runtime.scope", layer.Runtime.Scope, layer.Provenance)
		}

		// Merge runtime tools (last-wins per tool name)
		for _, tool := range layer.Runtime.Tools {
			runtimeToolsMap[tool.Name] = tool
			m.trackProvenance(merged, "runtime.tools", tool.Name, layer.Provenance)
		}

		// Merge runtime plugins (last-wins per plugin name)
		for _, plugin := range layer.Runtime.Plugins {
			runtimePluginsMap[plugin.Name] = plugin
			m.trackProvenance(merged, "runtime.plugins", plugin.Name, layer.Provenance)
		}

		// Merge shell config (scalars: last-wins)
		if layer.Shell.Default != "" {
			merged.Shell.Default = layer.Shell.Default
			m.trackProvenance(merged, "shell.default", layer.Shell.Default, layer.Provenance)
		}

		// Merge shells (last-wins per shell name)
		for _, sh := range layer.Shell.Shells {
			shellsMap[sh.Name] = sh
			m.trackProvenance(merged, "shell.shells", sh.Name, layer.Provenance)
		}

		// Merge starship (scalars: last-wins)
		if layer.Shell.Starship.Enabled {
			merged.Shell.Starship.Enabled = true
			m.trackProvenance(merged, "shell.starship.enabled", "true", layer.Provenance)
		}
		if layer.Shell.Starship.Preset != "" {
			merged.Shell.Starship.Preset = layer.Shell.Starship.Preset
			m.trackProvenance(merged, "shell.starship.preset", layer.Shell.Starship.Preset, layer.Provenance)
		}

		// Merge shell env (deep merge, last-wins per key)
		for key, value := range layer.Shell.Env {
			shellEnvMap[key] = value
			m.trackProvenance(merged, "shell.env", key, layer.Provenance)
		}

		// Merge shell aliases (deep merge, last-wins per key)
		for key, value := range layer.Shell.Aliases {
			shellAliasesMap[key] = value
			m.trackProvenance(merged, "shell.aliases", key, layer.Provenance)
		}

		// Merge nvim config (scalars: last-wins)
		if layer.Nvim.Preset != "" {
			merged.Nvim.Preset = layer.Nvim.Preset
			m.trackProvenance(merged, "nvim.preset", layer.Nvim.Preset, layer.Provenance)
		}
		if layer.Nvim.PluginManager != "" {
			merged.Nvim.PluginManager = layer.Nvim.PluginManager
			m.trackProvenance(merged, "nvim.plugin_manager", layer.Nvim.PluginManager, layer.Provenance)
		}
		if layer.Nvim.ConfigRepo != "" {
			merged.Nvim.ConfigRepo = layer.Nvim.ConfigRepo
			m.trackProvenance(merged, "nvim.config_repo", layer.Nvim.ConfigRepo, layer.Provenance)
		}
		if layer.Nvim.EnsureInstall {
			merged.Nvim.EnsureInstall = true
			m.trackProvenance(merged, "nvim.ensure_install", "true", layer.Provenance)
		}

		// Merge VSCode extensions (set union)
		for _, ext := range layer.VSCode.Extensions {
			// Check if already present
			found := false
			for _, existing := range merged.VSCode.Extensions {
				if existing == ext {
					found = true
					break
				}
			}
			if !found {
				merged.VSCode.Extensions = append(merged.VSCode.Extensions, ext)
			}
			m.trackProvenance(merged, "vscode.extensions", ext, layer.Provenance)
		}

		// Merge VSCode settings (deep merge, last-wins per key)
		if len(layer.VSCode.Settings) > 0 {
			if merged.VSCode.Settings == nil {
				merged.VSCode.Settings = make(map[string]interface{})
			}
			for key, value := range layer.VSCode.Settings {
				merged.VSCode.Settings[key] = value
				m.trackProvenance(merged, "vscode.settings", key, layer.Provenance)
			}
		}

		// Merge VSCode keybindings (set union by key+command)
		for _, kb := range layer.VSCode.Keybindings {
			// Check if already present
			found := false
			for _, existing := range merged.VSCode.Keybindings {
				if existing.Key == kb.Key && existing.Command == kb.Command {
					found = true
					break
				}
			}
			if !found {
				merged.VSCode.Keybindings = append(merged.VSCode.Keybindings, kb)
			}
			m.trackProvenance(merged, "vscode.keybindings", kb.Key, layer.Provenance)
		}
	}

	// Convert files map to slice
	for _, file := range filesMap {
		merged.Files = append(merged.Files, file)
	}

	// Convert aliases map to merged config
	if len(aliasesMap) > 0 {
		merged.Git.Aliases = aliasesMap
	}

	// Convert SSH hosts map to slice
	for _, host := range sshHostsMap {
		merged.SSH.Hosts = append(merged.SSH.Hosts, host)
	}

	// Convert runtime tools map to slice
	for _, tool := range runtimeToolsMap {
		merged.Runtime.Tools = append(merged.Runtime.Tools, tool)
	}

	// Convert runtime plugins map to slice
	for _, plugin := range runtimePluginsMap {
		merged.Runtime.Plugins = append(merged.Runtime.Plugins, plugin)
	}

	// Convert shell maps to slices
	for _, sh := range shellsMap {
		merged.Shell.Shells = append(merged.Shell.Shells, sh)
	}
	if len(shellEnvMap) > 0 {
		merged.Shell.Env = shellEnvMap
	}
	if len(shellAliasesMap) > 0 {
		merged.Shell.Aliases = shellAliasesMap
	}

	return merged, nil
}

func (m *Merger) trackProvenance(merged *MergedConfig, path, value, source string) {
	if merged.provenance[path] == nil {
		merged.provenance[path] = make(map[string]string)
	}
	merged.provenance[path][value] = source
}
