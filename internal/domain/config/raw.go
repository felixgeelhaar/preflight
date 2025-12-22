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
