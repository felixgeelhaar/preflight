package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CaptureConfigGenerator generates configuration files from captured items.
type CaptureConfigGenerator struct {
	targetDir     string
	smartSplit    bool
	splitStrategy SplitStrategy
	aiCategorizer AICategorizer
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
	default:
		// Provider not supported for layer generation (e.g., ssh, nvim)
		return false, nil
	}

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
}

type capturePackagesYAML struct {
	Brew *captureBrewYAML `yaml:"brew,omitempty"`
}

type captureBrewYAML struct {
	Taps     []string `yaml:"taps,omitempty"`
	Formulae []string `yaml:"formulae,omitempty"`
	Casks    []string `yaml:"casks,omitempty"`
}

type captureGitYAML struct {
	User *captureGitUserYAML `yaml:"user,omitempty"`
	Core *captureGitCoreYAML `yaml:"core,omitempty"`
	Init *captureGitInitYAML `yaml:"init,omitempty"`
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
	Default string                  `yaml:"default,omitempty"`
	Shells  []captureShellEntryYAML `yaml:"shells,omitempty"`
}

type captureShellEntryYAML struct {
	Name      string   `yaml:"name"`
	Framework string   `yaml:"framework,omitempty"`
	Theme     string   `yaml:"theme,omitempty"`
	Plugins   []string `yaml:"plugins,omitempty"`
}

type captureVSCodeYAML struct {
	Extensions []string `yaml:"extensions,omitempty"`
}

type captureRuntimeYAML struct {
	Tools []captureRuntimeToolYAML `yaml:"tools,omitempty"`
}

type captureRuntimeToolYAML struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version,omitempty"`
}
