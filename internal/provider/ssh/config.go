// Package ssh provides the SSH configuration provider for Preflight.
// It handles generating ~/.ssh/config from declarative configuration.
package ssh

import "fmt"

// Config represents SSH configuration.
type Config struct {
	Include  string
	Defaults DefaultsConfig
	Hosts    []HostConfig
	Matches  []MatchConfig
}

// DefaultsConfig represents global SSH defaults (Host *).
type DefaultsConfig struct {
	AddKeysToAgent      bool
	IdentitiesOnly      bool
	ForwardAgent        bool
	ServerAliveCountMax int
	ServerAliveInterval int
}

// HostConfig represents a Host block in SSH config.
type HostConfig struct {
	Host           string
	HostName       string
	User           string
	Port           int
	IdentityFile   string
	IdentitiesOnly bool
	ForwardAgent   bool
	ProxyCommand   string
	ProxyJump      string
	LocalForward   string
	RemoteForward  string
	RequestTTY     string
	AddKeysToAgent bool
	UseKeychain    bool
	IgnoreUnknown  string
}

// MatchConfig represents a Match block in SSH config.
type MatchConfig struct {
	Match        string
	HostName     string
	User         string
	IdentityFile string
	ProxyCommand string
	ProxyJump    string
}

// ConfigPath returns the path to the SSH config file.
func (c *Config) ConfigPath() string {
	return "~/.ssh/config"
}

// ParseConfig parses SSH configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{}

	// Parse include directive
	if include, ok := raw["include"].(string); ok {
		cfg.Include = include
	}

	// Parse defaults
	if defaults, ok := raw["defaults"].(map[string]interface{}); ok {
		if v, ok := defaults["addkeystoagent"].(bool); ok {
			cfg.Defaults.AddKeysToAgent = v
		}
		if v, ok := defaults["identitiesonly"].(bool); ok {
			cfg.Defaults.IdentitiesOnly = v
		}
		if v, ok := defaults["forwardagent"].(bool); ok {
			cfg.Defaults.ForwardAgent = v
		}
		if v, ok := defaults["serveralivecountmax"].(int); ok {
			cfg.Defaults.ServerAliveCountMax = v
		}
		if v, ok := defaults["serveraliveinterval"].(int); ok {
			cfg.Defaults.ServerAliveInterval = v
		}
	}

	// Parse hosts
	if hosts, ok := raw["hosts"].([]interface{}); ok {
		for i, h := range hosts {
			hostMap, ok := h.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid host entry at index %d", i)
			}

			host := HostConfig{}
			if v, ok := hostMap["host"].(string); ok {
				host.Host = v
			}
			if v, ok := hostMap["hostname"].(string); ok {
				host.HostName = v
			}
			if v, ok := hostMap["user"].(string); ok {
				host.User = v
			}
			if v, ok := hostMap["port"].(int); ok {
				host.Port = v
			}
			if v, ok := hostMap["identityfile"].(string); ok {
				host.IdentityFile = v
			}
			if v, ok := hostMap["identitiesonly"].(bool); ok {
				host.IdentitiesOnly = v
			}
			if v, ok := hostMap["forwardagent"].(bool); ok {
				host.ForwardAgent = v
			}
			if v, ok := hostMap["proxycommand"].(string); ok {
				host.ProxyCommand = v
			}
			if v, ok := hostMap["proxyjump"].(string); ok {
				host.ProxyJump = v
			}
			if v, ok := hostMap["localforward"].(string); ok {
				host.LocalForward = v
			}
			if v, ok := hostMap["remoteforward"].(string); ok {
				host.RemoteForward = v
			}
			if v, ok := hostMap["requesttty"].(string); ok {
				host.RequestTTY = v
			}
			if v, ok := hostMap["addkeystoagent"].(bool); ok {
				host.AddKeysToAgent = v
			}
			if v, ok := hostMap["usekeychain"].(bool); ok {
				host.UseKeychain = v
			}
			if v, ok := hostMap["ignoreunknown"].(string); ok {
				host.IgnoreUnknown = v
			}

			cfg.Hosts = append(cfg.Hosts, host)
		}
	}

	// Parse matches
	if matches, ok := raw["matches"].([]interface{}); ok {
		for i, m := range matches {
			matchMap, ok := m.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid match entry at index %d", i)
			}

			match := MatchConfig{}
			if v, ok := matchMap["match"].(string); ok {
				match.Match = v
			}
			if v, ok := matchMap["hostname"].(string); ok {
				match.HostName = v
			}
			if v, ok := matchMap["user"].(string); ok {
				match.User = v
			}
			if v, ok := matchMap["identityfile"].(string); ok {
				match.IdentityFile = v
			}
			if v, ok := matchMap["proxycommand"].(string); ok {
				match.ProxyCommand = v
			}
			if v, ok := matchMap["proxyjump"].(string); ok {
				match.ProxyJump = v
			}

			cfg.Matches = append(cfg.Matches, match)
		}
	}

	return cfg, nil
}
