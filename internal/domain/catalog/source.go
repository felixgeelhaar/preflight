package catalog

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Source errors.
var (
	ErrInvalidSource     = errors.New("invalid catalog source")
	ErrSourceNotFound    = errors.New("catalog source not found")
	ErrSourceUnreachable = errors.New("catalog source unreachable")
)

// SourceType represents the type of catalog source.
type SourceType string

// SourceType constants.
const (
	SourceTypeBuiltin SourceType = "builtin"
	SourceTypeURL     SourceType = "url"
	SourceTypeLocal   SourceType = "local"
)

// Source represents a catalog source location.
// It can be a URL or a local file path.
type Source struct {
	sourceType SourceType
	location   string
	name       string
}

// NewURLSource creates a new URL-based catalog source.
func NewURLSource(name, rawURL string) (Source, error) {
	if name == "" {
		return Source{}, fmt.Errorf("%w: name is required", ErrInvalidSource)
	}

	if rawURL == "" {
		return Source{}, fmt.Errorf("%w: URL is required", ErrInvalidSource)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return Source{}, fmt.Errorf("%w: invalid URL: %w", ErrInvalidSource, err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return Source{}, fmt.Errorf("%w: URL scheme must be http or https", ErrInvalidSource)
	}

	if parsed.Host == "" {
		return Source{}, fmt.Errorf("%w: URL must have a host", ErrInvalidSource)
	}

	return Source{
		sourceType: SourceTypeURL,
		location:   rawURL,
		name:       name,
	}, nil
}

// NewLocalSource creates a new local file-based catalog source.
func NewLocalSource(name, path string) (Source, error) {
	if name == "" {
		return Source{}, fmt.Errorf("%w: name is required", ErrInvalidSource)
	}

	if path == "" {
		return Source{}, fmt.Errorf("%w: path is required", ErrInvalidSource)
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return Source{}, fmt.Errorf("%w: cannot expand home directory: %w", ErrInvalidSource, err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Clean the path
	path = filepath.Clean(path)

	return Source{
		sourceType: SourceTypeLocal,
		location:   path,
		name:       name,
	}, nil
}

// NewBuiltinSource creates a source representing the embedded catalog.
func NewBuiltinSource() Source {
	return Source{
		sourceType: SourceTypeBuiltin,
		location:   "embedded",
		name:       "builtin",
	}
}

// Type returns the source type.
func (s Source) Type() SourceType {
	return s.sourceType
}

// Location returns the source location (URL or path).
func (s Source) Location() string {
	return s.location
}

// Name returns the source name.
func (s Source) Name() string {
	return s.name
}

// IsBuiltin returns true if this is the builtin catalog.
func (s Source) IsBuiltin() bool {
	return s.sourceType == SourceTypeBuiltin
}

// IsURL returns true if this is a URL source.
func (s Source) IsURL() bool {
	return s.sourceType == SourceTypeURL
}

// IsLocal returns true if this is a local file source.
func (s Source) IsLocal() bool {
	return s.sourceType == SourceTypeLocal
}

// IsZero returns true if the source is empty.
func (s Source) IsZero() bool {
	return s.sourceType == "" && s.location == "" && s.name == ""
}

// String returns a string representation.
func (s Source) String() string {
	return fmt.Sprintf("%s (%s: %s)", s.name, s.sourceType, s.location)
}

// ManifestURL returns the URL to the manifest file.
// For URL sources, appends /catalog-manifest.yaml.
// For local sources, returns the manifest path.
func (s Source) ManifestURL() string {
	switch s.sourceType {
	case SourceTypeURL:
		loc := strings.TrimSuffix(s.location, "/")
		return loc + "/catalog-manifest.yaml"
	case SourceTypeLocal:
		return filepath.Join(s.location, "catalog-manifest.yaml")
	default:
		return ""
	}
}

// CatalogURL returns the URL to the catalog file.
func (s Source) CatalogURL() string {
	switch s.sourceType {
	case SourceTypeURL:
		loc := strings.TrimSuffix(s.location, "/")
		return loc + "/catalog.yaml"
	case SourceTypeLocal:
		return filepath.Join(s.location, "catalog.yaml")
	default:
		return ""
	}
}
