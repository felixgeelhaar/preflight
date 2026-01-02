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
	Tmux       TmuxConfig
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
	// Calculate capacity hints for pre-allocation
	var formulaeCount, casksCount, tapsCount, ppasCount, aptPkgCount int
	var npmPkgCount, goToolsCount, pipPkgCount, gemCount, cratesCount int
	var filesCount, aliasesCount, includesCount, sshHostsCount, sshMatchesCount int
	var toolsCount, pluginsCount, shellsCount, envCount, aliasCount int
	var extCount, keybindingsCount int

	for _, layer := range layers {
		formulaeCount += len(layer.Packages.Brew.Formulae)
		casksCount += len(layer.Packages.Brew.Casks)
		tapsCount += len(layer.Packages.Brew.Taps)
		ppasCount += len(layer.Packages.Apt.PPAs)
		aptPkgCount += len(layer.Packages.Apt.Packages)
		npmPkgCount += len(layer.Packages.Npm.Packages)
		goToolsCount += len(layer.Packages.Go.Tools)
		pipPkgCount += len(layer.Packages.Pip.Packages)
		gemCount += len(layer.Packages.Gem.Gems)
		cratesCount += len(layer.Packages.Cargo.Crates)
		filesCount += len(layer.Files)
		aliasesCount += len(layer.Git.Aliases)
		includesCount += len(layer.Git.Includes)
		sshHostsCount += len(layer.SSH.Hosts)
		sshMatchesCount += len(layer.SSH.Matches)
		toolsCount += len(layer.Runtime.Tools)
		pluginsCount += len(layer.Runtime.Plugins)
		shellsCount += len(layer.Shell.Shells)
		envCount += len(layer.Shell.Env)
		aliasCount += len(layer.Shell.Aliases)
		extCount += len(layer.VSCode.Extensions)
		keybindingsCount += len(layer.VSCode.Keybindings)
	}

	merged := &MergedConfig{
		provenance: make(ProvenanceMap, 32), // Common paths are limited
	}

	// Pre-allocate maps with calculated capacity hints to avoid rehashing
	formulaeSet := make(map[string]bool, formulaeCount)
	casksSet := make(map[string]bool, casksCount)
	tapsSet := make(map[string]bool, tapsCount)
	ppasSet := make(map[string]bool, ppasCount)
	aptPackagesSet := make(map[string]bool, aptPkgCount)
	npmPackagesSet := make(map[string]bool, npmPkgCount)
	goToolsSet := make(map[string]bool, goToolsCount)
	pipPackagesSet := make(map[string]bool, pipPkgCount)
	gemsSet := make(map[string]bool, gemCount)
	cratesSet := make(map[string]bool, cratesCount)
	filesMap := make(map[string]FileDeclaration, filesCount)
	aliasesMap := make(map[string]string, aliasesCount)
	includesSet := make(map[string]bool, includesCount)
	sshHostsMap := make(map[string]SSHHostConfig, sshHostsCount)
	sshMatchesSet := make(map[string]bool, sshMatchesCount)
	runtimeToolsMap := make(map[string]RuntimeToolConfig, toolsCount)
	runtimePluginsMap := make(map[string]RuntimePluginConfig, pluginsCount)
	shellsMap := make(map[string]ShellConfigEntry, shellsCount)
	shellEnvMap := make(map[string]string, envCount)
	shellAliasesMap := make(map[string]string, aliasCount)
	vscodeExtensionsSet := make(map[string]bool, extCount)
	vscodeKeybindingsSet := make(map[string]bool, keybindingsCount)

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

		// Merge npm packages
		for _, pkg := range layer.Packages.Npm.Packages {
			if !npmPackagesSet[pkg] {
				npmPackagesSet[pkg] = true
				merged.Packages.Npm.Packages = append(merged.Packages.Npm.Packages, pkg)
			}
			m.trackProvenance(merged, "packages.npm.packages", pkg, layer.Provenance)
		}

		// Merge go tools
		for _, tool := range layer.Packages.Go.Tools {
			if !goToolsSet[tool] {
				goToolsSet[tool] = true
				merged.Packages.Go.Tools = append(merged.Packages.Go.Tools, tool)
			}
			m.trackProvenance(merged, "packages.go.tools", tool, layer.Provenance)
		}

		// Merge pip packages
		for _, pkg := range layer.Packages.Pip.Packages {
			if !pipPackagesSet[pkg] {
				pipPackagesSet[pkg] = true
				merged.Packages.Pip.Packages = append(merged.Packages.Pip.Packages, pkg)
			}
			m.trackProvenance(merged, "packages.pip.packages", pkg, layer.Provenance)
		}

		// Merge gem packages
		for _, gem := range layer.Packages.Gem.Gems {
			if !gemsSet[gem] {
				gemsSet[gem] = true
				merged.Packages.Gem.Gems = append(merged.Packages.Gem.Gems, gem)
			}
			m.trackProvenance(merged, "packages.gem.gems", gem, layer.Provenance)
		}

		// Merge cargo crates
		for _, crate := range layer.Packages.Cargo.Crates {
			if !cratesSet[crate] {
				cratesSet[crate] = true
				merged.Packages.Cargo.Crates = append(merged.Packages.Cargo.Crates, crate)
			}
			m.trackProvenance(merged, "packages.cargo.crates", crate, layer.Provenance)
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

		// Merge git config_source (scalar: last-wins)
		if layer.Git.ConfigSource != "" {
			merged.Git.ConfigSource = layer.Git.ConfigSource
			m.trackProvenance(merged, "git.config_source", layer.Git.ConfigSource, layer.Provenance)
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

		// Merge SSH config_source (scalar: last-wins)
		if layer.SSH.ConfigSource != "" {
			merged.SSH.ConfigSource = layer.SSH.ConfigSource
			m.trackProvenance(merged, "ssh.config_source", layer.SSH.ConfigSource, layer.Provenance)
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
		if layer.Shell.Starship.ConfigSource != "" {
			merged.Shell.Starship.ConfigSource = layer.Shell.Starship.ConfigSource
			m.trackProvenance(merged, "shell.starship.config_source", layer.Shell.Starship.ConfigSource, layer.Provenance)
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

		// Merge shell config_source (struct: last-wins per field)
		if layer.Shell.ConfigSource != nil {
			if merged.Shell.ConfigSource == nil {
				merged.Shell.ConfigSource = &ShellConfigSource{}
			}
			if layer.Shell.ConfigSource.Aliases != "" {
				merged.Shell.ConfigSource.Aliases = layer.Shell.ConfigSource.Aliases
				m.trackProvenance(merged, "shell.config_source.aliases", layer.Shell.ConfigSource.Aliases, layer.Provenance)
			}
			if layer.Shell.ConfigSource.Functions != "" {
				merged.Shell.ConfigSource.Functions = layer.Shell.ConfigSource.Functions
				m.trackProvenance(merged, "shell.config_source.functions", layer.Shell.ConfigSource.Functions, layer.Provenance)
			}
			if layer.Shell.ConfigSource.Env != "" {
				merged.Shell.ConfigSource.Env = layer.Shell.ConfigSource.Env
				m.trackProvenance(merged, "shell.config_source.env", layer.Shell.ConfigSource.Env, layer.Provenance)
			}
			if layer.Shell.ConfigSource.Dir != "" {
				merged.Shell.ConfigSource.Dir = layer.Shell.ConfigSource.Dir
				m.trackProvenance(merged, "shell.config_source.dir", layer.Shell.ConfigSource.Dir, layer.Provenance)
			}
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
		if layer.Nvim.ConfigSource != "" {
			merged.Nvim.ConfigSource = layer.Nvim.ConfigSource
			m.trackProvenance(merged, "nvim.config_source", layer.Nvim.ConfigSource, layer.Provenance)
		}

		// Merge nvim extra_plugins (set union)
		for _, plugin := range layer.Nvim.ExtraPlugins {
			// Deduplicate by checking if already present
			found := false
			for _, existing := range merged.Nvim.ExtraPlugins {
				if existing == plugin {
					found = true
					break
				}
			}
			if !found {
				merged.Nvim.ExtraPlugins = append(merged.Nvim.ExtraPlugins, plugin)
			}
			m.trackProvenance(merged, "nvim.extra_plugins", plugin, layer.Provenance)
		}

		// Merge VSCode extensions (set union) - O(n) with map lookup
		for _, ext := range layer.VSCode.Extensions {
			if !vscodeExtensionsSet[ext] {
				vscodeExtensionsSet[ext] = true
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

		// Merge VSCode keybindings (set union by key+command) - O(n) with map lookup
		for _, kb := range layer.VSCode.Keybindings {
			kbKey := kb.Key + ":" + kb.Command
			if !vscodeKeybindingsSet[kbKey] {
				vscodeKeybindingsSet[kbKey] = true
				merged.VSCode.Keybindings = append(merged.VSCode.Keybindings, kb)
			}
			m.trackProvenance(merged, "vscode.keybindings", kb.Key, layer.Provenance)
		}

		// Merge VSCode config_source (scalar: last-wins)
		if layer.VSCode.ConfigSource != "" {
			merged.VSCode.ConfigSource = layer.VSCode.ConfigSource
			m.trackProvenance(merged, "vscode.config_source", layer.VSCode.ConfigSource, layer.Provenance)
		}

		// Merge Tmux config (scalars: last-wins)
		if layer.Tmux.ConfigSource != "" {
			merged.Tmux.ConfigSource = layer.Tmux.ConfigSource
			m.trackProvenance(merged, "tmux.config_source", layer.Tmux.ConfigSource, layer.Provenance)
		}

		// Merge Tmux plugins (set union)
		for _, plugin := range layer.Tmux.Plugins {
			found := false
			for _, existing := range merged.Tmux.Plugins {
				if existing == plugin {
					found = true
					break
				}
			}
			if !found {
				merged.Tmux.Plugins = append(merged.Tmux.Plugins, plugin)
			}
			m.trackProvenance(merged, "tmux.plugins", plugin, layer.Provenance)
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
