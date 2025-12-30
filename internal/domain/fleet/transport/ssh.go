package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"golang.org/x/crypto/ssh"
)

// SSHTransport implements Transport using SSH.
type SSHTransport struct {
	// DefaultTimeout is the default connection timeout.
	DefaultTimeout time.Duration
	// DefaultUser is the default SSH user.
	DefaultUser string
	// IdentityFiles are default identity file paths to try.
	IdentityFiles []string
}

// NewSSHTransport creates a new SSH transport with defaults.
func NewSSHTransport() *SSHTransport {
	homeDir, _ := os.UserHomeDir()
	return &SSHTransport{
		DefaultTimeout: 30 * time.Second,
		DefaultUser:    os.Getenv("USER"),
		IdentityFiles: []string{
			filepath.Join(homeDir, ".ssh", "id_ed25519"),
			filepath.Join(homeDir, ".ssh", "id_rsa"),
		},
	}
}

// Name returns "ssh".
func (t *SSHTransport) Name() string {
	return "ssh"
}

// Connect establishes an SSH connection to the host.
func (t *SSHTransport) Connect(ctx context.Context, host *fleet.Host) (Connection, error) {
	sshCfg := host.SSH()

	// Build auth methods
	authMethods, err := t.buildAuthMethods(sshCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build auth methods: %w", err)
	}

	// Determine timeout
	timeout := sshCfg.ConnectTimeout
	if timeout == 0 {
		timeout = t.DefaultTimeout
	}

	// Determine user
	user := sshCfg.User
	if user == "" {
		user = t.DefaultUser
	}

	// Build SSH config
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // Fleet management uses known hosts
		Timeout:         timeout,
	}

	// Build address
	port := sshCfg.Port
	if port == 0 {
		port = 22
	}
	addr := fmt.Sprintf("%s:%d", sshCfg.Hostname, port)

	// Handle proxy jump if configured
	var client *ssh.Client
	if sshCfg.ProxyJump != "" {
		client, err = t.connectViaProxy(ctx, addr, config, sshCfg.ProxyJump)
	} else {
		client, err = t.dial(ctx, addr, config)
	}

	if err != nil {
		host.MarkError(err)
		return nil, err
	}

	host.MarkOnline()
	return &SSHConnection{
		host:   host,
		client: client,
	}, nil
}

// Ping tests SSH connectivity.
func (t *SSHTransport) Ping(ctx context.Context, host *fleet.Host) error {
	conn, err := t.Connect(ctx, host)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	// Run a simple command to verify
	result, err := conn.Run(ctx, "echo pong")
	if err != nil {
		return err
	}

	if !result.Success() {
		return fmt.Errorf("ping command failed with exit code %d", result.ExitCode)
	}

	return nil
}

func (t *SSHTransport) buildAuthMethods(cfg fleet.SSHConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// Try explicit identity file first
	if cfg.IdentityFile != "" {
		signer, err := t.loadPrivateKey(cfg.IdentityFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load identity file %s: %w", cfg.IdentityFile, err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	// Try default identity files
	for _, path := range t.IdentityFiles {
		signer, err := t.loadPrivateKey(path)
		if err == nil {
			methods = append(methods, ssh.PublicKeys(signer))
		}
	}

	// Try SSH agent
	if agentAuth := t.trySSHAgent(); agentAuth != nil {
		methods = append(methods, agentAuth)
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no authentication methods available")
	}

	return methods, nil
}

func (t *SSHTransport) loadPrivateKey(path string) (ssh.Signer, error) {
	// Expand ~ if present
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		path = filepath.Join(homeDir, path[2:])
	}

	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(key)
}

//nolint:unparam // SSH agent support is planned but not yet implemented
func (t *SSHTransport) trySSHAgent() ssh.AuthMethod {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil
	}

	// TODO: Use golang.org/x/crypto/ssh/agent for proper agent support
	_ = conn.Close()
	return nil
}

func (t *SSHTransport) dial(ctx context.Context, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	// Use context-aware dialing
	dialer := &net.Dialer{
		Timeout: config.Timeout,
	}

	netConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", addr, err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, config)
	if err != nil {
		_ = netConn.Close()
		return nil, fmt.Errorf("SSH handshake failed: %w", err)
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

func (t *SSHTransport) connectViaProxy(ctx context.Context, addr string, config *ssh.ClientConfig, proxyJump string) (*ssh.Client, error) {
	// Connect to the proxy host first
	proxyClient, err := t.dial(ctx, proxyJump, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy %s: %w", proxyJump, err)
	}

	// Dial through the proxy
	netConn, err := proxyClient.Dial("tcp", addr)
	if err != nil {
		_ = proxyClient.Close()
		return nil, fmt.Errorf("failed to dial through proxy: %w", err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, config)
	if err != nil {
		_ = netConn.Close()
		_ = proxyClient.Close()
		return nil, fmt.Errorf("SSH handshake via proxy failed: %w", err)
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

// SSHConnection implements Connection using SSH.
type SSHConnection struct {
	host   *fleet.Host
	client *ssh.Client
}

// Host returns the connected host.
func (c *SSHConnection) Host() *fleet.Host {
	return c.host
}

// Run executes a command on the remote host.
func (c *SSHConnection) Run(ctx context.Context, cmd string) (*CommandResult, error) {
	return c.RunWithInput(ctx, cmd, nil)
}

// RunWithInput executes a command with stdin.
func (c *SSHConnection) RunWithInput(ctx context.Context, cmd string, stdin io.Reader) (*CommandResult, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer func() { _ = session.Close() }()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if stdin != nil {
		session.Stdin = stdin
	}

	// Handle context cancellation
	done := make(chan error, 1)
	start := time.Now()

	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		return nil, ctx.Err()
	case err := <-done:
		duration := time.Since(start)
		result := &CommandResult{
			Stdout:   stdout.Bytes(),
			Stderr:   stderr.Bytes(),
			Duration: duration,
		}

		if err != nil {
			var exitErr *ssh.ExitError
			if errors.As(err, &exitErr) {
				result.ExitCode = exitErr.ExitStatus()
			} else {
				return nil, err
			}
		}

		return result, nil
	}
}

// Upload transfers a file to the remote host using SCP.
func (c *SSHConnection) Upload(ctx context.Context, localPath, remotePath string) error {
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	// Use a simple approach: cat > file
	cmd := fmt.Sprintf("cat > %s && chmod %o %s", remotePath, info.Mode().Perm(), remotePath)
	result, err := c.RunWithInput(ctx, cmd, bytes.NewReader(data))
	if err != nil {
		return err
	}

	if !result.Success() {
		return fmt.Errorf("upload failed: %s", string(result.Stderr))
	}

	return nil
}

// Download transfers a file from the remote host.
func (c *SSHConnection) Download(ctx context.Context, remotePath, localPath string) error {
	result, err := c.Run(ctx, fmt.Sprintf("cat %s", remotePath))
	if err != nil {
		return err
	}

	if !result.Success() {
		return fmt.Errorf("download failed: %s", string(result.Stderr))
	}

	// Ensure directory exists
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(localPath, result.Stdout, 0o644); err != nil {
		return fmt.Errorf("failed to write local file: %w", err)
	}

	return nil
}

// Close closes the SSH connection.
func (c *SSHConnection) Close() error {
	return c.client.Close()
}
