package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CaptureConfigGenerator generates configuration files from captured items.
type CaptureConfigGenerator struct {
	targetDir      string
	smartSplit     bool
	splitStrategy  SplitStrategy
	aiCategorizer  AICategorizer
	dotfilesResult *DotfilesCaptureResult
}

// NewCaptureConfigGenerator creates a new generator.
func NewCaptureConfigGenerator(targetDir string) *CaptureConfigGenerator {
	return &CaptureConfigGenerator{
		targetDir:     targetDir,
		smartSplit:    false,
		splitStrategy: SplitByCategory, // default
	}
}

// WithSmartSplit enables smart layer separation.
func (g *CaptureConfigGenerator) WithSmartSplit(enabled bool) *CaptureConfigGenerator {
	g.smartSplit = enabled
	return g
}

// WithSplitStrategy sets the split strategy for layer organization.
func (g *CaptureConfigGenerator) WithSplitStrategy(strategy SplitStrategy) *CaptureConfigGenerator {
	g.splitStrategy = strategy
	g.smartSplit = true // enable smart split when a strategy is set
	return g
}

// WithAICategorizer sets an AI categorizer for enhanced categorization.
func (g *CaptureConfigGenerator) WithAICategorizer(ai AICategorizer) *CaptureConfigGenerator {
	g.aiCategorizer = ai
	return g
}

// WithDotfiles sets the dotfiles capture result for config_source population.
func (g *CaptureConfigGenerator) WithDotfiles(result *DotfilesCaptureResult) *CaptureConfigGenerator {
	g.dotfilesResult = result
	return g
}

// GenerateFromCapture creates preflight configuration from captured items.
func (g *CaptureConfigGenerator) GenerateFromCapture(findings *CaptureFindings, target string) error {
	return g.GenerateFromCaptureWithContext(context.Background(), findings, target)
}

