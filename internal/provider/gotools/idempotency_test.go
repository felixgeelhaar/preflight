package gotools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestToolStep_Idempotent verifies running Apply twice on a Go ToolStep is
// safe. Unlike package-manager steps, gotools.Check uses os.Stat directly on
// $GOBIN/<binary>, so we must:
//   - point GOBIN at a temp dir, and
//   - have the stateful runner's `go install` handler create the binary
//     file on disk so the post-Apply Check succeeds.
func TestToolStep_Idempotent(t *testing.T) {
	gobin := t.TempDir()
	t.Setenv("GOBIN", gobin)

	runner := mocks.NewStatefulCommandRunner()
	runner.AddHandler("go", func(args []string) bool {
		return len(args) >= 1 && args[0] == "install"
	}, func(args []string) (ports.CommandResult, error) {
		// `go install <module>@<version>` — derive binary name from module.
		for _, a := range args[1:] {
			module := a
			if at := indexOf(a, "@"); at > 0 {
				module = a[:at]
			}
			binary := filepath.Base(module)
			path := filepath.Join(gobin, binary)
			if err := os.WriteFile(path, []byte("ok"), 0o755); err != nil {
				return ports.CommandResult{ExitCode: 1, Stderr: err.Error()}, err
			}
		}
		return ports.CommandResult{ExitCode: 0}, nil
	})

	step := NewToolStep(Tool{Module: "golang.org/x/tools/gopls"}, runner, nil)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}

func indexOf(s, sub string) int {
	for i := range len(s) - len(sub) + 1 {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
