package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_ID(t *testing.T) {
	p := &Plugin{
		Manifest: Manifest{Name: "docker"},
	}
	assert.Equal(t, "docker", p.ID())
}

func TestPlugin_String(t *testing.T) {
	p := &Plugin{
		Manifest: Manifest{Name: "docker", Version: "1.0.0"},
	}
	assert.Equal(t, "docker@1.0.0", p.String())
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	p := &Plugin{
		Manifest: Manifest{Name: "docker", Version: "1.0.0"},
		Enabled:  true,
	}

	err := r.Register(p)
	require.NoError(t, err)

	assert.Equal(t, 1, r.Count())
}

func TestRegistry_Register_NilPlugin(t *testing.T) {
	r := NewRegistry()
	err := r.Register(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestRegistry_Register_EmptyName(t *testing.T) {
	r := NewRegistry()
	p := &Plugin{Manifest: Manifest{Name: ""}}
	err := r.Register(p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	r := NewRegistry()

	p1 := &Plugin{Manifest: Manifest{Name: "docker", Version: "1.0.0"}}
	p2 := &Plugin{Manifest: Manifest{Name: "docker", Version: "2.0.0"}}

	err := r.Register(p1)
	require.NoError(t, err)

	err = r.Register(p2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	p := &Plugin{
		Manifest: Manifest{Name: "docker", Version: "1.0.0"},
	}
	_ = r.Register(p)

	got, ok := r.Get("docker")
	assert.True(t, ok)
	assert.Equal(t, "docker", got.Manifest.Name)

	_, ok = r.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	_ = r.Register(&Plugin{Manifest: Manifest{Name: "a"}})
	_ = r.Register(&Plugin{Manifest: Manifest{Name: "b"}})
	_ = r.Register(&Plugin{Manifest: Manifest{Name: "c"}})

	list := r.List()
	assert.Len(t, list, 3)
}

func TestRegistry_Enabled(t *testing.T) {
	r := NewRegistry()

	_ = r.Register(&Plugin{Manifest: Manifest{Name: "a"}, Enabled: true})
	_ = r.Register(&Plugin{Manifest: Manifest{Name: "b"}, Enabled: false})
	_ = r.Register(&Plugin{Manifest: Manifest{Name: "c"}, Enabled: true})

	enabled := r.Enabled()
	assert.Len(t, enabled, 2)
}

func TestRegistry_Remove(t *testing.T) {
	r := NewRegistry()

	_ = r.Register(&Plugin{Manifest: Manifest{Name: "docker"}})
	assert.Equal(t, 1, r.Count())

	removed := r.Remove("docker")
	assert.True(t, removed)
	assert.Equal(t, 0, r.Count())

	removed = r.Remove("nonexistent")
	assert.False(t, removed)
}

func TestManifest_IsConfigPlugin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest Manifest
		expected bool
	}{
		{
			name:     "empty type is config",
			manifest: Manifest{},
			expected: true,
		},
		{
			name:     "explicit config type",
			manifest: Manifest{Type: TypeConfig},
			expected: true,
		},
		{
			name:     "provider type is not config",
			manifest: Manifest{Type: TypeProvider},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.manifest.IsConfigPlugin())
		})
	}
}

func TestManifest_IsProviderPlugin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest Manifest
		expected bool
	}{
		{
			name:     "empty type is not provider",
			manifest: Manifest{},
			expected: false,
		},
		{
			name:     "config type is not provider",
			manifest: Manifest{Type: TypeConfig},
			expected: false,
		},
		{
			name:     "provider type is provider",
			manifest: Manifest{Type: TypeProvider},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.manifest.IsProviderPlugin())
		})
	}
}

func TestValidateManifest_ConfigPlugin_Valid(t *testing.T) {
	m := &Manifest{
		APIVersion:  "v1",
		Type:        TypeConfig,
		Name:        "my-team-config",
		Version:     "1.0.0",
		Description: "Team configuration plugin",
		Provides: Capabilities{
			Presets: []string{"team:backend", "team:frontend"},
		},
	}

	err := ValidateManifest(m)
	assert.NoError(t, err)
}

func TestValidateManifest_ConfigPlugin_WithCapabilityPacks(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Name:       "my-packs",
		Version:    "1.0.0",
		Provides: Capabilities{
			CapabilityPacks: []string{"go-developer", "rust-developer"},
		},
	}

	err := ValidateManifest(m)
	assert.NoError(t, err)
}

func TestValidateManifest_ConfigPlugin_Empty(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Type:       TypeConfig,
		Name:       "empty-plugin",
		Version:    "1.0.0",
		Provides:   Capabilities{},
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must provide at least one preset")
}

func TestValidateManifest_ConfigPlugin_WithWASM(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Type:       TypeConfig,
		Name:       "bad-config",
		Version:    "1.0.0",
		Provides: Capabilities{
			Presets: []string{"test:preset"},
		},
		WASM: &WASMConfig{
			Module:   "plugin.wasm",
			Checksum: "abc123",
		},
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "should not have 'wasm' configuration")
}

func TestValidateManifest_ProviderPlugin_Valid(t *testing.T) {
	m := &Manifest{
		APIVersion:  "v1",
		Type:        TypeProvider,
		Name:        "docker",
		Version:     "1.0.0",
		Description: "Docker provider for Preflight",
		Provides: Capabilities{
			Providers: []ProviderSpec{
				{Name: "docker", ConfigKey: "docker"},
			},
		},
		WASM: &WASMConfig{
			Module:   "plugin.wasm",
			Checksum: "sha256:abc123def456",
			Capabilities: []WASMCapability{
				{Name: "shell:execute", Justification: "Run docker commands"},
			},
		},
	}

	err := ValidateManifest(m)
	assert.NoError(t, err)
}

