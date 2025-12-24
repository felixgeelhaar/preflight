package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlatform_OS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		platform *Platform
		want     OS
	}{
		{
			name:     "darwin",
			platform: New(OSDarwin, "arm64", EnvNative),
			want:     OSDarwin,
		},
		{
			name:     "linux",
			platform: New(OSLinux, "amd64", EnvNative),
			want:     OSLinux,
		},
		{
			name:     "windows",
			platform: New(OSWindows, "amd64", EnvNative),
			want:     OSWindows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.platform.OS())
		})
	}
}

func TestPlatform_Environment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		platform *Platform
		want     Environment
	}{
		{
			name:     "native",
			platform: New(OSDarwin, "arm64", EnvNative),
			want:     EnvNative,
		},
		{
			name:     "wsl1",
			platform: NewWSL(EnvWSL1, "Ubuntu", "/mnt/c"),
			want:     EnvWSL1,
		},
		{
			name:     "wsl2",
			platform: NewWSL(EnvWSL2, "Ubuntu", "/mnt/c"),
			want:     EnvWSL2,
		},
		{
			name:     "docker",
			platform: New(OSLinux, "amd64", EnvDocker),
			want:     EnvDocker,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.platform.Environment())
		})
	}
}

func TestPlatform_IsChecks(t *testing.T) {
	t.Parallel()

	t.Run("IsWindows", func(t *testing.T) {
		t.Parallel()
		assert.True(t, New(OSWindows, "amd64", EnvNative).IsWindows())
		assert.False(t, New(OSLinux, "amd64", EnvNative).IsWindows())
		assert.False(t, New(OSDarwin, "arm64", EnvNative).IsWindows())
	})

	t.Run("IsMacOS", func(t *testing.T) {
		t.Parallel()
		assert.True(t, New(OSDarwin, "arm64", EnvNative).IsMacOS())
		assert.False(t, New(OSLinux, "amd64", EnvNative).IsMacOS())
		assert.False(t, New(OSWindows, "amd64", EnvNative).IsMacOS())
	})

	t.Run("IsLinux", func(t *testing.T) {
		t.Parallel()
		assert.True(t, New(OSLinux, "amd64", EnvNative).IsLinux())
		assert.True(t, NewWSL(EnvWSL2, "Ubuntu", "/mnt/c").IsLinux())
		assert.False(t, New(OSDarwin, "arm64", EnvNative).IsLinux())
		assert.False(t, New(OSWindows, "amd64", EnvNative).IsLinux())
	})

	t.Run("IsWSL", func(t *testing.T) {
		t.Parallel()
		assert.True(t, NewWSL(EnvWSL1, "Ubuntu", "/mnt/c").IsWSL())
		assert.True(t, NewWSL(EnvWSL2, "Ubuntu", "/mnt/c").IsWSL())
		assert.False(t, New(OSLinux, "amd64", EnvNative).IsWSL())
		assert.False(t, New(OSWindows, "amd64", EnvNative).IsWSL())
	})

	t.Run("IsWSL2", func(t *testing.T) {
		t.Parallel()
		assert.True(t, NewWSL(EnvWSL2, "Ubuntu", "/mnt/c").IsWSL2())
		assert.False(t, NewWSL(EnvWSL1, "Ubuntu", "/mnt/c").IsWSL2())
		assert.False(t, New(OSLinux, "amd64", EnvNative).IsWSL2())
	})

	t.Run("IsDocker", func(t *testing.T) {
		t.Parallel()
		assert.True(t, New(OSLinux, "amd64", EnvDocker).IsDocker())
		assert.False(t, New(OSLinux, "amd64", EnvNative).IsDocker())
	})

	t.Run("IsNative", func(t *testing.T) {
		t.Parallel()
		assert.True(t, New(OSDarwin, "arm64", EnvNative).IsNative())
		assert.False(t, NewWSL(EnvWSL2, "Ubuntu", "/mnt/c").IsNative())
		assert.False(t, New(OSLinux, "amd64", EnvDocker).IsNative())
	})
}

func TestPlatform_WSL(t *testing.T) {
	t.Parallel()

	t.Run("WSLDistro", func(t *testing.T) {
		t.Parallel()
		p := NewWSL(EnvWSL2, "Ubuntu-22.04", "/mnt/c")
		assert.Equal(t, "Ubuntu-22.04", p.WSLDistro())
	})

	t.Run("WindowsPath", func(t *testing.T) {
		t.Parallel()
		p := NewWSL(EnvWSL2, "Ubuntu", "/mnt/c")
		assert.Equal(t, "/mnt/c", p.WindowsPath())
	})

	t.Run("CanAccessWindows", func(t *testing.T) {
		t.Parallel()
		assert.True(t, NewWSL(EnvWSL2, "Ubuntu", "/mnt/c").CanAccessWindows())
		assert.False(t, New(OSLinux, "amd64", EnvNative).CanAccessWindows())
	})
}

func TestPlatform_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		platform *Platform
		want     string
	}{
		{
			name:     "macOS native",
			platform: New(OSDarwin, "arm64", EnvNative),
			want:     "darwin/arm64",
		},
		{
			name:     "linux native",
			platform: New(OSLinux, "amd64", EnvNative),
			want:     "linux/amd64",
		},
		{
			name:     "wsl2 ubuntu",
			platform: NewWSL(EnvWSL2, "Ubuntu", "/mnt/c"),
			want:     "linux/amd64/wsl2/Ubuntu",
		},
		{
			name:     "docker",
			platform: New(OSLinux, "amd64", EnvDocker),
			want:     "linux/amd64/docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.platform.String())
		})
	}
}

func TestPlatform_Arch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		arch string
		want string
	}{
		{"amd64", "amd64"},
		{"arm64", "arm64"},
		{"386", "386"},
	}

	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			t.Parallel()
			p := New(OSLinux, tt.arch, EnvNative)
			assert.Equal(t, tt.want, p.Arch())
		})
	}
}

func TestSetTestPlatform(t *testing.T) {
	// This test modifies global state, so don't run in parallel

	// Set a test platform
	testPlat := New(OSWindows, "amd64", EnvNative)
	SetTestPlatform(testPlat)

	detected, err := Detect()
	assert.NoError(t, err)
	assert.Equal(t, OSWindows, detected.OS())

	// Reset
	SetTestPlatform(nil)
}
