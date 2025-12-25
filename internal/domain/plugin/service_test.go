package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDiscoverer is a test double for PluginDiscoverer.
type mockDiscoverer struct {
	plugins []*Plugin
	errors  []DiscoveryError
	err     error
}

func (m *mockDiscoverer) Discover(_ context.Context) (*DiscoveryResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &DiscoveryResult{
		Plugins: m.plugins,
		Errors:  m.errors,
	}, nil
}

func (m *mockDiscoverer) LoadFromPath(_ string) (*Plugin, error) {
	return nil, ErrManifestNotFound
}

// mockSearcher is a test double for PluginSearcher.
type mockSearcher struct {
	results []SearchResult
	err     error
}

func (m *mockSearcher) Search(_ context.Context, _ SearchOptions) ([]SearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func TestNewService(t *testing.T) {
	t.Parallel()

	t.Run("default configuration", func(t *testing.T) {
		t.Parallel()
		s := NewService()
		assert.NotNil(t, s.Registry())
		assert.NotNil(t, s.discoverer)
		assert.NotNil(t, s.searcher)
	})

	t.Run("with custom discoverer", func(t *testing.T) {
		t.Parallel()
		mock := &mockDiscoverer{}
		s := NewService(WithDiscoverer(mock))
		assert.Equal(t, mock, s.discoverer)
	})

	t.Run("with custom searcher", func(t *testing.T) {
		t.Parallel()
		mock := &mockSearcher{}
		s := NewService(WithSearcher(mock))
		assert.Equal(t, mock, s.searcher)
	})
}

func TestDefaultService_Discover(t *testing.T) {
	t.Parallel()

	t.Run("discovers and registers plugins", func(t *testing.T) {
		t.Parallel()

		plugins := []*Plugin{
			{Manifest: Manifest{Name: "plugin-a", Version: "1.0.0"}},
			{Manifest: Manifest{Name: "plugin-b", Version: "2.0.0"}},
		}

		s := NewService(WithDiscoverer(&mockDiscoverer{plugins: plugins}))

		err := s.Discover(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 2, s.Registry().Count())

		p, ok := s.Get("plugin-a")
		assert.True(t, ok)
		assert.Equal(t, "1.0.0", p.Manifest.Version)
	})

	t.Run("skips duplicate plugins", func(t *testing.T) {
		t.Parallel()

		plugins := []*Plugin{
			{Manifest: Manifest{Name: "duplicate", Version: "1.0.0"}},
			{Manifest: Manifest{Name: "duplicate", Version: "2.0.0"}},
		}

		s := NewService(WithDiscoverer(&mockDiscoverer{plugins: plugins}))

		err := s.Discover(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 1, s.Registry().Count())
	})
}

func TestDefaultService_Search(t *testing.T) {
	t.Parallel()

	t.Run("returns search results", func(t *testing.T) {
		t.Parallel()

		results := []SearchResult{
			{Name: "docker", FullName: "example/docker"},
			{Name: "kubernetes", FullName: "example/kubernetes"},
		}

		s := NewService(WithSearcher(&mockSearcher{results: results}))

		got, err := s.Search(context.Background(), SearchOptions{Query: "test"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("returns error when no searcher", func(t *testing.T) {
		t.Parallel()

		s := &DefaultService{registry: NewRegistry()}

		_, err := s.Search(context.Background(), SearchOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no searcher configured")
	})
}

func TestDefaultService_List(t *testing.T) {
	t.Parallel()

	plugins := []*Plugin{
		{Manifest: Manifest{Name: "a"}},
		{Manifest: Manifest{Name: "b"}},
	}

	s := NewService(WithDiscoverer(&mockDiscoverer{plugins: plugins}))
	_ = s.Discover(context.Background())

	list := s.List()
	assert.Len(t, list, 2)
}

func TestDefaultService_Get(t *testing.T) {
	t.Parallel()

	plugins := []*Plugin{
		{Manifest: Manifest{Name: "docker", Version: "1.0.0"}},
	}

	s := NewService(WithDiscoverer(&mockDiscoverer{plugins: plugins}))
	_ = s.Discover(context.Background())

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		p, ok := s.Get("docker")
		assert.True(t, ok)
		assert.Equal(t, "docker", p.Manifest.Name)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		_, ok := s.Get("nonexistent")
		assert.False(t, ok)
	})
}

// mockInstaller is a test double for Installer.
type mockInstaller struct {
	installPlugin *Plugin
	installErr    error
	uninstallErr  error
	uninstalled   string
}

func (m *mockInstaller) Install(_ context.Context, _ string) (*Plugin, error) {
	if m.installErr != nil {
		return nil, m.installErr
	}
	return m.installPlugin, nil
}

func (m *mockInstaller) Uninstall(name string) error {
	m.uninstalled = name
	return m.uninstallErr
}

func TestDefaultService_Install(t *testing.T) {
	t.Parallel()

	t.Run("installs and registers plugin", func(t *testing.T) {
		t.Parallel()

		plugin := &Plugin{Manifest: Manifest{Name: "test-plugin", Version: "1.0.0"}}
		installer := &mockInstaller{installPlugin: plugin}
		s := NewService(WithInstaller(installer))

		result, err := s.Install(context.Background(), "https://github.com/example/plugin")
		require.NoError(t, err)
		assert.Equal(t, "test-plugin", result.Manifest.Name)
		assert.Equal(t, 1, s.Registry().Count())
	})

	t.Run("returns error when no installer", func(t *testing.T) {
		t.Parallel()

		s := NewService()

		_, err := s.Install(context.Background(), "https://github.com/example/plugin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no installer configured")
	})

	t.Run("returns error when install fails", func(t *testing.T) {
		t.Parallel()

		installer := &mockInstaller{installErr: assert.AnError}
		s := NewService(WithInstaller(installer))

		_, err := s.Install(context.Background(), "https://github.com/example/plugin")
		assert.Error(t, err)
	})
}

func TestDefaultService_Uninstall(t *testing.T) {
	t.Parallel()

	t.Run("uninstalls and removes from registry", func(t *testing.T) {
		t.Parallel()

		plugin := &Plugin{Manifest: Manifest{Name: "test-plugin", Version: "1.0.0"}}
		installer := &mockInstaller{}
		s := NewService(WithInstaller(installer))

		// Pre-register the plugin
		err := s.Registry().Register(plugin)
		require.NoError(t, err)
		assert.Equal(t, 1, s.Registry().Count())

		err = s.Uninstall("test-plugin")
		require.NoError(t, err)
		assert.Equal(t, "test-plugin", installer.uninstalled)
		assert.Equal(t, 0, s.Registry().Count())
	})

	t.Run("returns error when no installer", func(t *testing.T) {
		t.Parallel()

		s := NewService()

		err := s.Uninstall("test-plugin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no installer configured")
	})

	t.Run("returns error when uninstall fails", func(t *testing.T) {
		t.Parallel()

		installer := &mockInstaller{uninstallErr: assert.AnError}
		s := NewService(WithInstaller(installer))

		err := s.Uninstall("test-plugin")
		assert.Error(t, err)
	})
}

func TestDefaultService_Discover_Errors(t *testing.T) {
	t.Parallel()

	t.Run("returns error when no discoverer", func(t *testing.T) {
		t.Parallel()

		s := &DefaultService{registry: NewRegistry()}

		err := s.Discover(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no discoverer configured")
	})

	t.Run("returns error when discover fails", func(t *testing.T) {
		t.Parallel()

		discoverer := &mockDiscoverer{err: assert.AnError}
		s := NewService(WithDiscoverer(discoverer))

		err := s.Discover(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "discovering plugins")
	})
}

func TestWithInstaller(t *testing.T) {
	t.Parallel()

	installer := &mockInstaller{}
	s := NewService(WithInstaller(installer))

	assert.Equal(t, installer, s.installer)
}

// Context cancellation tests

func TestDefaultService_Discover_ContextCancelled(t *testing.T) {
	t.Parallel()

	plugins := []*Plugin{
		{Manifest: Manifest{Name: "plugin-a", Version: "1.0.0"}},
		{Manifest: Manifest{Name: "plugin-b", Version: "2.0.0"}},
	}

	s := NewService(WithDiscoverer(&mockDiscoverer{plugins: plugins}))

	// Create an already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := s.Discover(ctx)
	// The mock immediately returns plugins, so context cancellation
	// happens after discovery but during registration
	// Either passes or returns context.Canceled
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
	}
}

func TestDefaultService_Discover_WithValidContext(t *testing.T) {
	t.Parallel()

	plugins := []*Plugin{
		{Manifest: Manifest{Name: "plugin-a", Version: "1.0.0"}},
	}

	s := NewService(WithDiscoverer(&mockDiscoverer{plugins: plugins}))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := s.Discover(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, s.Registry().Count())
}

// contextCancellingDiscoverer simulates a slow discoverer that respects context
type contextCancellingDiscoverer struct {
	delay time.Duration
}

func (d *contextCancellingDiscoverer) Discover(ctx context.Context) (*DiscoveryResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(d.delay):
		return &DiscoveryResult{
			Plugins: []*Plugin{{Manifest: Manifest{Name: "test", Version: "1.0.0"}}},
		}, nil
	}
}

func (d *contextCancellingDiscoverer) LoadFromPath(_ string) (*Plugin, error) {
	return nil, ErrManifestNotFound
}

func TestDefaultService_Discover_ContextTimeout(t *testing.T) {
	t.Parallel()

	// Create a discoverer that takes 1 second
	discoverer := &contextCancellingDiscoverer{delay: 1 * time.Second}
	s := NewService(WithDiscoverer(discoverer))

	// Create a context that times out in 10ms
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := s.Discover(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