// GenerateFromCaptureWithContext creates preflight configuration from captured items with context.
func (g *CaptureConfigGenerator) GenerateFromCaptureWithContext(ctx context.Context, findings *CaptureFindings, target string) error {
	if target == "" {
		target = "default"
	}

	// Ensure target directory exists
	if err := os.MkdirAll(g.targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Ensure layers directory exists
	layersDir := filepath.Join(g.targetDir, "layers")
	if err := os.MkdirAll(layersDir, 0o755); err != nil {
		return fmt.Errorf("failed to create layers directory: %w", err)
	}

	if g.smartSplit {
		return g.generateSmartSplitLayers(ctx, findings, target)
	}

	// Generate manifest
	if err := g.generateManifest(target, []string{"captured"}); err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	// Generate layer from captured items
	if err := g.generateLayerFromCapture(findings); err != nil {
		return fmt.Errorf("failed to generate layer: %w", err)
	}

	return nil
}

// generateSmartSplitLayers creates multiple layer files organized by category.
func (g *CaptureConfigGenerator) generateSmartSplitLayers(ctx context.Context, findings *CaptureFindings, target string) error {
	// Select categorizer based on strategy
	categorizer := StrategyCategorizer(g.splitStrategy)

	// Get brew items for categorization
	byProvider := findings.ItemsByProvider()
	brewItems := byProvider["brew"]
	brewCasks := byProvider["brew-cask"]

	// Provider strategy groups by provider name directly
	if g.splitStrategy == SplitByProvider {
		return g.generateProviderSplitLayers(findings, target)
	}

	// Categorize brew items (formulae and casks together)
	allBrewItems := make([]CapturedItem, 0, len(brewItems)+len(brewCasks))
	allBrewItems = append(allBrewItems, brewItems...)
	allBrewItems = append(allBrewItems, brewCasks...)
	categorized := categorizer.Categorize(allBrewItems)

	// Use AI to categorize remaining items if available
	if g.aiCategorizer != nil && len(categorized.Uncategorized) > 0 {
		if err := CategorizeWithAI(ctx, categorized, g.aiCategorizer, g.splitStrategy); err != nil {
			// Log warning but continue without AI enhancement
			fmt.Printf("Warning: AI categorization failed: %v\n", err)
		}
	}

	categorized.SortItemsAlphabetically()

	// Build layer content map to merge brew packages with provider configs
	layerContent := make(map[string]*captureLayerYAML)
	createdLayers := make(map[string]bool)

	// First pass: populate with brew categorized items
	for _, layerName := range categorized.LayerOrder {
		items := categorized.Layers[layerName]
		if len(items) == 0 {
			continue
		}

		layer := g.buildLayerFromBrewItems(layerName, items)
		layerContent[layerName] = layer
		createdLayers[layerName] = true
	}

	// Second pass: merge provider configs into appropriate layers
	providerToLayer := map[string]string{
		"git":     "git",
		"shell":   "shell",
		"vscode":  "editor",
		"runtime": "runtime",
		"npm":     "dev-node",
		"go":      "dev-go",
		"pip":     "dev-python",
		"gem":     "dev-ruby",
		"cargo":   "dev-rust",
	}

	for provider, items := range byProvider {
		if provider == "brew" || provider == "brew-cask" || len(items) == 0 {
			continue
		}

		// Determine target layer name
		layerName := provider
		if mappedLayer, ok := providerToLayer[provider]; ok {
			layerName = mappedLayer
		}

		// Get or create layer
		layer := layerContent[layerName]
		if layer == nil {
			layer = &captureLayerYAML{Name: layerName}
			layerContent[layerName] = layer
			createdLayers[layerName] = true
		}

		// Add provider config to layer
		g.addProviderConfigToLayer(layer, provider, items)
	}

	// Write all layers to disk
	for layerName, layer := range layerContent {
		// Apply config_source from captured dotfiles before writing
		g.applyDotfilesToLayer(layer)
		description := categorizer.GetLayerDescription(layerName)
		if err := g.writeLayerFile(layerName, layer, description); err != nil {
			return fmt.Errorf("failed to write layer %s: %w", layerName, err)
		}
	}

	// Build ordered layer list from created layers
	layerNames := g.buildOrderedLayerList(categorized.LayerOrder, createdLayers)

	// Generate manifest with all layers
	if err := g.generateManifest(target, layerNames); err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	return nil
}

// buildLayerFromBrewItems creates a layer struct from brew items.
func (g *CaptureConfigGenerator) buildLayerFromBrewItems(name string, items []CapturedItem) *captureLayerYAML {
	layer := &captureLayerYAML{
		Name: name,
	}

	// Separate formulae from casks
	var formulae, casks []string
	for _, item := range items {
		if item.Provider == "brew-cask" {
			casks = append(casks, item.Name)
		} else {
			formulae = append(formulae, item.Name)
		}
	}

	// Only create packages section if we have items
	if len(formulae) > 0 || len(casks) > 0 {
		brew := &captureBrewYAML{}
		if len(formulae) > 0 {
			brew.Formulae = formulae
		}
		if len(casks) > 0 {
			brew.Casks = casks
		}
		layer.Packages = &capturePackagesYAML{
			Brew: brew,
		}
	}

	return layer
}

// addProviderConfigToLayer adds provider-specific config to a layer.
func (g *CaptureConfigGenerator) addProviderConfigToLayer(layer *captureLayerYAML, provider string, items []CapturedItem) {
	switch provider {
	case "git":
		layer.Git = g.generateGitFromCapture(items)
	case "shell":
		layer.Shell = g.generateShellFromCapture(items)
	case "vscode":
		extensions := make([]string, 0, len(items))
		for _, item := range items {
			extensions = append(extensions, item.Name)
		}
		layer.VSCode = &captureVSCodeYAML{
			Extensions: extensions,
		}
	case "runtime":
		layer.Runtime = g.generateRuntimeFromCapture(items)
	case "nvim":
		layer.Nvim = g.generateNvimFromCapture(items)
	case "ssh":
		layer.SSH = g.generateSSHFromCapture(items)
	case "npm":
		g.addNpmPackagesToLayer(layer, items)
	case "go":
		g.addGoToolsToLayer(layer, items)
	case "pip":
		g.addPipPackagesToLayer(layer, items)
	case "gem":
		g.addGemPackagesToLayer(layer, items)
	case "cargo":
		g.addCargoPackagesToLayer(layer, items)
	}
}

// applyDotfilesToLayer populates config_source fields from captured dotfiles.
func (g *CaptureConfigGenerator) applyDotfilesToLayer(layer *captureLayerYAML) {
	if g.dotfilesResult == nil {
		return
	}

	byProvider := g.dotfilesResult.DotfilesByProvider()

	// Helper to get the relative dotfiles path for a provider
	getConfigSource := func(provider string) string {
		if files, ok := byProvider[provider]; ok && len(files) > 0 {
			// Use the target directory relative path
			// e.g., "dotfiles/nvim" or "dotfiles.work/nvim"
			baseDir := g.dotfilesResult.TargetDir
			return baseDir + "/" + provider
		}
		return ""
	}

	// Apply config_source to each provider that has captured dotfiles
	if configSource := getConfigSource("nvim"); configSource != "" {
		if layer.Nvim == nil {
			layer.Nvim = &captureNvimYAML{}
		}
		layer.Nvim.ConfigSource = configSource
	}

	if configSource := getConfigSource("vscode"); configSource != "" {
		if layer.VSCode == nil {
			layer.VSCode = &captureVSCodeYAML{}
		}
		layer.VSCode.ConfigSource = configSource
	}

	if configSource := getConfigSource("git"); configSource != "" {
		if layer.Git == nil {
			layer.Git = &captureGitYAML{}
		}
		layer.Git.ConfigSource = configSource
	}

	if configSource := getConfigSource("ssh"); configSource != "" {
		if layer.SSH == nil {
			layer.SSH = &captureSSHYAML{}
		}
		layer.SSH.ConfigSource = configSource
	}

	if configSource := getConfigSource("tmux"); configSource != "" {
		if layer.Tmux == nil {
			layer.Tmux = &captureTmuxYAML{}
		}
		layer.Tmux.ConfigSource = configSource
	}

	// Handle shell config sources
	if configSource := getConfigSource("shell"); configSource != "" {
		if layer.Shell == nil {
			layer.Shell = &captureShellYAML{}
		}
		if layer.Shell.ConfigSource == nil {
			layer.Shell.ConfigSource = &captureShellConfigSourceYAML{}
		}
		layer.Shell.ConfigSource.Dir = configSource
	}

	// Handle starship separately (it's under shell but captured as separate provider)
	if configSource := getConfigSource("starship"); configSource != "" {
		if layer.Shell == nil {
			layer.Shell = &captureShellYAML{}
		}
		if layer.Shell.Starship == nil {
			layer.Shell.Starship = &captureStarshipYAML{}
		}
		layer.Shell.Starship.ConfigSource = configSource
	}
}

// writeLayerFile writes a layer to disk with optional description comment.
func (g *CaptureConfigGenerator) writeLayerFile(name string, layer *captureLayerYAML, description string) error {
	data, err := yaml.Marshal(layer)
	if err != nil {
		return err
	}

	var content string
	if description != "" {
		content = fmt.Sprintf("# %s\n%s", description, string(data))
	} else {
		content = string(data)
	}

	layerPath := filepath.Join(g.targetDir, "layers", name+".yaml")
	return os.WriteFile(layerPath, []byte(content), 0o644)
}

// buildOrderedLayerList creates an ordered, deduplicated list of layer names.
func (g *CaptureConfigGenerator) buildOrderedLayerList(categoryOrder []string, createdLayers map[string]bool) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(createdLayers))

	// First add category layers in order
	for _, name := range categoryOrder {
		if createdLayers[name] && !seen[name] {
			result = append(result, name)
			seen[name] = true
		}
	}

	// Then add any remaining created layers (non-brew providers)
	for name := range createdLayers {
		if !seen[name] {
			result = append(result, name)
			seen[name] = true
		}
	}

	return result
}

