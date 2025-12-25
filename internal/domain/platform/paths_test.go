package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathTranslator_ToWSL(t *testing.T) {
	t.Parallel()

	translator := NewPathTranslator(NewWSL(EnvWSL2, "Ubuntu", "/mnt/c"))

	tests := []struct {
		name        string
		windowsPath string
		want        string
		wantErr     bool
	}{
		{
			name:        "simple C drive path",
			windowsPath: "C:\\Users\\name",
			want:        "/mnt/c/Users/name",
			wantErr:     false,
		},
		{
			name:        "C drive with forward slashes",
			windowsPath: "C:/Users/name",
			want:        "/mnt/c/Users/name",
			wantErr:     false,
		},
		{
			name:        "D drive",
			windowsPath: "D:\\Projects\\code",
			want:        "/mnt/d/Projects/code",
			wantErr:     false,
		},
		{
			name:        "lowercase drive letter",
			windowsPath: "c:\\Users\\name",
			want:        "/mnt/c/Users/name",
			wantErr:     false,
		},
		{
			name:        "root of drive",
			windowsPath: "C:\\",
			want:        "/mnt/c",
			wantErr:     false,
		},
		{
			name:        "already unix path",
			windowsPath: "/home/user",
			want:        "/home/user",
			wantErr:     false,
		},
		{
			name:        "empty path",
			windowsPath: "",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "invalid path",
			windowsPath: "not a valid path",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "path with spaces",
			windowsPath: "C:\\Program Files\\App Name",
			want:        "/mnt/c/Program Files/App Name",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := translator.ToWSL(tt.windowsPath)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPathTranslator_ToWindows(t *testing.T) {
	t.Parallel()

	translator := NewPathTranslator(NewWSL(EnvWSL2, "Ubuntu", "/mnt/c"))

	tests := []struct {
		name    string
		wslPath string
		want    string
		wantErr bool
	}{
		{
			name:    "simple mnt path",
			wslPath: "/mnt/c/Users/name",
			want:    "C:\\Users\\name",
			wantErr: false,
		},
		{
			name:    "D drive",
			wslPath: "/mnt/d/Projects/code",
			want:    "D:\\Projects\\code",
			wantErr: false,
		},
		{
			name:    "root of drive",
			wslPath: "/mnt/c",
			want:    "C:\\",
			wantErr: false,
		},
		{
			name:    "WSL native path",
			wslPath: "/home/user/.config",
			want:    "\\\\wsl$\\Ubuntu/home/user/.config",
			wantErr: false,
		},
		{
			name:    "empty path",
			wslPath: "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := translator.ToWindows(tt.wslPath)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsWindowsPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{"C:\\Users\\name", true},
		{"C:/Users/name", true},
		{"D:\\Projects", true},
		{"c:\\lowercase", true},
		{"/home/user", false},
		{"/mnt/c/Users", false},
		{"relative/path", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsWindowsPath(tt.path))
		})
	}
}

func TestIsWSLMountPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{"/mnt/c/Users", true},
		{"/mnt/d/Projects", true},
		{"/mnt/e", true},
		{"/home/user", false},
		{"/mnt", false},
		{"/mnt/", false},
		{"C:\\Users", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsWSLMountPath(tt.path))
		})
	}
}

func TestPathTranslator_NormalizePath(t *testing.T) {
	t.Parallel()

	t.Run("windows platform", func(t *testing.T) {
		t.Parallel()
		translator := NewPathTranslator(New(OSWindows, "amd64", EnvNative))
		assert.Equal(t, "C:\\Users\\name", translator.NormalizePath("C:/Users/name"))
	})

	t.Run("unix platform", func(t *testing.T) {
		t.Parallel()
		translator := NewPathTranslator(New(OSLinux, "amd64", EnvNative))
		assert.Equal(t, "/home/user/path", translator.NormalizePath("/home/user\\path"))
	})
}