func TestValidateManifest_ProviderPlugin_MissingWASM(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Type:       TypeProvider,
		Name:       "docker",
		Version:    "1.0.0",
		Provides: Capabilities{
			Providers: []ProviderSpec{
				{Name: "docker", ConfigKey: "docker"},
			},
		},
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 'wasm' configuration")
}

func TestValidateManifest_ProviderPlugin_MissingModule(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Type:       TypeProvider,
		Name:       "docker",
		Version:    "1.0.0",
		WASM: &WASMConfig{
			Checksum: "abc123",
		},
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wasm.module is required")
}

func TestValidateManifest_ProviderPlugin_MissingChecksum(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Type:       TypeProvider,
		Name:       "docker",
		Version:    "1.0.0",
		WASM: &WASMConfig{
			Module: "plugin.wasm",
		},
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wasm.checksum is required")
}

func TestValidateManifest_ProviderPlugin_CapabilityMissingName(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Type:       TypeProvider,
		Name:       "docker",
		Version:    "1.0.0",
		Provides: Capabilities{
			Providers: []ProviderSpec{{Name: "docker", ConfigKey: "docker"}},
		},
		WASM: &WASMConfig{
			Module:   "plugin.wasm",
			Checksum: "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc1",
			Capabilities: []WASMCapability{
				{Justification: "Missing name"},
			},
		},
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wasm.capabilities[0].name is required")
}

func TestValidateManifest_ProviderPlugin_CapabilityMissingJustification(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Type:       TypeProvider,
		Name:       "docker",
		Version:    "1.0.0",
		Provides: Capabilities{
			Providers: []ProviderSpec{{Name: "docker", ConfigKey: "docker"}},
		},
		WASM: &WASMConfig{
			Module:   "plugin.wasm",
			Checksum: "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc1",
			Capabilities: []WASMCapability{
				{Name: "shell:execute"},
			},
		},
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wasm.capabilities[0].justification is required")
}

func TestValidateManifest_MissingAPIVersion(t *testing.T) {
	m := &Manifest{
		Name:    "docker",
		Version: "1.0.0",
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "apiVersion is required")
}

