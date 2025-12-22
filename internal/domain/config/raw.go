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
