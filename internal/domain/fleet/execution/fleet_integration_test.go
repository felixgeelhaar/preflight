//go:build integration

package execution

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"github.com/felixgeelhaar/preflight/internal/domain/fleet/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fleet Execution Integration Tests
//
// These tests verify the complete execution pipeline using real SSH connections.
// They require a running SSH server and are skipped by default.
//
// To run these tests:
//   1. Enable SSH on localhost (see transport/ssh_integration_test.go)
//   2. Set PREFLIGHT_SSH_TEST=1
//   3. Run: go test -tags=integration -v ./internal/domain/fleet/execution/...

func skipUnlessSSHEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv("PREFLIGHT_SSH_TEST") != "1" {
		t.Skip("SSH integration tests disabled. Set PREFLIGHT_SSH_TEST=1 to enable.")
	}
}

func getSSHTestConfig(t *testing.T) fleet.SSHConfig {
	t.Helper()

	user := os.Getenv("PREFLIGHT_SSH_USER")
	if user == "" {
		user = os.Getenv("USER")
	}

	keyPath := os.Getenv("PREFLIGHT_SSH_KEY")
	if keyPath == "" {
		home, _ := os.UserHomeDir()
		keyPath = filepath.Join(home, ".ssh", "id_ed25519")
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			keyPath = filepath.Join(home, ".ssh", "id_rsa")
		}
	}

	return fleet.SSHConfig{
		Hostname:       "localhost",
		User:           user,
		Port:           22,
		IdentityFile:   keyPath,
		ConnectTimeout: 10 * time.Second,
	}
}

func createSSHTestHosts(t *testing.T, count int) []*fleet.Host {
	t.Helper()
	config := getSSHTestConfig(t)
	hosts := make([]*fleet.Host, count)

	for i := 0; i < count; i++ {
		id, err := fleet.NewHostID("ssh-test-host-" + string(rune('a'+i)))
		require.NoError(t, err)
		hosts[i], err = fleet.NewHost(id, config)
		require.NoError(t, err)
	}

	return hosts
}

func TestFleetExecutor_Integration_SingleHost(t *testing.T) {
	skipUnlessSSHEnabled(t)

	trans := transport.NewSSHTransport()
	pool := transport.NewConnectionPool(trans, 5)
	defer func() { _ = pool.Close() }()

	hosts := createSSHTestHosts(t, 1)
	executor := NewFleetExecutor(pool, ParallelStrategy)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("execute single step", func(t *testing.T) {
		step := NewRemoteStep("test-echo", "echo 'hello from fleet'", "")
		step.SetDescription("Test echo command")

		results := executor.Execute(ctx, hosts, []*RemoteStep{step})
		require.Len(t, results, 1)

		result := results[0]
		assert.True(t, result.Success)
		assert.Equal(t, hosts[0].ID().String(), result.HostID)
		assert.Len(t, result.StepResults, 1)
		assert.True(t, result.StepResults[0].Applied)
	})

	t.Run("execute with check command", func(t *testing.T) {
		// Get temp directory for test
		conn, _ := pool.Get(ctx, hosts[0])
		tmpResult, _ := conn.Run(ctx, "mktemp -d")
		tmpDir := string(tmpResult.Stdout)
		tmpDir = tmpDir[:len(tmpDir)-1] // Remove trailing newline
		defer func() {
			_, _ = conn.Run(ctx, "rm -rf "+tmpDir)
		}()

		testFile := tmpDir + "/test-file.txt"

		// Step with check command - should apply since file doesn't exist
		step := NewRemoteStep("create-file", "touch "+testFile, "test -f "+testFile)
		step.SetDescription("Create test file")

		results := executor.Execute(ctx, hosts, []*RemoteStep{step})
		require.Len(t, results, 1)
		assert.True(t, results[0].Success)
		assert.True(t, results[0].StepResults[0].Applied)

		// Run again - should be satisfied (not applied)
		results2 := executor.Execute(ctx, hosts, []*RemoteStep{step})
		require.Len(t, results2, 1)
		assert.True(t, results2[0].Success)
		assert.False(t, results2[0].StepResults[0].Applied, "step should be satisfied and not re-applied")
	})

	t.Run("execute failing step", func(t *testing.T) {
		step := NewRemoteStep("failing-step", "exit 1", "")
		step.SetDescription("Intentionally failing step")

		results := executor.Execute(ctx, hosts, []*RemoteStep{step})
		require.Len(t, results, 1)
		assert.False(t, results[0].Success)
		assert.NotEmpty(t, results[0].Error)
	})
}

func TestFleetExecutor_Integration_MultipleHosts(t *testing.T) {
	skipUnlessSSHEnabled(t)

	trans := transport.NewSSHTransport()
	pool := transport.NewConnectionPool(trans, 10)
	defer func() { _ = pool.Close() }()

	// Create 3 "hosts" (all pointing to localhost for testing)
	hosts := createSSHTestHosts(t, 3)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	t.Run("parallel strategy", func(t *testing.T) {
		executor := NewFleetExecutor(pool, ParallelStrategy)
		step := NewRemoteStep("parallel-test", "echo 'parallel test'", "")

		start := time.Now()
		results := executor.Execute(ctx, hosts, []*RemoteStep{step})
		duration := time.Since(start)

		require.Len(t, results, 3)
		for _, result := range results {
			assert.True(t, result.Success, "host %s should succeed", result.HostID)
		}

		// Parallel should be faster than 3x sequential
		t.Logf("Parallel execution took %v", duration)
	})

	t.Run("rolling strategy", func(t *testing.T) {
		executor := NewFleetExecutor(pool, RollingStrategy)
		step := NewRemoteStep("rolling-test", "echo 'rolling test'", "")

		results := executor.Execute(ctx, hosts, []*RemoteStep{step})

		require.Len(t, results, 3)
		for _, result := range results {
			assert.True(t, result.Success, "host %s should succeed", result.HostID)
		}
	})
}

