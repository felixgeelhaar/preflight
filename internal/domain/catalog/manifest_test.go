package catalog

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestBuilder(t *testing.T) {
	t.Parallel()

	t.Run("minimal manifest", func(t *testing.T) {
		t.Parallel()
		m, err := NewManifestBuilder("test-catalog").Build()
		require.NoError(t, err)
		assert.Equal(t, "test-catalog", m.Name())
		assert.Equal(t, "1.0", m.Version())
		assert.Equal(t, HashAlgorithmSHA256, m.Algorithm())
	})

	t.Run("full manifest", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		m, err := NewManifestBuilder("full-catalog").
			WithVersion("2.0").
			WithDescription("A test catalog").
			WithAuthor("Test Author <test@example.com>").
			WithRepository("https://github.com/test/catalog").
			WithLicense("MIT").
			AddFile("catalog.yaml", "sha256:abc123").
			AddFile("presets/base.yaml", "sha256:def456").
			WithCreatedAt(now).
			WithUpdatedAt(now).
			Build()

		require.NoError(t, err)
		assert.Equal(t, "full-catalog", m.Name())
		assert.Equal(t, "2.0", m.Version())
		assert.Equal(t, "A test catalog", m.Description())
		assert.Equal(t, "Test Author <test@example.com>", m.Author())
		assert.Equal(t, "https://github.com/test/catalog", m.Repository())
		assert.Equal(t, "MIT", m.License())
		assert.Len(t, m.Files(), 2)
	})

	t.Run("empty name fails", func(t *testing.T) {
		t.Parallel()
		_, err := NewManifestBuilder("").Build()
		assert.ErrorIs(t, err, ErrInvalidManifest)
	})

	t.Run("unsupported algorithm fails", func(t *testing.T) {
		t.Parallel()
		_, err := NewManifestBuilder("test").
			WithAlgorithm("md5").
			Build()
		assert.ErrorIs(t, err, ErrUnsupportedAlgorithm)
	})
}

func TestManifest_GetFileHash(t *testing.T) {
	t.Parallel()

	m, _ := NewManifestBuilder("test").
		AddFile("catalog.yaml", "sha256:abc123").
		Build()

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()
		hash, ok := m.GetFileHash("catalog.yaml")
		assert.True(t, ok)
		assert.Equal(t, "sha256:abc123", hash)
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		_, ok := m.GetFileHash("missing.yaml")
		assert.False(t, ok)
	})
}

func TestManifest_HasFile(t *testing.T) {
	t.Parallel()

	m, _ := NewManifestBuilder("test").
		AddFile("catalog.yaml", "sha256:abc123").
		Build()

	assert.True(t, m.HasFile("catalog.yaml"))
	assert.False(t, m.HasFile("missing.yaml"))
}

func TestManifest_VerifyFile(t *testing.T) {
	t.Parallel()

	content := []byte("test content")
	hash := ComputeSHA256(content)

	m, _ := NewManifestBuilder("test").
		AddFile("test.yaml", hash).
		Build()

	t.Run("valid content", func(t *testing.T) {
		t.Parallel()
		err := m.VerifyFile("test.yaml", content)
		assert.NoError(t, err)
	})

	t.Run("invalid content", func(t *testing.T) {
		t.Parallel()
		err := m.VerifyFile("test.yaml", []byte("wrong content"))
		assert.ErrorIs(t, err, ErrIntegrityMismatch)
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		err := m.VerifyFile("missing.yaml", content)
		assert.ErrorIs(t, err, ErrMissingFile)
	})
}

func TestManifest_IsZero(t *testing.T) {
	t.Parallel()

	var zero Manifest
	assert.True(t, zero.IsZero())

	m, _ := NewManifestBuilder("test").Build()
	assert.False(t, m.IsZero())
}

func TestComputeSHA256(t *testing.T) {
	t.Parallel()

	hash := ComputeSHA256([]byte("hello world"))
	assert.True(t, strings.HasPrefix(hash, "sha256:"))
	assert.Len(t, hash, 7+64) // "sha256:" + 64 hex chars
}

func TestComputeSHA256Reader(t *testing.T) {
	t.Parallel()

	reader := strings.NewReader("hello world")
	hash, err := ComputeSHA256Reader(reader)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(hash, "sha256:"))

	// Should match direct computation
	direct := ComputeSHA256([]byte("hello world"))
	assert.Equal(t, direct, hash)
}

func TestValidateHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		hash    string
		wantErr bool
	}{
		{"valid sha256", "sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", false},
		{"too short", "abc", true},
		{"no colon", "sha256abc123", true},
		{"wrong algorithm", "md5:d41d8cd98f00b204e9800998ecf8427e", true},
		{"wrong length", "sha256:abc123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateHash(tt.hash)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
