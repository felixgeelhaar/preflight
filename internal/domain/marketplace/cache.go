package marketplace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Cache errors.
var (
	ErrCacheExpired = errors.New("cache expired")
	ErrCacheMiss    = errors.New("cache miss")
)

// CacheConfig configures the local cache behavior.
type CacheConfig struct {
	// BasePath is the base directory for cache files
	BasePath string
	// IndexTTL is how long the index cache is valid
	IndexTTL time.Duration
	// PackageTTL is how long package content cache is valid
	PackageTTL time.Duration
}

// DefaultCacheConfig returns sensible defaults.
func DefaultCacheConfig() CacheConfig {
	homeDir, _ := os.UserHomeDir()
	return CacheConfig{
		BasePath:   filepath.Join(homeDir, ".preflight", "marketplace", "cache"),
		IndexTTL:   1 * time.Hour,
		PackageTTL: 24 * time.Hour,
	}
}

// Cache provides local caching for marketplace data.
type Cache struct {
	config CacheConfig
}

// NewCache creates a new cache instance.
func NewCache(config CacheConfig) *Cache {
	return &Cache{config: config}
}

// indexPath returns the path to the cached index.
func (c *Cache) indexPath() string {
	return filepath.Join(c.config.BasePath, "index.json")
}

// indexMetaPath returns the path to the index metadata.
func (c *Cache) indexMetaPath() string {
	return filepath.Join(c.config.BasePath, "index.meta.json")
}

// packagePath returns the path to a cached package.
func (c *Cache) packagePath(id PackageID, version string) string {
	return filepath.Join(c.config.BasePath, "packages", id.String(), version+".tar.gz")
}

// packageMetaPath returns the path to package metadata.
func (c *Cache) packageMetaPath(id PackageID, version string) string {
	return filepath.Join(c.config.BasePath, "packages", id.String(), version+".meta.json")
}

// installedPath returns the path to installed packages manifest.
func (c *Cache) installedPath() string {
	return filepath.Join(c.config.BasePath, "installed.json")
}

// CacheMeta contains metadata about a cached item.
type CacheMeta struct {
	CachedAt time.Time `json:"cached_at"`
	Checksum string    `json:"checksum,omitempty"`
	Source   string    `json:"source,omitempty"`
}

// IsExpired returns true if the cache entry is expired.
func (m CacheMeta) IsExpired(ttl time.Duration) bool {
	return time.Since(m.CachedAt) > ttl
}

// EnsureDir ensures the cache directory exists.
func (c *Cache) EnsureDir() error {
	return os.MkdirAll(c.config.BasePath, 0o755)
}

// GetIndex returns the cached index if valid.
func (c *Cache) GetIndex() (*Index, error) {
	// Check metadata
	metaData, err := os.ReadFile(c.indexMetaPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("failed to read index meta: %w", err)
	}

	var meta CacheMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse index meta: %w", err)
	}

	if meta.IsExpired(c.config.IndexTTL) {
		return nil, ErrCacheExpired
	}

	// Read index
	indexData, err := os.ReadFile(c.indexPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	return ParseIndex(indexData)
}

// PutIndex caches the index.
func (c *Cache) PutIndex(idx *Index, source string) error {
	if err := c.EnsureDir(); err != nil {
		return err
	}

	// Write index
	data, err := idx.Marshal()
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.indexPath(), data, 0o644); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	// Write metadata
	meta := CacheMeta{
		CachedAt: time.Now(),
		Checksum: ComputeChecksum(data),
		Source:   source,
	}

	metaData, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.indexMetaPath(), metaData, 0o644); err != nil {
		return fmt.Errorf("failed to write index meta: %w", err)
	}

	return nil
}

// GetPackage returns cached package content if valid.
func (c *Cache) GetPackage(id PackageID, version string) ([]byte, error) {
	// Check metadata
	metaData, err := os.ReadFile(c.packageMetaPath(id, version))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("failed to read package meta: %w", err)
	}

	var meta CacheMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse package meta: %w", err)
	}

	if meta.IsExpired(c.config.PackageTTL) {
		return nil, ErrCacheExpired
	}

	// Read package
	data, err := os.ReadFile(c.packagePath(id, version))
	if err != nil {
		return nil, fmt.Errorf("failed to read package: %w", err)
	}

	// Verify checksum
	if ComputeChecksum(data) != meta.Checksum {
		return nil, ErrChecksumMismatch
	}

	return data, nil
}

// PutPackage caches package content.
func (c *Cache) PutPackage(id PackageID, version string, data []byte, source string) error {
	pkgDir := filepath.Dir(c.packagePath(id, version))
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		return fmt.Errorf("failed to create package dir: %w", err)
	}

	// Write package
	if err := os.WriteFile(c.packagePath(id, version), data, 0o644); err != nil {
		return fmt.Errorf("failed to write package: %w", err)
	}

	// Write metadata
	meta := CacheMeta{
		CachedAt: time.Now(),
		Checksum: ComputeChecksum(data),
		Source:   source,
	}

	metaData, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.packageMetaPath(id, version), metaData, 0o644); err != nil {
		return fmt.Errorf("failed to write package meta: %w", err)
	}

	return nil
}

// GetInstalled returns all installed packages.
func (c *Cache) GetInstalled() ([]InstalledPackage, error) {
	data, err := os.ReadFile(c.installedPath())
	if err != nil {
		if os.IsNotExist(err) {
			return []InstalledPackage{}, nil
		}
		return nil, fmt.Errorf("failed to read installed: %w", err)
	}

	var installed []InstalledPackage
	if err := json.Unmarshal(data, &installed); err != nil {
		return nil, fmt.Errorf("failed to parse installed: %w", err)
	}

	return installed, nil
}

// AddInstalled adds a package to the installed list.
func (c *Cache) AddInstalled(pkg InstalledPackage) error {
	installed, err := c.GetInstalled()
	if err != nil {
		return err
	}

	// Check if already installed
	for i, p := range installed {
		if p.Package.ID.Equals(pkg.Package.ID) {
			// Update existing
			installed[i] = pkg
			return c.saveInstalled(installed)
		}
	}

	// Add new
	installed = append(installed, pkg)
	return c.saveInstalled(installed)
}

// RemoveInstalled removes a package from the installed list.
func (c *Cache) RemoveInstalled(id PackageID) error {
	installed, err := c.GetInstalled()
	if err != nil {
		return err
	}

	var filtered []InstalledPackage
	for _, p := range installed {
		if !p.Package.ID.Equals(id) {
			filtered = append(filtered, p)
		}
	}

	return c.saveInstalled(filtered)
}

// GetInstalledPackage returns a specific installed package.
func (c *Cache) GetInstalledPackage(id PackageID) (InstalledPackage, bool, error) {
	installed, err := c.GetInstalled()
	if err != nil {
		return InstalledPackage{}, false, err
	}

	for _, p := range installed {
		if p.Package.ID.Equals(id) {
			return p, true, nil
		}
	}

	return InstalledPackage{}, false, nil
}

func (c *Cache) saveInstalled(installed []InstalledPackage) error {
	if err := c.EnsureDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(installed, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.installedPath(), data, 0o644)
}

// Clear removes all cached data.
func (c *Cache) Clear() error {
	return os.RemoveAll(c.config.BasePath)
}

// ClearIndex removes cached index only.
func (c *Cache) ClearIndex() error {
	_ = os.Remove(c.indexPath())
	_ = os.Remove(c.indexMetaPath())
	return nil
}