// generateProviderSplitLayers creates layer files organized by provider.
func (g *CaptureConfigGenerator) generateProviderSplitLayers(findings *CaptureFindings, target string) error {
	byProvider := findings.ItemsByProvider()
	createdLayers := make(map[string]bool)

	// Combine brew and brew-cask into a single "brew" layer
	brewItems := byProvider["brew"]
	brewCasks := byProvider["brew-cask"]
	if len(brewItems) > 0 || len(brewCasks) > 0 {
		if err := g.generateBrewProviderLayer("brew", brewItems, brewCasks); err != nil {
			return fmt.Errorf("failed to generate layer brew: %w", err)
		}
		createdLayers["brew"] = true
	}

	// Handle other providers
	for provider, items := range byProvider {
		if provider == "brew" || provider == "brew-cask" || len(items) == 0 {
			continue
		}

		created, err := g.generateProviderLayerIfSupported(provider, provider, items)
		if err != nil {
			return fmt.Errorf("failed to generate layer %s: %w", provider, err)
		}
		if created {
			createdLayers[provider] = true
		}
	}

	// Build layer names list
	layerNames := make([]string, 0, len(createdLayers))
	for name := range createdLayers {
		layerNames = append(layerNames, name)
	}

	// Generate manifest with all layers
	if err := g.generateManifest(target, layerNames); err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	return nil
}

// generateBrewProviderLayer creates a layer file with both formulae and casks.
func (g *CaptureConfigGenerator) generateBrewProviderLayer(name string, formulae, casks []CapturedItem) error {
	layer := captureLayerYAML{
		Name: name,
	}

	brew := &captureBrewYAML{}
	if len(formulae) > 0 {
		f := make([]string, len(formulae))
		for i, item := range formulae {
			f[i] = item.Name
		}
		brew.Formulae = f
	}
	if len(casks) > 0 {
		c := make([]string, len(casks))
		for i, item := range casks {
			c[i] = item.Name
		}
		brew.Casks = c
	}

	layer.Packages = &capturePackagesYAML{
		Brew: brew,
	}

	// Apply config_source from captured dotfiles before writing
	g.applyDotfilesToLayer(&layer)

	data, err := yaml.Marshal(layer)
	if err != nil {
		return err
	}

	layerPath := filepath.Join(g.targetDir, "layers", name+".yaml")
	return os.WriteFile(layerPath, data, 0o644)
}

// generateProviderLayerIfSupported creates a layer file for a non-brew provider.
// Returns true if the layer was created, false if the provider is not supported.
func (g *CaptureConfigGenerator) generateProviderLayerIfSupported(name, provider string, items []CapturedItem) (bool, error) {
	layer := captureLayerYAML{
		Name: name,
	}

	switch provider {
	case "git":
		layer.Git = g.generateGitFromCapture(items)
	case "shell":
		layer.Shell = g.generateShellFromCapture(items)
	case "vscode":
		extensions := make([]string, 0, len(items))
		for _, item := range items {
			extensions = append(extensions, item.Name)
		}
		layer.VSCode = &captureVSCodeYAML{
			Extensions: extensions,
		}
	case "runtime":
		layer.Runtime = g.generateRuntimeFromCapture(items)
	case "nvim":
		layer.Nvim = g.generateNvimFromCapture(items)
		if layer.Nvim == nil {
			return false, nil
		}
	case "ssh":
		layer.SSH = g.generateSSHFromCapture(items)
		if layer.SSH == nil {
			return false, nil
		}
	case "npm":
		g.addNpmPackagesToLayer(&layer, items)
	case "go":
		g.addGoToolsToLayer(&layer, items)
	case "pip":
		g.addPipPackagesToLayer(&layer, items)
	case "gem":
		g.addGemPackagesToLayer(&layer, items)
	case "cargo":
		g.addCargoPackagesToLayer(&layer, items)
	default:
		// Provider not supported for layer generation
		return false, nil
	}

	// Apply config_source from captured dotfiles before writing
	g.applyDotfilesToLayer(&layer)

	data, err := yaml.Marshal(layer)
	if err != nil {
		return false, err
	}

	layerPath := filepath.Join(g.targetDir, "layers", name+".yaml")
	if err := os.WriteFile(layerPath, data, 0o644); err != nil {
		return false, err
	}

	return true, nil
}

