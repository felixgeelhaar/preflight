package catalog

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ExternalLoader loads catalogs from external sources (URLs or local paths).
type ExternalLoader struct {
	client   *http.Client
	cacheDir string
}

// ExternalLoaderConfig configures the external loader.
type ExternalLoaderConfig struct {
	Timeout  time.Duration
	CacheDir string
}

// DefaultExternalLoaderConfig returns default configuration.
func DefaultExternalLoaderConfig() ExternalLoaderConfig {
	home, _ := os.UserHomeDir()
	return ExternalLoaderConfig{
		Timeout:  30 * time.Second,
		CacheDir: filepath.Join(home, ".preflight", "catalogs"),
	}
}

// NewExternalLoader creates a new external loader.
func NewExternalLoader(config ExternalLoaderConfig) *ExternalLoader {
	return &ExternalLoader{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		cacheDir: config.CacheDir,
	}
}

// Load loads a catalog from an external source.
// It first fetches and validates the manifest, then loads the catalog files.
func (l *ExternalLoader) Load(ctx context.Context, source Source) (*RegisteredCatalog, error) {
	if source.IsBuiltin() {
		return nil, fmt.Errorf("%w: cannot load builtin source externally", ErrInvalidSource)
	}

	// Fetch and parse manifest
	manifest, err := l.fetchManifest(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	// Fetch and verify catalog file
	catalogData, err := l.fetchAndVerify(ctx, source, "catalog.yaml", manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog: %w", err)
	}

	// Parse catalog
	catalog, err := l.parseCatalog(catalogData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse catalog: %w", err)
	}

	// Cache the catalog locally
	if err := l.cache(source, manifest, catalogData); err != nil {
		// Non-fatal, just log
		_ = err
	}

	return NewRegisteredCatalog(source, manifest, catalog), nil
}

// LoadFromCache loads a catalog from the local cache.
func (l *ExternalLoader) LoadFromCache(source Source) (*RegisteredCatalog, error) {
	cacheDir := l.sourceCacheDir(source)

	// Read manifest
	manifestPath := filepath.Join(cacheDir, "catalog-manifest.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSourceNotFound, err)
	}

	manifest, err := l.parseManifest(manifestData)
	if err != nil {
		return nil, err
	}

	// Read catalog
	catalogPath := filepath.Join(cacheDir, "catalog.yaml")
	catalogData, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSourceNotFound, err)
	}

	// Verify integrity
	if err := manifest.VerifyFile("catalog.yaml", catalogData); err != nil {
		return nil, err
	}

	catalog, err := l.parseCatalog(catalogData)
	if err != nil {
		return nil, err
	}

	return NewRegisteredCatalog(source, manifest, catalog), nil
}

// Verify verifies the integrity of a registered catalog.
func (l *ExternalLoader) Verify(ctx context.Context, rc *RegisteredCatalog) error {
	source := rc.Source()
	manifest := rc.Manifest()

	for _, file := range manifest.Files() {
		data, err := l.fetchFile(ctx, source, file.Path)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %w", file.Path, err)
		}

		if err := manifest.VerifyFile(file.Path, data); err != nil {
			return err
		}
	}

	rc.SetVerifiedAt(time.Now())
	return nil
}

// fetchManifest fetches and parses the manifest file.
func (l *ExternalLoader) fetchManifest(ctx context.Context, source Source) (Manifest, error) {
	data, err := l.fetchFile(ctx, source, "catalog-manifest.yaml")
	if err != nil {
		return Manifest{}, err
	}
	return l.parseManifest(data)
}

// fetchAndVerify fetches a file and verifies its integrity.
func (l *ExternalLoader) fetchAndVerify(ctx context.Context, source Source, path string, manifest Manifest) ([]byte, error) {
	data, err := l.fetchFile(ctx, source, path)
	if err != nil {
		return nil, err
	}

	if err := manifest.VerifyFile(path, data); err != nil {
		return nil, err
	}

	return data, nil
}

// fetchFile fetches a file from the source.
func (l *ExternalLoader) fetchFile(ctx context.Context, source Source, path string) ([]byte, error) {
	switch source.Type() {
	case SourceTypeURL:
		return l.fetchURL(ctx, source.Location()+"/"+path)
	case SourceTypeLocal:
		return os.ReadFile(filepath.Join(source.Location(), path))
	default:
		return nil, fmt.Errorf("%w: unsupported source type", ErrInvalidSource)
	}
}