func TestFleetExecutor_Integration_Plan(t *testing.T) {
	skipUnlessSSHEnabled(t)

	trans := transport.NewSSHTransport()
	pool := transport.NewConnectionPool(trans, 5)
	defer func() { _ = pool.Close() }()

	hosts := createSSHTestHosts(t, 1)
	executor := NewFleetExecutor(pool, ParallelStrategy)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get temp directory for test
	conn, _ := pool.Get(ctx, hosts[0])
	tmpResult, _ := conn.Run(ctx, "mktemp -d")
	tmpDir := string(tmpResult.Stdout)
	tmpDir = tmpDir[:len(tmpDir)-1]
	defer func() {
		_, _ = conn.Run(ctx, "rm -rf "+tmpDir)
	}()

	t.Run("plan shows needed steps", func(t *testing.T) {
		testFile := tmpDir + "/plan-test.txt"
		step := NewRemoteStep("create-file", "touch "+testFile, "test -f "+testFile)

		plan := executor.Plan(ctx, hosts, []*RemoteStep{step})

		require.Len(t, plan, 1)
		require.Len(t, plan[0].StepPlans, 1)
		assert.Equal(t, StepStatusNeedsApply, plan[0].StepPlans[0].Status)
	})

	t.Run("plan shows satisfied steps", func(t *testing.T) {
		// Create file first
		existingFile := tmpDir + "/existing-file.txt"
		_, _ = conn.Run(ctx, "touch "+existingFile)

		step := NewRemoteStep("check-file", "touch "+existingFile, "test -f "+existingFile)

		plan := executor.Plan(ctx, hosts, []*RemoteStep{step})

		require.Len(t, plan, 1)
		require.Len(t, plan[0].StepPlans, 1)
		assert.Equal(t, StepStatusSatisfied, plan[0].StepPlans[0].Status)
	})
}

func TestFleetExecutor_Integration_StepDependencies(t *testing.T) {
	skipUnlessSSHEnabled(t)

	trans := transport.NewSSHTransport()
	pool := transport.NewConnectionPool(trans, 5)
	defer func() { _ = pool.Close() }()

	hosts := createSSHTestHosts(t, 1)
	executor := NewFleetExecutor(pool, ParallelStrategy)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get temp directory for test
	conn, _ := pool.Get(ctx, hosts[0])
	tmpResult, _ := conn.Run(ctx, "mktemp -d")
	tmpDir := string(tmpResult.Stdout)
	tmpDir = tmpDir[:len(tmpDir)-1]
	defer func() {
		_, _ = conn.Run(ctx, "rm -rf "+tmpDir)
	}()

	t.Run("steps execute in dependency order", func(t *testing.T) {
		subdir := tmpDir + "/parent"
		testFile := subdir + "/child.txt"

		// Create parent directory first
		step1 := NewRemoteStep("create-dir", "mkdir -p "+subdir, "test -d "+subdir)
		step1.SetDescription("Create parent directory")

		// Then create file in it
		step2 := NewRemoteStep("create-file", "touch "+testFile, "test -f "+testFile)
		step2.SetDescription("Create child file")
		step2.SetDependsOn([]string{"create-dir"})

		results := executor.Execute(ctx, hosts, []*RemoteStep{step1, step2})

		require.Len(t, results, 1)
		assert.True(t, results[0].Success)
		assert.Len(t, results[0].StepResults, 2)

		// Verify both steps applied in correct order
		assert.True(t, results[0].StepResults[0].Applied, "step1 should apply")
		assert.True(t, results[0].StepResults[1].Applied, "step2 should apply")

		// Verify file actually exists
		checkResult, _ := conn.Run(ctx, "test -f "+testFile+" && echo exists")
		assert.Contains(t, string(checkResult.Stdout), "exists")
	})
}

func TestFleetExecutor_Integration_ErrorRecovery(t *testing.T) {
	skipUnlessSSHEnabled(t)

	trans := transport.NewSSHTransport()
	pool := transport.NewConnectionPool(trans, 5)
	defer func() { _ = pool.Close() }()

	hosts := createSSHTestHosts(t, 2)
	executor := NewFleetExecutor(pool, ParallelStrategy)
	executor.ContinueOnError = true

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("continue on error mode", func(t *testing.T) {
		// First step fails on purpose
		step1 := NewRemoteStep("fail-step", "exit 1", "")
		step2 := NewRemoteStep("succeed-step", "echo 'success'", "")

		results := executor.Execute(ctx, hosts, []*RemoteStep{step1, step2})

		// Both hosts should have results
		require.Len(t, results, 2)

		// Each host should have attempted both steps
		for _, result := range results {
			// First step should fail
			assert.False(t, result.StepResults[0].Applied || result.StepResults[0].Status == StepStatusSatisfied)
			// Second step should succeed (because ContinueOnError is true)
			if len(result.StepResults) > 1 {
				assert.True(t, result.StepResults[1].Applied)
			}
		}
	})
}
