// Package plugin provides plugin discovery, loading, and management.
package plugin

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// PluginType indicates the type of plugin.
//
//nolint:revive // Name kept for backward compatibility
type PluginType string

const (
	// TypeConfig is a configuration-only plugin (YAML presets/packs).
	TypeConfig PluginType = "config"
	// TypeProvider is a WASM-based provider plugin.
	TypeProvider PluginType = "provider"
)

// TrustLevel indicates the trust level of a plugin.
type TrustLevel string

const (
	// TrustBuiltin is embedded in Preflight binary.
	TrustBuiltin TrustLevel = "builtin"
	// TrustVerified is signed by a trusted key.
	TrustVerified TrustLevel = "verified"
	// TrustCommunity is hash-verified only.
	TrustCommunity TrustLevel = "community"
	// TrustUntrusted has no verification.
	TrustUntrusted TrustLevel = "untrusted"
)

// Manifest describes a plugin's metadata and capabilities.
type Manifest struct {
	// APIVersion is the plugin API version (e.g., "v1")
	APIVersion string `yaml:"apiVersion"`
	// Type is the plugin type: "config" or "provider"
	Type PluginType `yaml:"type,omitempty"`
	// Name is the plugin identifier (e.g., "docker", "kubernetes")
	Name string `yaml:"name"`
	// Version is the semantic version (e.g., "1.0.0")
	Version string `yaml:"version"`
	// Description is a brief description of the plugin
	Description string `yaml:"description,omitempty"`
	// Author is the plugin author
	Author string `yaml:"author,omitempty"`
	// License is the plugin license (e.g., "MIT", "Apache-2.0")
	License string `yaml:"license,omitempty"`
	// Homepage is the plugin homepage URL
	Homepage string `yaml:"homepage,omitempty"`
	// Repository is the source repository URL
	Repository string `yaml:"repository,omitempty"`
	// Keywords are searchable tags
	Keywords []string `yaml:"keywords,omitempty"`
	// Provides lists the capabilities this plugin offers
	Provides Capabilities `yaml:"provides"`
	// Requires lists dependencies on other plugins
	Requires []Dependency `yaml:"requires,omitempty"`
	// MinPreflightVersion is the minimum preflight version required
	MinPreflightVersion string `yaml:"minPreflightVersion,omitempty"`
	// WASM contains WASM-specific configuration (for provider plugins)
	WASM *WASMConfig `yaml:"wasm,omitempty"`
	// Signature contains cryptographic signature information
	Signature *SignatureInfo `yaml:"signature,omitempty"`
}

// WASMConfig contains WASM-specific plugin configuration.
type WASMConfig struct {
	// Module is the path to the WASM module (relative to plugin.yaml)
	Module string `yaml:"module"`
	// Checksum is the SHA256 hash of the WASM module
	Checksum string `yaml:"checksum"`
	// Capabilities required by the WASM module
	Capabilities []WASMCapability `yaml:"capabilities,omitempty"`
}

// WASMCapability describes a capability required by a WASM plugin.
type WASMCapability struct {
	// Name is the capability name (e.g., "files:read", "shell:execute")
	Name string `yaml:"name"`
	// Justification explains why this capability is needed
	Justification string `yaml:"justification"`
	// Optional indicates if the plugin can work without this capability
	Optional bool `yaml:"optional,omitempty"`
}

// SignatureInfo contains cryptographic signature information.
type SignatureInfo struct {
	// Type is the signature type: "ssh", "gpg", or "sigstore"
	Type string `yaml:"type"`
	// KeyID is the key identifier (fingerprint or ID)
	KeyID string `yaml:"keyId"`
	// Data is the base64-encoded signature
	Data string `yaml:"data"`
}

// IsConfigPlugin returns true if this is a configuration-only plugin.
func (m *Manifest) IsConfigPlugin() bool {
	return m.Type == TypeConfig || m.Type == ""
}

// IsProviderPlugin returns true if this is a WASM provider plugin.
func (m *Manifest) IsProviderPlugin() bool {
	return m.Type == TypeProvider
}

// Capabilities describes what a plugin provides.
type Capabilities struct {
	// Providers are custom provider implementations
	Providers []ProviderSpec `yaml:"providers,omitempty"`
	// Presets are catalog presets
	Presets []string `yaml:"presets,omitempty"`
	// CapabilityPacks are catalog capability packs
	CapabilityPacks []string `yaml:"capabilityPacks,omitempty"`
}