// generateManifest creates the preflight.yaml manifest file.
func (g *CaptureConfigGenerator) generateManifest(target string, layers []string) error {
	manifest := captureManifestYAML{
		Defaults: captureDefaultsYAML{
			Mode: "intent",
		},
		Targets: map[string][]string{
			target: layers,
		},
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(g.targetDir, "preflight.yaml")
	return os.WriteFile(manifestPath, data, 0o644)
}

// generateLayerFromCapture creates the layers/captured.yaml file from findings.
func (g *CaptureConfigGenerator) generateLayerFromCapture(findings *CaptureFindings) error {
	layer := captureLayerYAML{
		Name: "captured",
	}

	// Group items by provider
	byProvider := findings.ItemsByProvider()

	// Generate brew section (formulae and casks)
	brewItems := byProvider["brew"]
	caskItems := byProvider["brew-cask"]
	if len(brewItems) > 0 || len(caskItems) > 0 {
		brew := &captureBrewYAML{}

		if len(brewItems) > 0 {
			formulae := make([]string, 0, len(brewItems))
			for _, item := range brewItems {
				formulae = append(formulae, item.Name)
			}
			brew.Formulae = formulae
		}

		if len(caskItems) > 0 {
			casks := make([]string, 0, len(caskItems))
			for _, item := range caskItems {
				casks = append(casks, item.Name)
			}
			brew.Casks = casks
		}

		layer.Packages = &capturePackagesYAML{
			Brew: brew,
		}
	}

	// Generate git section
	if gitItems, ok := byProvider["git"]; ok && len(gitItems) > 0 {
		layer.Git = g.generateGitFromCapture(gitItems)
	}

	// Generate shell section (just note which files exist)
	if shellItems, ok := byProvider["shell"]; ok && len(shellItems) > 0 {
		layer.Shell = g.generateShellFromCapture(shellItems)
	}

	// Generate vscode section
	if vscodeItems, ok := byProvider["vscode"]; ok && len(vscodeItems) > 0 {
		extensions := make([]string, 0, len(vscodeItems))
		for _, item := range vscodeItems {
			extensions = append(extensions, item.Name)
		}
		layer.VSCode = &captureVSCodeYAML{
			Extensions: extensions,
		}
	}

	// Generate runtime section
	if runtimeItems, ok := byProvider["runtime"]; ok && len(runtimeItems) > 0 {
		layer.Runtime = g.generateRuntimeFromCapture(runtimeItems)
	}

	// Generate nvim section
	if nvimItems, ok := byProvider["nvim"]; ok && len(nvimItems) > 0 {
		layer.Nvim = g.generateNvimFromCapture(nvimItems)
	}

	// Generate ssh section
	if sshItems, ok := byProvider["ssh"]; ok && len(sshItems) > 0 {
		layer.SSH = g.generateSSHFromCapture(sshItems)
	}

	// Generate npm section
	if npmItems, ok := byProvider["npm"]; ok && len(npmItems) > 0 {
		g.addNpmPackagesToLayer(&layer, npmItems)
	}

	// Generate go section
	if goItems, ok := byProvider["go"]; ok && len(goItems) > 0 {
		g.addGoToolsToLayer(&layer, goItems)
	}

	// Generate pip section
	if pipItems, ok := byProvider["pip"]; ok && len(pipItems) > 0 {
		g.addPipPackagesToLayer(&layer, pipItems)
	}

	// Generate gem section
	if gemItems, ok := byProvider["gem"]; ok && len(gemItems) > 0 {
		g.addGemPackagesToLayer(&layer, gemItems)
	}

	// Generate cargo section
	if cargoItems, ok := byProvider["cargo"]; ok && len(cargoItems) > 0 {
		g.addCargoPackagesToLayer(&layer, cargoItems)
	}

	// Apply config_source from captured dotfiles before writing
	g.applyDotfilesToLayer(&layer)

	data, err := yaml.Marshal(layer)
	if err != nil {
		return err
	}

	layerPath := filepath.Join(g.targetDir, "layers", "captured.yaml")
	return os.WriteFile(layerPath, data, 0o644)
}

func (g *CaptureConfigGenerator) generateGitFromCapture(items []CapturedItem) *captureGitYAML {
	git := &captureGitYAML{}

	for _, item := range items {
		switch item.Name {
		case "user.name":
			if git.User == nil {
				git.User = &captureGitUserYAML{}
			}
			if s, ok := item.Value.(string); ok {
				git.User.Name = s
			}
		case "user.email":
			if git.User == nil {
				git.User = &captureGitUserYAML{}
			}
			if s, ok := item.Value.(string); ok {
				git.User.Email = s
			}
		case "core.editor":
			if git.Core == nil {
				git.Core = &captureGitCoreYAML{}
			}
			if s, ok := item.Value.(string); ok {
				git.Core.Editor = s
			}
		case "init.defaultBranch":
			if git.Init == nil {
				git.Init = &captureGitInitYAML{}
			}
			if s, ok := item.Value.(string); ok {
				git.Init.DefaultBranch = s
			}
		}
	}

	return git
}

func (g *CaptureConfigGenerator) generateShellFromCapture(items []CapturedItem) *captureShellYAML {
	shell := &captureShellYAML{
		Shells: make([]captureShellEntryYAML, 0),
	}

	hasZsh := false
	hasBash := false

	for _, item := range items {
		switch item.Name {
		case ".zshrc":
			hasZsh = true
		case ".bashrc", ".bash_profile":
			hasBash = true
		}
	}

	if hasZsh {
		shell.Default = "zsh"
		shell.Shells = append(shell.Shells, captureShellEntryYAML{Name: "zsh"})
	}

	if hasBash {
		if shell.Default == "" {
			shell.Default = "bash"
		}
		shell.Shells = append(shell.Shells, captureShellEntryYAML{Name: "bash"})
	}

	if len(shell.Shells) == 0 {
		return nil
	}

	return shell
}

func (g *CaptureConfigGenerator) generateRuntimeFromCapture(items []CapturedItem) *captureRuntimeYAML {
	runtime := &captureRuntimeYAML{
		Tools: make([]captureRuntimeToolYAML, 0, len(items)),
	}

	for _, item := range items {
		version := ""
		if s, ok := item.Value.(string); ok {
			version = s
		}
		runtime.Tools = append(runtime.Tools, captureRuntimeToolYAML{
			Name:    item.Name,
			Version: version,
		})
	}

	if len(runtime.Tools) == 0 {
		return nil
	}

	return runtime
}

// addNpmPackagesToLayer adds npm packages to a layer's packages section.
func (g *CaptureConfigGenerator) addNpmPackagesToLayer(layer *captureLayerYAML, items []CapturedItem) {
	if len(items) == 0 {
		return
	}

	packages := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.Value.(string); ok && s != "" {
			packages = append(packages, s)
		} else {
			packages = append(packages, item.Name)
		}
	}

	if len(packages) == 0 {
		return
	}

	if layer.Packages == nil {
		layer.Packages = &capturePackagesYAML{}
	}
	layer.Packages.Npm = &captureNpmYAML{
		Packages: packages,
	}
}

