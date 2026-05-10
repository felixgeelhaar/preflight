package npm

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestPackageStep_Idempotent verifies that running Apply twice on an npm
// global package install is safe. StatefulCommandRunner emits a JSON
// {"dependencies":{...}} document for `npm list -g --depth=0 --json` so the
// PackageStep's JSON-parsing Check can detect the post-install state.
func TestPackageStep_Idempotent(t *testing.T) {
	runner := mocks.NewStatefulCommandRunner()
	step := NewPackageStep(Package{Name: "typescript"}, runner, nil)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
