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

func TestService_Update(t *testing.T) {
	t.Parallel()

	// Create tar.gz packages for v1.0.0 and v2.0.0
	createPackage := func(version string) ([]byte, string) {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gw)

		content := []byte("version: " + version)
		hdr := &tar.Header{
			Name: "config.yaml",
			Mode: 0o644,
			Size: int64(len(content)),
		}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write(content)
		_ = tw.Close()
		_ = gw.Close()

		data := buf.Bytes()
		return data, ComputeChecksum(data)
	}

	pkg1, checksum1 := createPackage("1.0.0")
	pkg2, checksum2 := createPackage("2.0.0")

	indexData := `{
		"version": "1",
		"packages": [
			{
				"id": "update-test",
				"type": "preset",
				"title": "Update Test",
				"versions": [
					{"version": "2.0.0", "checksum": "` + checksum2 + `"},
					{"version": "1.0.0", "checksum": "` + checksum1 + `"}
				]
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/index.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(indexData))
		case "/v1/packages/update-test/1.0.0.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(pkg1)
		case "/v1/packages/update-test/2.0.0.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(pkg2)
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

	// Install v1.0.0
	_, err := service.Install(context.Background(), MustNewPackageID("update-test"), "1.0.0")
	require.NoError(t, err)

	// Update to latest (v2.0.0)
	updated, err := service.Update(context.Background(), MustNewPackageID("update-test"))
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", updated.Version)

	// Update again should return current version (already up to date)
	result, err := service.Update(context.Background(), MustNewPackageID("update-test"))
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", result.Version)
}

func TestService_Update_NotInstalled(t *testing.T) {
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

	// Update non-installed package should fail
	_, err := service.Update(context.Background(), MustNewPackageID("not-installed"))
	assert.ErrorIs(t, err, ErrNotInstalled)
}

func TestService_UpdateAll(t *testing.T) {
	t.Parallel()

	// Create packages
	createPackage := func(name, version string) ([]byte, string) {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gw)

		content := []byte(name + ": " + version)
		hdr := &tar.Header{
			Name: "config.yaml",
			Mode: 0o644,
			Size: int64(len(content)),
		}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write(content)
		_ = tw.Close()
		_ = gw.Close()

		data := buf.Bytes()
		return data, ComputeChecksum(data)
	}

	pkg1v1, checksum1v1 := createPackage("pkg1", "1.0.0")
	pkg1v2, checksum1v2 := createPackage("pkg1", "2.0.0")
	pkg2v1, checksum2v1 := createPackage("pkg2", "1.0.0")

	indexData := `{
		"version": "1",
		"packages": [
			{
				"id": "pkg1",
				"type": "preset",
				"title": "Package 1",
				"versions": [
					{"version": "2.0.0", "checksum": "` + checksum1v2 + `"},
					{"version": "1.0.0", "checksum": "` + checksum1v1 + `"}
				]
			},
			{
				"id": "pkg2",
				"type": "preset",
				"title": "Package 2",
				"versions": [
					{"version": "1.0.0", "checksum": "` + checksum2v1 + `"}
				]
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/index.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(indexData))
		case "/v1/packages/pkg1/1.0.0.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(pkg1v1)
		case "/v1/packages/pkg1/2.0.0.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(pkg1v2)
		case "/v1/packages/pkg2/1.0.0.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(pkg2v1)
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

	// Install pkg1 v1.0.0 and pkg2 v1.0.0
	_, err := service.Install(context.Background(), MustNewPackageID("pkg1"), "1.0.0")
	require.NoError(t, err)
	_, err = service.Install(context.Background(), MustNewPackageID("pkg2"), "1.0.0")
	require.NoError(t, err)

	// Update all - should update pkg1 to 2.0.0
	updated, err := service.UpdateAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, updated, 1)
	assert.Equal(t, "pkg1", updated[0].Package.ID.String())
	assert.Equal(t, "2.0.0", updated[0].Version)
}

func TestService_CheckUpdates(t *testing.T) {
	t.Parallel()

	createPackage := func(version string) ([]byte, string) {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gw)

		content := []byte("v" + version)
		hdr := &tar.Header{
			Name: "config.yaml",
			Mode: 0o644,
			Size: int64(len(content)),
		}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write(content)
		_ = tw.Close()
		_ = gw.Close()

		data := buf.Bytes()
		return data, ComputeChecksum(data)
	}

	pkg1, checksum1 := createPackage("1.0.0")
	_, checksum2 := createPackage("2.0.0")

	indexData := `{
		"version": "1",
		"packages": [
			{
				"id": "check-update-test",
				"type": "preset",
				"title": "Check Update Test",
				"versions": [
					{"version": "2.0.0", "checksum": "` + checksum2 + `", "changelog": "New features!"},
					{"version": "1.0.0", "checksum": "` + checksum1 + `"}
				]
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/index.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(indexData))
		case "/v1/packages/check-update-test/1.0.0.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(pkg1)
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

	// Install v1.0.0
	_, err := service.Install(context.Background(), MustNewPackageID("check-update-test"), "1.0.0")
	require.NoError(t, err)

	// Check for updates
	updates, err := service.CheckUpdates(context.Background())
	require.NoError(t, err)
	assert.Len(t, updates, 1)
	assert.Equal(t, "check-update-test", updates[0].Package.ID.String())
	assert.Equal(t, "1.0.0", updates[0].CurrentVersion)
	assert.Equal(t, "2.0.0", updates[0].LatestVersion)
	assert.Equal(t, "New features!", updates[0].Changelog)
}

func TestService_CheckUpdates_Empty(t *testing.T) {
	t.Parallel()

	// Mock server with empty index
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/index.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"packages":[]}`))
			return
		}
		http.NotFound(w, r)
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
			Timeout:     5 * time.Second,
		},
	}

	service := NewService(config)

	// No installed packages - should return empty
	updates, err := service.CheckUpdates(context.Background())
	require.NoError(t, err)
	assert.Empty(t, updates)
}
