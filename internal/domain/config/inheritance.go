package config

import (
	"fmt"
	"strings"
)

// LayerExtends represents layer inheritance configuration.
type LayerExtends struct {
	Parent   string   `yaml:"extends,omitempty"`  // Single parent layer
	Parents  []string `yaml:"parents,omitempty"`  // Multiple parent layers (merged in order)
	Override bool     `yaml:"override,omitempty"` // If true, child completely replaces parent values
}

// InheritableLayer extends Layer with inheritance support.
type InheritableLayer struct {
	Layer
	Extends LayerExtends `yaml:"-"`
}

// LayerResolver handles layer inheritance resolution.
type LayerResolver struct {
	layers    map[string]*Layer
	resolved  map[string]*Layer
	resolving map[string]bool // Tracks layers currently being resolved (cycle detection)
}

// NewLayerResolver creates a new layer resolver.
func NewLayerResolver() *LayerResolver {
	return &LayerResolver{
		layers:    make(map[string]*Layer),
		resolved:  make(map[string]*Layer),
		resolving: make(map[string]bool),
	}
}

// RegisterLayer adds a layer to the resolver.
func (r *LayerResolver) RegisterLayer(layer *Layer) {
	r.layers[layer.Name.String()] = layer
}

// Resolve resolves all layer inheritance and returns the fully merged layers.
func (r *LayerResolver) Resolve(layerNames []LayerName) ([]*Layer, error) {
	result := make([]*Layer, 0, len(layerNames))

	for _, name := range layerNames {
		resolved, err := r.resolveLayer(name.String())
		if err != nil {
			return nil, err
		}
		result = append(result, resolved)
	}

	return result, nil
}

// resolveLayer recursively resolves a single layer's inheritance.
func (r *LayerResolver) resolveLayer(name string) (*Layer, error) {
	// Check if already resolved
	if resolved, ok := r.resolved[name]; ok {
		return resolved, nil
	}

	// Check for cycles
	if r.resolving[name] {
		return nil, fmt.Errorf("circular layer inheritance detected: %s", name)
	}

	// Get the layer
	layer, ok := r.layers[name]
	if !ok {
		return nil, fmt.Errorf("layer not found: %s", name)
	}

	// Mark as resolving
	r.resolving[name] = true
	defer delete(r.resolving, name)

	// Check for parent layers
	parents := r.getParentLayers(layer)
	if len(parents) == 0 {
		// No inheritance, return as-is
		r.resolved[name] = layer
		return layer, nil
	}

	// Resolve parent layers first
	resolvedParents := make([]*Layer, 0, len(parents))
	for _, parentName := range parents {
		parent, err := r.resolveLayer(parentName)
		if err != nil {
			return nil, fmt.Errorf("resolving parent %s of %s: %w", parentName, name, err)
		}
		resolvedParents = append(resolvedParents, parent)
	}

	// Merge parent layers
	merged := r.mergeLayers(resolvedParents)

	// Merge child on top
	result := r.mergeLayer(merged, layer)
	result.Name = layer.Name
	result.Provenance = fmt.Sprintf("%s (extends: %s)", layer.Provenance, strings.Join(parents, ", "))

	r.resolved[name] = result
	return result, nil
}

// getParentLayers extracts parent layer names from a layer.
// This looks for an "extends" field in the raw config.
func (r *LayerResolver) getParentLayers(_ *Layer) []string {
	// In a real implementation, this would parse the "extends" field
	// from the raw YAML before it's fully parsed into a Layer struct.
	// For now, we return empty as inheritance info needs to be captured
	// during initial parsing.
	return nil
}

// mergeLayers merges multiple layers into one.
func (r *LayerResolver) mergeLayers(layers []*Layer) *Layer {
	if len(layers) == 0 {
		return &Layer{}
	}
	if len(layers) == 1 {
		return layers[0]
	}

	result := layers[0]
	for i := 1; i < len(layers); i++ {
		result = r.mergeLayer(result, layers[i])
	}
	return result
}

