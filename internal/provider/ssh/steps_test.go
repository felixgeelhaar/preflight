package ssh

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestSSHConfigStep_ID(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "github.com", User: "git"},
		},
	}

	step := NewConfigStep(cfg, fs)
	id := step.ID()

	if id.String() != "ssh:config" {
		t.Errorf("ID() = %q, want %q", id.String(), "ssh:config")
	}
}

func TestSSHConfigStep_DependsOn(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{}
	step := NewConfigStep(cfg, fs)

	deps := step.DependsOn()
	if len(deps) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(deps))
	}
}

func TestSSHConfigStep_Check_NotExists(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "github.com", User: "git"},
		},
	}

	step := NewConfigStep(cfg, fs)
	status, err := step.Check(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestSSHConfigStep_Check_ExistsWithSameContent(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "github.com", User: "git", IdentityFile: "~/.ssh/id_ed25519"},
		},
	}

	// Write expected content first
	step := NewConfigStep(cfg, fs)
	content := step.generateConfig()
	fs.SetFileContent(ports.ExpandPath("~/.ssh/config"), content)

	status, err := step.Check(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusSatisfied {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusSatisfied)
	}
}

func TestSSHConfigStep_Check_ExistsWithDifferentContent(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "github.com", User: "git"},
		},
	}

	// Write different content
	fs.SetFileContent(ports.ExpandPath("~/.ssh/config"), []byte("different content"))

	step := NewConfigStep(cfg, fs)
	status, err := step.Check(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != compiler.StatusNeedsApply {
		t.Errorf("Check() = %v, want %v", status, compiler.StatusNeedsApply)
	}
}

func TestSSHConfigStep_Apply(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{
				Host:         "github.com",
				HostName:     "github.com",
				User:         "git",
				IdentityFile: "~/.ssh/id_ed25519",
			},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Verify file was written
	path := ports.ExpandPath("~/.ssh/config")
	if !fs.Exists(path) {
		t.Error("Apply() did not create file")
	}

	content, err := fs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Check content contains expected values
	contentStr := string(content)
	if !contains(contentStr, "Host github.com") {
		t.Error("config should contain Host github.com")
	}
	if !contains(contentStr, "HostName github.com") {
		t.Error("config should contain HostName")
	}
	if !contains(contentStr, "User git") {
		t.Error("config should contain User")
	}
	if !contains(contentStr, "IdentityFile ~/.ssh/id_ed25519") {
		t.Error("config should contain IdentityFile")
	}
}

func TestSSHConfigStep_Apply_WithDefaults(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Defaults: DefaultsConfig{
			AddKeysToAgent:      true,
			IdentitiesOnly:      true,
			ServerAliveInterval: 60,
			ServerAliveCountMax: 3,
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	path := ports.ExpandPath("~/.ssh/config")
	content, _ := fs.ReadFile(path)
	contentStr := string(content)

	if !contains(contentStr, "Host *") {
		t.Error("config should contain Host * for defaults")
	}
	if !contains(contentStr, "AddKeysToAgent yes") {
		t.Error("config should contain AddKeysToAgent yes")
	}
	if !contains(contentStr, "IdentitiesOnly yes") {
		t.Error("config should contain IdentitiesOnly yes")
	}
	if !contains(contentStr, "ServerAliveInterval 60") {
		t.Error("config should contain ServerAliveInterval")
	}
	if !contains(contentStr, "ServerAliveCountMax 3") {
		t.Error("config should contain ServerAliveCountMax")
	}
}

func TestSSHConfigStep_Apply_WithInclude(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Include: "~/.ssh/config.d/*",
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	path := ports.ExpandPath("~/.ssh/config")
	content, _ := fs.ReadFile(path)
	contentStr := string(content)

	if !contains(contentStr, "Include ~/.ssh/config.d/*") {
		t.Error("config should contain Include directive")
	}
}

func TestSSHConfigStep_Apply_WithMatch(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Matches: []MatchConfig{
			{
				Match:        "host *.internal.company.com",
				ProxyCommand: "ssh -W %h:%p bastion",
			},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	path := ports.ExpandPath("~/.ssh/config")
	content, _ := fs.ReadFile(path)
	contentStr := string(content)

	if !contains(contentStr, "Match host *.internal.company.com") {
		t.Error("config should contain Match block")
	}
	if !contains(contentStr, "ProxyCommand ssh -W %h:%p bastion") {
		t.Error("config should contain ProxyCommand in Match block")
	}
}

func TestSSHConfigStep_Apply_WithProxyJump(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{
				Host:      "internal-server",
				HostName:  "10.0.0.5",
				User:      "admin",
				ProxyJump: "bastion.company.com",
			},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	path := ports.ExpandPath("~/.ssh/config")
	content, _ := fs.ReadFile(path)
	contentStr := string(content)

	if !contains(contentStr, "ProxyJump bastion.company.com") {
		t.Error("config should contain ProxyJump")
	}
}

func TestSSHConfigStep_Plan(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "github.com", User: "git"},
		},
	}

	step := NewConfigStep(cfg, fs)
	diff, err := step.Plan(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.IsEmpty() {
		t.Error("Plan() returned empty diff")
	}
}

