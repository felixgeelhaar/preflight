package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetadata_Valid(t *testing.T) {
	t.Parallel()

	meta, err := NewMetadata("Balanced Neovim", "A well-balanced Neovim configuration")

	require.NoError(t, err)
	assert.Equal(t, "Balanced Neovim", meta.Title())
	assert.Equal(t, "A well-balanced Neovim configuration", meta.Description())
	assert.Empty(t, meta.Tradeoffs())
	assert.Empty(t, meta.DocLinks())
}

func TestNewMetadata_EmptyTitle(t *testing.T) {
	t.Parallel()

	_, err := NewMetadata("", "Some description")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyTitle)
}

func TestNewMetadata_EmptyDescription(t *testing.T) {
	t.Parallel()

	_, err := NewMetadata("Some Title", "")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyDescription)
}

func TestMetadata_WithTradeoffs(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Balanced Neovim", "A well-balanced configuration")
	tradeoffs := []string{
		"More plugins mean longer startup time",
		"Requires Node.js for some LSP features",
	}

	updated := meta.WithTradeoffs(tradeoffs)

	// Original unchanged
	assert.Empty(t, meta.Tradeoffs())

	// New has tradeoffs
	assert.Equal(t, tradeoffs, updated.Tradeoffs())
}

func TestMetadata_WithDocLinks(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Balanced Neovim", "A well-balanced configuration")
	links := map[string]string{
		"Neovim":  "https://neovim.io",
		"LazyVim": "https://www.lazyvim.org",
	}

	updated := meta.WithDocLinks(links)

	// Original unchanged
	assert.Empty(t, meta.DocLinks())

	// New has doc links
	assert.Equal(t, links, updated.DocLinks())
}

func TestMetadata_WithTags(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Balanced Neovim", "A well-balanced configuration")
	tags := []string{"editor", "vim", "productivity"}

	updated := meta.WithTags(tags)

	assert.Empty(t, meta.Tags())
	assert.Equal(t, tags, updated.Tags())
}

func TestMetadata_HasTag(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Balanced Neovim", "A well-balanced configuration")
	meta = meta.WithTags([]string{"editor", "vim", "productivity"})

	assert.True(t, meta.HasTag("editor"))
	assert.True(t, meta.HasTag("vim"))
	assert.False(t, meta.HasTag("shell"))
}

func TestMetadata_IsZero(t *testing.T) {
	t.Parallel()

	var zero Metadata
	assert.True(t, zero.IsZero())

	nonZero, _ := NewMetadata("Title", "Description")
	assert.False(t, nonZero.IsZero())
}

func TestMetadata_Immutability(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Title", "Description")

	// Add tradeoffs
	tradeoffs := []string{"tradeoff1"}
	meta1 := meta.WithTradeoffs(tradeoffs)

	// Modify original slice
	tradeoffs[0] = "modified"

	// meta1 should not be affected
	assert.Equal(t, "tradeoff1", meta1.Tradeoffs()[0])
}

func TestMetadata_String(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Balanced Neovim", "A well-balanced configuration")

	assert.Equal(t, "Balanced Neovim: A well-balanced configuration", meta.String())
}
