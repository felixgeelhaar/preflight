package pip

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestPackageStep_Idempotent verifies running Apply twice on a pip
// PackageStep is safe. StatefulCommandRunner models the
// `pip install black` -> `pip show black` transition end-to-end.
func TestPackageStep_Idempotent(t *testing.T) {
	runner := mocks.NewStatefulCommandRunner()
	step := NewPackageStep(Package{Name: "black"}, runner, nil)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
