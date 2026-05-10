package scoop

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestBucketStep_Idempotent verifies running Apply twice on a scoop
// BucketStep is safe. StatefulCommandRunner models
// `scoop bucket add <name>` -> `scoop bucket list` transitions.
func TestBucketStep_Idempotent(t *testing.T) {
	runner := mocks.NewStatefulCommandRunner()
	step := NewBucketStep(Bucket{Name: "extras"}, runner, nil)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}

// TestPackageStep_Idempotent verifies running Apply twice on a scoop
// PackageStep is safe. StatefulCommandRunner models
// `scoop install <pkg>` -> `scoop list` transitions.
func TestPackageStep_Idempotent(t *testing.T) {
	runner := mocks.NewStatefulCommandRunner()
	step := NewPackageStep(Package{Name: "neovim"}, runner, nil)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
