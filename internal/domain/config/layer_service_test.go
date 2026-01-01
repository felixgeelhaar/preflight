package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLayerService_ValidateLayerPath(t *testing.T) {
	service := NewLayerService()

	// Create a temporary layer file for testing
	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "test.yaml")
	require.NoError(t, os.WriteFile(validFile, []byte("name: test\n"), 0644))

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid yaml file",
			path:    validFile,
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "invalid extension txt",
			path:    filepath.Join(tmpDir, "test.txt"),
			wantErr: true,
			errMsg:  "invalid layer file extension",
		},
		{
			name:    "file not found",
			path:    filepath.Join(tmpDir, "nonexistent.yaml"),
			wantErr: true,
			errMsg:  "layer file not found",
		},
		{
			name:    "directory instead of file",
			path:    tmpDir,
			wantErr: true,
			errMsg:  "invalid layer file extension",
		},
		{
			name:    "path traversal",
			path:    tmpDir + "/../etc/passwd.yaml", // Use raw string to preserve ".."
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name:    "null byte injection",
			path:    tmpDir + "/test\x00.yaml",
			wantErr: true,
			errMsg:  "null byte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateLayerPath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLayerPath_YmlExtension(t *testing.T) {
	tmpDir := t.TempDir()
	ymlFile := filepath.Join(tmpDir, "test.yml")
	require.NoError(t, os.WriteFile(ymlFile, []byte("name: test\n"), 0644))

	err := ValidateLayerPath(ymlFile)
	assert.NoError(t, err)
}

func TestValidateLayerPathWithBase(t *testing.T) {
	tmpDir := t.TempDir()
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0755))

	validFile := filepath.Join(layersDir, "base.yaml")
	require.NoError(t, os.WriteFile(validFile, []byte("name: base\n"), 0644))

	outsideFile := filepath.Join(tmpDir, "outside.yaml")
	require.NoError(t, os.WriteFile(outsideFile, []byte("name: outside\n"), 0644))

	tests := []struct {
		name     string
		path     string
		basePath string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid path within base",
			path:     validFile,
			basePath: layersDir,
			wantErr:  false,
		},
		{
			name:     "path escapes base",
			path:     outsideFile,
			basePath: layersDir,
			wantErr:  true,
			errMsg:   "escapes base directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLayerPathWithBase(tt.path, tt.basePath)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLayerPathWithBase_SymlinkEscape(t *testing.T) {
	tmpDir := t.TempDir()
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0755))

	// Create a file outside the layers directory
	outsideDir := filepath.Join(tmpDir, "outside")
	require.NoError(t, os.MkdirAll(outsideDir, 0755))
	outsideFile := filepath.Join(outsideDir, "secret.yaml")
	require.NoError(t, os.WriteFile(outsideFile, []byte("secret: data\n"), 0644))

	// Create a symlink inside layers that points outside
	symlinkPath := filepath.Join(layersDir, "escape.yaml")
	err := os.Symlink(outsideFile, symlinkPath)
	if err != nil {
		t.Skip("symlink creation not supported on this system")
	}

	// Validate should detect the symlink escape
	err = ValidateLayerPathWithBase(symlinkPath, layersDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "escapes base directory")
}

func TestContainsPathTraversal(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"normal/path/file.yaml", false},
		{"../escape/file.yaml", true},
		{"path/../file.yaml", true},
		{"path/./file.yaml", false},
		{"%2e%2e/encoded.yaml", true},
		{"%2E%2E/ENCODED.yaml", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := containsPathTraversal(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewLayerService(t *testing.T) {
	service := NewLayerService()
	assert.NotNil(t, service)
}

func TestValidateLayerPath_FileSizeLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that exceeds the max size
	largePath := filepath.Join(tmpDir, "large.yaml")
	largeContent := make([]byte, MaxLayerFileSize+1)
	// Fill with valid YAML content
	copy(largeContent, "name: large\ndata: ")
	require.NoError(t, os.WriteFile(largePath, largeContent, 0644))

	// Verify the file is actually larger than the max size
	info, err := os.Stat(largePath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(MaxLayerFileSize))

	// Validation should fail for oversized files
	err = ValidateLayerPath(largePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "layer file too large")

	// Create a file at exactly the max size (should pass)
	exactPath := filepath.Join(tmpDir, "exact.yaml")
	exactContent := make([]byte, MaxLayerFileSize)
	copy(exactContent, "name: exact\ndata: ")
	require.NoError(t, os.WriteFile(exactPath, exactContent, 0644))

	err = ValidateLayerPath(exactPath)
	assert.NoError(t, err)

	// Create a small file (should pass)
	smallPath := filepath.Join(tmpDir, "small.yaml")
	require.NoError(t, os.WriteFile(smallPath, []byte("name: small\n"), 0644))

	err = ValidateLayerPath(smallPath)
	assert.NoError(t, err)
}

func TestMaxLayerFileSize_Constant(t *testing.T) {
	// Verify the constant is set to 1MB
	assert.Equal(t, int64(1*1024*1024), int64(MaxLayerFileSize))
}
