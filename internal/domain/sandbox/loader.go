package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/preflight/internal/domain/capability"
)

// Loader errors.
var (
	ErrPluginManifestNotFound = errors.New("plugin manifest not found")
	ErrPluginModuleNotFound   = errors.New("plugin module not found")
	ErrPluginChecksumMismatch = errors.New("plugin checksum mismatch")
	ErrPluginManifestInvalid  = errors.New("plugin manifest invalid")
)

// PluginManifest describes a plugin and its requirements.
type PluginManifest struct {
	// ID is the unique plugin identifier
	ID string `yaml:"id"`

	// Name is the human-readable name
	Name string `yaml:"name"`

	// Version is the plugin version
	Version string `yaml:"version"`

	// Description of what the plugin does
	Description string `yaml:"description,omitempty"`

	// Author information
	Author string `yaml:"author,omitempty"`

	// Module is the path to the WASM module relative to manifest
	Module string `yaml:"module"`

	// Checksum is the SHA256 hash of the module
	Checksum string `yaml:"checksum"`

	// Capabilities required by the plugin
	Capabilities []ManifestCapability `yaml:"capabilities,omitempty"`
}

// ManifestCapability describes a capability requirement in manifest.
type ManifestCapability struct {
	// Capability name (e.g., "files:read")
	Name string `yaml:"name"`

	// Justification for why it's needed
	Justification string `yaml:"justification"`

	// Optional if the plugin can work without it
	Optional bool `yaml:"optional,omitempty"`
}

// Validate checks if the manifest is valid.
func (m *PluginManifest) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("%w: missing id", ErrPluginManifestInvalid)
	}
	if m.Name == "" {
		return fmt.Errorf("%w: missing name", ErrPluginManifestInvalid)
	}
	if m.Module == "" {
		return fmt.Errorf("%w: missing module path", ErrPluginManifestInvalid)
	}
	if m.Checksum == "" {
		return fmt.Errorf("%w: missing checksum", ErrPluginManifestInvalid)
	}
	return nil
}

// Loader loads plugins from the filesystem.
type Loader struct {
	// basePath is the directory containing plugins
	basePath string
}

// NewLoader creates a new plugin loader.
func NewLoader(basePath string) *Loader {
	return &Loader{basePath: basePath}
}

// LoadManifest loads a plugin manifest from a directory.
func (l *Loader) LoadManifest(pluginDir string) (*PluginManifest, error) {
	manifestPath := filepath.Join(l.basePath, pluginDir, "plugin.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrPluginManifestNotFound, manifestPath)
		}
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest PluginManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPluginManifestInvalid, err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// LoadPlugin loads a complete plugin from a directory.
func (l *Loader) LoadPlugin(ctx context.Context, pluginDir string) (*Plugin, error) {
	manifest, err := l.LoadManifest(pluginDir)
	if err != nil {
		return nil, err
	}

	return l.LoadPluginFromManifest(ctx, pluginDir, manifest)
}

// LoadPluginFromManifest loads a plugin using a pre-loaded manifest.
func (l *Loader) LoadPluginFromManifest(_ context.Context, pluginDir string, manifest *PluginManifest) (*Plugin, error) {
	// Load the WASM module
	modulePath := filepath.Join(l.basePath, pluginDir, manifest.Module)
	moduleData, err := os.ReadFile(modulePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrPluginModuleNotFound, modulePath)
		}
		return nil, fmt.Errorf("failed to read module: %w", err)
	}

	// Verify checksum
	actualChecksum := sha256Hex(moduleData)
	if actualChecksum != manifest.Checksum {
		return nil, fmt.Errorf("%w: expected %s, got %s",
			ErrPluginChecksumMismatch, manifest.Checksum, actualChecksum)
	}

	// Build capabilities from manifest
	var caps *capability.Requirements
	if len(manifest.Capabilities) > 0 {
		caps = capability.NewRequirements()
		for _, c := range manifest.Capabilities {
			parsedCap, err := capability.ParseCapability(c.Name)
			if err != nil {
				return nil, fmt.Errorf("invalid capability %q: %w", c.Name, err)
			}
			if c.Optional {
				caps.AddOptional(parsedCap, c.Justification)
			} else {
				caps.AddCapability(parsedCap, c.Justification)
			}
		}
	}

	return &Plugin{
		ID:           manifest.ID,
		Name:         manifest.Name,
		Version:      manifest.Version,
		Module:       moduleData,
		Capabilities: caps,
		Checksum:     manifest.Checksum,
	}, nil
}

// ListPlugins returns a list of plugin directories.
func (l *Loader) ListPlugins() ([]string, error) {
	entries, err := os.ReadDir(l.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	var plugins []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if plugin.yaml exists
		manifestPath := filepath.Join(l.basePath, entry.Name(), "plugin.yaml")
		if _, err := os.Stat(manifestPath); err == nil {
			plugins = append(plugins, entry.Name())
		}
	}

	return plugins, nil
}

// CalculateChecksum computes the SHA256 checksum of a file.
func CalculateChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return sha256Hex(data), nil
}

// sha256Hex computes SHA256 hash and returns hex string.
func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Executor runs plugins in a sandbox.
type Executor struct {
	runtime Runtime
	loader  *Loader
}

// NewExecutor creates a new plugin executor.
func NewExecutor(runtime Runtime, loader *Loader) *Executor {
	return &Executor{
		runtime: runtime,
		loader:  loader,
	}
}

// Run loads and executes a plugin.
func (e *Executor) Run(ctx context.Context, pluginDir string, config Config, input []byte) (*ExecutionResult, error) {
	// Load the plugin
	plugin, err := e.loader.LoadPlugin(ctx, pluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	// Create sandbox
	sandbox, err := e.runtime.NewSandbox(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}
	defer func() { _ = sandbox.Close() }()

	// Validate plugin against sandbox policy
	if err := sandbox.Validate(ctx, plugin); err != nil {
		return nil, fmt.Errorf("plugin validation failed: %w", err)
	}

	// Execute
	return sandbox.Execute(ctx, plugin, input)
}

// ValidatePlugin loads and validates a plugin without executing it.
func (e *Executor) ValidatePlugin(ctx context.Context, pluginDir string, config Config) error {
	plugin, err := e.loader.LoadPlugin(ctx, pluginDir)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	sandbox, err := e.runtime.NewSandbox(config)
	if err != nil {
		return fmt.Errorf("failed to create sandbox: %w", err)
	}
	defer func() { _ = sandbox.Close() }()

	return sandbox.Validate(ctx, plugin)
}
