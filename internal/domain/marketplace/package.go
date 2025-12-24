// Package marketplace provides functionality for discovering, installing,
// and managing community presets, capability packs, and layer templates.
package marketplace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Package type constants.
const (
	PackageTypePreset         = "preset"
	PackageTypeCapabilityPack = "capability-pack"
	PackageTypeLayerTemplate  = "layer-template"
)

// Package errors.
var (
	ErrInvalidPackage     = errors.New("invalid package")
	ErrPackageNotFound    = errors.New("package not found")
	ErrVersionNotFound    = errors.New("version not found")
	ErrChecksumMismatch   = errors.New("checksum mismatch")
	ErrInvalidChecksum    = errors.New("invalid checksum format")
	ErrInvalidPackageType = errors.New("invalid package type")
)

// PackageID uniquely identifies a package in the marketplace.
type PackageID struct {
	name string
}

// NewPackageID creates a new PackageID.
func NewPackageID(name string) (PackageID, error) {
	if name == "" {
		return PackageID{}, fmt.Errorf("%w: name cannot be empty", ErrInvalidPackage)
	}
	// Package names must be lowercase alphanumeric with hyphens
	for _, c := range name {
		isLowercase := c >= 'a' && c <= 'z'
		isDigit := c >= '0' && c <= '9'
		isHyphen := c == '-'
		if !isLowercase && !isDigit && !isHyphen {
			return PackageID{}, fmt.Errorf("%w: invalid character in name: %c", ErrInvalidPackage, c)
		}
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return PackageID{}, fmt.Errorf("%w: name cannot start or end with hyphen", ErrInvalidPackage)
	}
	return PackageID{name: name}, nil
}

// MustNewPackageID creates a PackageID, panicking on error.
func MustNewPackageID(name string) PackageID {
	id, err := NewPackageID(name)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the package ID as a string.
func (id PackageID) String() string {
	return id.name
}

// MarshalJSON implements json.Marshaler.
func (id PackageID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.name + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *PackageID) UnmarshalJSON(data []byte) error {
	// Remove quotes
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		id.name = string(data[1 : len(data)-1])
		return nil
	}
	return fmt.Errorf("invalid package ID: %s", string(data))
}

// IsZero returns true if this is a zero-value PackageID.
func (id PackageID) IsZero() bool {
	return id.name == ""
}

// Equals returns true if this ID equals another.
func (id PackageID) Equals(other PackageID) bool {
	return id.name == other.name
}

// Provenance tracks the origin and verification status of a package.
type Provenance struct {
	Author     string    `json:"author" yaml:"author"`
	Repository string    `json:"repository" yaml:"repository"`
	License    string    `json:"license" yaml:"license"`
	Verified   bool      `json:"verified" yaml:"verified"`
	SignedBy   string    `json:"signed_by,omitempty" yaml:"signed_by,omitempty"`
	SignedAt   time.Time `json:"signed_at,omitempty" yaml:"signed_at,omitempty"`
}

// IsZero returns true if provenance is empty.
func (p Provenance) IsZero() bool {
	return p.Author == "" && p.Repository == ""
}

// PackageVersion represents a specific version of a package.
type PackageVersion struct {
	Version    string    `json:"version" yaml:"version"`
	Checksum   string    `json:"checksum" yaml:"checksum"` // SHA256 hex
	ReleasedAt time.Time `json:"released_at" yaml:"released_at"`
	Changelog  string    `json:"changelog,omitempty" yaml:"changelog,omitempty"`
	MinVersion string    `json:"min_preflight_version,omitempty" yaml:"min_preflight_version,omitempty"`
}

// ValidateChecksum verifies that the given data matches this version's checksum.
func (v PackageVersion) ValidateChecksum(data []byte) error {
	if v.Checksum == "" {
		return ErrInvalidChecksum
	}
	hash := sha256.Sum256(data)
	computed := hex.EncodeToString(hash[:])
	if computed != v.Checksum {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, v.Checksum, computed)
	}
	return nil
}

// ComputeChecksum calculates the SHA256 checksum of data.
func ComputeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Package represents a distributable unit in the marketplace.
type Package struct {
	ID          PackageID        `json:"id" yaml:"id"`
	Type        string           `json:"type" yaml:"type"` // preset, capability-pack, layer-template
	Title       string           `json:"title" yaml:"title"`
	Description string           `json:"description" yaml:"description"`
	Keywords    []string         `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Provenance  Provenance       `json:"provenance" yaml:"provenance"`
	Versions    []PackageVersion `json:"versions" yaml:"versions"`
	Downloads   int              `json:"downloads" yaml:"downloads"`
	Stars       int              `json:"stars" yaml:"stars"`
	CreatedAt   time.Time        `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" yaml:"updated_at"`
}

// LatestVersion returns the most recent version.
func (p Package) LatestVersion() (PackageVersion, bool) {
	if len(p.Versions) == 0 {
		return PackageVersion{}, false
	}
	return p.Versions[0], true
}

// GetVersion returns a specific version by version string.
func (p Package) GetVersion(version string) (PackageVersion, bool) {
	for _, v := range p.Versions {
		if v.Version == version {
			return v, true
		}
	}
	return PackageVersion{}, false
}

// IsValid returns true if the package has required fields.
func (p Package) IsValid() bool {
	if p.ID.IsZero() {
		return false
	}
	if p.Type == "" || !isValidPackageType(p.Type) {
		return false
	}
	if p.Title == "" {
		return false
	}
	if len(p.Versions) == 0 {
		return false
	}
	return true
}

// MatchesQuery returns true if the package matches the search query.
func (p Package) MatchesQuery(query string) bool {
	query = strings.ToLower(query)

	// Check ID
	if strings.Contains(strings.ToLower(p.ID.String()), query) {
		return true
	}

	// Check title
	if strings.Contains(strings.ToLower(p.Title), query) {
		return true
	}

	// Check description
	if strings.Contains(strings.ToLower(p.Description), query) {
		return true
	}

	// Check keywords
	for _, kw := range p.Keywords {
		if strings.Contains(strings.ToLower(kw), query) {
			return true
		}
	}

	// Check author
	if strings.Contains(strings.ToLower(p.Provenance.Author), query) {
		return true
	}

	return false
}

func isValidPackageType(t string) bool {
	switch t {
	case PackageTypePreset, PackageTypeCapabilityPack, PackageTypeLayerTemplate:
		return true
	default:
		return false
	}
}

// InstalledPackage represents a package that has been installed locally.
type InstalledPackage struct {
	Package     Package   `json:"package" yaml:"package"`
	Version     string    `json:"version" yaml:"version"`
	InstalledAt time.Time `json:"installed_at" yaml:"installed_at"`
	Path        string    `json:"path" yaml:"path"`
	AutoUpdate  bool      `json:"auto_update" yaml:"auto_update"`
}

// NeedsUpdate returns true if a newer version is available.
func (ip InstalledPackage) NeedsUpdate() bool {
	latest, ok := ip.Package.LatestVersion()
	if !ok {
		return false
	}
	return latest.Version != ip.Version
}
