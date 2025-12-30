//go:build integration

package transport

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SSH Integration Tests
//
// These tests require a running SSH server and are skipped by default.
// To run these tests:
//
//   1. Ensure SSH is accessible on localhost:
//      - macOS: System Preferences > Sharing > Remote Login
//      - Linux: sudo systemctl start sshd
//
//   2. Set environment variable:
//      export PREFLIGHT_SSH_TEST=1
//
//   3. Run with integration tag:
//      go test -tags=integration -v ./internal/domain/fleet/transport/...
//
// Environment variables:
//   PREFLIGHT_SSH_TEST      - Set to "1" to enable SSH tests
//   PREFLIGHT_SSH_HOST      - SSH hostname (default: localhost)
//   PREFLIGHT_SSH_USER      - SSH username (default: current user)
//   PREFLIGHT_SSH_PORT      - SSH port (default: 22)
//   PREFLIGHT_SSH_KEY       - Path to SSH private key (default: ~/.ssh/id_ed25519)

func skipUnlessSSHEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv("PREFLIGHT_SSH_TEST") != "1" {
		t.Skip("SSH integration tests disabled. Set PREFLIGHT_SSH_TEST=1 to enable.")
	}
}

func getSSHTestConfig(t *testing.T) fleet.SSHConfig {
	t.Helper()

	host := os.Getenv("PREFLIGHT_SSH_HOST")
	if host == "" {
		host = "localhost"
	}

	user := os.Getenv("PREFLIGHT_SSH_USER")
	if user == "" {
		user = os.Getenv("USER")
	}

	port := 22
	if portStr := os.Getenv("PREFLIGHT_SSH_PORT"); portStr != "" {
		// Parse port if needed
		port = 22
	}

	keyPath := os.Getenv("PREFLIGHT_SSH_KEY")
	if keyPath == "" {
		home, _ := os.UserHomeDir()
		keyPath = filepath.Join(home, ".ssh", "id_ed25519")
		// Fall back to id_rsa if ed25519 doesn't exist
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			keyPath = filepath.Join(home, ".ssh", "id_rsa")
		}
	}

	return fleet.SSHConfig{
		Hostname:       host,
		User:           user,
		Port:           port,
		IdentityFile:   keyPath,
		ConnectTimeout: 10 * time.Second,
	}
}

func createSSHTestHost(t *testing.T) *fleet.Host {
	t.Helper()
	config := getSSHTestConfig(t)
	id, err := fleet.NewHostID("ssh-test-host")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, config)
	require.NoError(t, err)
	return host
}

