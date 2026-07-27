package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The guard's whole value depends on this being true inside `go test`. A false
// answer here silently re-arms the daemon self-exec, and the failure mode is a
// fork bomb that leaves the test output looking green (see selfexec.go).
func TestUnderTest_IsTrueInATestBinary(t *testing.T) {
	if !underTest() {
		self, _ := os.Executable()
		t.Fatalf("underTest() must be true inside `go test` (executable=%q)", self)
	}
}

// Reaching the daemon launch from a test must fail loudly rather than spawn, and
// rather than quietly pass. The error names the cause and the way out, because
// whoever trips this will be reading it with no idea why their test hung the
// machine.
func TestRunAgentStart_RefusesToDaemonizeUnderTest(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	orig := struct {
		schedule, remediation string
		foreground            bool
	}{
		agentSchedule, agentRemediation, agentForeground,
	}
	defer func() {
		agentSchedule, agentRemediation, agentForeground = orig.schedule, orig.remediation, orig.foreground
	}()

	// Valid everywhere, so nothing short-circuits before the daemon branch —
	// this is precisely the arrangement the existing tests avoid by failing
	// early, and the one a future test could introduce without realising.
	agentSchedule = "30m"
	agentRemediation = "notify"
	agentForeground = false

	err := runAgentStart(nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errNoDaemonUnderTest)
}
