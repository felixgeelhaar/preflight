package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewURLSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		srcName string
		url     string
		wantErr bool
	}{
		{"valid https", "test", "https://example.com/catalog", false},
		{"valid http", "test", "http://example.com/catalog", false},
		{"empty name", "", "https://example.com", true},
		{"empty url", "test", "", true},
		{"invalid scheme", "test", "ftp://example.com", true},
		{"no host", "test", "https:///path", true},
		{"invalid url", "test", "://invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src, err := NewURLSource(tt.srcName, tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidSource)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.srcName, src.Name())
				assert.Equal(t, tt.url, src.Location())
				assert.Equal(t, SourceTypeURL, src.Type())
				assert.True(t, src.IsURL())
				assert.False(t, src.IsLocal())
				assert.False(t, src.IsBuiltin())
			}
		})
	}
}

func TestNewLocalSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		srcName string
		path    string
		wantErr bool
	}{
		{"valid path", "local", "/path/to/catalog", false},
		{"valid relative", "local", "./catalog", false},
		{"empty name", "", "/path", true},
		{"empty path", "local", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src, err := NewLocalSource(tt.srcName, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.srcName, src.Name())
				assert.Equal(t, SourceTypeLocal, src.Type())
				assert.True(t, src.IsLocal())
				assert.False(t, src.IsURL())
			}
		})
	}
}

func TestNewBuiltinSource(t *testing.T) {
	t.Parallel()

	src := NewBuiltinSource()
	assert.Equal(t, "builtin", src.Name())
	assert.Equal(t, "embedded", src.Location())
	assert.Equal(t, SourceTypeBuiltin, src.Type())
	assert.True(t, src.IsBuiltin())
	assert.False(t, src.IsURL())
	assert.False(t, src.IsLocal())
}

func TestSource_ManifestURL(t *testing.T) {
	t.Parallel()

	t.Run("url source", func(t *testing.T) {
		t.Parallel()
		src, _ := NewURLSource("test", "https://example.com/catalog")
		assert.Equal(t, "https://example.com/catalog/catalog-manifest.yaml", src.ManifestURL())
	})

	t.Run("url source with trailing slash", func(t *testing.T) {
		t.Parallel()
		src, _ := NewURLSource("test", "https://example.com/catalog/")
		assert.Equal(t, "https://example.com/catalog/catalog-manifest.yaml", src.ManifestURL())
	})

	t.Run("local source", func(t *testing.T) {
		t.Parallel()
		src, _ := NewLocalSource("test", "/path/to/catalog")
		assert.Contains(t, src.ManifestURL(), "catalog-manifest.yaml")
	})

	t.Run("builtin source", func(t *testing.T) {
		t.Parallel()
		src := NewBuiltinSource()
		assert.Empty(t, src.ManifestURL())
	})
}

func TestSource_CatalogURL(t *testing.T) {
	t.Parallel()

	src, _ := NewURLSource("test", "https://example.com/catalog")
	assert.Equal(t, "https://example.com/catalog/catalog.yaml", src.CatalogURL())
}

func TestSource_IsZero(t *testing.T) {
	t.Parallel()

	var zero Source
	assert.True(t, zero.IsZero())

	src, _ := NewURLSource("test", "https://example.com")
	assert.False(t, src.IsZero())
}

func TestSource_String(t *testing.T) {
	t.Parallel()

	src, _ := NewURLSource("my-catalog", "https://example.com")
	str := src.String()
	assert.Contains(t, str, "my-catalog")
	assert.Contains(t, str, "url")
}