// addGoToolsToLayer adds go tools to a layer's packages section.
func (g *CaptureConfigGenerator) addGoToolsToLayer(layer *captureLayerYAML, items []CapturedItem) {
	if len(items) == 0 {
		return
	}

	tools := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.Value.(string); ok && s != "" {
			tools = append(tools, s)
		} else {
			tools = append(tools, item.Name)
		}
	}

	if len(tools) == 0 {
		return
	}

	if layer.Packages == nil {
		layer.Packages = &capturePackagesYAML{}
	}
	layer.Packages.Go = &captureGoYAML{
		Tools: tools,
	}
}

// addPipPackagesToLayer adds pip packages to a layer's packages section.
func (g *CaptureConfigGenerator) addPipPackagesToLayer(layer *captureLayerYAML, items []CapturedItem) {
	if len(items) == 0 {
		return
	}

	packages := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.Value.(string); ok && s != "" {
			packages = append(packages, s)
		} else {
			packages = append(packages, item.Name)
		}
	}

	if len(packages) == 0 {
		return
	}

	if layer.Packages == nil {
		layer.Packages = &capturePackagesYAML{}
	}
	layer.Packages.Pip = &capturePipYAML{
		Packages: packages,
	}
}

// addGemPackagesToLayer adds gem packages to a layer's packages section.
func (g *CaptureConfigGenerator) addGemPackagesToLayer(layer *captureLayerYAML, items []CapturedItem) {
	if len(items) == 0 {
		return
	}

	gems := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.Value.(string); ok && s != "" {
			gems = append(gems, s)
		} else {
			gems = append(gems, item.Name)
		}
	}

	if len(gems) == 0 {
		return
	}

	if layer.Packages == nil {
		layer.Packages = &capturePackagesYAML{}
	}
	layer.Packages.Gem = &captureGemYAML{
		Gems: gems,
	}
}

// addCargoPackagesToLayer adds cargo crates to a layer's packages section.
func (g *CaptureConfigGenerator) addCargoPackagesToLayer(layer *captureLayerYAML, items []CapturedItem) {
	if len(items) == 0 {
		return
	}

	crates := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.Value.(string); ok && s != "" {
			crates = append(crates, s)
		} else {
			crates = append(crates, item.Name)
		}
	}

	if len(crates) == 0 {
		return
	}

	if layer.Packages == nil {
		layer.Packages = &capturePackagesYAML{}
	}
	layer.Packages.Cargo = &captureCargoYAML{
		Crates: crates,
	}
}

func (g *CaptureConfigGenerator) generateNvimFromCapture(items []CapturedItem) *captureNvimYAML {
	if len(items) == 0 {
		return nil
	}

	nvim := &captureNvimYAML{}

	for _, item := range items {
		switch item.Name {
		case "config":
			if configPath, ok := item.Value.(string); ok {
				nvim.ConfigPath = configPath
				nvim.Preset = detectNvimPreset(configPath)
				nvim.PluginManager = detectPluginManager(configPath)
				nvim.ConfigManaged = isGitManaged(configPath)
			}
		case "lazy-lock.json":
			if lockPath, ok := item.Value.(string); ok {
				nvim.PluginCount = countLazyPlugins(lockPath)
				if nvim.PluginManager == "" {
					nvim.PluginManager = "lazy.nvim"
				}
			}
		case "packer_compiled.lua":
			if nvim.PluginManager == "" {
				nvim.PluginManager = "packer"
			}
		case ".vimrc":
			if nvim.Preset == "" {
				nvim.Preset = "legacy"
			}
		}
	}

	// Set default preset if we have a config but couldn't detect type
	if nvim.ConfigPath != "" && nvim.Preset == "" {
		nvim.Preset = "custom"
	}

	return nvim
}

