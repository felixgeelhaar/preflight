package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSourceRegistry(t *testing.T) {
	t.Parallel()

	reg := NewSourceRegistry()

	assert.NotNil(t, reg)
	assert.Empty(t, reg.Available())
	assert.Empty(t, reg.Names())
}

func TestSourceRegistry_Register(t *testing.T) {
	t.Parallel()

	reg := NewSourceRegistry()
	source := newMockSource(t, "aws", true)

	reg.Register(source)

	assert.Len(t, reg.Names(), 1)
	assert.Equal(t, []string{"aws"}, reg.Names())
}

func TestSourceRegistry_RegisterMultiple(t *testing.T) {
	t.Parallel()

	reg := NewSourceRegistry()
	reg.Register(newMockSource(t, "aws", true))
	reg.Register(newMockSource(t, "gcp", false))
	reg.Register(newMockSource(t, "azure", true))

	assert.Len(t, reg.Names(), 3)
	assert.Equal(t, []string{"aws", "gcp", "azure"}, reg.Names())
}

func TestSourceRegistry_Available(t *testing.T) {
	t.Parallel()

	reg := NewSourceRegistry()
	reg.Register(newMockSource(t, "aws", true))
	reg.Register(newMockSource(t, "gcp", false))
	reg.Register(newMockSource(t, "azure", true))

	available := reg.Available()
	assert.Len(t, available, 2)
	assert.Equal(t, "aws", available[0].Name())
	assert.Equal(t, "azure", available[1].Name())
}

func TestSourceRegistry_Get(t *testing.T) {
	t.Parallel()

	reg := NewSourceRegistry()
	aws := newMockSource(t, "aws", true)
	gcp := newMockSource(t, "gcp", false)
	reg.Register(aws)
	reg.Register(gcp)

	t.Run("existing available source", func(t *testing.T) {
		t.Parallel()
		source := reg.Get("aws")
		require.NotNil(t, source)
		assert.Equal(t, "aws", source.Name())
	})

	t.Run("existing unavailable source", func(t *testing.T) {
		t.Parallel()
		source := reg.Get("gcp")
		assert.Nil(t, source)
	})

	t.Run("nonexistent source", func(t *testing.T) {
		t.Parallel()
		source := reg.Get("nonexistent")
		assert.Nil(t, source)
	})
}

func TestSourceRegistry_First(t *testing.T) {
	t.Parallel()

	t.Run("with available sources", func(t *testing.T) {
		t.Parallel()
		reg := NewSourceRegistry()
		reg.Register(newMockSource(t, "aws", true))
		reg.Register(newMockSource(t, "gcp", true))

		first := reg.First()
		require.NotNil(t, first)
		assert.Equal(t, "aws", first.Name())
	})

	t.Run("no available sources", func(t *testing.T) {
		t.Parallel()
		reg := NewSourceRegistry()
		reg.Register(newMockSource(t, "gcp", false))

		first := reg.First()
		assert.Nil(t, first)
	})

	t.Run("empty registry", func(t *testing.T) {
		t.Parallel()
		reg := NewSourceRegistry()

		first := reg.First()
		assert.Nil(t, first)
	})
}

func TestSourceRegistry_Names(t *testing.T) {
	t.Parallel()

	reg := NewSourceRegistry()
	assert.Empty(t, reg.Names())

	reg.Register(newMockSource(t, "aws", true))
	reg.Register(newMockSource(t, "gcp", false))

	names := reg.Names()
	assert.Equal(t, []string{"aws", "gcp"}, names)
}

func TestSourceRegistry_AvailableNames(t *testing.T) {
	t.Parallel()

	reg := NewSourceRegistry()
	reg.Register(newMockSource(t, "aws", true))
	reg.Register(newMockSource(t, "gcp", false))
	reg.Register(newMockSource(t, "azure", true))

	names := reg.AvailableNames()
	assert.Equal(t, []string{"aws", "azure"}, names)
}
