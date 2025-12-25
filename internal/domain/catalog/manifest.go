package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"
)

// Manifest errors.
var (
	ErrInvalidManifest      = errors.New("invalid catalog manifest")
	ErrIntegrityMismatch    = errors.New("integrity hash mismatch")
	ErrMissingFile          = errors.New("file missing from manifest")
	ErrUnsupportedAlgorithm = errors.New("unsupported hash algorithm")
)

// HashAlgorithm represents a supported hash algorithm.
type HashAlgorithm string

// HashAlgorithm constants.
const (
	HashAlgorithmSHA256 HashAlgorithm = "sha256"
)

// FileHash represents a file and its integrity hash.
type FileHash struct {
	Path string `yaml:"path" json:"path"`
	Hash string `yaml:"hash" json:"hash"`
}

// Manifest represents a catalog manifest with integrity information.
// The manifest is used to verify that catalog files have not been tampered with.
type Manifest struct {
	version     string
	name        string
	description string
	author      string
	repository  string
	license     string
	algorithm   HashAlgorithm
	files       []FileHash
	createdAt   time.Time
	updatedAt   time.Time
}

// ManifestBuilder builds a Manifest.
type ManifestBuilder struct {
	manifest Manifest
}

// NewManifestBuilder creates a new ManifestBuilder.
func NewManifestBuilder(name string) *ManifestBuilder {
	return &ManifestBuilder{
		manifest: Manifest{
			version:   "1.0",
			name:      name,
			algorithm: HashAlgorithmSHA256,
			createdAt: time.Now(),
			updatedAt: time.Now(),
		},
	}
}

// WithVersion sets the manifest version.
func (b *ManifestBuilder) WithVersion(version string) *ManifestBuilder {
	b.manifest.version = version
	return b
}

// WithDescription sets the description.
func (b *ManifestBuilder) WithDescription(description string) *ManifestBuilder {
	b.manifest.description = description
	return b
}

// WithAuthor sets the author.
func (b *ManifestBuilder) WithAuthor(author string) *ManifestBuilder {
	b.manifest.author = author
	return b
}

// WithRepository sets the repository URL.
func (b *ManifestBuilder) WithRepository(repository string) *ManifestBuilder {
	b.manifest.repository = repository
	return b
}

// WithLicense sets the license.
func (b *ManifestBuilder) WithLicense(license string) *ManifestBuilder {
	b.manifest.license = license
	return b
}

// WithAlgorithm sets the hash algorithm.
func (b *ManifestBuilder) WithAlgorithm(algorithm HashAlgorithm) *ManifestBuilder {
	b.manifest.algorithm = algorithm
	return b
}

// WithFiles sets the file hashes.
func (b *ManifestBuilder) WithFiles(files []FileHash) *ManifestBuilder {
	b.manifest.files = files
	return b
}

// AddFile adds a file hash.
func (b *ManifestBuilder) AddFile(path, hash string) *ManifestBuilder {
	b.manifest.files = append(b.manifest.files, FileHash{Path: path, Hash: hash})
	return b
}

// WithCreatedAt sets the creation time.
func (b *ManifestBuilder) WithCreatedAt(t time.Time) *ManifestBuilder {
	b.manifest.createdAt = t
	return b
}

// WithUpdatedAt sets the update time.
func (b *ManifestBuilder) WithUpdatedAt(t time.Time) *ManifestBuilder {
	b.manifest.updatedAt = t
	return b
}

// Build creates the Manifest.
func (b *ManifestBuilder) Build() (Manifest, error) {
	if b.manifest.name == "" {
		return Manifest{}, fmt.Errorf("%w: name is required", ErrInvalidManifest)
	}

	if b.manifest.algorithm != HashAlgorithmSHA256 {
		return Manifest{}, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, b.manifest.algorithm)
	}

	return b.manifest, nil
}

// Version returns the manifest version.
func (m Manifest) Version() string {
	return m.version
}

// Name returns the catalog name.
func (m Manifest) Name() string {
	return m.name
}

// Description returns the catalog description.
func (m Manifest) Description() string {
	return m.description
}

// Author returns the catalog author.
func (m Manifest) Author() string {
	return m.author
}

// Repository returns the source repository URL.
func (m Manifest) Repository() string {
	return m.repository
}

// License returns the license.
func (m Manifest) License() string {
	return m.license
}

// Algorithm returns the hash algorithm.
func (m Manifest) Algorithm() HashAlgorithm {
	return m.algorithm
}

// Files returns the file hashes.
func (m Manifest) Files() []FileHash {
	result := make([]FileHash, len(m.files))
	copy(result, m.files)
	return result
}

// CreatedAt returns the creation time.
func (m Manifest) CreatedAt() time.Time {
	return m.createdAt
}

// UpdatedAt returns the update time.
func (m Manifest) UpdatedAt() time.Time {
	return m.updatedAt
}

// IsZero returns true if the manifest is empty.
func (m Manifest) IsZero() bool {
	return m.name == "" && len(m.files) == 0
}

// GetFileHash returns the hash for a specific file.
func (m Manifest) GetFileHash(path string) (string, bool) {
	for _, f := range m.files {
		if f.Path == path {
			return f.Hash, true
		}
	}
	return "", false
}

// HasFile returns true if the file is in the manifest.
func (m Manifest) HasFile(path string) bool {
	_, ok := m.GetFileHash(path)
	return ok
}

// VerifyFile verifies that content matches the expected hash for a file.
func (m Manifest) VerifyFile(path string, content []byte) error {
	expected, ok := m.GetFileHash(path)
	if !ok {
		return fmt.Errorf("%w: %s", ErrMissingFile, path)
	}

	actual := ComputeSHA256(content)
	if actual != expected {
		return fmt.Errorf("%w: %s (expected %s, got %s)", ErrIntegrityMismatch, path, expected, actual)
	}

	return nil
}

// ComputeSHA256 computes the SHA256 hash of data.
func ComputeSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}

// ComputeSHA256Reader computes the SHA256 hash from a reader.
func ComputeSHA256Reader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// ValidateHash validates that a hash string is properly formatted.
func ValidateHash(hash string) error {
	if len(hash) < 8 {
		return fmt.Errorf("%w: hash too short", ErrIntegrityMismatch)
	}

	parts := splitHash(hash)
	if len(parts) != 2 {
		return fmt.Errorf("%w: invalid hash format (expected algo:hash)", ErrIntegrityMismatch)
	}

	if parts[0] != "sha256" {
		return fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, parts[0])
	}

	if len(parts[1]) != 64 {
		return fmt.Errorf("%w: invalid SHA256 hash length", ErrIntegrityMismatch)
	}

	return nil
}

func splitHash(hash string) []string {
	for i, c := range hash {
		if c == ':' {
			return []string{hash[:i], hash[i+1:]}
		}
	}
	return []string{hash}
}
