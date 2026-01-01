// Package config provides domain services for configuration management.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LayerService provides domain operations for layer files.
type LayerService struct{}

// NewLayerService creates a new LayerService.
func NewLayerService() *LayerService {
	return &LayerService{}
}

// ValidateLayerPath validates a layer file path.
// It ensures the path is a valid YAML file that exists and is not a symlink escaping a base directory.
func (s *LayerService) ValidateLayerPath(path string) error {
	return ValidateLayerPath(path)
}

// ValidateLayerPathWithBase validates a layer path is within the expected base directory.
// This prevents path traversal and symlink escape attacks.
func (s *LayerService) ValidateLayerPathWithBase(path, basePath string) error {
	return ValidateLayerPathWithBase(path, basePath)
}

// ValidateLayerPath validates a layer file path.
// It ensures the path is a valid YAML file that exists and is not a directory.
func ValidateLayerPath(path string) error {
	// Check for empty path
	if path == "" {
		return fmt.Errorf("layer path cannot be empty")
	}

	// Check for null bytes (security)
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("invalid layer path: contains null byte")
	}

	// Check for path traversal sequences
	if containsPathTraversal(path) {
		return fmt.Errorf("invalid layer path: path traversal detected")
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("invalid layer file extension: %s (expected .yaml or .yml)", ext)
	}

	// Check file exists and is a regular file
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("layer file not found: %s", path)
		}
		return fmt.Errorf("cannot access layer file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but got directory: %s", path)
	}

	return nil
}

// ValidateLayerPathWithBase validates a layer path is within the expected base directory.
// This prevents symlink escape attacks by resolving symlinks and checking containment.
func ValidateLayerPathWithBase(path, basePath string) error {
	// Basic validation first
	if err := ValidateLayerPath(path); err != nil {
		return err
	}

	// Resolve symlinks for the file path
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("cannot resolve layer path: %w", err)
	}

	// Resolve symlinks for the base path
	resolvedBase, err := filepath.EvalSymlinks(basePath)
	if err != nil {
		// If base doesn't exist, use cleaned path
		resolvedBase = filepath.Clean(basePath)
	}

	// Clean and check if resolved path is within base
	cleanPath := filepath.Clean(resolvedPath)
	cleanBase := filepath.Clean(resolvedBase)

	// Ensure the path is within the base directory
	if !strings.HasPrefix(cleanPath, cleanBase+string(filepath.Separator)) && cleanPath != cleanBase {
		return fmt.Errorf("layer path escapes base directory: %s", path)
	}

	return nil
}

// containsPathTraversal checks for common path traversal patterns.
func containsPathTraversal(path string) bool {
	// Check for ".." sequences in the original path BEFORE normalization
	// This catches both explicit "../" and embedded "/.." patterns
	if strings.Contains(path, "..") {
		return true
	}

	// Check for URL-encoded traversal
	if strings.Contains(path, "%2e%2e") || strings.Contains(path, "%2E%2E") {
		return true
	}

	return false
}
