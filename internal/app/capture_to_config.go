package app

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CaptureConfigGenerator generates configuration files from captured items.
type CaptureConfigGenerator struct {
	targetDir  string
	smartSplit bool
}

// NewCaptureConfigGenerator creates a new generator.
func NewCaptureConfigGenerator(targetDir string) *CaptureConfigGenerator {
	return &CaptureConfigGenerator{
		targetDir:  targetDir,
		smartSplit: false,
	}
}

// WithSmartSplit enables smart layer separation.
func (g *CaptureConfigGenerator) WithSmartSplit(enabled bool) *CaptureConfigGenerator {
	g.smartSplit = enabled
	return g
}

// GenerateFromCapture creates preflight configuration from captured items.
func (g *CaptureConfigGenerator) GenerateFromCapture(findings *CaptureFindings, target string) error {
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
		return g.generateSmartSplitLayers(findings, target)
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
func (g *CaptureConfigGenerator) generateSmartSplitLayers(findings *CaptureFindings, target string) error {
	categorizer := NewLayerCategorizer()

	// Get brew items for categorization
	byProvider := findings.ItemsByProvider()
	brewItems := byProvider["brew"]

	// Categorize brew items
	categorized := categorizer.Categorize(brewItems)
	categorized.SortItemsAlphabetically()

	// Collect layer names for manifest
	layerNames := make([]string, 0, len(categorized.LayerOrder))

	// Generate a layer file for each category with brew items
	for _, layerName := range categorized.LayerOrder {
		items := categorized.Layers[layerName]
		if len(items) == 0 {
			continue
		}

		if err := g.generateCategoryLayer(layerName, items, categorizer.GetLayerDescription(layerName)); err != nil {
			return fmt.Errorf("failed to generate layer %s: %w", layerName, err)
		}
		layerNames = append(layerNames, layerName)
	}

	// Handle non-brew providers (git, shell, vscode, runtime) in separate layers
	for provider, items := range byProvider {
		if provider == "brew" || len(items) == 0 {
			continue
		}

		layerName := provider
		if err := g.generateProviderLayer(layerName, provider, items); err != nil {
			return fmt.Errorf("failed to generate layer %s: %w", layerName, err)
		}
		layerNames = append(layerNames, layerName)
	}

	// Generate manifest with all layers
	if err := g.generateManifest(target, layerNames); err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	return nil
}

// generateCategoryLayer creates a layer file for a category of brew packages.
func (g *CaptureConfigGenerator) generateCategoryLayer(name string, items []CapturedItem, description string) error {
	layer := captureLayerYAML{
		Name: name,
	}

	formulae := make([]string, 0, len(items))
	for _, item := range items {
		formulae = append(formulae, item.Name)
	}

	layer.Packages = &capturePackagesYAML{
		Brew: &captureBrewYAML{
			Formulae: formulae,
		},
	}

	data, err := yaml.Marshal(layer)
	if err != nil {
		return err
	}

	// Add description as YAML comment
	content := fmt.Sprintf("# %s\n%s", description, string(data))

	layerPath := filepath.Join(g.targetDir, "layers", name+".yaml")
	return os.WriteFile(layerPath, []byte(content), 0o644)
}

// generateProviderLayer creates a layer file for a non-brew provider.
func (g *CaptureConfigGenerator) generateProviderLayer(name, provider string, items []CapturedItem) error {
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
		return nil // Skip unknown providers
	}

	data, err := yaml.Marshal(layer)
	if err != nil {
		return err
	}

	layerPath := filepath.Join(g.targetDir, "layers", name+".yaml")
	return os.WriteFile(layerPath, data, 0o644)
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

	// Generate brew section
	if brewItems, ok := byProvider["brew"]; ok && len(brewItems) > 0 {
		formulae := make([]string, 0, len(brewItems))
		for _, item := range brewItems {
			formulae = append(formulae, item.Name)
		}
		layer.Packages = &capturePackagesYAML{
			Brew: &captureBrewYAML{
				Formulae: formulae,
			},
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