// mergeLayer merges child layer on top of parent layer.
func (r *LayerResolver) mergeLayer(parent, child *Layer) *Layer {
	result := &Layer{
		Name:       child.Name,
		Provenance: child.Provenance,
	}

	// Merge packages
	result.Packages = r.mergePackages(parent.Packages, child.Packages)

	// Merge files (child takes precedence for same paths)
	result.Files = r.mergeFiles(parent.Files, child.Files)

	// Merge git config
	result.Git = r.mergeGit(parent.Git, child.Git)

	// Merge SSH config
	result.SSH = r.mergeSSH(parent.SSH, child.SSH)

	// Merge runtime config
	result.Runtime = r.mergeRuntime(parent.Runtime, child.Runtime)

	// Merge shell config
	result.Shell = r.mergeShell(parent.Shell, child.Shell)

	// Merge editor configs
	result.Nvim = r.mergeNvim(parent.Nvim, child.Nvim)
	result.VSCode = r.mergeVSCode(parent.VSCode, child.VSCode)

	return result
}

// mergePackages merges package sets.
func (r *LayerResolver) mergePackages(parent, child PackageSet) PackageSet {
	return PackageSet{
		Brew: BrewPackages{
			Taps:     uniqueStrings(append(parent.Brew.Taps, child.Brew.Taps...)),
			Formulae: uniqueStrings(append(parent.Brew.Formulae, child.Brew.Formulae...)),
			Casks:    uniqueStrings(append(parent.Brew.Casks, child.Brew.Casks...)),
		},
		Apt: AptPackages{
			PPAs:     uniqueStrings(append(parent.Apt.PPAs, child.Apt.PPAs...)),
			Packages: uniqueStrings(append(parent.Apt.Packages, child.Apt.Packages...)),
		},
	}
}

// mergeFiles merges file declarations (child takes precedence).
func (r *LayerResolver) mergeFiles(parent, child []FileDeclaration) []FileDeclaration {
	fileMap := make(map[string]FileDeclaration)

	for _, f := range parent {
		fileMap[f.Path] = f
	}
	for _, f := range child {
		fileMap[f.Path] = f
	}

	result := make([]FileDeclaration, 0, len(fileMap))
	for _, f := range fileMap {
		result = append(result, f)
	}
	return result
}

// mergeGit merges git configurations.
func (r *LayerResolver) mergeGit(parent, child GitConfig) GitConfig {
	result := parent

	// User config (child overrides)
	if child.User.Name != "" {
		result.User.Name = child.User.Name
	}
	if child.User.Email != "" {
		result.User.Email = child.User.Email
	}
	if child.User.SigningKey != "" {
		result.User.SigningKey = child.User.SigningKey
	}

	// Core config
	if child.Core.Editor != "" {
		result.Core.Editor = child.Core.Editor
	}
	if child.Core.AutoCRLF != "" {
		result.Core.AutoCRLF = child.Core.AutoCRLF
	}
	if child.Core.ExcludesFile != "" {
		result.Core.ExcludesFile = child.Core.ExcludesFile
	}

	// Commit config
	if child.Commit.GPGSign {
		result.Commit.GPGSign = true
	}

	// GPG config
	if child.GPG.Format != "" {
		result.GPG.Format = child.GPG.Format
	}
	if child.GPG.Program != "" {
		result.GPG.Program = child.GPG.Program
	}

	// Merge aliases
	if result.Aliases == nil {
		result.Aliases = make(map[string]string)
	}
	for k, v := range child.Aliases {
		result.Aliases[k] = v
	}

	// Merge includes
	result.Includes = append(result.Includes, child.Includes...)

	return result
}

// mergeSSH merges SSH configurations.
func (r *LayerResolver) mergeSSH(parent, child SSHConfig) SSHConfig {
	result := parent

	if child.Include != "" {
		result.Include = child.Include
	}

	// Merge defaults (child overrides)
	if child.Defaults.AddKeysToAgent {
		result.Defaults.AddKeysToAgent = true
	}
	if child.Defaults.IdentitiesOnly {
		result.Defaults.IdentitiesOnly = true
	}
	if child.Defaults.ForwardAgent {
		result.Defaults.ForwardAgent = true
	}
	if child.Defaults.ServerAliveInterval > 0 {
		result.Defaults.ServerAliveInterval = child.Defaults.ServerAliveInterval
	}
	if child.Defaults.ServerAliveCountMax > 0 {
		result.Defaults.ServerAliveCountMax = child.Defaults.ServerAliveCountMax
	}

	// Merge hosts (child takes precedence for same host)
	hostMap := make(map[string]SSHHostConfig)
	for _, h := range parent.Hosts {
		hostMap[h.Host] = h
	}
	for _, h := range child.Hosts {
		hostMap[h.Host] = h
	}
	result.Hosts = make([]SSHHostConfig, 0, len(hostMap))
	for _, h := range hostMap {
		result.Hosts = append(result.Hosts, h)
	}

	// Merge matches
	allMatches := make([]SSHMatchConfig, 0, len(parent.Matches)+len(child.Matches))
	allMatches = append(allMatches, parent.Matches...)
	allMatches = append(allMatches, child.Matches...)
	result.Matches = allMatches

	return result
}

