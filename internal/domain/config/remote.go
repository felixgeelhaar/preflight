package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RemoteLayerSource represents a remote layer location.
type RemoteLayerSource struct {
	URL       string `yaml:"url,omitempty"`       // HTTP/HTTPS URL
	GitRepo   string `yaml:"git,omitempty"`       // Git repository URL
	GitRef    string `yaml:"ref,omitempty"`       // Git ref (branch, tag, commit)
	GitPath   string `yaml:"path,omitempty"`      // Path within repo
	CacheTTL  string `yaml:"cache_ttl,omitempty"` // Cache duration (e.g., "1h", "24h")
	Integrity string `yaml:"integrity,omitempty"` // SHA256 hash for verification
}

// RemoteLayerConfig extends manifest with remote layer definitions.
type RemoteLayerConfig struct {
	Remotes map[string]RemoteLayerSource `yaml:"remote_layers,omitempty"`
}

// RemoteLoader handles fetching and caching remote layers.
type RemoteLoader struct {
	cacheDir   string
	httpClient *http.Client
}

// NewRemoteLoader creates a new RemoteLoader with the specified cache directory.
func NewRemoteLoader(cacheDir string) *RemoteLoader {
	return &RemoteLoader{
		cacheDir: cacheDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LoadRemoteLayer fetches a layer from a remote source.
func (r *RemoteLoader) LoadRemoteLayer(ctx context.Context, name string, source RemoteLayerSource) (*Layer, error) {
	// Determine cache path
	cacheKey := r.cacheKey(source)
	cachePath := filepath.Join(r.cacheDir, "remote-layers", cacheKey+".yaml")

	// Check cache validity
	if r.isCacheValid(cachePath, source.CacheTTL) {
		return r.loadFromCache(cachePath, name)
	}

	// Fetch from remote
	var data []byte
	var err error

	switch {
	case source.URL != "":
		data, err = r.fetchFromURL(ctx, source.URL)
	case source.GitRepo != "":
		data, err = r.fetchFromGit(ctx, source)
	default:
		return nil, fmt.Errorf("remote layer %s: must specify url or git", name)
	}

	if err != nil {
		// Try to use stale cache if fetch fails
		if _, statErr := os.Stat(cachePath); statErr == nil {
			return r.loadFromCache(cachePath, name)
		}
		return nil, fmt.Errorf("remote layer %s: %w", name, err)
	}

	// Verify integrity if specified
	if source.Integrity != "" {
		hash := sha256.Sum256(data)
		actual := hex.EncodeToString(hash[:])
		if !strings.EqualFold(actual, source.Integrity) {
			return nil, fmt.Errorf("remote layer %s: integrity check failed (expected %s, got %s)",
				name, source.Integrity, actual)
		}
	}

	// Cache the data
	if err := r.saveToCache(cachePath, data); err != nil {
		// Log warning but continue
		fmt.Fprintf(os.Stderr, "warning: failed to cache remote layer %s: %v\n", name, err)
	}

	// Parse and return
	layer, err := ParseLayer(data)
	if err != nil {
		return nil, fmt.Errorf("remote layer %s: parse error: %w", name, err)
	}

	layer.SetProvenance(fmt.Sprintf("remote:%s", r.sourceDescription(source)))
	return layer, nil
}

// cacheKey generates a unique cache key for a remote source.
func (r *RemoteLoader) cacheKey(source RemoteLayerSource) string {
	var key string
	if source.URL != "" {
		key = source.URL
	} else {
		key = fmt.Sprintf("%s@%s:%s", source.GitRepo, source.GitRef, source.GitPath)
	}
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:16])
}

// isCacheValid checks if the cached file is still valid.
func (r *RemoteLoader) isCacheValid(cachePath string, ttlStr string) bool {
	info, err := os.Stat(cachePath)
	if err != nil {
		return false
	}

	ttl := 24 * time.Hour // Default TTL
	if ttlStr != "" {
		if parsed, err := time.ParseDuration(ttlStr); err == nil {
			ttl = parsed
		}
	}

	return time.Since(info.ModTime()) < ttl
}

// loadFromCache loads a layer from the cache.
func (r *RemoteLoader) loadFromCache(cachePath, _ string) (*Layer, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	layer, err := ParseLayer(data)
	if err != nil {
		return nil, err
	}

	layer.SetProvenance(fmt.Sprintf("cache:%s", cachePath))
	return layer, nil
}

// saveToCache saves layer data to the cache.
func (r *RemoteLoader) saveToCache(cachePath string, data []byte) error {
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(cachePath, data, 0644)
}

// fetchFromURL fetches layer content from an HTTP/HTTPS URL.
func (r *RemoteLoader) fetchFromURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// fetchFromGit fetches layer content from a Git repository.
func (r *RemoteLoader) fetchFromGit(_ context.Context, source RemoteLayerSource) ([]byte, error) {
	// Clone to temp directory
	tmpDir, err := os.MkdirTemp("", "preflight-remote-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ref := source.GitRef
	if ref == "" {
		ref = "HEAD"
	}

	// Use sparse checkout for efficiency
	_ = filepath.Join(tmpDir, "repo") // repoPath reserved for future git clone implementation

	// For now, we'll use a simple approach - this could be enhanced with go-git
	// In production, you'd want to use the git CLI or go-git library
	layerPath := source.GitPath
	if layerPath == "" {
		layerPath = "preflight.yaml"
	}

	// Placeholder: In a real implementation, clone and read
	return nil, fmt.Errorf("git remote layers require git clone implementation (repo: %s, ref: %s, path: %s)",
		source.GitRepo, ref, layerPath)
}

// sourceDescription returns a human-readable description of the source.
func (r *RemoteLoader) sourceDescription(source RemoteLayerSource) string {
	if source.URL != "" {
		return source.URL
	}
	desc := source.GitRepo
	if source.GitRef != "" {
		desc += "@" + source.GitRef
	}
	if source.GitPath != "" {
		desc += ":" + source.GitPath
	}
	return desc
}

// ParseRemoteLayerConfig parses remote layer configuration from YAML.
func ParseRemoteLayerConfig(data []byte) (*RemoteLayerConfig, error) {
	var config RemoteLayerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
