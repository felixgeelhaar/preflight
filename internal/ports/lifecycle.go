package ports

import "context"

// FileLifecycle provides file lifecycle management operations.
// It handles snapshots before file modifications and tracks applied files for drift detection.
type FileLifecycle interface {
	// BeforeModify takes a snapshot of the file before modification.
	// Returns nil if the file doesn't exist (nothing to snapshot).
	BeforeModify(ctx context.Context, path string) error

	// AfterApply records that a file was applied by preflight.
	// This enables drift detection for the file.
	AfterApply(ctx context.Context, path, sourceLayer string) error
}

// NoopLifecycle is a no-op implementation of FileLifecycle.
// Use this when lifecycle management is not needed.
type NoopLifecycle struct{}

// BeforeModify does nothing.
func (n *NoopLifecycle) BeforeModify(_ context.Context, _ string) error {
	return nil
}

// AfterApply does nothing.
func (n *NoopLifecycle) AfterApply(_ context.Context, _, _ string) error {
	return nil
}

// Ensure NoopLifecycle implements FileLifecycle.
var _ FileLifecycle = (*NoopLifecycle)(nil)