func TestSSHConfigStep_Explain(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "github.com", User: "git"},
		},
	}

	step := NewConfigStep(cfg, fs)
	explanation := step.Explain(compiler.ExplainContext{})

	if explanation.Summary() == "" {
		t.Error("Explain() returned empty summary")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Validation error tests

func TestSSHConfigStep_Apply_InvalidInclude(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Include: "invalid\ninclude", // Contains newline - invalid
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err == nil {
		t.Error("Apply() should return error for invalid Include")
	}
}

func TestSSHConfigStep_Apply_InvalidHostPattern(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "invalid\nhost"}, // Contains newline - invalid
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err == nil {
		t.Error("Apply() should return error for invalid Host pattern")
	}
}

func TestSSHConfigStep_Apply_InvalidHostName(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "myhost", HostName: "invalid\nhostname"},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err == nil {
		t.Error("Apply() should return error for invalid HostName")
	}
}

func TestSSHConfigStep_Apply_InvalidUser(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "myhost", User: "invalid\nuser"},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err == nil {
		t.Error("Apply() should return error for invalid User")
	}
}

func TestSSHConfigStep_Apply_InvalidIdentityFile(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "myhost", IdentityFile: "../../../etc/passwd"},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err == nil {
		t.Error("Apply() should return error for invalid IdentityFile")
	}
}

func TestSSHConfigStep_Apply_InvalidProxyCommand(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "myhost", ProxyCommand: "ssh; rm -rf /"},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err == nil {
		t.Error("Apply() should return error for invalid ProxyCommand")
	}
}

func TestSSHConfigStep_Apply_InvalidProxyJump(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "myhost", ProxyJump: "invalid\nproxyjump"},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err == nil {
		t.Error("Apply() should return error for invalid ProxyJump")
	}
}

func TestSSHConfigStep_Apply_InvalidLocalForward(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "myhost", LocalForward: "invalid\nforward"},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err == nil {
		t.Error("Apply() should return error for invalid LocalForward")
	}
}

func TestSSHConfigStep_Apply_InvalidRemoteForward(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "myhost", RemoteForward: "invalid\nforward"},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err == nil {
		t.Error("Apply() should return error for invalid RemoteForward")
	}
}

func TestSSHConfigStep_Apply_InvalidRequestTTY(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "myhost", RequestTTY: "invalid\ntty"},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err == nil {
		t.Error("Apply() should return error for invalid RequestTTY")
	}
}

func TestSSHConfigStep_Apply_InvalidIgnoreUnknown(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "myhost", IgnoreUnknown: "invalid\nunknown"},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err == nil {
		t.Error("Apply() should return error for invalid IgnoreUnknown")
	}
}

func TestSSHConfigStep_Apply_WithAllHostFields(t *testing.T) {
	fs := mocks.NewFileSystem()
	cfg := &Config{
		Hosts: []HostConfig{
			{
				Host:          "fullhost",
				HostName:      "full.example.com",
				User:          "admin",
				IdentityFile:  "~/.ssh/id_rsa",
				Port:          2222,
				LocalForward:  "8080:localhost:80",
				RemoteForward: "9090:localhost:90",
				RequestTTY:    "yes",
				IgnoreUnknown: "SomeOption",
			},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	path := ports.ExpandPath("~/.ssh/config")
	content, _ := fs.ReadFile(path)
	contentStr := string(content)

	if !contains(contentStr, "Port 2222") {
		t.Error("config should contain Port")
	}
	if !contains(contentStr, "LocalForward 8080:localhost:80") {
		t.Error("config should contain LocalForward")
	}
	if !contains(contentStr, "RemoteForward 9090:localhost:90") {
		t.Error("config should contain RemoteForward")
	}
	if !contains(contentStr, "RequestTTY yes") {
		t.Error("config should contain RequestTTY")
	}
	if !contains(contentStr, "IgnoreUnknown SomeOption") {
		t.Error("config should contain IgnoreUnknown")
	}
}

func TestSSHConfigStep_Apply_WithSSHDirCreation(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Clear any existing .ssh directory
	cfg := &Config{
		Hosts: []HostConfig{
			{Host: "github.com", User: "git"},
		},
	}

	step := NewConfigStep(cfg, fs)
	err := step.Apply(compiler.RunContext{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Verify .ssh directory was created
	sshDir := ports.ExpandPath("~/.ssh")
	if !fs.Exists(sshDir) {
		t.Error("Apply() should create .ssh directory")
	}
}