func TestValidateManifest_UnsupportedAPIVersion(t *testing.T) {
	m := &Manifest{
		APIVersion: "v2",
		Name:       "docker",
		Version:    "1.0.0",
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported apiVersion")
}

func TestValidateManifest_MissingName(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Version:    "1.0.0",
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestValidateManifest_MissingVersion(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Name:       "docker",
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestValidateManifest_ProviderMissingName(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Type:       TypeProvider,
		Name:       "docker",
		Version:    "1.0.0",
		Provides: Capabilities{
			Providers: []ProviderSpec{
				{ConfigKey: "docker"},
			},
		},
		WASM: &WASMConfig{
			Module:   "plugin.wasm",
			Checksum: "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc1",
		},
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provides.providers[0].name is required")
}

func TestValidateManifest_ProviderMissingConfigKey(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Type:       TypeProvider,
		Name:       "docker",
		Version:    "1.0.0",
		Provides: Capabilities{
			Providers: []ProviderSpec{
				{Name: "docker"},
			},
		},
		WASM: &WASMConfig{
			Module:   "plugin.wasm",
			Checksum: "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc1",
		},
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configKey is required")
}

func TestManifest_FullConfigExample(t *testing.T) {
	m := Manifest{
		APIVersion:          "v1",
		Type:                TypeConfig,
		Name:                "kubernetes-config",
		Version:             "1.2.3",
		Description:         "Kubernetes configuration for Preflight",
		Author:              "K8s Team",
		License:             "Apache-2.0",
		Homepage:            "https://example.com/preflight-k8s",
		Repository:          "https://github.com/example/preflight-k8s-config",
		Keywords:            []string{"kubernetes", "k8s", "container", "orchestration"},
		MinPreflightVersion: "2.0.0",
		Provides: Capabilities{
			Presets:         []string{"k8s:dev", "k8s:prod"},
			CapabilityPacks: []string{"k8s-developer"},
		},
		Requires: []Dependency{
			{Name: "docker", Version: ">=1.0.0"},
		},
	}

	err := ValidateManifest(&m)
	assert.NoError(t, err)

	assert.Equal(t, "kubernetes-config", m.Name)
	assert.True(t, m.IsConfigPlugin())
	assert.Len(t, m.Provides.Presets, 2)
	assert.Len(t, m.Requires, 1)
}

func TestManifest_FullProviderExample(t *testing.T) {
	m := Manifest{
		APIVersion:          "v1",
		Type:                TypeProvider,
		Name:                "kubernetes",
		Version:             "1.2.3",
		Description:         "Kubernetes tooling for Preflight",
		Author:              "K8s Team",
		License:             "Apache-2.0",
		Homepage:            "https://example.com/preflight-k8s",
		Repository:          "https://github.com/example/preflight-k8s",
		Keywords:            []string{"kubernetes", "k8s", "container", "orchestration"},
		MinPreflightVersion: "2.0.0",
		Provides: Capabilities{
			Providers: []ProviderSpec{
				{Name: "kubectl", ConfigKey: "kubernetes.kubectl", Description: "kubectl installation"},
				{Name: "helm", ConfigKey: "kubernetes.helm", Description: "Helm chart management"},
			},
			Presets:         []string{"k8s:dev", "k8s:prod"},
			CapabilityPacks: []string{"k8s-developer"},
		},
		Requires: []Dependency{
			{Name: "docker", Version: ">=1.0.0"},
		},
		WASM: &WASMConfig{
			Module:   "plugin.wasm",
			Checksum: "sha256:abc123def456789",
			Capabilities: []WASMCapability{
				{Name: "shell:execute", Justification: "Run kubectl and helm commands"},
				{Name: "files:write", Justification: "Write kubeconfig files"},
			},
		},
	}

	err := ValidateManifest(&m)
	assert.NoError(t, err)

	assert.Equal(t, "kubernetes", m.Name)
	assert.True(t, m.IsProviderPlugin())
	assert.Len(t, m.Provides.Providers, 2)
	assert.Len(t, m.Provides.Presets, 2)
	assert.Len(t, m.Requires, 1)
	assert.Len(t, m.WASM.Capabilities, 2)
}

func TestPlugin_LoadedAt(t *testing.T) {
	before := time.Now()
	p := &Plugin{
		Manifest: Manifest{Name: "test"},
		LoadedAt: time.Now(),
	}
	after := time.Now()

	assert.True(t, p.LoadedAt.After(before) || p.LoadedAt.Equal(before))
	assert.True(t, p.LoadedAt.Before(after) || p.LoadedAt.Equal(after))
}

// Test Clone methods for deep copy

func TestPlugin_Clone(t *testing.T) {
	t.Parallel()

	original := &Plugin{
		Manifest: Manifest{
			APIVersion:  "v1",
			Type:        TypeProvider,
			Name:        "docker",
			Version:     "1.0.0",
			Keywords:    []string{"docker", "container"},
			Description: "Docker provider",
			Provides: Capabilities{
				Providers:       []ProviderSpec{{Name: "docker", ConfigKey: "docker"}},
				Presets:         []string{"docker:default"},
				CapabilityPacks: []string{"docker-pack"},
			},
			Requires: []Dependency{{Name: "base", Version: ">=1.0.0"}},
			WASM: &WASMConfig{
				Module:   "plugin.wasm",
				Checksum: "abc123",
				Capabilities: []WASMCapability{
					{Name: "shell:execute", Justification: "Run docker"},
				},
			},
			Signature: &SignatureInfo{
				Type:  "ssh",
				KeyID: "key123",
				Data:  "signaturedata",
			},
		},
		Path:     "/path/to/plugin",
		Enabled:  true,
		LoadedAt: time.Now(),
	}

	clone := original.Clone()

	// Verify basic fields are copied
	assert.Equal(t, original.Manifest.Name, clone.Manifest.Name)
	assert.Equal(t, original.Path, clone.Path)
	assert.Equal(t, original.Enabled, clone.Enabled)

	// Modify original slices - clone should not be affected
	original.Manifest.Keywords = append(original.Manifest.Keywords, "modified")
	assert.NotEqual(t, len(original.Manifest.Keywords), len(clone.Manifest.Keywords))

	original.Manifest.Provides.Presets[0] = "modified"
	assert.NotEqual(t, original.Manifest.Provides.Presets[0], clone.Manifest.Provides.Presets[0])

	// Modify original WASM - clone should not be affected
	original.Manifest.WASM.Module = "modified.wasm"
	assert.NotEqual(t, original.Manifest.WASM.Module, clone.Manifest.WASM.Module)

	// Modify original Signature - clone should not be affected
	original.Manifest.Signature.Data = "modified"
	assert.NotEqual(t, original.Manifest.Signature.Data, clone.Manifest.Signature.Data)
}

func TestPlugin_Clone_Nil(t *testing.T) {
	t.Parallel()

	var p *Plugin
	assert.Nil(t, p.Clone())
}

func TestWASMConfig_Clone(t *testing.T) {
	t.Parallel()

	original := &WASMConfig{
		Module:   "plugin.wasm",
		Checksum: "abc123",
		Capabilities: []WASMCapability{
			{Name: "shell:execute", Justification: "Run commands"},
		},
	}

	clone := original.Clone()
	assert.Equal(t, original.Module, clone.Module)

	// Modify original capabilities
	original.Capabilities = append(original.Capabilities, WASMCapability{Name: "new"})
	assert.NotEqual(t, len(original.Capabilities), len(clone.Capabilities))
}

func TestWASMConfig_Clone_Nil(t *testing.T) {
	t.Parallel()

	var w *WASMConfig
	assert.Nil(t, w.Clone())
}

func TestSignatureInfo_Clone_Nil(t *testing.T) {
	t.Parallel()

	var s *SignatureInfo
	assert.Nil(t, s.Clone())
}

func TestCapabilities_Clone(t *testing.T) {
	t.Parallel()

	original := Capabilities{
		Providers:       []ProviderSpec{{Name: "test"}},
		Presets:         []string{"preset1"},
		CapabilityPacks: []string{"pack1"},
	}

	clone := original.Clone()

	original.Providers = append(original.Providers, ProviderSpec{Name: "new"})
	original.Presets[0] = "modified"

	assert.NotEqual(t, len(original.Providers), len(clone.Providers))
	assert.NotEqual(t, original.Presets[0], clone.Presets[0])
}

// Test Semver validation

func TestValidateSemver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{name: "valid basic", version: "1.0.0", wantErr: false},
		{name: "valid with v prefix", version: "v1.0.0", wantErr: false},
		{name: "valid with V prefix", version: "V1.0.0", wantErr: false},
		{name: "valid with prerelease", version: "1.0.0-alpha", wantErr: false},
		{name: "valid with prerelease dot", version: "1.0.0-alpha.1", wantErr: false},
		{name: "valid with build", version: "1.0.0+build.123", wantErr: false},
		{name: "valid with prerelease and build", version: "1.0.0-beta.2+build.456", wantErr: false},
		{name: "valid zero version", version: "0.0.0", wantErr: false},
		{name: "valid large numbers", version: "100.200.300", wantErr: false},
		{name: "empty string", version: "", wantErr: true},
		{name: "invalid format", version: "1.0", wantErr: true},
		{name: "invalid characters", version: "1.0.0.beta", wantErr: true},
		{name: "leading zeros major", version: "01.0.0", wantErr: true},
		{name: "leading zeros minor", version: "1.00.0", wantErr: true},
		{name: "leading zeros patch", version: "1.0.00", wantErr: true},
		{name: "negative number", version: "-1.0.0", wantErr: true},
		{name: "just text", version: "version", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSemver(tt.version)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test Checksum verification

func TestVerifyChecksum(t *testing.T) {
	t.Parallel()

	data := []byte("test plugin data")
	hash := sha256.Sum256(data)
	validChecksum := hex.EncodeToString(hash[:])

	t.Run("valid checksum lowercase", func(t *testing.T) {
		t.Parallel()
		err := VerifyChecksum(data, validChecksum)
		assert.NoError(t, err)
	})

	t.Run("valid checksum uppercase", func(t *testing.T) {
		t.Parallel()
		// Use correct empty data hash (uppercase)
		err := VerifyChecksum([]byte{}, "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855")
		assert.NoError(t, err)
	})

	t.Run("valid mixed case checksum", func(t *testing.T) {
		t.Parallel()
		// Mix of upper and lowercase should work
		emptyHash := sha256.Sum256([]byte{})
		mixedCase := strings.ToUpper(hex.EncodeToString(emptyHash[:]))[:32] +
			strings.ToLower(hex.EncodeToString(emptyHash[:]))[32:]
		err := VerifyChecksum([]byte{}, mixedCase)
		assert.NoError(t, err)
	})

	t.Run("empty data", func(t *testing.T) {
		t.Parallel()
		emptyHash := sha256.Sum256([]byte{})
		emptyChecksum := hex.EncodeToString(emptyHash[:])
		err := VerifyChecksum([]byte{}, emptyChecksum)
		assert.NoError(t, err)
	})

	t.Run("empty checksum", func(t *testing.T) {
		t.Parallel()
		err := VerifyChecksum(data, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("wrong checksum", func(t *testing.T) {
		t.Parallel()
		wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
		err := VerifyChecksum(data, wrongChecksum)
		assert.Error(t, err)
		var checksumErr *ChecksumError
		require.ErrorAs(t, err, &checksumErr)
		assert.Equal(t, checksumErr.Expected, wrongChecksum)
	})
}

func TestVerifyChecksum_FormatValidation(t *testing.T) {
	t.Parallel()

	data := []byte("test data")

	t.Run("too short checksum", func(t *testing.T) {
		t.Parallel()
		err := VerifyChecksum(data, "abc123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid checksum length")
		assert.Contains(t, err.Error(), "expected 64")
		assert.Contains(t, err.Error(), "got 6")
	})

	t.Run("too long checksum", func(t *testing.T) {
		t.Parallel()
		longChecksum := strings.Repeat("a", 128)
		err := VerifyChecksum(data, longChecksum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid checksum length")
		assert.Contains(t, err.Error(), "expected 64")
		assert.Contains(t, err.Error(), "got 128")
	})

	t.Run("invalid hex characters", func(t *testing.T) {
		t.Parallel()
		// 64 chars but contains 'g' which is not valid hex
		invalidChecksum := strings.Repeat("g", 64)
		err := VerifyChecksum(data, invalidChecksum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid checksum character")
	})

	t.Run("spaces in checksum", func(t *testing.T) {
		t.Parallel()
		// 64 chars with spaces
		invalidChecksum := strings.Repeat("a ", 32)[:64]
		err := VerifyChecksum(data, invalidChecksum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid checksum character")
	})

	t.Run("special characters", func(t *testing.T) {
		t.Parallel()
		// 64 chars with special characters
		invalidChecksum := strings.Repeat("a!", 32)
		err := VerifyChecksum(data, invalidChecksum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid checksum character")
	})

	t.Run("valid hex boundary characters", func(t *testing.T) {
		t.Parallel()
		// Test all valid hex chars
		validChars := "0123456789abcdefABCDEF0123456789abcdefABCDEF0123456789abcdefABCD"
		// This should pass format validation but fail matching
		err := VerifyChecksum(data, validChars)
		// Will get ChecksumError since it doesn't match
		assert.Error(t, err)
		assert.True(t, IsChecksumError(err))
	})
}

// Test concurrent registry access

func TestRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 3) // writers, readers, listers

	// Concurrent writers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				name := "plugin-" + string(rune('a'+id%26))
				p := &Plugin{Manifest: Manifest{Name: name, Version: "1.0.0"}}
				_ = r.Register(p) // May fail with duplicate, that's OK
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, _ = r.Get("plugin-a")
				_, _ = r.Get("nonexistent")
			}
		}()
	}

	// Concurrent listers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = r.List()
				_ = r.Enabled()
				_ = r.Count()
			}
		}()
	}

	wg.Wait()

	// Verify registry is still consistent
	assert.GreaterOrEqual(t, r.Count(), 1)
	assert.LessOrEqual(t, r.Count(), 26) // Max 26 unique plugins (a-z)
}

func TestRegistry_Get_ReturnsDeepCopy(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	original := &Plugin{
		Manifest: Manifest{
			Name:     "test",
			Keywords: []string{"original"},
		},
		Enabled: true,
	}
	_ = r.Register(original)

	// Get a copy
	copy1, ok := r.Get("test")
	require.True(t, ok)

	// Modify the copy
	copy1.Manifest.Keywords = append(copy1.Manifest.Keywords, "modified")
	copy1.Enabled = false

	// Get another copy - should not be affected
	copy2, ok := r.Get("test")
	require.True(t, ok)
	assert.Len(t, copy2.Manifest.Keywords, 1) // Still original length
	assert.True(t, copy2.Enabled)             // Still original value
}

func TestRegistry_List_ReturnsDeterministicOrder(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	// Register in random order
	names := []string{"zebra", "apple", "mango", "banana"}
	for _, name := range names {
		_ = r.Register(&Plugin{Manifest: Manifest{Name: name}})
	}

	// List should return sorted order
	list := r.List()
	assert.Equal(t, "apple", list[0].Manifest.Name)
	assert.Equal(t, "banana", list[1].Manifest.Name)
	assert.Equal(t, "mango", list[2].Manifest.Name)
	assert.Equal(t, "zebra", list[3].Manifest.Name)
}

// Tests for signature verification

func TestVerifySignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		manifest  *Manifest
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "nil manifest",
			manifest:  nil,
			wantErr:   true,
			errSubstr: "manifest cannot be nil",
		},
		{
			name:      "no signature",
			manifest:  &Manifest{Name: "test"},
			wantErr:   true,
			errSubstr: "no signature present",
		},
		{
			name: "empty signature type",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{KeyID: "abc", Data: "base64data"},
			},
			wantErr:   true,
			errSubstr: "signature type is required",
		},
		{
			name: "unsupported signature type",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "rsa", KeyID: "abc", Data: "base64data"},
			},
			wantErr:   true,
			errSubstr: "unsupported signature type",
		},
		{
			name: "missing key ID",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "ssh", Data: "base64data"},
			},
			wantErr:   true,
			errSubstr: "keyId is required",
		},
		{
			name: "missing signature data",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "ssh", KeyID: "abc"},
			},
			wantErr:   true,
			errSubstr: "signature data is required",
		},
		{
			name: "signature data too short",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "ssh", KeyID: "abc", Data: "ab"},
			},
			wantErr:   true,
			errSubstr: "signature data is too short",
		},
		{
			name: "ssh signature requires allowed_signers file",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "ssh", KeyID: "SHA256:abc123", Data: "dGVzdHNpZ25hdHVyZWRhdGE="}, // valid base64
			},
			wantErr:   true,
			errSubstr: "allowed_signers", // No allowed_signers file configured
		},
		{
			name: "gpg signature with invalid base64 fails",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "gpg", KeyID: "ABCD1234", Data: "not-valid-base64!"},
			},
			wantErr:   true,
			errSubstr: "invalid signature encoding",
		},
		{
			name: "sigstore signature verification fails",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "sigstore", KeyID: "sigstore-key", Data: "dGVzdHNpZ25hdHVyZWRhdGE="}, // valid base64
			},
			wantErr:   true,
			errSubstr: "Sigstore", // matches both "cosign not found...Sigstore" and "Sigstore signature verification failed"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := VerifySignature(tt.manifest, nil)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				assert.True(t, IsSignatureError(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestVerifySignatureStructure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		manifest  *Manifest
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "nil manifest",
			manifest:  nil,
			wantErr:   true,
			errSubstr: "manifest cannot be nil",
		},
		{
			name:      "nil signature",
			manifest:  &Manifest{Name: "test"},
			wantErr:   true,
			errSubstr: "no signature present",
		},
		{
			name: "empty signature type",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{KeyID: "abc", Data: "data"},
			},
			wantErr:   true,
			errSubstr: "type is required",
		},
		{
			name: "invalid signature type",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "invalid", KeyID: "abc", Data: "data"},
			},
			wantErr:   true,
			errSubstr: "unsupported signature type",
		},
		{
			name: "missing keyId",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "ssh", Data: "data"},
			},
			wantErr:   true,
			errSubstr: "keyId is required",
		},
		{
			name: "missing signature data",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "ssh", KeyID: "abc"},
			},
			wantErr:   true,
			errSubstr: "signature data is required",
		},
		{
			name: "signature data too short",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "ssh", KeyID: "abc", Data: "ab"},
			},
			wantErr:   true,
			errSubstr: "signature data is too short",
		},
		{
			name: "valid ssh signature structure",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "ssh", KeyID: "SHA256:abc123", Data: "base64signaturedata"},
			},
			wantErr: false,
		},
		{
			name: "valid gpg signature structure",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "gpg", KeyID: "ABCD1234", Data: "base64gpgdata"},
			},
			wantErr: false,
		},
		{
			name: "valid sigstore signature structure",
			manifest: &Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "sigstore", KeyID: "sigstore-key", Data: "base64sigstoredata"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := VerifySignatureStructure(tt.manifest)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestVerifySignatureWithConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil config uses default", func(t *testing.T) {
		t.Parallel()
		manifest := &Manifest{
			Name:      "test",
			Signature: &SignatureInfo{Type: "ssh", KeyID: "test@example.com", Data: "dGVzdA=="},
		}
		// Should fail because no allowed_signers file exists
		err := VerifySignatureWithConfig(manifest, []byte("test"), nil)
		require.Error(t, err)
		assert.True(t, IsSignatureError(err))
	})

	t.Run("trusted key ID bypasses verification", func(t *testing.T) {
		t.Parallel()
		manifest := &Manifest{
			Name:      "test",
			Signature: &SignatureInfo{Type: "ssh", KeyID: "trusted@example.com", Data: "dGVzdA=="},
		}
		config := &VerificationConfig{
			TrustedKeyIDs: []string{"trusted@example.com"},
		}
		// Should succeed because key ID is explicitly trusted
		err := VerifySignatureWithConfig(manifest, []byte("test"), config)
		require.NoError(t, err)
	})

	t.Run("untrusted key ID requires verification", func(t *testing.T) {
		t.Parallel()
		manifest := &Manifest{
			Name:      "test",
			Signature: &SignatureInfo{Type: "ssh", KeyID: "untrusted@example.com", Data: "dGVzdA=="},
		}
		config := &VerificationConfig{
			TrustedKeyIDs: []string{"other@example.com"},
		}
		// Should fail because key ID is not trusted and no allowed_signers
		err := VerifySignatureWithConfig(manifest, []byte("test"), config)
		require.Error(t, err)
	})

	t.Run("invalid structure fails before verification", func(t *testing.T) {
		t.Parallel()
		manifest := &Manifest{
			Name:      "test",
			Signature: &SignatureInfo{Type: "invalid", KeyID: "test", Data: "dGVzdA=="},
		}
		err := VerifySignatureWithConfig(manifest, []byte("test"), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported signature type")
	})
}