// detectNvimPreset checks for known distribution markers.
func detectNvimPreset(configPath string) string {
	// Check for LazyVim
	lazyVimMarker := filepath.Join(configPath, "lazyvim.json")
	if _, err := os.Stat(lazyVimMarker); err == nil {
		return "lazyvim"
	}

	// Check for LazyVim in lazy-lock.json
	lazyLock := filepath.Join(configPath, "lazy-lock.json")
	if data, err := os.ReadFile(lazyLock); err == nil {
		if strings.Contains(string(data), "LazyVim") {
			return "lazyvim"
		}
	}

	// Check for NvChad
	nvChadMarker := filepath.Join(configPath, "lua", "core")
	customDir := filepath.Join(configPath, "lua", "custom")
	if _, err := os.Stat(nvChadMarker); err == nil {
		if _, err := os.Stat(customDir); err == nil {
			return "nvchad"
		}
	}

	// Check for AstroNvim
	astroMarker := filepath.Join(configPath, "lua", "astronvim")
	if _, err := os.Stat(astroMarker); err == nil {
		return "astronvim"
	}

	// Check for LunarVim (usually at ~/.local/share/lunarvim)
	if strings.Contains(configPath, "lvim") || strings.Contains(configPath, "lunarvim") {
		return "lunarvim"
	}

	return "custom"
}

// detectPluginManager checks what plugin manager is used.
func detectPluginManager(configPath string) string {
	// Check for lazy.nvim
	lazyLock := filepath.Join(configPath, "lazy-lock.json")
	if _, err := os.Stat(lazyLock); err == nil {
		return "lazy.nvim"
	}

	// Check for packer
	packerCompiled := filepath.Join(configPath, "plugin", "packer_compiled.lua")
	if _, err := os.Stat(packerCompiled); err == nil {
		return "packer"
	}

	// Check in init.lua for plugin manager references
	initLua := filepath.Join(configPath, "init.lua")
	if data, err := os.ReadFile(initLua); err == nil {
		content := string(data)
		if strings.Contains(content, "lazy.nvim") || strings.Contains(content, "folke/lazy") {
			return "lazy.nvim"
		}
		if strings.Contains(content, "packer") || strings.Contains(content, "wbthomason/packer") {
			return "packer"
		}
		if strings.Contains(content, "vim-plug") || strings.Contains(content, "junegunn/vim-plug") {
			return "vim-plug"
		}
	}

	return ""
}

// countLazyPlugins counts plugins in lazy-lock.json.
func countLazyPlugins(lockPath string) int {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return 0
	}

	var plugins map[string]interface{}
	if err := json.Unmarshal(data, &plugins); err != nil {
		return 0
	}

	return len(plugins)
}

// isGitManaged checks if a directory is under git version control.
func isGitManaged(path string) bool {
	gitDir := filepath.Join(path, ".git")
	if info, err := os.Stat(gitDir); err == nil {
		return info.IsDir()
	}
	return false
}

func (g *CaptureConfigGenerator) generateSSHFromCapture(items []CapturedItem) *captureSSHYAML {
	if len(items) == 0 {
		return nil
	}

	ssh := &captureSSHYAML{}

	for _, item := range items {
		if item.Name == "config" {
			if configPath, ok := item.Value.(string); ok {
				ssh.ConfigPath = configPath
				hosts, defaults := parseSSHConfig(configPath)
				ssh.Hosts = hosts
				ssh.Defaults = defaults
			}
		}
	}

	// Detect SSH keys
	if ssh.ConfigPath != "" {
		sshDir := filepath.Dir(ssh.ConfigPath)
		ssh.Keys = detectSSHKeys(sshDir)
	}

	if ssh.ConfigPath == "" && len(ssh.Hosts) == 0 && len(ssh.Keys) == 0 {
		return nil
	}

	return ssh
}

// parseSSHConfig parses ~/.ssh/config into hosts and defaults.
func parseSSHConfig(configPath string) ([]captureSSHHostYAML, *captureSSHDefaults) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, nil
	}
	defer func() { _ = file.Close() }()

	var hosts []captureSSHHostYAML
	defaults := &captureSSHDefaults{}
	var currentHost *captureSSHHostYAML

	scanner := bufio.NewScanner(file)
	inGlobalSection := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key-value pairs
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			parts = strings.SplitN(line, "\t", 2)
			if len(parts) < 2 {
				continue
			}
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Handle Host directive
		if strings.EqualFold(key, "Host") {
			// Save previous host if exists
			if currentHost != nil {
				hosts = append(hosts, *currentHost)
			}

			// Skip wildcard patterns for now
			if strings.Contains(value, "*") {
				currentHost = nil
				continue
			}

			inGlobalSection = false
			currentHost = &captureSSHHostYAML{
				Host: value,
			}
			continue
		}

		// Handle options
		if inGlobalSection {
			// Global defaults
			switch strings.ToLower(key) {
			case "addkeystoagent":
				defaults.AddKeysToAgent = value
			case "usekeychain":
				defaults.UseKeychain = value
			case "identitiesonly":
				defaults.IdentitiesOnly = value
			case "serveraliveinterval":
				defaults.ServerAliveInterval = value
			}
		} else if currentHost != nil {
			// Host-specific options
			switch strings.ToLower(key) {
			case "hostname":
				currentHost.HostName = value
			case "user":
				currentHost.User = value
			case "identityfile":
				currentHost.IdentityFile = value
			case "port":
				currentHost.Port = value
			}
		}
	}

	// Don't forget the last host
	if currentHost != nil {
		hosts = append(hosts, *currentHost)
	}

	// Return nil defaults if empty
	if defaults.AddKeysToAgent == "" && defaults.UseKeychain == "" &&
		defaults.IdentitiesOnly == "" && defaults.ServerAliveInterval == "" {
		defaults = nil
	}

	return hosts, defaults
}