func TestSSHTransport_Integration_Ping(t *testing.T) {
	skipUnlessSSHEnabled(t)

	transport := NewSSHTransport()
	host := createSSHTestHost(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := transport.Ping(ctx, host)
	require.NoError(t, err, "SSH ping should succeed")
	assert.Equal(t, fleet.HostStatusOnline, host.Status())
}

func TestSSHTransport_Integration_Connect(t *testing.T) {
	skipUnlessSSHEnabled(t)

	transport := NewSSHTransport()
	host := createSSHTestHost(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err, "SSH connection should succeed")
	defer func() { _ = conn.Close() }()

	assert.Equal(t, host, conn.Host())
	assert.Equal(t, fleet.HostStatusOnline, host.Status())
}

func TestSSHConnection_Integration_Run(t *testing.T) {
	skipUnlessSSHEnabled(t)

	transport := NewSSHTransport()
	host := createSSHTestHost(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	t.Run("echo command", func(t *testing.T) {
		result, err := conn.Run(ctx, "echo 'hello from SSH'")
		require.NoError(t, err)
		assert.True(t, result.Success())
		assert.Contains(t, string(result.Stdout), "hello from SSH")
	})

	t.Run("pwd command", func(t *testing.T) {
		result, err := conn.Run(ctx, "pwd")
		require.NoError(t, err)
		assert.True(t, result.Success())
		assert.NotEmpty(t, strings.TrimSpace(string(result.Stdout)))
	})

	t.Run("exit code", func(t *testing.T) {
		result, err := conn.Run(ctx, "exit 42")
		require.NoError(t, err)
		assert.False(t, result.Success())
		assert.Equal(t, 42, result.ExitCode)
	})

	t.Run("stderr output", func(t *testing.T) {
		result, err := conn.Run(ctx, "echo 'error message' >&2")
		require.NoError(t, err)
		assert.Contains(t, string(result.Stderr), "error message")
	})

	t.Run("multiline output", func(t *testing.T) {
		result, err := conn.Run(ctx, "printf 'line1\\nline2\\nline3'")
		require.NoError(t, err)
		assert.True(t, result.Success())
		lines := strings.Split(strings.TrimSpace(string(result.Stdout)), "\n")
		assert.Len(t, lines, 3)
	})

	t.Run("environment variable", func(t *testing.T) {
		result, err := conn.Run(ctx, "echo $HOME")
		require.NoError(t, err)
		assert.True(t, result.Success())
		assert.NotEmpty(t, strings.TrimSpace(string(result.Stdout)))
	})
}

func TestSSHConnection_Integration_RunWithInput(t *testing.T) {
	skipUnlessSSHEnabled(t)

	transport := NewSSHTransport()
	host := createSSHTestHost(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	t.Run("cat with stdin", func(t *testing.T) {
		input := "hello from stdin\n"
		result, err := conn.RunWithInput(ctx, "cat", strings.NewReader(input))
		require.NoError(t, err)
		assert.True(t, result.Success())
		assert.Equal(t, input, string(result.Stdout))
	})

	t.Run("wc with stdin", func(t *testing.T) {
		input := "one\ntwo\nthree\n"
		result, err := conn.RunWithInput(ctx, "wc -l", strings.NewReader(input))
		require.NoError(t, err)
		assert.True(t, result.Success())
		assert.Contains(t, string(result.Stdout), "3")
	})
}

func TestSSHConnection_Integration_FileOperations(t *testing.T) {
	skipUnlessSSHEnabled(t)

	transport := NewSSHTransport()
	host := createSSHTestHost(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Create local temp directory
	localDir := t.TempDir()

	// Get remote temp directory
	result, err := conn.Run(ctx, "mktemp -d")
	require.NoError(t, err)
	remoteDir := strings.TrimSpace(string(result.Stdout))

	// Ensure cleanup of remote temp directory
	defer func() {
		_, _ = conn.Run(ctx, "rm -rf "+remoteDir)
	}()

	t.Run("upload file", func(t *testing.T) {
		// Create local file
		localPath := filepath.Join(localDir, "upload-test.txt")
		content := "test content for upload\n"
		err := os.WriteFile(localPath, []byte(content), 0o644)
		require.NoError(t, err)

		// Upload
		remotePath := filepath.Join(remoteDir, "uploaded.txt")
		err = conn.Upload(ctx, localPath, remotePath)
		require.NoError(t, err)

		// Verify on remote
		result, err := conn.Run(ctx, "cat "+remotePath)
		require.NoError(t, err)
		assert.True(t, result.Success())
		assert.Equal(t, content, string(result.Stdout))
	})

	t.Run("download file", func(t *testing.T) {
		// Create file on remote
		content := "test content for download\n"
		remotePath := filepath.Join(remoteDir, "download-source.txt")
		_, err := conn.Run(ctx, "printf '"+content+"' > "+remotePath)
		require.NoError(t, err)

		// Download
		localPath := filepath.Join(localDir, "downloaded.txt")
		err = conn.Download(ctx, remotePath, localPath)
		require.NoError(t, err)

		// Verify locally
		data, err := os.ReadFile(localPath)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("upload binary file", func(t *testing.T) {
		// Create local binary file
		localPath := filepath.Join(localDir, "binary-test.bin")
		content := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		err := os.WriteFile(localPath, content, 0o644)
		require.NoError(t, err)

		// Upload
		remotePath := filepath.Join(remoteDir, "uploaded.bin")
		err = conn.Upload(ctx, localPath, remotePath)
		require.NoError(t, err)

		// Verify on remote via md5
		localMD5Result, _ := conn.Run(ctx, "md5 -q "+localPath+" 2>/dev/null || md5sum "+localPath+" | cut -d' ' -f1")
		remoteMD5Result, _ := conn.Run(ctx, "md5 -q "+remotePath+" 2>/dev/null || md5sum "+remotePath+" | cut -d' ' -f1")
		assert.Equal(t,
			strings.TrimSpace(string(localMD5Result.Stdout)),
			strings.TrimSpace(string(remoteMD5Result.Stdout)),
		)
	})
}

func TestSSHConnection_Integration_ContextCancellation(t *testing.T) {
	skipUnlessSSHEnabled(t)

	transport := NewSSHTransport()
	host := createSSHTestHost(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	t.Run("cancelled before completion", func(t *testing.T) {
		cmdCtx, cmdCancel := context.WithCancel(ctx)

		// Start a long-running command
		done := make(chan struct{})
		var runResult *CommandResult
		var runErr error

		go func() {
			runResult, runErr = conn.Run(cmdCtx, "sleep 60")
			close(done)
		}()

		// Cancel after a short delay
		time.Sleep(100 * time.Millisecond)
		cmdCancel()

		// Wait for command to finish
		select {
		case <-done:
			// Command should have been cancelled
			if runErr == nil && runResult != nil {
				// Some implementations may return non-zero exit code instead of error
				assert.False(t, runResult.Success(), "cancelled command should not succeed")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("command did not respect context cancellation")
		}
	})
}

func TestSSHConnection_Integration_Idempotence(t *testing.T) {
	skipUnlessSSHEnabled(t)

	transport := NewSSHTransport()
	host := createSSHTestHost(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Test idempotent command pattern (used by RemoteStep)
	t.Run("idempotent create directory", func(t *testing.T) {
		tmpDir, _ := conn.Run(ctx, "mktemp -d")
		testDir := strings.TrimSpace(string(tmpDir.Stdout)) + "/idempotent-test"
		defer func() {
			_, _ = conn.Run(ctx, "rm -rf "+testDir)
		}()

		// Check command (should fail first time)
		checkCmd := "test -d " + testDir
		result1, _ := conn.Run(ctx, checkCmd)
		assert.False(t, result1.Success(), "directory should not exist yet")

		// Apply command
		applyCmd := "mkdir -p " + testDir
		result2, err := conn.Run(ctx, applyCmd)
		require.NoError(t, err)
		assert.True(t, result2.Success())

		// Check command (should pass now)
		result3, _ := conn.Run(ctx, checkCmd)
		assert.True(t, result3.Success(), "directory should exist now")

		// Apply again (should be idempotent)
		result4, err := conn.Run(ctx, applyCmd)
		require.NoError(t, err)
		assert.True(t, result4.Success())
	})
}

func TestSSHTransport_Integration_MultipleConnections(t *testing.T) {
	skipUnlessSSHEnabled(t)

	transport := NewSSHTransport()
	host := createSSHTestHost(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Test multiple sequential connections
	t.Run("sequential connections", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			conn, err := transport.Connect(ctx, host)
			require.NoError(t, err, "connection %d should succeed", i)

			result, err := conn.Run(ctx, "echo test")
			require.NoError(t, err)
			assert.True(t, result.Success())

			err = conn.Close()
			require.NoError(t, err)
		}
	})

	// Test connection pool behavior
	t.Run("connection pool", func(t *testing.T) {
		pool := NewConnectionPool(transport, 5)
		defer func() { _ = pool.Close() }()

		// Get connection
		conn1, err := pool.Get(ctx, host)
		require.NoError(t, err)

		// Should return same connection
		conn2, err := pool.Get(ctx, host)
		require.NoError(t, err)
		assert.Equal(t, conn1, conn2)

		// Execute command through pooled connection
		result, err := conn1.Run(ctx, "hostname")
		require.NoError(t, err)
		assert.True(t, result.Success())
	})
}

func TestSSHTransport_Integration_ErrorHandling(t *testing.T) {
	skipUnlessSSHEnabled(t)

	transport := NewSSHTransport()

	t.Run("invalid host", func(t *testing.T) {
		id, _ := fleet.NewHostID("invalid-host")
		host, _ := fleet.NewHost(id, fleet.SSHConfig{
			Hostname:       "invalid.nonexistent.host.example",
			User:           "nobody",
			Port:           22,
			ConnectTimeout: 2 * time.Second,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := transport.Ping(ctx, host)
		assert.Error(t, err, "ping to invalid host should fail")
		assert.Equal(t, fleet.HostStatusError, host.Status())
	})

	t.Run("invalid port", func(t *testing.T) {
		id, _ := fleet.NewHostID("invalid-port")
		host, _ := fleet.NewHost(id, fleet.SSHConfig{
			Hostname:       "localhost",
			User:           os.Getenv("USER"),
			Port:           65000, // Unlikely to have SSH
			ConnectTimeout: 2 * time.Second,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := transport.Connect(ctx, host)
		assert.Error(t, err, "connection to invalid port should fail")
	})
}