func TestDefaultVerificationConfig(t *testing.T) {
	t.Parallel()

	config := DefaultVerificationConfig()
	require.NotNil(t, config)
	assert.Contains(t, config.SSHAllowedSignersFile, "allowed_signers")
	assert.Empty(t, config.GPGKeyring)
	assert.Empty(t, config.SigstoreTrustedRoots)
}

func TestDetermineTrustLevelWithVerification(t *testing.T) {
	t.Parallel()

	t.Run("trusted key ID grants TrustVerified", func(t *testing.T) {
		t.Parallel()
		plugin := &Plugin{
			Manifest: Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "ssh", KeyID: "trusted@example.com", Data: "dGVzdGRhdGFoZXJl"},
			},
		}
		config := &VerificationConfig{
			TrustedKeyIDs: []string{"trusted@example.com"},
		}
		level := DetermineTrustLevelWithVerification(plugin, []byte("test"), config)
		assert.Equal(t, TrustVerified, level)
	})

	t.Run("no verification config still grants TrustCommunity for valid signature", func(t *testing.T) {
		t.Parallel()
		plugin := &Plugin{
			Manifest: Manifest{
				Name:      "test",
				Signature: &SignatureInfo{Type: "ssh", KeyID: "test@example.com", Data: "dGVzdGRhdGFoZXJl"},
			},
		}
		level := DetermineTrustLevelWithVerification(plugin, nil, nil)
		assert.Equal(t, TrustCommunity, level)
	})

	t.Run("nil plugin returns TrustUntrusted", func(t *testing.T) {
		t.Parallel()
		level := DetermineTrustLevelWithVerification(nil, nil, nil)
		assert.Equal(t, TrustUntrusted, level)
	})

	t.Run("builtin path returns TrustBuiltin", func(t *testing.T) {
		t.Parallel()
		plugin := &Plugin{Path: "builtin:core"}
		level := DetermineTrustLevelWithVerification(plugin, nil, nil)
		assert.Equal(t, TrustBuiltin, level)
	})
}