func TestPathTranslator_JoinPath(t *testing.T) {
	t.Parallel()

	t.Run("windows platform", func(t *testing.T) {
		t.Parallel()
		translator := NewPathTranslator(New(OSWindows, "amd64", EnvNative))
		assert.Equal(t, "C:\\Users\\name", translator.JoinPath("C:", "Users", "name"))
	})

	t.Run("unix platform", func(t *testing.T) {
		t.Parallel()
		translator := NewPathTranslator(New(OSLinux, "amd64", EnvNative))
		assert.Equal(t, "/home/user/name", translator.JoinPath("/home", "user", "name"))
	})
}

func TestPathTranslator_WindowsHome(t *testing.T) {
	t.Parallel()

	t.Run("not WSL returns error", func(t *testing.T) {
		t.Parallel()
		translator := NewPathTranslator(New(OSLinux, "amd64", EnvNative))
		_, err := translator.WindowsHome()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not running in WSL")
	})

	t.Run("macOS returns error", func(t *testing.T) {
		t.Parallel()
		translator := NewPathTranslator(New(OSDarwin, "arm64", EnvNative))
		_, err := translator.WindowsHome()
		assert.Error(t, err)
	})

	t.Run("Windows returns error", func(t *testing.T) {
		t.Parallel()
		translator := NewPathTranslator(New(OSWindows, "amd64", EnvNative))
		_, err := translator.WindowsHome()
		assert.Error(t, err)
	})

	t.Run("Docker returns error", func(t *testing.T) {
		t.Parallel()
		translator := NewPathTranslator(New(OSLinux, "amd64", EnvDocker))
		_, err := translator.WindowsHome()
		assert.Error(t, err)
	})
}

func TestNewPathTranslator(t *testing.T) {
	t.Parallel()

	t.Run("creates translator with platform", func(t *testing.T) {
		t.Parallel()
		p := NewWSL(EnvWSL2, "Ubuntu", "/mnt/c")
		translator := NewPathTranslator(p)
		assert.NotNil(t, translator)
		assert.Equal(t, "/mnt/c", translator.windowsRoot)
	})

	t.Run("creates translator for non-WSL platform", func(t *testing.T) {
		t.Parallel()
		p := New(OSDarwin, "arm64", EnvNative)
		translator := NewPathTranslator(p)
		assert.NotNil(t, translator)
		assert.Empty(t, translator.windowsRoot)
	})
}

func TestPathTranslator_ToWindows_EdgeCases(t *testing.T) {
	t.Parallel()

	translator := NewPathTranslator(NewWSL(EnvWSL2, "Ubuntu", "/mnt/c"))

	t.Run("invalid mnt path returns error", func(t *testing.T) {
		t.Parallel()
		// Invalid: no drive letter after /mnt/
		_, err := translator.ToWindows("/mnt//invalid")
		assert.Error(t, err)
	})

	t.Run("mnt with multi-char drive returns error", func(t *testing.T) {
		t.Parallel()
		_, err := translator.ToWindows("/mnt/abc/path")
		assert.Error(t, err)
	})
}

func TestPathTranslator_ToWSLWithWslpath(t *testing.T) {
	t.Parallel()

	// wslpath doesn't exist on macOS, so this should fall back to ToWSL
	translator := NewPathTranslator(New(OSLinux, "amd64", EnvNative))
	result, err := translator.ToWSLWithWslpath("C:\\Users\\name")
	require.NoError(t, err)
	assert.Equal(t, "/mnt/c/Users/name", result)
}

func TestPathTranslator_ToWindowsWithWslpath(t *testing.T) {
	t.Parallel()

	// wslpath doesn't exist on macOS, so this should fall back to ToWindows
	translator := NewPathTranslator(NewWSL(EnvWSL2, "Ubuntu", "/mnt/c"))
	result, err := translator.ToWindowsWithWslpath("/mnt/c/Users/name")
	require.NoError(t, err)
	assert.Equal(t, "C:\\Users\\name", result)
}

func TestIsWSLMountPath_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"uppercase drive", "/mnt/C/Users", true},
		{"numeric after mnt", "/mnt/1/path", false},
		{"special char after mnt", "/mnt/@/path", false},
		{"just mnt with slash", "/mnt/", false},
		{"mnt without trailing", "/mnt", false},
		{"different prefix", "/var/mnt/c", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsWSLMountPath(tt.path))
		})
	}
}