// ProviderSpec describes a provider implementation.
type ProviderSpec struct {
	// Name is the provider name (e.g., "docker")
	Name string `yaml:"name"`
	// ConfigKey is the config section this provider handles
	ConfigKey string `yaml:"configKey"`
	// Description describes what this provider does
	Description string `yaml:"description,omitempty"`
}

// Dependency describes a plugin dependency.
type Dependency struct {
	// Name is the required plugin name
	Name string `yaml:"name"`
	// Version is a semver constraint (e.g., ">=1.0.0")
	Version string `yaml:"version,omitempty"`
}

// Plugin represents a loaded plugin.
type Plugin struct {
	// Manifest contains the plugin metadata
	Manifest Manifest
	// Path is the plugin's installation path
	Path string
	// Enabled indicates if the plugin is enabled
	Enabled bool
	// LoadedAt is when the plugin was loaded
	LoadedAt time.Time
}

// ID returns the plugin identifier.
func (p *Plugin) ID() string {
	return p.Manifest.Name
}

// String returns a human-readable plugin description.
func (p *Plugin) String() string {
	return fmt.Sprintf("%s@%s", p.Manifest.Name, p.Manifest.Version)
}

// Registry manages installed plugins with thread-safe access.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]*Plugin
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]*Plugin),
	}
}

// Register adds a plugin to the registry.
// Returns ErrNilPlugin if plugin is nil, ErrEmptyPluginName if name is empty,
// or PluginExistsError if a plugin with the same name is already registered.
func (r *Registry) Register(plugin *Plugin) error {
	if plugin == nil {
		return ErrNilPlugin
	}
	if plugin.Manifest.Name == "" {
		return ErrEmptyPluginName
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[plugin.Manifest.Name]; exists {
		return &PluginExistsError{Name: plugin.Manifest.Name}
	}
	r.plugins[plugin.Manifest.Name] = plugin
	return nil
}

// Get returns a plugin by name.
// Returns a deep copy of the plugin to prevent data races.
func (r *Registry) Get(name string) (*Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.plugins[name]
	if !ok {
		return nil, false
	}
	// Return a deep copy to prevent data races
	return plugin.Clone(), true
}

// List returns all registered plugins.
// Returns deep copies sorted by name for deterministic ordering.
func (r *Registry) List() []*Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		// Return deep copies to prevent data races
		plugins = append(plugins, p.Clone())
	}

	// Sort by name for deterministic ordering
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Manifest.Name < plugins[j].Manifest.Name
	})

	return plugins
}

// Enabled returns all enabled plugins.
// Returns deep copies sorted by name for deterministic ordering.
func (r *Registry) Enabled() []*Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]*Plugin, 0)
	for _, p := range r.plugins {
		if p.Enabled {
			// Return deep copies to prevent data races
			plugins = append(plugins, p.Clone())
		}
	}

	// Sort by name for deterministic ordering
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Manifest.Name < plugins[j].Manifest.Name
	})

	return plugins
}

// Remove removes a plugin from the registry.
func (r *Registry) Remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; exists {
		delete(r.plugins, name)
		return true
	}
	return false
}

// Count returns the number of registered plugins.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.plugins)
}

// ValidateManifest checks if a manifest is valid.
// Returns detailed error messages with examples and documentation links.
func ValidateManifest(m *Manifest) error {
	ve := &ValidationError{}

	// Required field validations with examples
	if m.APIVersion == "" {
		ve.Add("apiVersion is required. Example: apiVersion: v1")
	} else if m.APIVersion != "v1" {
		ve.Addf("unsupported apiVersion: %q (only 'v1' is currently supported)", m.APIVersion)
	}

	if m.Name == "" {
		ve.Add("name is required. Example: name: my-plugin")
	} else if err := validatePluginNameFormat(m.Name); err != nil {
		ve.Add(err.Error())
	}

	if m.Version == "" {
		ve.Add("version is required. Example: version: 1.0.0 (use semantic versioning)")
	} else if err := ValidateSemver(m.Version); err != nil {
		ve.Addf("version %q is not valid semantic versioning. Examples: 1.0.0, 1.2.3-beta.1, 2.0.0+build.123", m.Version)
	}

	// Return early if basic fields are invalid
	if ve.HasErrors() {
		return ve
	}

	// Validate type-specific requirements
	if m.IsProviderPlugin() {
		if err := validateProviderManifest(m); err != nil {
			// Merge validation errors
			var verr *ValidationError
			if errors.As(err, &verr) {
				ve.Errors = append(ve.Errors, verr.Errors...)
			} else {
				ve.Add(err.Error())
			}
		}
	} else {
		if err := validateConfigManifest(m); err != nil {
			var verr *ValidationError
			if errors.As(err, &verr) {
				ve.Errors = append(ve.Errors, verr.Errors...)
			} else {
				ve.Add(err.Error())
			}
		}
	}

	if ve.HasErrors() {
		return ve
	}

	return nil
}