// Tests for trust level

func TestTrustLevelOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level    TrustLevel
		expected int
	}{
		{TrustBuiltin, 4},
		{TrustVerified, 3},
		{TrustCommunity, 2},
		{TrustUntrusted, 1},
		{TrustLevel("unknown"), 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, TrustLevelOrder(tt.level))
		})
	}
}

func TestDefaultTrustPolicy(t *testing.T) {
	t.Parallel()

	policy := DefaultTrustPolicy()
	assert.Equal(t, TrustCommunity, policy.MinLevel)
	assert.Nil(t, policy.AllowedLevels)
}

func TestStrictTrustPolicy(t *testing.T) {
	t.Parallel()

	policy := StrictTrustPolicy()
	assert.Equal(t, TrustVerified, policy.MinLevel)
	assert.Nil(t, policy.AllowedLevels)
}

func TestEnforceTrustLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		plugin  *Plugin
		policy  TrustPolicy
		wantErr bool
	}{
		{
			name:    "nil plugin",
			plugin:  nil,
			policy:  DefaultTrustPolicy(),
			wantErr: true,
		},
		{
			name: "builtin meets any requirement",
			plugin: &Plugin{
				Path:     "builtin:core",
				Manifest: Manifest{Name: "core"},
			},
			policy:  StrictTrustPolicy(),
			wantErr: false,
		},
		{
			name: "signed plugin does not meet strict requirement (no crypto verification yet)",
			plugin: &Plugin{
				Manifest: Manifest{
					Name: "test",
					Signature: &SignatureInfo{
						Type:  "ssh",
						KeyID: "abc",
						Data:  "base64data",
					},
				},
			},
			policy:  StrictTrustPolicy(),
			wantErr: true, // TrustVerified requires actual crypto verification, signed only gets TrustCommunity
		},
		{
			name: "community does not meet strict requirement",
			plugin: &Plugin{
				Manifest: Manifest{
					Name: "test",
					WASM: &WASMConfig{Checksum: "abc123"},
				},
			},
			policy:  StrictTrustPolicy(),
			wantErr: true,
		},
		{
			name: "community meets default requirement",
			plugin: &Plugin{
				Manifest: Manifest{
					Name:     "test",
					Provides: Capabilities{Presets: []string{"test"}},
				},
			},
			policy:  DefaultTrustPolicy(),
			wantErr: false,
		},
		{
			name: "explicit allowed levels - in list",
			plugin: &Plugin{
				Manifest: Manifest{
					Name:     "test",
					Provides: Capabilities{Presets: []string{"test"}},
				},
			},
			policy: TrustPolicy{
				AllowedLevels: []TrustLevel{TrustCommunity, TrustBuiltin},
			},
			wantErr: false,
		},
		{
			name: "explicit allowed levels - not in list",
			plugin: &Plugin{
				Manifest: Manifest{Name: "test"},
			},
			policy: TrustPolicy{
				AllowedLevels: []TrustLevel{TrustVerified},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := EnforceTrustLevel(tt.plugin, tt.policy)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDetermineTrustLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plugin   *Plugin
		expected TrustLevel
	}{
		{
			name:     "nil plugin",
			plugin:   nil,
			expected: TrustUntrusted,
		},
		{
			name:     "builtin path",
			plugin:   &Plugin{Path: "builtin:core"},
			expected: TrustBuiltin,
		},
		{
			name: "signed plugin structure valid (no crypto verification yet)",
			plugin: &Plugin{
				Manifest: Manifest{
					Signature: &SignatureInfo{
						Type:  "ssh",
						KeyID: "abc",
						Data:  "base64data",
					},
				},
			},
			expected: TrustCommunity, // TrustVerified requires actual crypto verification
		},
		{
			name: "wasm with checksum",
			plugin: &Plugin{
				Manifest: Manifest{
					WASM: &WASMConfig{Checksum: "abc123"},
				},
			},
			expected: TrustCommunity,
		},
		{
			name: "config with presets",
			plugin: &Plugin{
				Manifest: Manifest{
					Provides: Capabilities{Presets: []string{"test"}},
				},
			},
			expected: TrustCommunity,
		},
		{
			name: "config with capability packs",
			plugin: &Plugin{
				Manifest: Manifest{
					Provides: Capabilities{CapabilityPacks: []string{"test"}},
				},
			},
			expected: TrustCommunity,
		},
		{
			name:     "untrusted by default",
			plugin:   &Plugin{Manifest: Manifest{Name: "test"}},
			expected: TrustUntrusted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, DetermineTrustLevel(tt.plugin))
		})
	}
}

