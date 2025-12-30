package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Tag
		wantErr bool
	}{
		{
			name:  "simple tag",
			input: "darwin",
			want:  Tag("darwin"),
		},
		{
			name:  "tag with hyphen",
			input: "arm-64",
			want:  Tag("arm-64"),
		},
		{
			name:  "tag with numbers",
			input: "node18",
			want:  Tag("node18"),
		},
		{
			name:  "uppercase converted to lowercase",
			input: "Darwin",
			want:  Tag("darwin"),
		},
		{
			name:  "whitespace trimmed",
			input: "  production  ",
			want:  Tag("production"),
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "starts with number",
			input:   "123abc",
			wantErr: true,
		},
		{
			name:    "contains underscore",
			input:   "my_tag",
			wantErr: true,
		},
		{
			name:    "contains space",
			input:   "my tag",
			wantErr: true,
		},
		{
			name:    "ends with hyphen",
			input:   "tag-",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewTag(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMustTag(t *testing.T) {
	t.Parallel()

	t.Run("valid tag", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() {
			tag := MustTag("production")
			assert.Equal(t, Tag("production"), tag)
		})
	})

	t.Run("invalid tag panics", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			MustTag("")
		})
	})
}

func TestTag_String(t *testing.T) {
	t.Parallel()
	tag := Tag("darwin")
	assert.Equal(t, "darwin", tag.String())
}

func TestNewTags(t *testing.T) {
	t.Parallel()

	t.Run("creates tags from strings", func(t *testing.T) {
		t.Parallel()
		tags, err := NewTags("darwin", "production", "arm64")
		require.NoError(t, err)
		assert.Len(t, tags, 3)
		assert.Contains(t, tags, Tag("darwin"))
		assert.Contains(t, tags, Tag("production"))
		assert.Contains(t, tags, Tag("arm64"))
	})

	t.Run("deduplicates tags", func(t *testing.T) {
		t.Parallel()
		tags, err := NewTags("darwin", "darwin", "linux")
		require.NoError(t, err)
		assert.Len(t, tags, 2)
	})

	t.Run("returns error for invalid tag", func(t *testing.T) {
		t.Parallel()
		_, err := NewTags("valid", "in valid")
		assert.Error(t, err)
	})

	t.Run("empty input returns empty tags", func(t *testing.T) {
		t.Parallel()
		tags, err := NewTags()
		require.NoError(t, err)
		assert.Empty(t, tags)
	})
}

func TestTags_Contains(t *testing.T) {
	t.Parallel()
	tags, _ := NewTags("darwin", "production")

	assert.True(t, tags.Contains(Tag("darwin")))
	assert.True(t, tags.Contains(Tag("production")))
	assert.False(t, tags.Contains(Tag("linux")))
}

func TestTags_ContainsAny(t *testing.T) {
	t.Parallel()
	tags, _ := NewTags("darwin", "production")

	t.Run("contains one", func(t *testing.T) {
		t.Parallel()
		other, _ := NewTags("linux", "darwin")
		assert.True(t, tags.ContainsAny(other))
	})

	t.Run("contains none", func(t *testing.T) {
		t.Parallel()
		other, _ := NewTags("linux", "staging")
		assert.False(t, tags.ContainsAny(other))
	})

	t.Run("empty other returns false", func(t *testing.T) {
		t.Parallel()
		assert.False(t, tags.ContainsAny(Tags{}))
	})
}

func TestTags_ContainsAll(t *testing.T) {
	t.Parallel()
	tags, _ := NewTags("darwin", "production", "arm64")

	t.Run("contains all", func(t *testing.T) {
		t.Parallel()
		other, _ := NewTags("darwin", "production")
		assert.True(t, tags.ContainsAll(other))
	})

	t.Run("missing one", func(t *testing.T) {
		t.Parallel()
		other, _ := NewTags("darwin", "linux")
		assert.False(t, tags.ContainsAll(other))
	})

	t.Run("empty other returns true", func(t *testing.T) {
		t.Parallel()
		assert.True(t, tags.ContainsAll(Tags{}))
	})
}

func TestTags_Strings(t *testing.T) {
	t.Parallel()
	tags, _ := NewTags("darwin", "production")
	strings := tags.Strings()

	assert.Len(t, strings, 2)
	assert.Contains(t, strings, "darwin")
	assert.Contains(t, strings, "production")
}

func TestTags_Union(t *testing.T) {
	t.Parallel()
	tags1, _ := NewTags("darwin", "production")
	tags2, _ := NewTags("linux", "production")

	union := tags1.Union(tags2)

	assert.Len(t, union, 3)
	assert.True(t, union.Contains(Tag("darwin")))
	assert.True(t, union.Contains(Tag("linux")))
	assert.True(t, union.Contains(Tag("production")))
}

func TestTags_Intersect(t *testing.T) {
	t.Parallel()
	tags1, _ := NewTags("darwin", "production", "arm64")
	tags2, _ := NewTags("linux", "production", "arm64")

	intersect := tags1.Intersect(tags2)

	assert.Len(t, intersect, 2)
	assert.True(t, intersect.Contains(Tag("production")))
	assert.True(t, intersect.Contains(Tag("arm64")))
	assert.False(t, intersect.Contains(Tag("darwin")))
}