// validatePluginNameFormat checks if a plugin name follows naming conventions.
func validatePluginNameFormat(name string) error {
	if len(name) < 2 {
		return fmt.Errorf("plugin name %q is too short (minimum 2 characters)", name)
	}
	if len(name) > 64 {
		return fmt.Errorf("plugin name %q is too long (maximum 64 characters)", name)
	}

	// Must start with a letter
	firstChar := name[0]
	isFirstCharLetter := (firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z')
	if !isFirstCharLetter {
		return fmt.Errorf("plugin name %q must start with a letter", name)
	}

	// Only allow alphanumeric, hyphens, and underscores
	for i, c := range name {
		isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'
		isAllowedSpecial := c == '-' || c == '_'
		if !isLetter && !isDigit && !isAllowedSpecial {
			return fmt.Errorf("plugin name %q contains invalid character %q at position %d (only letters, numbers, hyphens, and underscores allowed)", name, c, i)
		}
	}

	return nil
}

// validateProviderManifest validates a provider (WASM) plugin manifest.
func validateProviderManifest(m *Manifest) error {
	ve := &ValidationError{}

	// Provider plugins must have WASM config
	if m.WASM == nil {
		ve.Add("provider plugin requires 'wasm' configuration. Example:\n" +
			"  wasm:\n" +
			"    module: plugin.wasm\n" +
			"    checksum: sha256:abc123...")
		return ve
	}

	if m.WASM.Module == "" {
		ve.Add("wasm.module is required (path to WASM file). Example: module: plugin.wasm")
	}

	if m.WASM.Checksum == "" {
		ve.Add("wasm.checksum is required (SHA256 hash of WASM file). Example: checksum: sha256:abc123...")
	} else if !strings.HasPrefix(strings.ToLower(m.WASM.Checksum), "sha256:") && len(m.WASM.Checksum) != 64 {
		ve.Addf("wasm.checksum should be a SHA256 hash (64 hex characters or sha256: prefix). Got: %s", m.WASM.Checksum)
	}

	// Validate WASM capabilities
	for i, c := range m.WASM.Capabilities {
		if c.Name == "" {
			ve.Addf("wasm.capabilities[%d].name is required. Available: files:read, files:write, shell:execute, net:http", i)
		}
		if c.Justification == "" && !c.Optional {
			ve.Addf("wasm.capabilities[%d].justification is required for capability %q (explain why it's needed)", i, c.Name)
		}
	}

	// Validate provider specs
	if len(m.Provides.Providers) == 0 {
		ve.Add("provider plugin must define at least one provider in 'provides.providers'. Example:\n" +
			"  provides:\n" +
			"    providers:\n" +
			"      - name: docker\n" +
			"        configKey: docker")
	}

	for i, p := range m.Provides.Providers {
		if p.Name == "" {
			ve.Addf("provides.providers[%d].name is required", i)
		}
		if p.ConfigKey == "" {
			ve.Addf("provides.providers[%d].configKey is required (the config section this provider handles)", i)
		}
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

// validateConfigManifest validates a configuration-only plugin manifest.
func validateConfigManifest(m *Manifest) error {
	ve := &ValidationError{}

	// Config plugins must provide at least presets or capability packs
	hasPresets := len(m.Provides.Presets) > 0
	hasPacks := len(m.Provides.CapabilityPacks) > 0
	hasProviders := len(m.Provides.Providers) > 0

	if !hasPresets && !hasPacks && !hasProviders {
		ve.Add("config plugin must provide at least one preset, capability pack, or provider config. Example:\n" +
			"  provides:\n" +
			"    presets:\n" +
			"      - nvim:balanced\n" +
			"    capabilityPacks:\n" +
			"      - go-developer")
	}

	// Config plugins should NOT have WASM config
	if m.WASM != nil {
		ve.Add("config plugins should not have 'wasm' configuration. " +
			"To create a WASM provider plugin, add 'type: provider' to your manifest.")
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

// semverRegex matches semantic version strings (simplified).
// Matches: 1.0.0, 1.0.0-alpha, 1.0.0-alpha.1, 1.0.0+build.123
var semverRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
	`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
	`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// ValidateSemver checks if a version string is valid semantic versioning.
func ValidateSemver(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	// Strip optional 'v' prefix
	v := version
	if len(v) > 0 && (v[0] == 'v' || v[0] == 'V') {
		v = v[1:]
	}
	if !semverRegex.MatchString(v) {
		return fmt.Errorf("invalid semantic version: %s", version)
	}
	return nil
}

// VerifyChecksum verifies a WASM module's SHA256 checksum.
// Expected checksum must be a valid hex-encoded SHA256 hash (64 characters).
func VerifyChecksum(data []byte, expectedChecksum string) error {
	if expectedChecksum == "" {
		return fmt.Errorf("checksum cannot be empty")
	}

	// Validate checksum format: must be exactly 64 hex characters (SHA256)
	if len(expectedChecksum) != 64 {
		return fmt.Errorf("invalid checksum length: expected 64 characters (SHA256), got %d", len(expectedChecksum))
	}

	// Validate all characters are valid hex
	for i, c := range expectedChecksum {
		isHexDigit := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !isHexDigit {
			return fmt.Errorf("invalid checksum character at position %d: %c (must be 0-9, a-f, or A-F)", i, c)
		}
	}

	// Compute SHA256
	hash := sha256.Sum256(data)
	actualChecksum := hex.EncodeToString(hash[:])

	// Normalize to lowercase for consistent comparison
	expectedLower := strings.ToLower(expectedChecksum)
	actualLower := strings.ToLower(actualChecksum)

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(actualLower), []byte(expectedLower)) != 1 {
		return &ChecksumError{
			Expected: expectedChecksum,
			Actual:   actualChecksum,
		}
	}
	return nil
}

// Clone creates a deep copy of the Plugin.
func (p *Plugin) Clone() *Plugin {
	if p == nil {
		return nil
	}
	return &Plugin{
		Manifest: p.Manifest.Clone(),
		Path:     p.Path,
		Enabled:  p.Enabled,
		LoadedAt: p.LoadedAt,
	}
}

// Clone creates a deep copy of the Manifest.
func (m Manifest) Clone() Manifest {
	clone := Manifest{
		APIVersion:          m.APIVersion,
		Type:                m.Type,
		Name:                m.Name,
		Version:             m.Version,
		Description:         m.Description,
		Author:              m.Author,
		License:             m.License,
		Homepage:            m.Homepage,
		Repository:          m.Repository,
		MinPreflightVersion: m.MinPreflightVersion,
	}

	// Deep copy slices
	if m.Keywords != nil {
		clone.Keywords = make([]string, len(m.Keywords))
		copy(clone.Keywords, m.Keywords)
	}

	// Deep copy Provides
	clone.Provides = m.Provides.Clone()

	// Deep copy Requires
	if m.Requires != nil {
		clone.Requires = make([]Dependency, len(m.Requires))
		copy(clone.Requires, m.Requires)
	}

	// Deep copy WASM config
	if m.WASM != nil {
		clone.WASM = m.WASM.Clone()
	}

	// Deep copy Signature
	if m.Signature != nil {
		clone.Signature = m.Signature.Clone()
	}

	return clone
}

// Clone creates a deep copy of Capabilities.
func (c Capabilities) Clone() Capabilities {
	clone := Capabilities{}

	if c.Providers != nil {
		clone.Providers = make([]ProviderSpec, len(c.Providers))
		copy(clone.Providers, c.Providers)
	}

	if c.Presets != nil {
		clone.Presets = make([]string, len(c.Presets))
		copy(clone.Presets, c.Presets)
	}

	if c.CapabilityPacks != nil {
		clone.CapabilityPacks = make([]string, len(c.CapabilityPacks))
		copy(clone.CapabilityPacks, c.CapabilityPacks)
	}

	return clone
}

// Clone creates a deep copy of WASMConfig.
func (w *WASMConfig) Clone() *WASMConfig {
	if w == nil {
		return nil
	}
	clone := &WASMConfig{
		Module:   w.Module,
		Checksum: w.Checksum,
	}
	if w.Capabilities != nil {
		clone.Capabilities = make([]WASMCapability, len(w.Capabilities))
		copy(clone.Capabilities, w.Capabilities)
	}
	return clone
}

// Clone creates a deep copy of SignatureInfo.
func (s *SignatureInfo) Clone() *SignatureInfo {
	if s == nil {
		return nil
	}
	return &SignatureInfo{
		Type:  s.Type,
		KeyID: s.KeyID,
		Data:  s.Data,
	}
}

// VerifySignature verifies a plugin's cryptographic signature using the default config.
// This function first validates that the signature is well-formed, then attempts
// actual cryptographic verification using available tools (ssh-keygen, gpg, cosign).
//
// For custom configuration (trusted keys, keyring paths), use VerifySignatureWithConfig.
func VerifySignature(manifest *Manifest, manifestData []byte) error {
	// Use default verification config
	config := DefaultVerificationConfig()
	return VerifySignatureWithConfig(manifest, manifestData, config)
}

// VerifySignatureStructure validates that a signature has the correct structure
// without performing actual cryptographic verification.
// Use this when you only need to check if a signature is well-formed.
func VerifySignatureStructure(manifest *Manifest) error {
	if manifest == nil {
		return &SignatureError{Reason: "manifest cannot be nil"}
	}
	if manifest.Signature == nil {
		return &SignatureError{Reason: "no signature present"}
	}

	sig := manifest.Signature

	// Validate signature type
	switch sig.Type {
	case "ssh", "gpg", "sigstore":
		// Valid signature types
	case "":
		return &SignatureError{Reason: "signature type is required"}
	default:
		return &SignatureError{Reason: fmt.Sprintf("unsupported signature type: %s (use ssh, gpg, or sigstore)", sig.Type)}
	}

	// Validate key ID
	if sig.KeyID == "" {
		return &SignatureError{Reason: "signature keyId is required"}
	}

	// Validate signature data
	if sig.Data == "" {
		return &SignatureError{Reason: "signature data is required"}
	}

	// Validate base64 encoding of signature data
	if len(sig.Data) < 4 {
		return &SignatureError{Reason: "signature data is too short"}
	}

	return nil
}

// TrustLevelOrder returns the numeric order of a trust level (higher = more trusted).
func TrustLevelOrder(level TrustLevel) int {
	switch level {
	case TrustBuiltin:
		return 4
	case TrustVerified:
		return 3
	case TrustCommunity:
		return 2
	case TrustUntrusted:
		return 1
	default:
		return 0
	}
}

// TrustPolicy defines what trust levels are allowed for operations.
type TrustPolicy struct {
	// MinLevel is the minimum trust level required
	MinLevel TrustLevel
	// AllowedLevels is an explicit list of allowed levels (if set, overrides MinLevel)
	AllowedLevels []TrustLevel
}

// DefaultTrustPolicy returns the default trust policy (community and above).
func DefaultTrustPolicy() TrustPolicy {
	return TrustPolicy{MinLevel: TrustCommunity}
}

// StrictTrustPolicy returns a strict trust policy (verified and above).
func StrictTrustPolicy() TrustPolicy {
	return TrustPolicy{MinLevel: TrustVerified}
}

// EnforceTrustLevel checks if a plugin's trust level meets the policy requirements.
func EnforceTrustLevel(plugin *Plugin, policy TrustPolicy) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}

	trustLevel := DetermineTrustLevel(plugin)

	// Check explicit allowed levels first
	if len(policy.AllowedLevels) > 0 {
		for _, allowed := range policy.AllowedLevels {
			if trustLevel == allowed {
				return nil
			}
		}
		return &TrustError{
			PluginName: plugin.Manifest.Name,
			Level:      trustLevel,
			Required:   policy.MinLevel,
			Reason:     "trust level not in allowed list",
		}
	}

	// Check minimum trust level
	if TrustLevelOrder(trustLevel) < TrustLevelOrder(policy.MinLevel) {
		return &TrustError{
			PluginName: plugin.Manifest.Name,
			Level:      trustLevel,
			Required:   policy.MinLevel,
			Reason:     fmt.Sprintf("plugin trust level %q is below required minimum %q", trustLevel, policy.MinLevel),
		}
	}

	return nil
}

// DetermineTrustLevel determines the trust level of a plugin based on its source and verification.
// This version does not perform cryptographic verification - use DetermineTrustLevelWithVerification
// for full signature verification.
func DetermineTrustLevel(plugin *Plugin) TrustLevel {
	return DetermineTrustLevelWithVerification(plugin, nil, nil)
}

// DetermineTrustLevelWithVerification determines trust level with optional cryptographic verification.
// If manifestData and config are provided, actual signature verification is attempted.
// If verification succeeds, TrustVerified is returned for signed plugins.
func DetermineTrustLevelWithVerification(plugin *Plugin, manifestData []byte, config *VerificationConfig) TrustLevel {
	if plugin == nil {
		return TrustUntrusted
	}

	// Builtin plugins are embedded in the binary
	if strings.HasPrefix(plugin.Path, "builtin:") {
		return TrustBuiltin
	}

	// Check if plugin has a valid signature
	if plugin.Manifest.Signature != nil {
		// First check structure
		if err := VerifySignatureStructure(&plugin.Manifest); err != nil {
			// Invalid signature structure - fall through to other checks
			goto checkOther
		}

		// If we have manifest data and config, attempt cryptographic verification
		if len(manifestData) > 0 && config != nil {
			if err := VerifySignatureWithConfig(&plugin.Manifest, manifestData, config); err == nil {
				return TrustVerified // Cryptographic verification succeeded!
			}
			// Verification failed - still grant community trust for valid structure
		}

		// Valid structure but no crypto verification - grant community trust
		return TrustCommunity
	}

checkOther:
	// Check if plugin has integrity verification (checksum)
	if plugin.Manifest.WASM != nil && plugin.Manifest.WASM.Checksum != "" {
		return TrustCommunity
	}

	// Config plugins with presets are community-level
	if len(plugin.Manifest.Provides.Presets) > 0 || len(plugin.Manifest.Provides.CapabilityPacks) > 0 {
		return TrustCommunity
	}

	return TrustUntrusted
}

// AllowedCapabilities defines which WASM capabilities are permitted.
var AllowedCapabilities = map[string]bool{
	// File system (read-only by default)
	"files:read":  true,
	"files:write": true,
	"files:stat":  true,

	// Environment (read-only)
	"env:read": true,

	// Shell execution (requires justification)
	"shell:execute": true,

	// Network (restricted)
	"net:http": true,

	// System info (read-only)
	"sys:info": true,

	// Preflight APIs
	"preflight:config": true,
	"preflight:state":  true,
}

// DangerousCapabilities are capabilities that require extra scrutiny.
var DangerousCapabilities = map[string]bool{
	"shell:execute": true,
	"files:write":   true,
	"net:http":      true,
}

// ValidateCapabilities checks if the requested WASM capabilities are allowed.
func ValidateCapabilities(caps []WASMCapability) error {
	if len(caps) == 0 {
		return nil
	}

	var errors []string

	for _, cap := range caps {
		// Check if capability is in allowed list
		if !AllowedCapabilities[cap.Name] {
			errors = append(errors, fmt.Sprintf("capability %q is not recognized", cap.Name))
			continue
		}

		// Dangerous capabilities require justification
		if DangerousCapabilities[cap.Name] && cap.Justification == "" {
			errors = append(errors, fmt.Sprintf("dangerous capability %q requires a justification", cap.Name))
		}
	}

	if len(errors) > 0 {
		return &CapabilityError{
			Capability: "multiple",
			Reason:     strings.Join(errors, "; "),
		}
	}

	return nil
}

// DefaultValidator implements the Validator interface.
type DefaultValidator struct {
	// AllowedCaps overrides the default allowed capabilities
	AllowedCaps map[string]bool
}

// NewValidator creates a new manifest validator.
func NewValidator() *DefaultValidator {
	return &DefaultValidator{
		AllowedCaps: AllowedCapabilities,
	}
}

// Validate checks if a manifest is valid.
func (v *DefaultValidator) Validate(manifest *Manifest) error {
	return ValidateManifest(manifest)
}

// ValidateCapabilities checks if requested WASM capabilities are allowed.
func (v *DefaultValidator) ValidateCapabilities(caps []WASMCapability) error {
	if len(caps) == 0 {
		return nil
	}

	allowedCaps := v.AllowedCaps
	if allowedCaps == nil {
		allowedCaps = AllowedCapabilities
	}

	var errors []string

	for _, cap := range caps {
		if !allowedCaps[cap.Name] {
			errors = append(errors, fmt.Sprintf("capability %q is not allowed", cap.Name))
			continue
		}

		if DangerousCapabilities[cap.Name] && cap.Justification == "" {
			errors = append(errors, fmt.Sprintf("capability %q requires justification", cap.Name))
		}
	}

	if len(errors) > 0 {
		return &CapabilityError{
			Capability: "validation",
			Reason:     strings.Join(errors, "; "),
		}
	}

	return nil
}

// Ensure DefaultValidator implements Validator interface.
var _ Validator = (*DefaultValidator)(nil)

// VerificationConfig provides configuration for cryptographic signature verification.
type VerificationConfig struct {
	// SSHAllowedSignersFile is the path to the SSH allowed_signers file.
	// Format: <principal> <key-type> <public-key>
	// Example: plugin@example.com ssh-ed25519 AAAAC3NzaC1lZDI1...
	SSHAllowedSignersFile string

	// GPGKeyring is the path to the GPG keyring directory.
	// If empty, uses the default GPG keyring (~/.gnupg).
	GPGKeyring string

	// SigstoreTrustedRoots is the path to Sigstore trusted root certificates.
	// If empty, uses the public Sigstore infrastructure.
	SigstoreTrustedRoots string

	// TrustedKeyIDs is a list of explicitly trusted key IDs.
	// These bypass the normal verification and grant TrustVerified.
	TrustedKeyIDs []string
}

// DefaultVerificationConfig returns a default verification config.
// Uses standard locations for key storage.
func DefaultVerificationConfig() *VerificationConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		return &VerificationConfig{}
	}

	return &VerificationConfig{
		SSHAllowedSignersFile: filepath.Join(home, ".preflight", "allowed_signers"),
		GPGKeyring:            "", // Use system default
		SigstoreTrustedRoots:  "", // Use public Sigstore
	}
}

