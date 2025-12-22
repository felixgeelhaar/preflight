package config

// ProvenanceMap tracks which layer each value came from.
type ProvenanceMap map[string]map[string]string

// MergedConfig is the result of merging multiple layers.
type MergedConfig struct {
	Packages   PackageSet
	Files      []FileDeclaration
	Git        GitConfig
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
	filesMap := make(map[string]FileDeclaration)
	aliasesMap := make(map[string]string)
	includesSet := make(map[string]bool)

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

		// Merge apt packages
		for _, pkg := range layer.Packages.Apt.Packages {
			// Simple append for apt packages (dedup could be added)
			merged.Packages.Apt.Packages = append(merged.Packages.Apt.Packages, pkg)
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
	}

	// Convert files map to slice
	for _, file := range filesMap {
		merged.Files = append(merged.Files, file)
	}

	// Convert aliases map to merged config
	if len(aliasesMap) > 0 {
		merged.Git.Aliases = aliasesMap
	}

	return merged, nil
}

func (m *Merger) trackProvenance(merged *MergedConfig, path, value, source string) {
	if merged.provenance[path] == nil {
		merged.provenance[path] = make(map[string]string)
	}
	merged.provenance[path][value] = source
}