// detectSSHKeys finds SSH key files in the .ssh directory.
func detectSSHKeys(sshDir string) []captureSSHKeyYAML {
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil
	}

	keys := make([]captureSSHKeyYAML, 0, len(entries))
	seen := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip known non-key files
		if name == "config" || name == "known_hosts" || name == "authorized_keys" ||
			strings.HasSuffix(name, ".pub") {
			continue
		}

		// Check if this looks like a private key
		keyPath := filepath.Join(sshDir, name)
		if !looksLikePrivateKey(keyPath) {
			continue
		}

		// Avoid duplicates (e.g., if we process both id_rsa and id_rsa.pub)
		if seen[name] {
			continue
		}
		seen[name] = true

		key := captureSSHKeyYAML{
			Name: name,
			Type: detectKeyType(keyPath),
		}

		// Check for passphrase by looking at the key file header
		key.HasPassphrase = keyHasPassphrase(keyPath)

		// Try to get comment from public key
		pubKeyPath := keyPath + ".pub"
		if data, err := os.ReadFile(pubKeyPath); err == nil {
			parts := strings.Fields(string(data))
			if len(parts) >= 3 {
				key.Comment = parts[2]
			}
		}

		keys = append(keys, key)
	}

	return keys
}

// looksLikePrivateKey checks if a file appears to be an SSH private key.
func looksLikePrivateKey(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	// Read first line
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		return strings.Contains(line, "PRIVATE KEY")
	}
	return false
}

// detectKeyType determines the type of SSH key.
func detectKeyType(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.Contains(line, "OPENSSH"):
			// Modern OpenSSH format - check public key for type
			pubKeyPath := path + ".pub"
			if data, err := os.ReadFile(pubKeyPath); err == nil {
				content := string(data)
				if strings.HasPrefix(content, "ssh-ed25519") {
					return "ed25519"
				}
				if strings.HasPrefix(content, "ssh-rsa") {
					return "rsa"
				}
				if strings.HasPrefix(content, "ecdsa-") {
					return "ecdsa"
				}
			}
			return "ed25519" // Default for modern keys
		case strings.Contains(line, "RSA"):
			return "rsa"
		case strings.Contains(line, "EC"):
			return "ecdsa"
		case strings.Contains(line, "DSA"):
			return "dsa"
		}
	}
	return ""
}

// keyHasPassphrase checks if a private key is encrypted.
func keyHasPassphrase(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := string(data)
	return strings.Contains(content, "ENCRYPTED")
}

// YAML structure types for marshaling

type captureManifestYAML struct {
	Defaults captureDefaultsYAML `yaml:"defaults,omitempty"`
	Targets  map[string][]string `yaml:"targets"`
}

type captureDefaultsYAML struct {
	Mode string `yaml:"mode,omitempty"`
}

type captureLayerYAML struct {
	Name     string               `yaml:"name"`
	Packages *capturePackagesYAML `yaml:"packages,omitempty"`
	Git      *captureGitYAML      `yaml:"git,omitempty"`
	Shell    *captureShellYAML    `yaml:"shell,omitempty"`
	VSCode   *captureVSCodeYAML   `yaml:"vscode,omitempty"`
	Runtime  *captureRuntimeYAML  `yaml:"runtime,omitempty"`
	Nvim     *captureNvimYAML     `yaml:"nvim,omitempty"`
	SSH      *captureSSHYAML      `yaml:"ssh,omitempty"`
	Tmux     *captureTmuxYAML     `yaml:"tmux,omitempty"`
}

type capturePackagesYAML struct {
	Brew  *captureBrewYAML  `yaml:"brew,omitempty"`
	Npm   *captureNpmYAML   `yaml:"npm,omitempty"`
	Go    *captureGoYAML    `yaml:"go,omitempty"`
	Pip   *capturePipYAML   `yaml:"pip,omitempty"`
	Gem   *captureGemYAML   `yaml:"gem,omitempty"`
	Cargo *captureCargoYAML `yaml:"cargo,omitempty"`
}

type captureNpmYAML struct {
	Packages []string `yaml:"packages,omitempty"`
}

type captureGoYAML struct {
	Tools []string `yaml:"tools,omitempty"`
}

type capturePipYAML struct {
	Packages []string `yaml:"packages,omitempty"`
}

type captureGemYAML struct {
	Gems []string `yaml:"gems,omitempty"`
}

type captureCargoYAML struct {
	Crates []string `yaml:"crates,omitempty"`
}