// verifySSHSignature verifies an SSH signature using ssh-keygen.
func verifySSHSignature(manifest *Manifest, manifestData []byte, config *VerificationConfig) error {
	if config == nil || config.SSHAllowedSignersFile == "" {
		return &SignatureError{
			Reason: "SSH verification requires an allowed_signers file: " +
				"create ~/.preflight/allowed_signers with trusted public keys",
		}
	}

	// Check if allowed_signers file exists
	if _, err := os.Stat(config.SSHAllowedSignersFile); os.IsNotExist(err) {
		return &SignatureError{
			Reason: fmt.Sprintf("SSH allowed_signers file not found: %s", config.SSHAllowedSignersFile),
		}
	}

	// Check if ssh-keygen is available
	sshKeygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		return &SignatureError{
			Reason: "ssh-keygen not found: required for SSH signature verification",
		}
	}

	// Decode the base64 signature
	sigBytes, err := base64.StdEncoding.DecodeString(manifest.Signature.Data)
	if err != nil {
		return &SignatureError{
			Reason: fmt.Sprintf("invalid signature encoding: %v", err),
		}
	}

	// Create temp files for verification
	tempDir, err := os.MkdirTemp("", "preflight-verify-*")
	if err != nil {
		return &SignatureError{Reason: fmt.Sprintf("creating temp directory: %v", err)}
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	dataFile := filepath.Join(tempDir, "manifest")
	sigFile := filepath.Join(tempDir, "manifest.sig")

	if err := os.WriteFile(dataFile, manifestData, 0600); err != nil {
		return &SignatureError{Reason: fmt.Sprintf("writing manifest data: %v", err)}
	}
	if err := os.WriteFile(sigFile, sigBytes, 0600); err != nil {
		return &SignatureError{Reason: fmt.Sprintf("writing signature: %v", err)}
	}

	// Run ssh-keygen -Y verify
	// -f: allowed_signers file
	// -I: identity (principal) to verify against
	// -n: namespace (use "preflight" for plugin signatures)
	// -s: signature file
	cmd := exec.Command(sshKeygen, "-Y", "verify",
		"-f", config.SSHAllowedSignersFile,
		"-I", manifest.Signature.KeyID,
		"-n", "preflight",
		"-s", sigFile,
	)
	cmd.Stdin = bytes.NewReader(manifestData)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return &SignatureError{
			Reason: fmt.Sprintf("SSH signature verification failed: %s", errMsg),
		}
	}

	return nil
}

