// Package git provides the Git provider for managing git configuration.
package git

import (
	"crypto/sha256"
	"fmt"
)

// Config represents the git section of the configuration.
type Config struct {
	Path     string            // Custom path for .gitconfig (default: ~/.gitconfig)
	User     UserConfig        // [user] section
	Core     CoreConfig        // [core] section
	Commit   CommitConfig      // [commit] section
	GPG      GPGConfig         // [gpg] section
	Aliases  map[string]string // [alias] section
	Includes []Include         // [includeIf] directives for identity separation
}

// UserConfig represents the [user] section.
type UserConfig struct {
	Name       string
	Email      string
	SigningKey string
}

// CoreConfig represents the [core] section.
type CoreConfig struct {
	Editor       string
	AutoCRLF     string
	ExcludesFile string
}

// CommitConfig represents the [commit] section.
type CommitConfig struct {
	GPGSign bool
}

// GPGConfig represents the [gpg] section.
type GPGConfig struct {
	Format  string // openpgp, x509, ssh
	Program string
}

// Include represents a conditional include directive.
type Include struct {
	Path     string // Path to included config file
	IfConfig string // Condition (e.g., "gitdir:~/work/")
}

// ID returns a unique identifier for this include.
func (i Include) ID() string {
	h := sha256.Sum256([]byte(i.Path + i.IfConfig))
	return fmt.Sprintf("%x", h[:8])
}

// ConfigPath returns the path for the git config file.
func (c Config) ConfigPath() string {
	if c.Path != "" {
		return c.Path
	}
	return "~/.gitconfig"
}

// ParseConfig parses the git configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Aliases:  make(map[string]string),
		Includes: make([]Include, 0),
	}

	// Parse user config
	if user, ok := raw["user"].(map[string]interface{}); ok {
		if name, ok := user["name"].(string); ok {
			cfg.User.Name = name
		}
		if email, ok := user["email"].(string); ok {
			cfg.User.Email = email
		}
		if signingKey, ok := user["signingkey"].(string); ok {
			cfg.User.SigningKey = signingKey
		}
	}

	// Parse core config
	if core, ok := raw["core"].(map[string]interface{}); ok {
		if editor, ok := core["editor"].(string); ok {
			cfg.Core.Editor = editor
		}
		if autocrlf, ok := core["autocrlf"].(string); ok {
			cfg.Core.AutoCRLF = autocrlf
		}
		if excludesFile, ok := core["excludesfile"].(string); ok {
			cfg.Core.ExcludesFile = excludesFile
		}
	}

	// Parse commit config
	if commit, ok := raw["commit"].(map[string]interface{}); ok {
		if gpgSign, ok := commit["gpgsign"].(bool); ok {
			cfg.Commit.GPGSign = gpgSign
		}
	}

	// Parse gpg config
	if gpg, ok := raw["gpg"].(map[string]interface{}); ok {
		if format, ok := gpg["format"].(string); ok {
			cfg.GPG.Format = format
		}
		if program, ok := gpg["program"].(string); ok {
			cfg.GPG.Program = program
		}
	}

	// Parse aliases
	if aliases, ok := raw["alias"].(map[string]interface{}); ok {
		for name, cmd := range aliases {
			if cmdStr, ok := cmd.(string); ok {
				cfg.Aliases[name] = cmdStr
			}
		}
	}

	// Parse includes
	if includes, ok := raw["includes"]; ok {
		includesList, ok := includes.([]interface{})
		if !ok {
			return nil, fmt.Errorf("includes must be a list")
		}
		for _, inc := range includesList {
			incMap, ok := inc.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("include must be an object")
			}
			include := Include{}
			if path, ok := incMap["path"].(string); ok {
				include.Path = path
			}
			if ifConfig, ok := incMap["ifconfig"].(string); ok {
				include.IfConfig = ifConfig
			}
			cfg.Includes = append(cfg.Includes, include)
		}
	}

	return cfg, nil
}
