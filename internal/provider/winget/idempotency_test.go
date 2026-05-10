package winget

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestPackageStep_Idempotent verifies that running Apply twice on a winget
// PackageStep is safe. StatefulCommandRunner models the
// `winget install --id <ID>` -> `winget list --id <ID>` transition by
// keying installation state on the package's --id flag value.
func TestPackageStep_Idempotent(t *testing.T) {
	runner := mocks.NewStatefulCommandRunner()
	step := NewPackageStep(Package{ID: "Microsoft.PowerToys"}, runner, nil)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
