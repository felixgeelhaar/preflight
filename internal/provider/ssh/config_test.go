package ssh

import (
	"testing"
)

func TestParseConfig_EmptyMap_ReturnsEmptyConfig(t *testing.T) {
	raw := map[string]interface{}{}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(cfg.Hosts) != 0 {
		t.Errorf("Hosts len = %d, want 0", len(cfg.Hosts))
	}
}

func TestParseConfig_WithHosts_ParsesHostBlocks(t *testing.T) {
	raw := map[string]interface{}{
		"hosts": []interface{}{
			map[string]interface{}{
				"host":         "github.com",
				"hostname":     "github.com",
				"user":         "git",
				"identityfile": "~/.ssh/id_ed25519",
			},
			map[string]interface{}{
				"host":         "work",
				"hostname":     "git.company.com",
				"user":         "developer",
				"identityfile": "~/.ssh/id_work",
				"port":         22,
			},
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(cfg.Hosts) != 2 {
		t.Fatalf("Hosts len = %d, want 2", len(cfg.Hosts))
	}

	// Check first host
	if cfg.Hosts[0].Host != "github.com" {
		t.Errorf("Hosts[0].Host = %q, want %q", cfg.Hosts[0].Host, "github.com")
	}
	if cfg.Hosts[0].HostName != "github.com" {
		t.Errorf("Hosts[0].HostName = %q, want %q", cfg.Hosts[0].HostName, "github.com")
	}
	if cfg.Hosts[0].User != "git" {
		t.Errorf("Hosts[0].User = %q, want %q", cfg.Hosts[0].User, "git")
	}
	if cfg.Hosts[0].IdentityFile != "~/.ssh/id_ed25519" {
		t.Errorf("Hosts[0].IdentityFile = %q, want %q", cfg.Hosts[0].IdentityFile, "~/.ssh/id_ed25519")
	}

	// Check second host
	if cfg.Hosts[1].Host != "work" {
		t.Errorf("Hosts[1].Host = %q, want %q", cfg.Hosts[1].Host, "work")
	}
	if cfg.Hosts[1].Port != 22 {
		t.Errorf("Hosts[1].Port = %d, want %d", cfg.Hosts[1].Port, 22)
	}
}

func TestParseConfig_WithGlobalOptions_ParsesDefaults(t *testing.T) {
	raw := map[string]interface{}{
		"defaults": map[string]interface{}{
			"addkeystoagent":      true,
			"identitiesonly":      true,
			"forwardagent":        false,
			"serveralivecountmax": 3,
			"serveraliveinterval": 60,
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if !cfg.Defaults.AddKeysToAgent {
		t.Error("Defaults.AddKeysToAgent = false, want true")
	}
	if !cfg.Defaults.IdentitiesOnly {
		t.Error("Defaults.IdentitiesOnly = false, want true")
	}
	if cfg.Defaults.ForwardAgent {
		t.Error("Defaults.ForwardAgent = true, want false")
	}
	if cfg.Defaults.ServerAliveCountMax != 3 {
		t.Errorf("Defaults.ServerAliveCountMax = %d, want 3", cfg.Defaults.ServerAliveCountMax)
	}
	if cfg.Defaults.ServerAliveInterval != 60 {
		t.Errorf("Defaults.ServerAliveInterval = %d, want 60", cfg.Defaults.ServerAliveInterval)
	}
}

func TestParseConfig_WithMatch_ParsesMatchBlocks(t *testing.T) {
	raw := map[string]interface{}{
		"matches": []interface{}{
			map[string]interface{}{
				"match":        "host *.internal.company.com",
				"proxycommand": "ssh -W %h:%p bastion.company.com",
			},
		},
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(cfg.Matches) != 1 {
		t.Fatalf("Matches len = %d, want 1", len(cfg.Matches))
	}

	if cfg.Matches[0].Match != "host *.internal.company.com" {
		t.Errorf("Matches[0].Match = %q, want %q", cfg.Matches[0].Match, "host *.internal.company.com")
	}
	if cfg.Matches[0].ProxyCommand != "ssh -W %h:%p bastion.company.com" {
		t.Errorf("Matches[0].ProxyCommand = %q, want %q", cfg.Matches[0].ProxyCommand, "ssh -W %h:%p bastion.company.com")
	}
}

func TestConfig_ConfigPath_ReturnsSSHConfigPath(t *testing.T) {
	cfg := &Config{}
	if cfg.ConfigPath() != "~/.ssh/config" {
		t.Errorf("ConfigPath() = %q, want %q", cfg.ConfigPath(), "~/.ssh/config")
	}
}

func TestParseConfig_WithInclude_ParsesIncludeDirective(t *testing.T) {
	raw := map[string]interface{}{
		"include": "~/.ssh/config.d/*",
	}

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.Include != "~/.ssh/config.d/*" {
		t.Errorf("Include = %q, want %q", cfg.Include, "~/.ssh/config.d/*")
	}
}