// mergeRuntime merges runtime configurations.
func (r *LayerResolver) mergeRuntime(parent, child RuntimeConfig) RuntimeConfig {
	result := parent

	if child.Backend != "" {
		result.Backend = child.Backend
	}
	if child.Scope != "" {
		result.Scope = child.Scope
	}

	// Merge tools (child versions take precedence)
	toolMap := make(map[string]RuntimeToolConfig)
	for _, t := range parent.Tools {
		toolMap[t.Name] = t
	}
	for _, t := range child.Tools {
		toolMap[t.Name] = t
	}
	result.Tools = make([]RuntimeToolConfig, 0, len(toolMap))
	for _, t := range toolMap {
		result.Tools = append(result.Tools, t)
	}

	// Merge plugins
	allPlugins := make([]RuntimePluginConfig, 0, len(parent.Plugins)+len(child.Plugins))
	allPlugins = append(allPlugins, parent.Plugins...)
	allPlugins = append(allPlugins, child.Plugins...)
	result.Plugins = allPlugins

	return result
}

// mergeShell merges shell configurations.
func (r *LayerResolver) mergeShell(parent, child ShellConfig) ShellConfig {
	result := parent

	if child.Default != "" {
		result.Default = child.Default
	}

	// Merge shells by name
	shellMap := make(map[string]ShellConfigEntry)
	for _, s := range parent.Shells {
		shellMap[s.Name] = s
	}
	for _, s := range child.Shells {
		shellMap[s.Name] = s
	}
	result.Shells = make([]ShellConfigEntry, 0, len(shellMap))
	for _, s := range shellMap {
		result.Shells = append(result.Shells, s)
	}

	// Starship config (child overrides)
	if child.Starship.Enabled {
		result.Starship.Enabled = true
	}
	if child.Starship.Preset != "" {
		result.Starship.Preset = child.Starship.Preset
	}

	// Merge env and aliases
	if result.Env == nil {
		result.Env = make(map[string]string)
	}
	for k, v := range child.Env {
		result.Env[k] = v
	}

	if result.Aliases == nil {
		result.Aliases = make(map[string]string)
	}
	for k, v := range child.Aliases {
		result.Aliases[k] = v
	}

	return result
}

// mergeNvim merges Neovim configurations.
func (r *LayerResolver) mergeNvim(parent, child NvimConfig) NvimConfig {
	result := parent

	if child.Preset != "" {
		result.Preset = child.Preset
	}
	if child.PluginManager != "" {
		result.PluginManager = child.PluginManager
	}
	if child.ConfigRepo != "" {
		result.ConfigRepo = child.ConfigRepo
	}
	if child.EnsureInstall {
		result.EnsureInstall = true
	}

	return result
}

// mergeVSCode merges VSCode configurations.
func (r *LayerResolver) mergeVSCode(parent, child VSCodeConfig) VSCodeConfig {
	result := parent

	// Merge extensions
	result.Extensions = uniqueStrings(append(parent.Extensions, child.Extensions...))

	// Merge settings (child overrides)
	if result.Settings == nil {
		result.Settings = make(map[string]interface{})
	}
	for k, v := range child.Settings {
		result.Settings[k] = v
	}

	// Merge keybindings
	allKeybindings := make([]VSCodeKeybinding, 0, len(parent.Keybindings)+len(child.Keybindings))
	allKeybindings = append(allKeybindings, parent.Keybindings...)
	allKeybindings = append(allKeybindings, child.Keybindings...)
	result.Keybindings = allKeybindings

	return result
}

// uniqueStrings returns a deduplicated slice of strings.
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(input))

	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}
