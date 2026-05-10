package chocolatey

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestPackageStep_Idempotent verifies running Apply twice on a chocolatey
// PackageStep is safe. StatefulCommandRunner models the
// `choco install <name>` -> `choco list --local-only --exact <name>`
// transition end-to-end.
func TestPackageStep_Idempotent(t *testing.T) {
	runner := mocks.NewStatefulCommandRunner()
	step := NewPackageStep(Package{Name: "git"}, runner, nil, nil)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
