// Package lock provides lockfile management for reproducible builds.
package lock

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// Integrity errors.
var (
	ErrUnsupportedAlgorithm = errors.New("unsupported hash algorithm")
	ErrEmptyHash            = errors.New("hash cannot be empty")
	ErrInvalidHash          = errors.New("invalid hash format")
)

// Supported hash algorithms.
const (
	AlgorithmSHA256 = "sha256"
	AlgorithmSHA512 = "sha512"
)

// hashLengths maps algorithm to expected hex string length.
var hashLengths = map[string]int{
	AlgorithmSHA256: 64,  // 256 bits = 32 bytes = 64 hex chars
	AlgorithmSHA512: 128, // 512 bits = 64 bytes = 128 hex chars
}

// Integrity represents a content integrity hash.
// It is an immutable value object.
type Integrity struct {
	algorithm string
	hash      string
}

// NewIntegrity creates a new Integrity value object.
// Returns an error if the algorithm is unsupported or the hash is invalid.
func NewIntegrity(algorithm, hash string) (Integrity, error) {
	expectedLen, ok := hashLengths[algorithm]
	if !ok {
		return Integrity{}, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, algorithm)
	}

	if hash == "" {
		return Integrity{}, ErrEmptyHash
	}

	// Validate hex encoding
	if _, err := hex.DecodeString(hash); err != nil {
		return Integrity{}, fmt.Errorf("%w: invalid hex encoding", ErrInvalidHash)
	}

	// Validate length
	if len(hash) != expectedLen {
		return Integrity{}, fmt.Errorf("%w: expected %d chars for %s, got %d",
			ErrInvalidHash, expectedLen, algorithm, len(hash))
	}

	return Integrity{
		algorithm: algorithm,
		hash:      hash,
	}, nil
}

// IntegrityFromData computes an integrity hash from data.
func IntegrityFromData(algorithm string, data []byte) Integrity {
	var hash string

	switch algorithm {
	case AlgorithmSHA256:
		h := sha256.Sum256(data)
		hash = hex.EncodeToString(h[:])
	case AlgorithmSHA512:
		h := sha512.Sum512(data)
		hash = hex.EncodeToString(h[:])
	default:
		// Default to SHA256 if unknown algorithm
		h := sha256.Sum256(data)
		hash = hex.EncodeToString(h[:])
		algorithm = AlgorithmSHA256
	}

	return Integrity{
		algorithm: algorithm,
		hash:      hash,
	}
}

// ParseIntegrity parses an integrity string in the format "algorithm:hash".
func ParseIntegrity(s string) (Integrity, error) {
	if s == "" {
		return Integrity{}, fmt.Errorf("%w: empty string", ErrInvalidHash)
	}

	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return Integrity{}, fmt.Errorf("%w: missing colon separator", ErrInvalidHash)
	}

	if parts[0] == "" {
		return Integrity{}, fmt.Errorf("%w: empty algorithm", ErrInvalidHash)
	}

	if parts[1] == "" {
		return Integrity{}, fmt.Errorf("%w: empty hash", ErrInvalidHash)
	}

	return NewIntegrity(parts[0], parts[1])
}

// Algorithm returns the hash algorithm.
func (i Integrity) Algorithm() string {
	return i.algorithm
}

// Hash returns the hex-encoded hash value.
func (i Integrity) Hash() string {
	return i.hash
}

// String returns the integrity in "algorithm:hash" format.
func (i Integrity) String() string {
	return i.algorithm + ":" + i.hash
}

// IsZero returns true if this is a zero-value Integrity.
func (i Integrity) IsZero() bool {
	return i.algorithm == "" && i.hash == ""
}

// Verify checks if the given data matches this integrity hash.
func (i Integrity) Verify(data []byte) bool {
	if i.IsZero() {
		return false
	}

	computed := IntegrityFromData(i.algorithm, data)
	return computed.hash == i.hash
}
