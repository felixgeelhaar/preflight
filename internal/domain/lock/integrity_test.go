package lock

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIntegrity_ValidSHA256(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, err := NewIntegrity("sha256", hash)

	require.NoError(t, err)
	assert.Equal(t, "sha256", integrity.Algorithm())
	assert.Equal(t, hash, integrity.Hash())
}

func TestNewIntegrity_ValidSHA512(t *testing.T) {
	t.Parallel()

	hash := "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"
	integrity, err := NewIntegrity("sha512", hash)

	require.NoError(t, err)
	assert.Equal(t, "sha512", integrity.Algorithm())
	assert.Equal(t, hash, integrity.Hash())
}

func TestNewIntegrity_UnsupportedAlgorithm(t *testing.T) {
	t.Parallel()

	_, err := NewIntegrity("md5", "d41d8cd98f00b204e9800998ecf8427e")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedAlgorithm)
}

func TestNewIntegrity_EmptyHash(t *testing.T) {
	t.Parallel()

	_, err := NewIntegrity("sha256", "")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyHash)
}

func TestNewIntegrity_InvalidHexHash(t *testing.T) {
	t.Parallel()

	_, err := NewIntegrity("sha256", "not-valid-hex!")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidHash)
}

func TestNewIntegrity_WrongLengthHash(t *testing.T) {
	t.Parallel()

	// SHA256 should be 64 hex chars, this is too short
	_, err := NewIntegrity("sha256", "e3b0c44298fc1c149afbf4c8996fb924")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidHash)
}

func TestIntegrity_Verify_Success(t *testing.T) {
	t.Parallel()

	data := []byte("hello world")
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	integrity, err := NewIntegrity("sha256", hashStr)
	require.NoError(t, err)

	assert.True(t, integrity.Verify(data))
}

func TestIntegrity_Verify_Failure(t *testing.T) {
	t.Parallel()

	data := []byte("hello world")
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	integrity, err := NewIntegrity("sha256", hashStr)
	require.NoError(t, err)

	differentData := []byte("different data")
	assert.False(t, integrity.Verify(differentData))
}

func TestIntegrity_String(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	integrity, err := NewIntegrity("sha256", hash)
	require.NoError(t, err)

	expected := "sha256:" + hash
	assert.Equal(t, expected, integrity.String())
}

func TestIntegrity_IsZero(t *testing.T) {
	t.Parallel()

	var zero Integrity
	assert.True(t, zero.IsZero())

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	nonZero, _ := NewIntegrity("sha256", hash)
	assert.False(t, nonZero.IsZero())
}

func TestIntegrityFromData(t *testing.T) {
	t.Parallel()

	data := []byte("hello world")
	integrity := IntegrityFromData("sha256", data)

	assert.Equal(t, "sha256", integrity.Algorithm())
	assert.True(t, integrity.Verify(data))
}

func TestParseIntegrity_Valid(t *testing.T) {
	t.Parallel()

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	input := "sha256:" + hash

	integrity, err := ParseIntegrity(input)
	require.NoError(t, err)
	assert.Equal(t, "sha256", integrity.Algorithm())
	assert.Equal(t, hash, integrity.Hash())
}

func TestParseIntegrity_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"no colon", "sha256abc123"},
		{"empty hash", "sha256:"},
		{"empty algorithm", ":abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseIntegrity(tt.input)
			require.Error(t, err)
		})
	}
}
