package drift

import (
	"context"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Detector checks for drift in managed files.
type Detector struct {
	fs    ports.FileSystem
	state *AppliedState
}

// NewDetector creates a new Detector.
func NewDetector(fs ports.FileSystem, state *AppliedState) *Detector {
	return &Detector{
		fs:    fs,
		state: state,
	}
}

// Detect checks a single file for drift.
func (d *Detector) Detect(ctx context.Context, path string) (Drift, error) {
	_ = ctx // Reserved for future cancellation support

	// Check if file is tracked
	fileState, tracked := d.state.GetFile(path)
	if !tracked {
		// File not tracked, no drift to report
		return NewDrift(path, "", "", fileState.AppliedAt, TypeNone), nil
	}

	// Check if file exists
	if !d.fs.Exists(path) {
		// File was deleted - this is drift
		return NewDrift(path, "", fileState.AppliedHash, fileState.AppliedAt, TypeManual), nil
	}

	// Get current hash
	currentHash, err := d.fs.FileHash(path)
	if err != nil {
		return Drift{}, err
	}

	// Compare hashes
	if currentHash == fileState.AppliedHash {
		// No drift
		return NewDrift(path, currentHash, fileState.AppliedHash, fileState.AppliedAt, TypeNone), nil
	}

	// Drift detected - classify it
	driftType := d.classifyDrift(path)

	return NewDrift(path, currentHash, fileState.AppliedHash, fileState.AppliedAt, driftType), nil
}

// DetectAll checks all tracked files for drift.
func (d *Detector) DetectAll(ctx context.Context) ([]Drift, error) {
	files := d.state.ListFiles()
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}

	return d.DetectPaths(ctx, paths)
}

// DetectPaths checks specific paths for drift.
func (d *Detector) DetectPaths(ctx context.Context, paths []string) ([]Drift, error) {
	var drifts []Drift

	for _, path := range paths {
		drift, err := d.Detect(ctx, path)
		if err != nil {
			return nil, err
		}

		if drift.HasDrift() {
			drifts = append(drifts, drift)
		}
	}

	return drifts, nil
}

// classifyDrift attempts to determine the source of drift.
func (d *Detector) classifyDrift(path string) Type {
	// If file doesn't exist, it was manually deleted
	if !d.fs.Exists(path) {
		return TypeManual
	}

	// For now, we can't distinguish between manual and external modifications
	// Future enhancement: check file metadata, git history, etc.
	return TypeUnknown
}
