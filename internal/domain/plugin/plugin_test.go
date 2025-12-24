package plugin

import (
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

func TestValidateManifest_Valid(t *testing.T) {
	m := &Manifest{
		APIVersion:  "v1",
		Name:        "docker",
		Version:     "1.0.0",
		Description: "Docker provider for Preflight",
		Author:      "Example Author",
		License:     "MIT",
		Provides: Capabilities{
			Providers: []ProviderSpec{
				{Name: "docker", ConfigKey: "docker"},
			},
		},
	}

	err := ValidateManifest(m)
	assert.NoError(t, err)
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
		Name:       "docker",
		Version:    "1.0.0",
		Provides: Capabilities{
			Providers: []ProviderSpec{
				{ConfigKey: "docker"},
			},
		},
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider 0: name is required")
}

func TestValidateManifest_ProviderMissingConfigKey(t *testing.T) {
	m := &Manifest{
		APIVersion: "v1",
		Name:       "docker",
		Version:    "1.0.0",
		Provides: Capabilities{
			Providers: []ProviderSpec{
				{Name: "docker"},
			},
		},
	}

	err := ValidateManifest(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configKey is required")
}

func TestManifest_FullExample(t *testing.T) {
	m := Manifest{
		APIVersion:          "v1",
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
	}

	err := ValidateManifest(&m)
	assert.NoError(t, err)

	assert.Equal(t, "kubernetes", m.Name)
	assert.Len(t, m.Provides.Providers, 2)
	assert.Len(t, m.Provides.Presets, 2)
	assert.Len(t, m.Requires, 1)
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
