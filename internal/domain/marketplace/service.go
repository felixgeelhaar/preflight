package marketplace

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Service errors.
var (
	ErrAlreadyInstalled = errors.New("package already installed")
	ErrNotInstalled     = errors.New("package not installed")
	ErrInstallFailed    = errors.New("installation failed")
)

// ServiceConfig configures the marketplace service.
type ServiceConfig struct {
	// InstallPath is where packages are installed
	InstallPath string
	// CacheConfig configures the local cache
	CacheConfig CacheConfig
	// ClientConfig configures the HTTP client
	ClientConfig ClientConfig
	// OfflineMode disables network access
	OfflineMode bool
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() ServiceConfig {
	homeDir, _ := os.UserHomeDir()
	return ServiceConfig{
		InstallPath:  filepath.Join(homeDir, ".preflight", "marketplace", "installed"),
		CacheConfig:  DefaultCacheConfig(),
		ClientConfig: DefaultClientConfig(),
		OfflineMode:  false,
	}
}

// Service provides marketplace operations.
type Service struct {
	config ServiceConfig
	client *Client
	cache  *Cache
}

// NewService creates a new marketplace service.
func NewService(config ServiceConfig) *Service {
	return &Service{
		config: config,
		client: NewClient(config.ClientConfig),
		cache:  NewCache(config.CacheConfig),
	}
}

// Search finds packages matching the query.
func (s *Service) Search(ctx context.Context, query string) ([]Package, error) {
	idx, err := s.getIndex(ctx)
	if err != nil {
		return nil, err
	}

	return idx.Search(query), nil
}

// SearchByType finds packages of a specific type.
func (s *Service) SearchByType(ctx context.Context, pkgType string) ([]Package, error) {
	idx, err := s.getIndex(ctx)
	if err != nil {
		return nil, err
	}

	return idx.SearchByType(pkgType), nil
}

// Get returns a package by ID.
func (s *Service) Get(ctx context.Context, id PackageID) (*Package, error) {
	idx, err := s.getIndex(ctx)
	if err != nil {
		return nil, err
	}

	pkg, ok := idx.Get(id)
	if !ok {
		return nil, ErrPackageNotFound
	}

	return &pkg, nil
}

// Install downloads and installs a package.
func (s *Service) Install(ctx context.Context, id PackageID, version string) (*InstalledPackage, error) {
	// Check if already installed
	existing, found, err := s.cache.GetInstalledPackage(id)
	if err != nil {
		return nil, err
	}
	if found && existing.Version == version {
		return nil, fmt.Errorf("%w: %s@%s", ErrAlreadyInstalled, id, version)
	}

	// Get package metadata
	pkg, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Resolve version
	var pkgVersion PackageVersion
	if version == "" || version == "latest" {
		v, ok := pkg.LatestVersion()
		if !ok {
			return nil, fmt.Errorf("no versions available for %s", id)
		}
		pkgVersion = v
	} else {
		v, ok := pkg.GetVersion(version)
		if !ok {
			return nil, fmt.Errorf("%w: %s@%s", ErrVersionNotFound, id, version)
		}
		pkgVersion = v
	}

	// Download package
	data, err := s.downloadPackage(ctx, id, pkgVersion.Version)
	if err != nil {
		return nil, err
	}

	// Verify checksum
	if err := pkgVersion.ValidateChecksum(data); err != nil {
		return nil, err
	}

	// Extract to install path
	installPath := filepath.Join(s.config.InstallPath, id.String(), pkgVersion.Version)
	if err := s.extractPackage(data, installPath); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInstallFailed, err)
	}

	// Record installation
	installed := InstalledPackage{
		Package:     *pkg,
		Version:     pkgVersion.Version,
		InstalledAt: time.Now(),
		Path:        installPath,
		AutoUpdate:  false,
	}

	if err := s.cache.AddInstalled(installed); err != nil {
		return nil, err
	}

	return &installed, nil
}

// Uninstall removes an installed package.
func (s *Service) Uninstall(id PackageID) error {
	pkg, found, err := s.cache.GetInstalledPackage(id)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: %s", ErrNotInstalled, id)
	}

	// Remove installed files
	if pkg.Path != "" {
		if err := os.RemoveAll(pkg.Path); err != nil {
			return fmt.Errorf("failed to remove package files: %w", err)
		}
	}

	// Remove from installed list
	return s.cache.RemoveInstalled(id)
}

// Update updates an installed package to the latest version.
func (s *Service) Update(ctx context.Context, id PackageID) (*InstalledPackage, error) {
	existing, found, err := s.cache.GetInstalledPackage(id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("%w: %s", ErrNotInstalled, id)
	}

	// Get latest version
	pkg, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	latest, ok := pkg.LatestVersion()
	if !ok {
		return nil, fmt.Errorf("no versions available for %s", id)
	}

	// Check if update needed
	if latest.Version == existing.Version {
		return &existing, nil // Already up to date
	}

	// Uninstall old version
	if err := s.Uninstall(id); err != nil {
		return nil, err
	}

	// Install new version
	return s.Install(ctx, id, latest.Version)
}

