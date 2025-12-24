package marketplace

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	t.Parallel()

	config := DefaultServiceConfig()
	service := NewService(config)
	assert.NotNil(t, service)
}

func TestDefaultServiceConfig(t *testing.T) {
	t.Parallel()

	config := DefaultServiceConfig()
	assert.NotEmpty(t, config.InstallPath)
	assert.False(t, config.OfflineMode)
}

func TestService_Search(t *testing.T) {
	t.Parallel()

	// Create mock server
	indexData := `{
		"version": "1",
		"packages": [
			{
				"id": "nvim-config",
				"type": "preset",
				"title": "Neovim Config",
				"versions": [{"version": "1.0.0"}]
			},
			{
				"id": "vscode-config",
				"type": "preset",
				"title": "VS Code Config",
				"versions": [{"version": "1.0.0"}]
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/index.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(indexData))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: server.URL,
			Timeout:     10 * time.Second,
			UserAgent:   "test/1.0",
		},
		OfflineMode: false,
	}

	service := NewService(config)

	// Search for nvim
	results, err := service.Search(context.Background(), "nvim")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "nvim-config", results[0].ID.String())

	// Search for all
	results, err = service.Search(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestService_SearchByType(t *testing.T) {
	t.Parallel()

	indexData := `{
		"version": "1",
		"packages": [
			{
				"id": "my-preset",
				"type": "preset",
				"title": "My Preset",
				"versions": [{"version": "1.0.0"}]
			},
			{
				"id": "my-pack",
				"type": "capability-pack",
				"title": "My Pack",
				"versions": [{"version": "1.0.0"}]
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/index.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(indexData))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: server.URL,
			Timeout:     10 * time.Second,
		},
	}

	service := NewService(config)

	// Search for presets
	results, err := service.SearchByType(context.Background(), PackageTypePreset)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "my-preset", results[0].ID.String())

	// Search for capability packs
	results, err = service.SearchByType(context.Background(), PackageTypeCapabilityPack)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "my-pack", results[0].ID.String())
}

func TestService_Get(t *testing.T) {
	t.Parallel()

	indexData := `{
		"version": "1",
		"packages": [
			{
				"id": "my-package",
				"type": "preset",
				"title": "My Package",
				"description": "A test package",
				"versions": [{"version": "1.0.0"}]
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/index.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(indexData))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: server.URL,
			Timeout:     10 * time.Second,
		},
	}

	service := NewService(config)

	// Get existing package
	pkg, err := service.Get(context.Background(), MustNewPackageID("my-package"))
	require.NoError(t, err)
	assert.Equal(t, "My Package", pkg.Title)
	assert.Equal(t, "A test package", pkg.Description)

	// Get non-existing package
	_, err = service.Get(context.Background(), MustNewPackageID("not-found"))
	assert.ErrorIs(t, err, ErrPackageNotFound)
}

func TestService_Install(t *testing.T) {
	t.Parallel()

	// Create a tar.gz package
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add a file
	content := []byte("test file content")
	hdr := &tar.Header{
		Name: "config.yaml",
		Mode: 0o644,
		Size: int64(len(content)),
	}
	_ = tw.WriteHeader(hdr)
	_, _ = tw.Write(content)
	_ = tw.Close()
	_ = gw.Close()

	packageData := buf.Bytes()
	checksum := ComputeChecksum(packageData)

	indexData := `{
		"version": "1",
		"packages": [
			{
				"id": "install-test",
				"type": "preset",
				"title": "Install Test",
				"versions": [{"version": "1.0.0", "checksum": "` + checksum + `"}]
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/index.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(indexData))
		case "/v1/packages/install-test/1.0.0.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(packageData)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: server.URL,
			Timeout:     10 * time.Second,
		},
	}

	service := NewService(config)

	// Install package
	installed, err := service.Install(context.Background(), MustNewPackageID("install-test"), "latest")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", installed.Version)
	assert.Equal(t, "Install Test", installed.Package.Title)

	// Try to install again - should fail
	_, err = service.Install(context.Background(), MustNewPackageID("install-test"), "1.0.0")
	assert.ErrorIs(t, err, ErrAlreadyInstalled)
}

func TestService_List(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: DefaultClientConfig(),
	}

	service := NewService(config)

	// Initially empty
	installed, err := service.List()
	require.NoError(t, err)
	assert.Empty(t, installed)
}

func TestService_Statistics(t *testing.T) {
	t.Parallel()

	indexData := `{
		"version": "1",
		"packages": [
			{
				"id": "pkg1",
				"type": "preset",
				"title": "Preset 1",
				"downloads": 100,
				"versions": [{"version": "1.0.0"}]
			},
			{
				"id": "pkg2",
				"type": "capability-pack",
				"title": "Pack 1",
				"downloads": 50,
				"versions": [{"version": "1.0.0"}]
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/index.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(indexData))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: server.URL,
			Timeout:     10 * time.Second,
		},
	}

	service := NewService(config)

	stats, err := service.Statistics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalPackages)
	assert.Equal(t, 1, stats.Presets)
	assert.Equal(t, 1, stats.CapabilityPacks)
	assert.Equal(t, 150, stats.TotalDownloads)
}

func TestService_OfflineMode(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: "http://unreachable.invalid",
			Timeout:     1 * time.Second,
		},
		OfflineMode: true,
	}

	service := NewService(config)

	// No cached index - should fail
	_, err := service.Search(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "offline mode")

	// RefreshIndex should fail in offline mode
	err = service.RefreshIndex(context.Background())
	assert.ErrorIs(t, err, ErrNetworkError)
}

func TestService_RefreshIndex(t *testing.T) {
	t.Parallel()

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path == "/v1/index.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version": "1", "packages": []}`))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: server.URL,
			Timeout:     10 * time.Second,
		},
	}

	service := NewService(config)

	// First fetch
	err := service.RefreshIndex(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, calls)

	// Refresh again
	err = service.RefreshIndex(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestService_Uninstall(t *testing.T) {
	t.Parallel()

	// Create a tar.gz package
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("test content")
	hdr := &tar.Header{
		Name: "test.txt",
		Mode: 0o644,
		Size: int64(len(content)),
	}
	_ = tw.WriteHeader(hdr)
	_, _ = tw.Write(content)
	_ = tw.Close()
	_ = gw.Close()

	packageData := buf.Bytes()
	checksum := ComputeChecksum(packageData)

	indexData := `{
		"version": "1",
		"packages": [
			{
				"id": "uninstall-test",
				"type": "preset",
				"title": "Uninstall Test",
				"versions": [{"version": "1.0.0", "checksum": "` + checksum + `"}]
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/index.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(indexData))
		case "/v1/packages/uninstall-test/1.0.0.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(packageData)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: server.URL,
			Timeout:     10 * time.Second,
		},
	}

	service := NewService(config)

	// Install first
	_, err := service.Install(context.Background(), MustNewPackageID("uninstall-test"), "latest")
	require.NoError(t, err)

	// Verify installed
	installed, err := service.List()
	require.NoError(t, err)
	assert.Len(t, installed, 1)

	// Uninstall
	err = service.Uninstall(MustNewPackageID("uninstall-test"))
	require.NoError(t, err)

	// Verify uninstalled
	installed, err = service.List()
	require.NoError(t, err)
	assert.Empty(t, installed)

	// Try to uninstall again - should fail
	err = service.Uninstall(MustNewPackageID("uninstall-test"))
	assert.ErrorIs(t, err, ErrNotInstalled)
}
