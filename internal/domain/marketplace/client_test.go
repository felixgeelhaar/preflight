package marketplace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	config := DefaultClientConfig()
	client := NewClient(config)
	assert.NotNil(t, client)
}

func TestDefaultClientConfig(t *testing.T) {
	t.Parallel()

	config := DefaultClientConfig()
	assert.Equal(t, DefaultRegistryURL, config.RegistryURL)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Contains(t, config.UserAgent, "preflight")
}

func TestClient_FetchIndex(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/index.json", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"version": "1",
			"packages": [
				{
					"id": "test-pkg",
					"type": "preset",
					"title": "Test Package",
					"versions": [{"version": "1.0.0"}]
				}
			]
		}`))
	}))
	defer server.Close()

	config := ClientConfig{
		RegistryURL: server.URL,
		Timeout:     10 * time.Second,
		UserAgent:   "test/1.0",
	}
	client := NewClient(config)

	idx, err := client.FetchIndex(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "1", idx.Version)
	assert.Len(t, idx.Packages, 1)
	assert.Equal(t, "test-pkg", idx.Packages[0].ID.String())
}

func TestClient_FetchPackage(t *testing.T) {
	t.Parallel()

	packageData := []byte("package content data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/packages/my-package/1.0.0.tar.gz", r.URL.Path)
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(packageData)
	}))
	defer server.Close()

	config := ClientConfig{
		RegistryURL: server.URL,
		Timeout:     10 * time.Second,
	}
	client := NewClient(config)

	data, err := client.FetchPackage(context.Background(), MustNewPackageID("my-package"), "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, packageData, data)
}

func TestClient_FetchPackageMetadata(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/packages/meta-test/metadata.json", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "meta-test",
			"type": "preset",
			"title": "Metadata Test",
			"description": "A package for testing metadata",
			"versions": [{"version": "2.0.0"}]
		}`))
	}))
	defer server.Close()

	config := ClientConfig{
		RegistryURL: server.URL,
		Timeout:     10 * time.Second,
	}
	client := NewClient(config)

	pkg, err := client.FetchPackageMetadata(context.Background(), MustNewPackageID("meta-test"))
	require.NoError(t, err)
	assert.Equal(t, "Metadata Test", pkg.Title)
	assert.Equal(t, "A package for testing metadata", pkg.Description)
}

func TestClient_Search(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/search", r.URL.Path)
		assert.Equal(t, "nvim", r.URL.Query().Get("q"))
		assert.Equal(t, "preset", r.URL.Query().Get("type"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"version": "1",
			"packages": [
				{
					"id": "nvim-config",
					"type": "preset",
					"title": "Neovim Config",
					"versions": [{"version": "1.0.0"}]
				}
			]
		}`))
	}))
	defer server.Close()

	config := ClientConfig{
		RegistryURL: server.URL,
		Timeout:     10 * time.Second,
	}
	client := NewClient(config)

	results, err := client.Search(context.Background(), "nvim", SearchOptions{
		Type:  "preset",
		Limit: 10,
	})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "nvim-config", results[0].ID.String())
}

func TestClient_Ping(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := ClientConfig{
		RegistryURL: server.URL,
		Timeout:     10 * time.Second,
	}
	client := NewClient(config)

	err := client.Ping(context.Background())
	require.NoError(t, err)
}

func TestClient_Ping_Unhealthy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := ClientConfig{
		RegistryURL: server.URL,
		Timeout:     10 * time.Second,
	}
	client := NewClient(config)

	err := client.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unhealthy")
}

func TestClient_HTTPErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{"not found", http.StatusNotFound, ErrPackageNotFound},
		{"unauthorized", http.StatusUnauthorized, ErrUnauthorized},
		{"forbidden", http.StatusForbidden, ErrUnauthorized},
		{"rate limited", http.StatusTooManyRequests, ErrRateLimited},
		{"server error", http.StatusInternalServerError, ErrServerError},
		{"bad gateway", http.StatusBadGateway, ErrServerError},
		{"service unavailable", http.StatusServiceUnavailable, ErrServerError},
		{"other error", http.StatusTeapot, ErrFetchFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			config := ClientConfig{
				RegistryURL: server.URL,
				Timeout:     10 * time.Second,
			}
			client := NewClient(config)

			_, err := client.FetchIndex(context.Background())
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestClient_AuthToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-token-123", auth)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version": "1", "packages": []}`))
	}))
	defer server.Close()

	config := ClientConfig{
		RegistryURL: server.URL,
		Timeout:     10 * time.Second,
		AuthToken:   "test-token-123",
	}
	client := NewClient(config)

	_, err := client.FetchIndex(context.Background())
	require.NoError(t, err)
}

func TestClient_UserAgent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		assert.Equal(t, "custom-agent/1.0", ua)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version": "1", "packages": []}`))
	}))
	defer server.Close()

	config := ClientConfig{
		RegistryURL: server.URL,
		Timeout:     10 * time.Second,
		UserAgent:   "custom-agent/1.0",
	}
	client := NewClient(config)

	_, err := client.FetchIndex(context.Background())
	require.NoError(t, err)
}

func TestClient_NetworkError(t *testing.T) {
	t.Parallel()

	config := ClientConfig{
		RegistryURL: "http://localhost:0", // Invalid port
		Timeout:     1 * time.Second,
	}
	client := NewClient(config)

	_, err := client.FetchIndex(context.Background())
	assert.ErrorIs(t, err, ErrNetworkError)
}

func TestClient_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	config := ClientConfig{
		RegistryURL: server.URL,
		Timeout:     10 * time.Second,
	}
	client := NewClient(config)

	_, err := client.FetchIndex(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}
