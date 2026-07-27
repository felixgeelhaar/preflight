package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
)

// errNoDaemonUnderTest is returned instead of launching the detached daemon when
// running under `go test`.
//
// Deliberately an error rather than a silent nil: a test that reaches the daemon
// launch has almost certainly not meant to, and should be told exactly that
// rather than seeing a pass. Silently succeeding would trade a fork bomb for a
// test that asserts nothing — a quieter bug, not a fixed one.
var errNoDaemonUnderTest = errors.New(
	"refusing to launch the agent daemon from a test binary: re-execing a test binary " +
		"runs the whole suite instead of the requested subcommand (see selfexec.go); " +
		"use --foreground, or exercise the daemon path through an integration test that " +
		"runs the built binary")

// underTest reports whether this process is a binary built by `go test`.
//
// It guards the one place preflight re-execs itself: `agent start` without
// --foreground re-runs this binary as a detached daemon (see runAgentStart).
//
// Under `go test` that pattern is a fork bomb. os.Executable() resolves to the
// compiled test binary, and a Go test binary handed a subcommand it does not
// recognise does NOT error — it ignores the unknown arguments and runs the whole
// suite. So a detached self-exec from a test starts a second copy of the suite,
// which reaches the same call site and starts a third. Every child is detached
// with nil stdio, so the test output looks completely normal while the machine
// fills up behind it.
//
// This is currently latent here, not active: the three tests that call
// runAgentStart all fail earlier — at the experimental gate, at schedule
// parsing, and at remediation parsing — so none reach the spawn. The guard is
// preventative. It costs nothing and removes a trap where the failure mode is
// wildly out of proportion to the mistake: one future test that supplies a
// valid schedule and policy would arm it.
//
// The same bug was live in a sibling project (klarlabs-studio/mnemos#307) and
// reached ~1000 stray processes and a load average near 900 before anyone
// noticed, because nothing about a green test run suggests it.
//
// Two independent signals, because each alone has a gap: the testing package
// registers its -test.* flags only in a test binary, and `.test` is what the Go
// toolchain names that binary. The asymmetry is deliberate — a false positive
// costs one skipped daemon launch in a test run, where nothing depends on it; a
// false negative costs the above.
func underTest() bool {
	if flag.Lookup("test.v") != nil {
		return true
	}
	self, err := os.Executable()
	return err == nil && strings.HasSuffix(filepath.Base(self), ".test")
}
