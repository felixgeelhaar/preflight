package config

// Raw converts MergedConfig to a raw map format for providers.
func (m *MergedConfig) Raw() map[string]interface{} {
	raw := make(map[string]interface{})

	// Convert brew packages - use []interface{} for compatibility with parsers
	brew := make(map[string]interface{})
	brew["taps"] = toInterfaceSlice(m.Packages.Brew.Taps)
	brew["formulae"] = toInterfaceSlice(m.Packages.Brew.Formulae)
	brew["casks"] = toInterfaceSlice(m.Packages.Brew.Casks)
	raw["brew"] = brew

	// Convert apt packages
	apt := make(map[string]interface{})
	apt["ppas"] = toInterfaceSlice(m.Packages.Apt.PPAs)
	apt["packages"] = toInterfaceSlice(m.Packages.Apt.Packages)
	raw["apt"] = apt

	// Convert files - transform FileDeclaration to provider format
	// For now, map generated files to links, templates to templates
	files := make(map[string]interface{})
	var links []interface{}
	var templates []interface{}

	for _, f := range m.Files {
		switch f.Mode {
		case FileModeGenerated, FileModeBYO:
			// Generated/BYO files become links
			links = append(links, map[string]interface{}{
				"src":  f.Template,
				"dest": f.Path,
			})
		case FileModeTemplate:
			// Template files become templates
			templates = append(templates, map[string]interface{}{
				"src":  f.Template,
				"dest": f.Path,
			})
		}
	}

	files["links"] = links
	files["templates"] = templates
	files["copies"] = []interface{}{} // Empty for now
	raw["files"] = files

	// Convert git config
	git := make(map[string]interface{})

	// User section
	user := make(map[string]interface{})
	if m.Git.User.Name != "" {
		user["name"] = m.Git.User.Name
	}
	if m.Git.User.Email != "" {
		user["email"] = m.Git.User.Email
	}
	if m.Git.User.SigningKey != "" {
		user["signingkey"] = m.Git.User.SigningKey
	}
	if len(user) > 0 {
		git["user"] = user
	}

	// Core section
	core := make(map[string]interface{})
	if m.Git.Core.Editor != "" {
		core["editor"] = m.Git.Core.Editor
	}
	if m.Git.Core.AutoCRLF != "" {
		core["autocrlf"] = m.Git.Core.AutoCRLF
	}
	if m.Git.Core.ExcludesFile != "" {
		core["excludesfile"] = m.Git.Core.ExcludesFile
	}
	if len(core) > 0 {
		git["core"] = core
	}

	// Commit section
	commit := make(map[string]interface{})
	if m.Git.Commit.GPGSign {
		commit["gpgsign"] = true
	}
	if len(commit) > 0 {
		git["commit"] = commit
	}

	// GPG section
	gpg := make(map[string]interface{})
	if m.Git.GPG.Format != "" {
		gpg["format"] = m.Git.GPG.Format
	}
	if m.Git.GPG.Program != "" {
		gpg["program"] = m.Git.GPG.Program
	}
	if len(gpg) > 0 {
		git["gpg"] = gpg
	}

	// Aliases
	if len(m.Git.Aliases) > 0 {
		aliases := make(map[string]interface{})
		for k, v := range m.Git.Aliases {
			aliases[k] = v
		}
		git["alias"] = aliases
	}

	// Includes
	if len(m.Git.Includes) > 0 {
		var includes []interface{}
		for _, inc := range m.Git.Includes {
			incMap := map[string]interface{}{
				"path": inc.Path,
			}
			if inc.IfConfig != "" {
				incMap["ifconfig"] = inc.IfConfig
			}
			includes = append(includes, incMap)
		}
		git["includes"] = includes
	}

	if len(git) > 0 {
		raw["git"] = git
	}

	// Convert SSH config
	ssh := make(map[string]interface{})

	// Include directive
	if m.SSH.Include != "" {
		ssh["include"] = m.SSH.Include
	}

	// Defaults section
	defaults := make(map[string]interface{})
	if m.SSH.Defaults.AddKeysToAgent {
		defaults["addkeystoagent"] = true
	}
	if m.SSH.Defaults.IdentitiesOnly {
		defaults["identitiesonly"] = true
	}
	if m.SSH.Defaults.ForwardAgent {
		defaults["forwardagent"] = true
	}
	if m.SSH.Defaults.ServerAliveInterval > 0 {
		defaults["serveraliveinterval"] = m.SSH.Defaults.ServerAliveInterval
	}
	if m.SSH.Defaults.ServerAliveCountMax > 0 {
		defaults["serveralivecountmax"] = m.SSH.Defaults.ServerAliveCountMax
	}
	if len(defaults) > 0 {
		ssh["defaults"] = defaults
	}

	// Hosts section
	if len(m.SSH.Hosts) > 0 {
		var hosts []interface{}
		for _, h := range m.SSH.Hosts {
			hostMap := map[string]interface{}{
				"host": h.Host,
			}
			if h.HostName != "" {
				hostMap["hostname"] = h.HostName
			}
			if h.User != "" {
				hostMap["user"] = h.User
			}
			if h.Port > 0 {
				hostMap["port"] = h.Port
			}
			if h.IdentityFile != "" {
				hostMap["identityfile"] = h.IdentityFile
			}
			if h.IdentitiesOnly {
				hostMap["identitiesonly"] = true
			}
			if h.ForwardAgent {
				hostMap["forwardagent"] = true
			}
			if h.ProxyCommand != "" {
				hostMap["proxycommand"] = h.ProxyCommand
			}
			if h.ProxyJump != "" {
				hostMap["proxyjump"] = h.ProxyJump
			}
			if h.LocalForward != "" {
				hostMap["localforward"] = h.LocalForward
			}
			if h.RemoteForward != "" {
				hostMap["remoteforward"] = h.RemoteForward
			}
			if h.AddKeysToAgent {
				hostMap["addkeystoagent"] = true
			}
			if h.UseKeychain {
				hostMap["usekeychain"] = true
			}
			hosts = append(hosts, hostMap)
		}
		ssh["hosts"] = hosts
	}

	// Matches section
	if len(m.SSH.Matches) > 0 {
		var matches []interface{}
		for _, match := range m.SSH.Matches {
			matchMap := map[string]interface{}{
				"match": match.Match,
			}
			if match.HostName != "" {
				matchMap["hostname"] = match.HostName
			}
			if match.User != "" {
				matchMap["user"] = match.User
			}
			if match.IdentityFile != "" {
				matchMap["identityfile"] = match.IdentityFile
			}
			if match.ProxyCommand != "" {
				matchMap["proxycommand"] = match.ProxyCommand
			}
			if match.ProxyJump != "" {
				matchMap["proxyjump"] = match.ProxyJump
			}
			matches = append(matches, matchMap)
		}
		ssh["matches"] = matches
	}

	if len(ssh) > 0 {
		raw["ssh"] = ssh
	}

	// Convert runtime config
	runtime := make(map[string]interface{})

	if m.Runtime.Backend != "" {
		runtime["backend"] = m.Runtime.Backend
	}
	if m.Runtime.Scope != "" {
		runtime["scope"] = m.Runtime.Scope
	}

	// Tools section
	if len(m.Runtime.Tools) > 0 {
		var tools []interface{}
		for _, t := range m.Runtime.Tools {
			toolMap := map[string]interface{}{
				"name":    t.Name,
				"version": t.Version,
			}
			tools = append(tools, toolMap)
		}
		runtime["tools"] = tools
	}

	// Plugins section
	if len(m.Runtime.Plugins) > 0 {
		var plugins []interface{}
		for _, p := range m.Runtime.Plugins {
			pluginMap := map[string]interface{}{
				"name": p.Name,
			}
			if p.URL != "" {
				pluginMap["url"] = p.URL
			}
			plugins = append(plugins, pluginMap)
		}
		runtime["plugins"] = plugins
	}

	if len(runtime) > 0 {
		raw["runtime"] = runtime
	}

	// Convert shell config
	shell := make(map[string]interface{})

	if m.Shell.Default != "" {
		shell["default"] = m.Shell.Default
	}

	// Shells section
	if len(m.Shell.Shells) > 0 {
		var shells []interface{}
		for _, s := range m.Shell.Shells {
			shellMap := map[string]interface{}{
				"name": s.Name,
			}
			if s.Framework != "" {
				shellMap["framework"] = s.Framework
			}
			if s.Theme != "" {
				shellMap["theme"] = s.Theme
			}
			if len(s.Plugins) > 0 {
				shellMap["plugins"] = toInterfaceSlice(s.Plugins)
			}
			if len(s.CustomPlugins) > 0 {
				var customPlugins []interface{}
				for _, cp := range s.CustomPlugins {
					customPlugins = append(customPlugins, map[string]interface{}{
						"name": cp.Name,
						"repo": cp.Repo,
					})
				}
				shellMap["custom_plugins"] = customPlugins
			}
			shells = append(shells, shellMap)
		}
		shell["shells"] = shells
	}

	// Starship section
	if m.Shell.Starship.Enabled {
		starship := map[string]interface{}{
			"enabled": true,
		}
		if m.Shell.Starship.Preset != "" {
			starship["preset"] = m.Shell.Starship.Preset
		}
		shell["starship"] = starship
	}

	// Env section
	if len(m.Shell.Env) > 0 {
		env := make(map[string]interface{})
		for k, v := range m.Shell.Env {
			env[k] = v
		}
		shell["env"] = env
	}

	// Aliases section
	if len(m.Shell.Aliases) > 0 {
		aliases := make(map[string]interface{})
		for k, v := range m.Shell.Aliases {
			aliases[k] = v
		}
		shell["aliases"] = aliases
	}

	if len(shell) > 0 {
		raw["shell"] = shell
	}

	// Convert nvim config
	nvim := make(map[string]interface{})

	if m.Nvim.Preset != "" {
		nvim["preset"] = m.Nvim.Preset
	}
	if m.Nvim.PluginManager != "" {
		nvim["plugin_manager"] = m.Nvim.PluginManager
	}
	if m.Nvim.ConfigRepo != "" {
		nvim["config_repo"] = m.Nvim.ConfigRepo
	}
	if m.Nvim.EnsureInstall {
		nvim["ensure_install"] = true
	}

	if len(nvim) > 0 {
		raw["nvim"] = nvim
	}

	// Convert VSCode config
	vscode := make(map[string]interface{})

	if len(m.VSCode.Extensions) > 0 {
		vscode["extensions"] = toInterfaceSlice(m.VSCode.Extensions)
	}

	if len(m.VSCode.Settings) > 0 {
		settings := make(map[string]interface{})
		for k, v := range m.VSCode.Settings {
			settings[k] = v
		}
		vscode["settings"] = settings
	}

	if len(m.VSCode.Keybindings) > 0 {
		var keybindings []interface{}
		for _, kb := range m.VSCode.Keybindings {
			kbMap := map[string]interface{}{
				"key":     kb.Key,
				"command": kb.Command,
			}
			if kb.When != "" {
				kbMap["when"] = kb.When
			}
			if kb.Args != "" {
				kbMap["args"] = kb.Args
			}
			keybindings = append(keybindings, kbMap)
		}
		vscode["keybindings"] = keybindings
	}

	if len(vscode) > 0 {
		raw["vscode"] = vscode
	}

	return raw
}

// toInterfaceSlice converts a []string to []interface{}.
func toInterfaceSlice(ss []string) []interface{} {
	result := make([]interface{}, len(ss))
	for i, s := range ss {
		result[i] = s
	}
	return result
}