type captureBrewYAML struct {
	Taps     []string `yaml:"taps,omitempty"`
	Formulae []string `yaml:"formulae,omitempty"`
	Casks    []string `yaml:"casks,omitempty"`
}

type captureGitYAML struct {
	User         *captureGitUserYAML `yaml:"user,omitempty"`
	Core         *captureGitCoreYAML `yaml:"core,omitempty"`
	Init         *captureGitInitYAML `yaml:"init,omitempty"`
	ConfigSource string              `yaml:"config_source,omitempty"` // Path to gitconfig.d directory (e.g., "dotfiles/git")
}

type captureGitUserYAML struct {
	Name  string `yaml:"name,omitempty"`
	Email string `yaml:"email,omitempty"`
}

type captureGitCoreYAML struct {
	Editor string `yaml:"editor,omitempty"`
}

type captureGitInitYAML struct {
	DefaultBranch string `yaml:"defaultBranch,omitempty"`
}

type captureShellYAML struct {
	Default      string                        `yaml:"default,omitempty"`
	Shells       []captureShellEntryYAML       `yaml:"shells,omitempty"`
	ConfigSource *captureShellConfigSourceYAML `yaml:"config_source,omitempty"` // Paths to shell config files
	Starship     *captureStarshipYAML          `yaml:"starship,omitempty"`      // Starship prompt configuration
}

type captureShellEntryYAML struct {
	Name      string   `yaml:"name"`
	Framework string   `yaml:"framework,omitempty"`
	Theme     string   `yaml:"theme,omitempty"`
	Plugins   []string `yaml:"plugins,omitempty"`
}

// captureShellConfigSourceYAML represents paths to shell configuration files.
type captureShellConfigSourceYAML struct {
	Aliases   string `yaml:"aliases,omitempty"`   // Path to aliases file
	Functions string `yaml:"functions,omitempty"` // Path to functions file
	Env       string `yaml:"env,omitempty"`       // Path to env file
	Dir       string `yaml:"dir,omitempty"`       // Path to shell config directory
}

// captureStarshipYAML represents Starship prompt configuration.
type captureStarshipYAML struct {
	ConfigSource string `yaml:"config_source,omitempty"` // Path to starship.toml (e.g., "dotfiles/starship")
}

// captureTmuxYAML represents Tmux configuration.
type captureTmuxYAML struct {
	ConfigSource string `yaml:"config_source,omitempty"` // Path to tmux config (e.g., "dotfiles/tmux")
}

type captureVSCodeYAML struct {
	Extensions   []string `yaml:"extensions,omitempty"`
	ConfigSource string   `yaml:"config_source,omitempty"` // Path to dotfiles (e.g., "dotfiles/vscode")
}

type captureRuntimeYAML struct {
	Tools []captureRuntimeToolYAML `yaml:"tools,omitempty"`
}

type captureRuntimeToolYAML struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version,omitempty"`
}

// Nvim YAML types

type captureNvimYAML struct {
	Preset        string `yaml:"preset,omitempty"`         // lazyvim, nvchad, astrovim, custom
	ConfigPath    string `yaml:"config_path,omitempty"`    // Path to config directory
	PluginManager string `yaml:"plugin_manager,omitempty"` // lazy.nvim, packer, vim-plug
	PluginCount   int    `yaml:"plugin_count,omitempty"`   // Number of plugins detected
	ConfigManaged bool   `yaml:"config_managed,omitempty"` // Config is under version control
	ConfigSource  string `yaml:"config_source,omitempty"`  // Path to dotfiles (e.g., "dotfiles/nvim")
}

// SSH YAML types

type captureSSHYAML struct {
	ConfigPath   string               `yaml:"config_path,omitempty"`   // Path to SSH config
	ConfigSource string               `yaml:"config_source,omitempty"` // Path to dotfiles (e.g., "dotfiles/ssh")
	Defaults     *captureSSHDefaults  `yaml:"defaults,omitempty"`      // Global SSH options
	Hosts        []captureSSHHostYAML `yaml:"hosts,omitempty"`         // Host configurations
	Keys         []captureSSHKeyYAML  `yaml:"keys,omitempty"`          // Key references (never content)
}

type captureSSHDefaults struct {
	AddKeysToAgent      string `yaml:"AddKeysToAgent,omitempty"`
	UseKeychain         string `yaml:"UseKeychain,omitempty"`
	IdentitiesOnly      string `yaml:"IdentitiesOnly,omitempty"`
	ServerAliveInterval string `yaml:"ServerAliveInterval,omitempty"`
}

type captureSSHHostYAML struct {
	Host         string `yaml:"host"`                    // Host alias (matches SSHHostConfig.Host)
	HostName     string `yaml:"hostname,omitempty"`      // Actual hostname (may be redacted)
	User         string `yaml:"user,omitempty"`          // Username
	IdentityFile string `yaml:"identity_file,omitempty"` // Path to key file
	Port         string `yaml:"port,omitempty"`          // Port if non-standard
}

type captureSSHKeyYAML struct {
	Name          string `yaml:"name"`                     // Key filename
	Type          string `yaml:"type,omitempty"`           // ed25519, rsa, ecdsa
	HasPassphrase bool   `yaml:"has_passphrase,omitempty"` // Whether key is encrypted
	Comment       string `yaml:"comment,omitempty"`        // Key comment/email
}
