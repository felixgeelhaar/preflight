package shell

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

// TestFrameworkStep_OhMyZsh_Idempotent verifies running Apply twice on a
// shell.FrameworkStep is safe. The framework's Check uses fs.Exists on
// ~/.oh-my-zsh, so the test wires the runner's install handler to populate
// that path on the fake filesystem when the install script runs — modeling
// what the real curl-pipe-bash installer does on disk.
func TestFrameworkStep_OhMyZsh_Idempotent(t *testing.T) {
	// Pin HOME so ports.ExpandPath produces a deterministic path.
	home := t.TempDir()
	t.Setenv("HOME", home)

	fs := mocks.NewFileSystem()
	runner := mocks.NewStatefulCommandRunner()

	// When the install script runs, materialize the framework directory in fs.
	runner.AddHandler("/bin/bash", func(args []string) bool {
		return len(args) >= 2 && args[0] == "-c"
	}, func(_ []string) (ports.CommandResult, error) {
		fs.MkdirAll(filepath.Join(home, ".oh-my-zsh"), os.ModePerm) //nolint:errcheck
		return ports.CommandResult{ExitCode: 0}, nil
	})

	step := NewFrameworkStepWith(Entry{Name: "zsh", Framework: "oh-my-zsh"}, fs, runner)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}

// TestFrameworkStep_Fisher_Idempotent: same contract for the fish-shell
// fisher framework, whose Check looks for fisher.fish under
// ~/.config/fish/functions/.
func TestFrameworkStep_Fisher_Idempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	fs := mocks.NewFileSystem()
	runner := mocks.NewStatefulCommandRunner()

	runner.AddHandler("fish", func(args []string) bool {
		return len(args) >= 2 && args[0] == "-c"
	}, func(_ []string) (ports.CommandResult, error) {
		dir := filepath.Join(home, ".config", "fish", "functions")
		fs.MkdirAll(dir, os.ModePerm)                                    //nolint:errcheck
		fs.WriteFile(filepath.Join(dir, "fisher.fish"), []byte{}, 0o644) //nolint:errcheck
		return ports.CommandResult{ExitCode: 0}, nil
	})

	step := NewFrameworkStepWith(Entry{Name: "fish", Framework: "fisher"}, fs, runner)
	ctx := compiler.NewRunContext(context.Background())

	testutil.AssertStepIsIdempotent(t, step, ctx)
}