// verifyGPGSignature verifies a GPG signature using gpg.
func verifyGPGSignature(manifest *Manifest, manifestData []byte, config *VerificationConfig) error {
	// Check if gpg is available
	gpgPath, err := exec.LookPath("gpg")
	if err != nil {
		return &SignatureError{
			Reason: "gpg not found: required for GPG signature verification",
		}
	}

	// Decode the base64 signature
	sigBytes, err := base64.StdEncoding.DecodeString(manifest.Signature.Data)
	if err != nil {
		return &SignatureError{
			Reason: fmt.Sprintf("invalid signature encoding: %v", err),
		}
	}

	// Create temp files for verification
	tempDir, err := os.MkdirTemp("", "preflight-verify-*")
	if err != nil {
		return &SignatureError{Reason: fmt.Sprintf("creating temp directory: %v", err)}
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	dataFile := filepath.Join(tempDir, "manifest")
	sigFile := filepath.Join(tempDir, "manifest.sig")

	if err := os.WriteFile(dataFile, manifestData, 0600); err != nil {
		return &SignatureError{Reason: fmt.Sprintf("writing manifest data: %v", err)}
	}
	if err := os.WriteFile(sigFile, sigBytes, 0600); err != nil {
		return &SignatureError{Reason: fmt.Sprintf("writing signature: %v", err)}
	}

	// Build gpg command
	args := []string{"--verify", sigFile, dataFile}
	if config != nil && config.GPGKeyring != "" {
		args = append([]string{"--homedir", config.GPGKeyring}, args...)
	}

	cmd := exec.Command(gpgPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return &SignatureError{
			Reason: fmt.Sprintf("GPG signature verification failed: %s", errMsg),
		}
	}

	// Verify the key ID matches
	// GPG output contains the key ID used for signing
	output := stderr.String()
	if manifest.Signature.KeyID != "" && !strings.Contains(output, manifest.Signature.KeyID) {
		// Check if key ID matches (GPG may use short or long key IDs)
		keyIDUpper := strings.ToUpper(manifest.Signature.KeyID)
		if !strings.Contains(strings.ToUpper(output), keyIDUpper) {
			return &SignatureError{
				Reason: fmt.Sprintf("signature key ID mismatch: expected %s", manifest.Signature.KeyID),
			}
		}
	}

	return nil
}

// verifySigstoreSignature verifies a Sigstore signature.
func verifySigstoreSignature(manifest *Manifest, manifestData []byte, config *VerificationConfig) error {
	// Check if cosign is available
	cosignPath, err := exec.LookPath("cosign")
	if err != nil {
		return &SignatureError{
			Reason: "cosign not found: install sigstore/cosign for Sigstore signature verification " +
				"(https://docs.sigstore.dev/cosign/installation/)",
		}
	}

	// Decode the base64 signature bundle
	sigBytes, err := base64.StdEncoding.DecodeString(manifest.Signature.Data)
	if err != nil {
		return &SignatureError{
			Reason: fmt.Sprintf("invalid signature encoding: %v", err),
		}
	}

	// Create temp files for verification
	tempDir, err := os.MkdirTemp("", "preflight-verify-*")
	if err != nil {
		return &SignatureError{Reason: fmt.Sprintf("creating temp directory: %v", err)}
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	dataFile := filepath.Join(tempDir, "manifest")
	bundleFile := filepath.Join(tempDir, "manifest.bundle")

	if err := os.WriteFile(dataFile, manifestData, 0600); err != nil {
		return &SignatureError{Reason: fmt.Sprintf("writing manifest data: %v", err)}
	}
	if err := os.WriteFile(bundleFile, sigBytes, 0600); err != nil {
		return &SignatureError{Reason: fmt.Sprintf("writing signature bundle: %v", err)}
	}

	// Run cosign verify-blob
	args := []string{
		"verify-blob",
		"--bundle", bundleFile,
		dataFile,
	}

	// Add trusted roots if specified
	if config != nil && config.SigstoreTrustedRoots != "" {
		args = append(args, "--trusted-root", config.SigstoreTrustedRoots)
	}

	// Verify against the certificate identity (keyID contains the email/identity)
	if manifest.Signature.KeyID != "" {
		args = append(args, "--certificate-identity", manifest.Signature.KeyID)
		// Use GitHub OIDC issuer by default for plugin signatures
		args = append(args, "--certificate-oidc-issuer", "https://token.actions.githubusercontent.com")
	}

	cmd := exec.Command(cosignPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return &SignatureError{
			Reason: fmt.Sprintf("Sigstore signature verification failed: %s", errMsg),
		}
	}

	return nil
}

// VerifySignatureWithConfig verifies a plugin's cryptographic signature using the provided config.
// Returns nil if verification succeeds, or an error describing why verification failed.
func VerifySignatureWithConfig(manifest *Manifest, manifestData []byte, config *VerificationConfig) error {
	// First validate structure
	if err := VerifySignatureStructure(manifest); err != nil {
		return err
	}

	// Check if key ID is explicitly trusted (bypass verification)
	if config != nil && len(config.TrustedKeyIDs) > 0 {
		for _, trustedID := range config.TrustedKeyIDs {
			if manifest.Signature.KeyID == trustedID {
				return nil // Explicitly trusted
			}
		}
	}

	// Dispatch to type-specific verifier
	switch manifest.Signature.Type {
	case "ssh":
		return verifySSHSignature(manifest, manifestData, config)
	case "gpg":
		return verifyGPGSignature(manifest, manifestData, config)
	case "sigstore":
		return verifySigstoreSignature(manifest, manifestData, config)
	default:
		return &SignatureError{
			Reason: fmt.Sprintf("unsupported signature type: %s", manifest.Signature.Type),
		}
	}
}