// Tests for WASM capability validation

func TestValidateCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		caps    []WASMCapability
		wantErr bool
	}{
		{
			name:    "nil caps",
			caps:    nil,
			wantErr: false,
		},
		{
			name:    "empty caps",
			caps:    []WASMCapability{},
			wantErr: false,
		},
		{
			name: "valid read-only capability",
			caps: []WASMCapability{
				{Name: "files:read", Justification: "need to read config"},
			},
			wantErr: false,
		},
		{
			name: "dangerous capability with justification",
			caps: []WASMCapability{
				{Name: "shell:execute", Justification: "need to run git commands"},
			},
			wantErr: false,
		},
		{
			name: "dangerous capability without justification",
			caps: []WASMCapability{
				{Name: "shell:execute"},
			},
			wantErr: true,
		},
		{
			name: "unrecognized capability",
			caps: []WASMCapability{
				{Name: "unknown:capability"},
			},
			wantErr: true,
		},
		{
			name: "multiple valid capabilities",
			caps: []WASMCapability{
				{Name: "files:read", Justification: "read config"},
				{Name: "env:read", Justification: "read env vars"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateCapabilities(tt.caps)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, IsCapabilityError(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDefaultValidator(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	require.NotNil(t, v)
	assert.NotNil(t, v.AllowedCaps)

	t.Run("validate manifest", func(t *testing.T) {
		t.Parallel()

		m := &Manifest{
			APIVersion: "v1",
			Name:       "test",
			Version:    "1.0.0",
			Provides:   Capabilities{Presets: []string{"test"}},
		}
		err := v.Validate(m)
		require.NoError(t, err)
	})

	t.Run("validate capabilities", func(t *testing.T) {
		t.Parallel()

		caps := []WASMCapability{
			{Name: "files:read", Justification: "need to read"},
		}
		err := v.ValidateCapabilities(caps)
		require.NoError(t, err)
	})

	t.Run("validate invalid capability", func(t *testing.T) {
		t.Parallel()

		caps := []WASMCapability{
			{Name: "invalid:cap"},
		}
		err := v.ValidateCapabilities(caps)
		require.Error(t, err)
	})
}

// Tests for plugin name format validation

func TestValidatePluginNameFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple name", "docker", false},
		{"valid with hyphen", "my-plugin", false},
		{"valid with underscore", "my_plugin", false},
		{"valid with numbers", "plugin123", false},
		{"too short", "a", true},
		{"too long", "a" + strings.Repeat("b", 64), true},
		{"starts with number", "123plugin", true},
		{"starts with hyphen", "-plugin", true},
		{"contains space", "my plugin", true},
		{"contains dot", "my.plugin", true},
		{"contains slash", "my/plugin", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validatePluginNameFormat(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Tests for TrustError

func TestTrustError(t *testing.T) {
	t.Parallel()

	err := &TrustError{
		PluginName: "dangerous-plugin",
		Level:      TrustUntrusted,
		Required:   TrustVerified,
		Reason:     "plugin signature not verified",
	}

	assert.Contains(t, err.Error(), "dangerous-plugin")
	assert.Contains(t, err.Error(), "untrusted")
	assert.Contains(t, err.Error(), "verified")
}

func TestIsTrustError(t *testing.T) {
	t.Parallel()

	t.Run("returns true for TrustError", func(t *testing.T) {
		t.Parallel()
		err := &TrustError{PluginName: "test", Level: TrustUntrusted, Required: TrustVerified}
		assert.True(t, IsTrustError(err))
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		t.Parallel()
		err := &ValidationError{Errors: []string{"test error"}}
		assert.False(t, IsTrustError(err))
	})
}

// Tests for ManifestSizeError

func TestManifestSizeError(t *testing.T) {
	t.Parallel()

	err := &ManifestSizeError{
		Size:  512 * 1024,
		Limit: 256 * 1024,
	}

	errMsg := err.Error()
	assert.Contains(t, errMsg, "524288") // 512 * 1024 bytes
	assert.Contains(t, errMsg, "262144") // 256 * 1024 bytes
}

func TestIsManifestSizeError(t *testing.T) {
	t.Parallel()

	t.Run("returns true for ManifestSizeError", func(t *testing.T) {
		t.Parallel()
		err := &ManifestSizeError{Size: 1024, Limit: 512}
		assert.True(t, IsManifestSizeError(err))
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		t.Parallel()
		err := &ValidationError{Errors: []string{"test error"}}
		assert.False(t, IsManifestSizeError(err))
	})
}

// Tests for sanitizeAPIError

func TestSanitizeAPIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantMsg    string
	}{
		{"unauthorized", 401, "authentication required"},
		{"forbidden", 403, "rate limit exceeded"},
		{"not found", 404, "not found"},
		{"unprocessable entity", 422, "validation failed"},
		{"service unavailable", 503, "temporarily unavailable"},
		{"rate limited (default)", 429, "status 429"},
		{"server error (default)", 500, "status 500"},
		{"unknown status", 418, "status 418"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := sanitizeAPIError(tt.statusCode, nil)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

// Tests for CreateInstallPlan

func TestCreateInstallPlan(t *testing.T) {
	t.Parallel()

	t.Run("creates plan from manifest", func(t *testing.T) {
		t.Parallel()

		manifest := &Manifest{
			Name:        "docker-provider",
			Version:     "1.0.0",
			Description: "Docker provider for preflight",
			Author:      "Test Author",
			Type:        TypeProvider,
			WASM: &WASMConfig{
				Module: "provider.wasm",
				Capabilities: []WASMCapability{
					{Name: "fs_read", Justification: "Read docker config"},
				},
			},
			Provides: Capabilities{
				Providers: []ProviderSpec{
					{Name: "docker", ConfigKey: "docker"},
				},
			},
		}

		plan := CreateInstallPlan(manifest, "github.com/example/docker-plugin")

		assert.Equal(t, "github.com/example/docker-plugin", plan.Source)
		assert.Equal(t, manifest, plan.Plugin)
		assert.Len(t, plan.Capabilities, 1)
		assert.Contains(t, plan.Actions, "Install plugin docker-provider@1.0.0")
	})

	t.Run("with nil manifest returns nil", func(t *testing.T) {
		t.Parallel()
		plan := CreateInstallPlan(nil, "source")
		assert.Nil(t, plan)
	})

	t.Run("adds warning for dangerous capabilities", func(t *testing.T) {
		t.Parallel()

		manifest := &Manifest{
			Name:    "risky-plugin",
			Version: "1.0.0",
			Type:    TypeProvider,
			WASM: &WASMConfig{
				Module: "provider.wasm",
				Capabilities: []WASMCapability{
					{Name: "shell:execute", Justification: "Run shell commands"},
				},
			},
			Provides: Capabilities{
				Providers: []ProviderSpec{
					{Name: "risky", ConfigKey: "risky"},
				},
			},
		}

		plan := CreateInstallPlan(manifest, "source")

		found := false
		for _, w := range plan.Warnings {
			if strings.Contains(w, "shell:execute") {
				found = true
				break
			}
		}
		assert.True(t, found, "should warn about shell:execute capability")
	})

	t.Run("signed plugin gets TrustVerified", func(t *testing.T) {
		t.Parallel()

		manifest := &Manifest{
			Name:      "signed-plugin",
			Version:   "1.0.0",
			Type:      TypeConfig,
			Signature: &SignatureInfo{Type: "gpg", KeyID: "ABC123", Data: "dGVzdA==dGVzdA=="},
		}

		plan := CreateInstallPlan(manifest, "source")
		assert.Equal(t, TrustVerified, plan.TrustLevel)
		assert.Contains(t, plan.Actions, "Verify plugin signature")
	})

	t.Run("unsigned plugin gets warning", func(t *testing.T) {
		t.Parallel()

		manifest := &Manifest{
			Name:    "unsigned-plugin",
			Version: "1.0.0",
			Type:    TypeConfig,
		}

		plan := CreateInstallPlan(manifest, "source")
		assert.Equal(t, TrustCommunity, plan.TrustLevel)
		assert.Contains(t, plan.Warnings, "Plugin is not signed - verify source manually")
	})

	t.Run("with dependencies with version", func(t *testing.T) {
		t.Parallel()

		manifest := &Manifest{
			Name:    "dep-plugin",
			Version: "1.0.0",
			Type:    TypeConfig,
			Requires: []Dependency{
				{Name: "core-plugin", Version: "2.0.0"},
			},
		}

		plan := CreateInstallPlan(manifest, "source")
		found := false
		for _, action := range plan.Actions {
			if strings.Contains(action, "core-plugin@2.0.0") {
				found = true
				break
			}
		}
		assert.True(t, found, "should include versioned dependency")
	})

	t.Run("with dependencies without version", func(t *testing.T) {
		t.Parallel()

		manifest := &Manifest{
			Name:    "dep-plugin",
			Version: "1.0.0",
			Type:    TypeConfig,
			Requires: []Dependency{
				{Name: "other-plugin"},
			},
		}

		plan := CreateInstallPlan(manifest, "source")
		found := false
		for _, action := range plan.Actions {
			if strings.Contains(action, "other-plugin") && !strings.Contains(action, "@") {
				found = true
				break
			}
		}
		assert.True(t, found, "should include unversioned dependency")
	})

	t.Run("with presets and capability packs", func(t *testing.T) {
		t.Parallel()

		manifest := &Manifest{
			Name:    "config-plugin",
			Version: "1.0.0",
			Type:    TypeConfig,
			Provides: Capabilities{
				Presets:         []string{"preset1", "preset2"},
				CapabilityPacks: []string{"pack1"},
			},
		}

		plan := CreateInstallPlan(manifest, "source")
		foundPresets := false
		foundPacks := false
		for _, action := range plan.Actions {
			if strings.Contains(action, "2 preset") {
				foundPresets = true
			}
			if strings.Contains(action, "1 capability pack") {
				foundPacks = true
			}
		}
		assert.True(t, foundPresets, "should register presets")
		assert.True(t, foundPacks, "should register capability packs")
	})
}

// Tests for FormatInstallPlan

func TestFormatInstallPlan(t *testing.T) {
	t.Parallel()

	t.Run("formats plan with all sections", func(t *testing.T) {
		t.Parallel()

		plan := &InstallPlan{
			Source: "github.com/example/plugin",
			Plugin: &Manifest{
				Name:        "example-plugin",
				Version:     "1.0.0",
				Description: "An example plugin",
				Author:      "Test Author",
			},
			TrustLevel: TrustCommunity,
			Capabilities: []WASMCapability{
				{Name: "fs_read", Justification: "Read config files"},
			},
			Warnings: []string{"Uses network capability"},
			Actions:  []string{"Install plugin 'example-plugin' v1.0.0"},
		}

		output := FormatInstallPlan(plan)

		assert.Contains(t, output, "example-plugin")
		assert.Contains(t, output, "1.0.0")
		assert.Contains(t, output, "An example plugin")
		assert.Contains(t, output, "community")
		assert.Contains(t, output, "fs_read")
		assert.Contains(t, output, "Uses network capability")
	})

	t.Run("nil plan returns empty string", func(t *testing.T) {
		t.Parallel()
		output := FormatInstallPlan(nil)
		assert.Empty(t, output)
	})
}