// UpdateAll updates all installed packages.
func (s *Service) UpdateAll(ctx context.Context) ([]InstalledPackage, error) {
	installed, err := s.cache.GetInstalled()
	if err != nil {
		return nil, err
	}

	var updated []InstalledPackage
	for _, pkg := range installed {
		result, err := s.Update(ctx, pkg.Package.ID)
		if err != nil {
			// Log but continue
			continue
		}
		if result.Version != pkg.Version {
			updated = append(updated, *result)
		}
	}

	return updated, nil
}

// List returns all installed packages.
func (s *Service) List() ([]InstalledPackage, error) {
	return s.cache.GetInstalled()
}

// CheckUpdates returns packages that have updates available.
func (s *Service) CheckUpdates(ctx context.Context) ([]UpdateInfo, error) {
	installed, err := s.cache.GetInstalled()
	if err != nil {
		return nil, err
	}

	idx, err := s.getIndex(ctx)
	if err != nil {
		return nil, err
	}

	var updates []UpdateInfo
	for _, inst := range installed {
		pkg, ok := idx.Get(inst.Package.ID)
		if !ok {
			continue
		}

		latest, ok := pkg.LatestVersion()
		if !ok {
			continue
		}

		if latest.Version != inst.Version {
			updates = append(updates, UpdateInfo{
				Package:        pkg,
				CurrentVersion: inst.Version,
				LatestVersion:  latest.Version,
				Changelog:      latest.Changelog,
			})
		}
	}

	return updates, nil
}

// UpdateInfo describes an available update.
type UpdateInfo struct {
	Package        Package
	CurrentVersion string
	LatestVersion  string
	Changelog      string
}

// Statistics returns index statistics.
func (s *Service) Statistics(ctx context.Context) (*IndexStats, error) {
	idx, err := s.getIndex(ctx)
	if err != nil {
		return nil, err
	}

	stats := idx.Statistics()
	return &stats, nil
}

// RefreshIndex forces a refresh of the cached index.
func (s *Service) RefreshIndex(ctx context.Context) error {
	if s.config.OfflineMode {
		return ErrNetworkError
	}

	idx, err := s.client.FetchIndex(ctx)
	if err != nil {
		return err
	}

	return s.cache.PutIndex(idx, s.config.ClientConfig.RegistryURL)
}

// getIndex returns the index, using cache if valid.
func (s *Service) getIndex(ctx context.Context) (*Index, error) {
	// Try cache first
	idx, err := s.cache.GetIndex()
	if err == nil {
		return idx, nil
	}

	// Cache miss or expired - fetch if online
	if s.config.OfflineMode {
		if errors.Is(err, ErrCacheMiss) {
			return nil, fmt.Errorf("no cached index available in offline mode")
		}
		// Use expired cache in offline mode
		idx, _ = s.cache.GetIndex()
		if idx != nil {
			return idx, nil
		}
		return nil, fmt.Errorf("no cached index available in offline mode")
	}

	// Fetch from network
	idx, err = s.client.FetchIndex(ctx)
	if err != nil {
		return nil, err
	}

	// Cache for next time
	_ = s.cache.PutIndex(idx, s.config.ClientConfig.RegistryURL)

	return idx, nil
}

// downloadPackage downloads a package, using cache if available.
func (s *Service) downloadPackage(ctx context.Context, id PackageID, version string) ([]byte, error) {
	// Try cache first
	data, err := s.cache.GetPackage(id, version)
	if err == nil {
		return data, nil
	}

	if s.config.OfflineMode {
		return nil, fmt.Errorf("package %s@%s not cached and offline mode enabled", id, version)
	}

	// Fetch from network
	data, err = s.client.FetchPackage(ctx, id, version)
	if err != nil {
		return nil, err
	}

	// Cache for next time
	_ = s.cache.PutPackage(id, version, data, s.config.ClientConfig.RegistryURL)

	return data, nil
}

// extractPackage extracts a tar.gz package to the target directory.
func (s *Service) extractPackage(data []byte, targetDir string) error {
	// Create target directory
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create gzip reader
	gr, err := gzip.NewReader(&bytesReader{data: data})
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gr.Close() }()

	// Create tar reader
	tr := tar.NewReader(gr)

	// Extract files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Sanitize path to prevent directory traversal
		target := filepath.Join(targetDir, filepath.Clean(header.Name))
		cleanTargetDir := filepath.Clean(targetDir) + string(filepath.Separator)
		if !strings.HasPrefix(target, cleanTargetDir) && target != filepath.Clean(targetDir) {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			_ = f.Close()
		}
	}

	return nil
}

// bytesReader wraps a byte slice to implement io.Reader.
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
