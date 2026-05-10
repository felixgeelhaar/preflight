package files

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

// TestLinkStep_Idempotent verifies that re-running Apply on a LinkStep is safe:
// after the first Apply the link exists; the second Apply must succeed and
// leave the system in StatusSatisfied. Guards the product's #1 contract:
// "re-running apply is always safe".
func TestLinkStep_Idempotent(t *testing.T) {
	fs := mocks.NewFileSystem()
	link := Link{
		Src:   "/dotfiles/.zshrc",
		Dest:  "/home/user/.zshrc",
		Force: true,
	}
	step := NewLinkStep(link, fs, nil)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}

// TestCopyStep_Idempotent: Apply copies a file; re-Apply must not fail and
// must not regress Check() back to StatusNeedsApply.
func TestCopyStep_Idempotent(t *testing.T) {
	fs := mocks.NewFileSystem()
	fs.AddFile("/src/file.txt", "hello")

	cp := Copy{
		Src:  "/src/file.txt",
		Dest: "/dest/file.txt",
		Mode: "0644",
	}
	step := NewCopyStep(cp, fs, nil)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
