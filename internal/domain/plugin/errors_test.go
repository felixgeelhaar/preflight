package plugin

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluginExistsError(t *testing.T) {
	t.Parallel()

	err := &PluginExistsError{Name: "docker"}
	assert.Equal(t, `plugin "docker" already registered`, err.Error())
}

func TestValidationError(t *testing.T) {
	t.Parallel()

	t.Run("single error", func(t *testing.T) {
		t.Parallel()
		err := &ValidationError{Errors: []string{"name is required"}}
		assert.Equal(t, "name is required", err.Error())
	})

	t.Run("multiple errors", func(t *testing.T) {
		t.Parallel()
		err := &ValidationError{Errors: []string{"name is required", "version is required"}}
		assert.Equal(t, "validation failed: name is required; version is required", err.Error())
	})

	t.Run("add error", func(t *testing.T) {
		t.Parallel()
		err := &ValidationError{}
		err.Add("first error")
		assert.Len(t, err.Errors, 1)
		assert.True(t, err.HasErrors())
	})

	t.Run("addf formatted error", func(t *testing.T) {
		t.Parallel()
		err := &ValidationError{}
		err.Addf("invalid field: %s", "name")
		assert.Equal(t, "invalid field: name", err.Errors[0])
	})

	t.Run("has errors empty", func(t *testing.T) {
		t.Parallel()
		err := &ValidationError{}
		assert.False(t, err.HasErrors())
	})
}

func TestDiscoveryError(t *testing.T) {
	t.Parallel()

	underlying := errors.New("permission denied")
	err := &DiscoveryError{Path: "/path/to/plugin", Err: underlying}

	assert.Equal(t, "loading plugin at /path/to/plugin: permission denied", err.Error())
	assert.ErrorIs(t, err, underlying)
}

func TestDiscoveryResult(t *testing.T) {
	t.Parallel()

	t.Run("no errors", func(t *testing.T) {
		t.Parallel()
		result := &DiscoveryResult{Plugins: []*Plugin{{}}}
		assert.False(t, result.HasErrors())
	})

	t.Run("with errors", func(t *testing.T) {
		t.Parallel()
		result := &DiscoveryResult{
			Errors: []DiscoveryError{{Path: "/test", Err: errors.New("test")}},
		}
		assert.True(t, result.HasErrors())
	})
}

func TestPathTraversalError(t *testing.T) {
	t.Parallel()

	err := &PathTraversalError{Path: "../../../etc/passwd"}
	assert.Equal(t, "path traversal detected in: ../../../etc/passwd", err.Error())
}

func TestInvalidURLError(t *testing.T) {
	t.Parallel()

	err := &InvalidURLError{URL: "ftp://example.com", Reason: "unsupported scheme"}
	assert.Equal(t, `invalid URL "ftp://example.com": unsupported scheme`, err.Error())
}

func TestIsPluginExists(t *testing.T) {
	t.Parallel()

	t.Run("plugin exists error", func(t *testing.T) {
		t.Parallel()
		err := &PluginExistsError{Name: "docker"}
		assert.True(t, IsPluginExists(err))
	})

	t.Run("other error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("some other error")
		assert.False(t, IsPluginExists(err))
	})
}

func TestIsValidationError(t *testing.T) {
	t.Parallel()

	t.Run("validation error", func(t *testing.T) {
		t.Parallel()
		err := &ValidationError{Errors: []string{"test"}}
		assert.True(t, IsValidationError(err))
	})

	t.Run("other error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("some other error")
		assert.False(t, IsValidationError(err))
	})
}

func TestIsPathTraversal(t *testing.T) {
	t.Parallel()

	t.Run("path traversal error", func(t *testing.T) {
		t.Parallel()
		err := &PathTraversalError{Path: ".."}
		assert.True(t, IsPathTraversal(err))
	})

	t.Run("other error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("some other error")
		assert.False(t, IsPathTraversal(err))
	})
}

func TestChecksumError(t *testing.T) {
	t.Parallel()

	err := &ChecksumError{
		Expected: "abc123",
		Actual:   "def456",
	}
	assert.Equal(t, "checksum mismatch: expected abc123, got def456", err.Error())
}

func TestIsChecksumError(t *testing.T) {
	t.Parallel()

	t.Run("checksum error", func(t *testing.T) {
		t.Parallel()
		err := &ChecksumError{Expected: "abc", Actual: "def"}
		assert.True(t, IsChecksumError(err))
	})

	t.Run("other error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("some other error")
		assert.False(t, IsChecksumError(err))
	})
}

func TestSignatureError(t *testing.T) {
	t.Parallel()

	err := &SignatureError{Reason: "invalid signature format"}
	assert.Equal(t, "signature verification failed: invalid signature format", err.Error())
}

func TestIsSignatureError(t *testing.T) {
	t.Parallel()

	t.Run("signature error", func(t *testing.T) {
		t.Parallel()
		err := &SignatureError{Reason: "test"}
		assert.True(t, IsSignatureError(err))
	})

	t.Run("other error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("some other error")
		assert.False(t, IsSignatureError(err))
	})
}

func TestCapabilityError(t *testing.T) {
	t.Parallel()

	err := &CapabilityError{Capability: "shell:execute", Reason: "not in allow list"}
	assert.Equal(t, `capability "shell:execute" not allowed: not in allow list`, err.Error())
}

func TestIsCapabilityError(t *testing.T) {
	t.Parallel()

	t.Run("capability error", func(t *testing.T) {
		t.Parallel()
		err := &CapabilityError{Capability: "test", Reason: "denied"}
		assert.True(t, IsCapabilityError(err))
	})

	t.Run("other error", func(t *testing.T) {
		t.Parallel()
		err := errors.New("some other error")
		assert.False(t, IsCapabilityError(err))
	})
}