// fetchURL fetches data from a URL.
func (l *ExternalLoader) fetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/yaml, text/yaml, */*")
	req.Header.Set("User-Agent", "preflight/1.0")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSourceUnreachable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrSourceNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrSourceUnreachable, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// parseManifest parses manifest YAML data.
func (l *ExternalLoader) parseManifest(data []byte) (Manifest, error) {
	var dto manifestDTO
	if err := yaml.Unmarshal(data, &dto); err != nil {
		return Manifest{}, fmt.Errorf("%w: invalid YAML: %w", ErrInvalidManifest, err)
	}

	builder := NewManifestBuilder(dto.Name).
		WithVersion(dto.Version).
		WithDescription(dto.Description).
		WithAuthor(dto.Author).
		WithRepository(dto.Repository).
		WithLicense(dto.License)

	if dto.Integrity.Algorithm != "" {
		builder.WithAlgorithm(HashAlgorithm(dto.Integrity.Algorithm))
	}

	for path, hash := range dto.Integrity.Files {
		builder.AddFile(path, hash)
	}

	if !dto.CreatedAt.IsZero() {
		builder.WithCreatedAt(dto.CreatedAt)
	}
	if !dto.UpdatedAt.IsZero() {
		builder.WithUpdatedAt(dto.UpdatedAt)
	}

	return builder.Build()
}

// parseCatalog parses catalog YAML data.
func (l *ExternalLoader) parseCatalog(data []byte) (*Catalog, error) {
	var dto catalogDTO
	if err := yaml.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("failed to parse catalog YAML: %w", err)
	}

	cat := NewCatalog()

	// Load presets
	for _, p := range dto.Presets {
		preset, err := parsePresetDTO(p)
		if err != nil {
			return nil, fmt.Errorf("failed to parse preset %s: %w", p.ID, err)
		}
		if err := cat.AddPreset(preset); err != nil {
			return nil, fmt.Errorf("failed to add preset %s: %w", p.ID, err)
		}
	}

	// Load capability packs
	for _, cp := range dto.Packs {
		pack, err := parseCapabilityPackDTO(cp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse capability pack %s: %w", cp.ID, err)
		}
		if err := cat.AddPack(pack); err != nil {
			return nil, fmt.Errorf("failed to add capability pack %s: %w", cp.ID, err)
		}
	}

	return cat, nil
}

// cache stores a catalog in the local cache.
func (l *ExternalLoader) cache(source Source, manifest Manifest, catalogData []byte) error {
	cacheDir := l.sourceCacheDir(source)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	// Write manifest
	manifestData, err := yaml.Marshal(manifestToDTO(manifest))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "catalog-manifest.yaml"), manifestData, 0o644); err != nil {
		return err
	}

	// Write catalog
	return os.WriteFile(filepath.Join(cacheDir, "catalog.yaml"), catalogData, 0o644)
}

// sourceCacheDir returns the cache directory for a source.
func (l *ExternalLoader) sourceCacheDir(source Source) string {
	return filepath.Join(l.cacheDir, source.Name())
}

// ClearCache removes cached data for a source.
func (l *ExternalLoader) ClearCache(source Source) error {
	return os.RemoveAll(l.sourceCacheDir(source))
}

// manifestDTO is the data transfer object for manifest YAML.
type manifestDTO struct {
	Version     string       `yaml:"version"`
	Name        string       `yaml:"name"`
	Description string       `yaml:"description,omitempty"`
	Author      string       `yaml:"author,omitempty"`
	Repository  string       `yaml:"repository,omitempty"`
	License     string       `yaml:"license,omitempty"`
	Integrity   integrityDTO `yaml:"integrity"`
	CreatedAt   time.Time    `yaml:"created_at,omitempty"`
	UpdatedAt   time.Time    `yaml:"updated_at,omitempty"`
}

type integrityDTO struct {
	Algorithm string            `yaml:"algorithm"`
	Files     map[string]string `yaml:"files"`
}

// catalogDTO for parsing external catalog YAML.
type catalogDTO struct {
	Presets []presetDTO         `yaml:"presets"`
	Packs   []capabilityPackDTO `yaml:"capability_packs"`
}

type presetDTO struct {
	ID         string                 `yaml:"id"`
	Metadata   metadataDTO            `yaml:"metadata"`
	Difficulty string                 `yaml:"difficulty"`
	Config     map[string]interface{} `yaml:"config"`
	Requires   []string               `yaml:"requires"`
	Conflicts  []string               `yaml:"conflicts"`
}

type metadataDTO struct {
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	Tradeoffs   []string          `yaml:"tradeoffs"`
	DocLinks    map[string]string `yaml:"doc_links"`
	Tags        []string          `yaml:"tags"`
}

type capabilityPackDTO struct {
	ID       string      `yaml:"id"`
	Metadata metadataDTO `yaml:"metadata"`
	Presets  []string    `yaml:"presets"`
	Tools    []string    `yaml:"tools"`
}

func manifestToDTO(m Manifest) manifestDTO {
	files := make(map[string]string)
	for _, f := range m.Files() {
		files[f.Path] = f.Hash
	}
	return manifestDTO{
		Version:     m.Version(),
		Name:        m.Name(),
		Description: m.Description(),
		Author:      m.Author(),
		Repository:  m.Repository(),
		License:     m.License(),
		Integrity: integrityDTO{
			Algorithm: string(m.Algorithm()),
			Files:     files,
		},
		CreatedAt: m.CreatedAt(),
		UpdatedAt: m.UpdatedAt(),
	}
}

func parsePresetDTO(dto presetDTO) (Preset, error) {
	id, err := ParsePresetID(dto.ID)
	if err != nil {
		return Preset{}, err
	}

	meta, err := parseMetadataDTO(dto.Metadata)
	if err != nil {
		return Preset{}, err
	}

	difficulty, err := ParseDifficultyLevel(dto.Difficulty)
	if err != nil {
		return Preset{}, err
	}

	preset, err := NewPreset(id, meta, difficulty, dto.Config)
	if err != nil {
		return Preset{}, err
	}

	// Parse requires
	if len(dto.Requires) > 0 {
		requires := make([]PresetID, 0, len(dto.Requires))
		for _, r := range dto.Requires {
			reqID, err := ParsePresetID(r)
			if err != nil {
				return Preset{}, fmt.Errorf("invalid requires %s: %w", r, err)
			}
			requires = append(requires, reqID)
		}
		preset = preset.WithRequires(requires)
	}

	// Parse conflicts
	if len(dto.Conflicts) > 0 {
		conflicts := make([]PresetID, 0, len(dto.Conflicts))
		for _, c := range dto.Conflicts {
			conflictID, err := ParsePresetID(c)
			if err != nil {
				return Preset{}, fmt.Errorf("invalid conflict %s: %w", c, err)
			}
			conflicts = append(conflicts, conflictID)
		}
		preset = preset.WithConflicts(conflicts)
	}

	return preset, nil
}

func parseMetadataDTO(dto metadataDTO) (Metadata, error) {
	meta, err := NewMetadata(dto.Title, dto.Description)
	if err != nil {
		return Metadata{}, err
	}

	if len(dto.Tradeoffs) > 0 {
		meta = meta.WithTradeoffs(dto.Tradeoffs)
	}

	if len(dto.DocLinks) > 0 {
		meta = meta.WithDocLinks(dto.DocLinks)
	}

	if len(dto.Tags) > 0 {
		meta = meta.WithTags(dto.Tags)
	}

	return meta, nil
}

func parseCapabilityPackDTO(dto capabilityPackDTO) (CapabilityPack, error) {
	meta, err := parseMetadataDTO(dto.Metadata)
	if err != nil {
		return CapabilityPack{}, err
	}

	pack, err := NewCapabilityPack(dto.ID, meta)
	if err != nil {
		return CapabilityPack{}, err
	}

	// Parse preset IDs
	if len(dto.Presets) > 0 {
		presets := make([]PresetID, 0, len(dto.Presets))
		for _, p := range dto.Presets {
			presetID, err := ParsePresetID(p)
			if err != nil {
				return CapabilityPack{}, fmt.Errorf("invalid preset %s: %w", p, err)
			}
			presets = append(presets, presetID)
		}
		pack = pack.WithPresets(presets)
	}

	if len(dto.Tools) > 0 {
		pack = pack.WithTools(dto.Tools)
	}

	return pack, nil
}
