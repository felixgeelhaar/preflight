package targeting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		wantType PatternType
		wantErr  bool
	}{
		{
			name:     "literal pattern",
			raw:      "server01",
			wantType: PatternTypeLiteral,
		},
		{
			name:     "glob wildcard",
			raw:      "server-*",
			wantType: PatternTypeGlob,
		},
		{
			name:     "glob question mark",
			raw:      "server-0?",
			wantType: PatternTypeGlob,
		},
		{
			name:     "glob bracket",
			raw:      "server-[0-9]",
			wantType: PatternTypeGlob,
		},
		{
			name:     "regex pattern",
			raw:      "~server-\\d+",
			wantType: PatternTypeRegex,
		},
		{
			name:    "empty pattern",
			raw:     "",
			wantErr: true,
		},
		{
			name:    "invalid regex",
			raw:     "~[invalid",
			wantErr: true,
		},
		{
			name:    "invalid glob",
			raw:     "[invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := NewPattern(tt.raw)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, p.Type())
			assert.Equal(t, tt.raw, p.Raw())
		})
	}
}

func TestPattern_Match_Literal(t *testing.T) {
	t.Parallel()

	p, _ := NewPattern("server01")

	assert.True(t, p.Match("server01"))
	assert.False(t, p.Match("server02"))
	assert.False(t, p.Match("SERVER01"))
	assert.False(t, p.Match("server01-extra"))
}

func TestPattern_Match_Glob(t *testing.T) {
	t.Parallel()

	t.Run("asterisk wildcard", func(t *testing.T) {
		t.Parallel()
		p, _ := NewPattern("server-*")
		assert.True(t, p.Match("server-01"))
		assert.True(t, p.Match("server-prod"))
		assert.True(t, p.Match("server-"))
		assert.False(t, p.Match("server01"))
		assert.False(t, p.Match("other-01"))
	})

	t.Run("question mark wildcard", func(t *testing.T) {
		t.Parallel()
		p, _ := NewPattern("server-0?")
		assert.True(t, p.Match("server-01"))
		assert.True(t, p.Match("server-09"))
		assert.False(t, p.Match("server-001"))
		assert.False(t, p.Match("server-0"))
	})

	t.Run("bracket pattern", func(t *testing.T) {
		t.Parallel()
		p, _ := NewPattern("server-[0-2]")
		assert.True(t, p.Match("server-0"))
		assert.True(t, p.Match("server-1"))
		assert.True(t, p.Match("server-2"))
		assert.False(t, p.Match("server-3"))
	})

	t.Run("complex glob", func(t *testing.T) {
		t.Parallel()
		p, _ := NewPattern("web-*.prod")
		assert.True(t, p.Match("web-01.prod"))
		assert.True(t, p.Match("web-server.prod"))
		assert.False(t, p.Match("web-01.staging"))
	})
}

func TestPattern_Match_Regex(t *testing.T) {
	t.Parallel()

	t.Run("digit pattern", func(t *testing.T) {
		t.Parallel()
		p, _ := NewPattern("~server-\\d+")
		assert.True(t, p.Match("server-01"))
		assert.True(t, p.Match("server-999"))
		assert.False(t, p.Match("server-abc"))
	})

	t.Run("word boundary", func(t *testing.T) {
		t.Parallel()
		p, _ := NewPattern("~^web-[a-z]+$")
		assert.True(t, p.Match("web-server"))
		assert.True(t, p.Match("web-app"))
		assert.False(t, p.Match("web-01"))
		assert.False(t, p.Match("web-server-01"))
	})

	t.Run("alternation", func(t *testing.T) {
		t.Parallel()
		p, _ := NewPattern("~^(web|api)-")
		assert.True(t, p.Match("web-01"))
		assert.True(t, p.Match("api-server"))
		assert.False(t, p.Match("db-01"))
	})
}

func TestPattern_MatchAny(t *testing.T) {
	t.Parallel()

	p, _ := NewPattern("server-*")

	assert.True(t, p.MatchAny([]string{"db-01", "server-01", "cache-01"}))
	assert.False(t, p.MatchAny([]string{"db-01", "cache-01"}))
	assert.False(t, p.MatchAny([]string{}))
}

func TestNewPatterns(t *testing.T) {
	t.Parallel()

	t.Run("valid patterns", func(t *testing.T) {
		t.Parallel()
		ps, err := NewPatterns("server-*", "db-*", "cache-01")
		require.NoError(t, err)
		assert.Len(t, ps, 3)
	})

	t.Run("invalid pattern", func(t *testing.T) {
		t.Parallel()
		_, err := NewPatterns("valid", "~[invalid")
		assert.Error(t, err)
	})
}

func TestPatterns_MatchAny(t *testing.T) {
	t.Parallel()

	ps, _ := NewPatterns("web-*", "api-*")

	assert.True(t, ps.MatchAny("web-01"))
	assert.True(t, ps.MatchAny("api-server"))
	assert.False(t, ps.MatchAny("db-01"))
}

func TestPatterns_MatchAll(t *testing.T) {
	t.Parallel()

	// Patterns that could overlap
	ps, _ := NewPatterns("~.*-01$", "web-*")

	assert.True(t, ps.MatchAll("web-01"))
	assert.False(t, ps.MatchAll("web-02")) // Doesn't match first pattern
	assert.False(t, ps.MatchAll("api-01")) // Doesn't match second pattern
}

func TestPatterns_FilterMatching(t *testing.T) {
	t.Parallel()

	ps, _ := NewPatterns("web-*", "api-*")
	input := []string{"web-01", "api-server", "db-01", "web-02", "cache-01"}

	result := ps.FilterMatching(input)

	assert.Len(t, result, 3)
	assert.Contains(t, result, "web-01")
	assert.Contains(t, result, "web-02")
	assert.Contains(t, result, "api-server")
}
