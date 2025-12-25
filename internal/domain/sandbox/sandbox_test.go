package sandbox

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/felixgeelhaar/preflight/internal/domain/capability"
)

func TestMode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ModeFull, Mode("full"))
	assert.Equal(t, ModeRestricted, Mode("restricted"))
	assert.Equal(t, ModeTrusted, Mode("trusted"))
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	assert.Equal(t, ModeRestricted, cfg.Mode)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.NotNil(t, cfg.Policy)
	assert.False(t, cfg.AllowNetwork)
	assert.False(t, cfg.AllowFileSystem)
}

func TestDefaultLimits(t *testing.T) {
	t.Parallel()

	limits := DefaultLimits()

	assert.Equal(t, uint64(64*1024*1024), limits.MaxMemoryBytes)
	assert.Equal(t, 10*time.Second, limits.MaxCPUTime)
	assert.Equal(t, 16, limits.MaxFileDescriptors)
	assert.Equal(t, int64(1024*1024), limits.MaxOutputBytes)
}

func TestFullIsolationConfig(t *testing.T) {
	t.Parallel()

	cfg := FullIsolationConfig()

	assert.Equal(t, ModeFull, cfg.Mode)
	assert.Equal(t, 10*time.Second, cfg.Timeout)
	assert.False(t, cfg.AllowNetwork)
	assert.False(t, cfg.AllowFileSystem)
	assert.NotNil(t, cfg.Policy)
}

func TestRestrictedLimits(t *testing.T) {
	t.Parallel()

	limits := RestrictedLimits()

	assert.Equal(t, uint64(16*1024*1024), limits.MaxMemoryBytes)
	assert.Equal(t, 5*time.Second, limits.MaxCPUTime)
	assert.Equal(t, 4, limits.MaxFileDescriptors)
	assert.Equal(t, int64(256*1024), limits.MaxOutputBytes)
}

func TestTrustedConfig(t *testing.T) {
	t.Parallel()

	cfg := TrustedConfig()

	assert.Equal(t, ModeTrusted, cfg.Mode)
	assert.Equal(t, 5*time.Minute, cfg.Timeout)
	assert.True(t, cfg.AllowNetwork)
	assert.True(t, cfg.AllowFileSystem)
	assert.NotNil(t, cfg.Policy)
}

func TestTrustedLimits(t *testing.T) {
	t.Parallel()

	limits := TrustedLimits()

	assert.Equal(t, uint64(256*1024*1024), limits.MaxMemoryBytes)
	assert.Equal(t, 60*time.Second, limits.MaxCPUTime)
	assert.Equal(t, 64, limits.MaxFileDescriptors)
	assert.Equal(t, int64(10*1024*1024), limits.MaxOutputBytes)
}

func TestPlugin_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid plugin", func(t *testing.T) {
		t.Parallel()

		p := &Plugin{
			ID:      "test-plugin",
			Name:    "Test Plugin",
			Version: "1.0.0",
			Module:  []byte{0x00, 0x61, 0x73, 0x6d}, // WASM magic bytes
		}

		err := p.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing ID", func(t *testing.T) {
		t.Parallel()

		p := &Plugin{
			Name:   "Test Plugin",
			Module: []byte{0x00, 0x61, 0x73, 0x6d},
		}

		err := p.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ID")
	})

	t.Run("missing name", func(t *testing.T) {
		t.Parallel()

		p := &Plugin{
			ID:     "test-plugin",
			Module: []byte{0x00, 0x61, 0x73, 0x6d},
		}

		err := p.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("missing module", func(t *testing.T) {
		t.Parallel()

		p := &Plugin{
			ID:   "test-plugin",
			Name: "Test Plugin",
		}

		err := p.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "module")
	})
}

func TestExecutionResult(t *testing.T) {
	t.Parallel()

	result := &ExecutionResult{
		Success:  true,
		Output:   []byte("hello"),
		Duration: 100 * time.Millisecond,
		ResourceUsage: ResourceUsage{
			MemoryBytes:     1024,
			CPUTime:         50 * time.Millisecond,
			FileDescriptors: 2,
			OutputBytes:     5,
		},
	}

	assert.True(t, result.Success)
	assert.Equal(t, []byte("hello"), result.Output)
	assert.Equal(t, 100*time.Millisecond, result.Duration)
	assert.Equal(t, uint64(1024), result.ResourceUsage.MemoryBytes)
}

func TestPlugin_WithCapabilities(t *testing.T) {
	t.Parallel()

	caps := capability.NewRequirements()
	caps.AddCapability(capability.CapFilesRead, "Read config files")
	caps.AddCapability(capability.CapPackagesBrew, "Install packages")

	p := &Plugin{
		ID:           "test-plugin",
		Name:         "Test Plugin",
		Version:      "1.0.0",
		Module:       []byte{0x00, 0x61, 0x73, 0x6d},
		Capabilities: caps,
	}

	err := p.Validate()
	assert.NoError(t, err)
	assert.Equal(t, 2, p.Capabilities.Count())
}
