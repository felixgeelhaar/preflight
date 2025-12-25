package capability

import (
	"fmt"
)

// SecurityConfig holds security settings from preflight.yaml.
type SecurityConfig struct {
	BlockedCapabilities []string   `yaml:"blocked_capabilities,omitempty"`
	CSPDeny             []CSPEntry `yaml:"csp_deny,omitempty"`
	CSPWarn             []CSPEntry `yaml:"csp_warn,omitempty"`
	RequireApproval     *bool      `yaml:"require_approval,omitempty"`
}

// CSPEntry represents a CSP rule entry in config.
type CSPEntry struct {
	Pattern string `yaml:"pattern"`
	Reason  string `yaml:"reason"`
}

// DefaultSecurityConfig returns a default security configuration.
func DefaultSecurityConfig() SecurityConfig {
	requireApproval := true
	return SecurityConfig{
		BlockedCapabilities: []string{},
		CSPDeny:             []CSPEntry{},
		CSPWarn:             []CSPEntry{},
		RequireApproval:     &requireApproval,
	}
}

// ToPolicy converts security config to a Policy.
func (s *SecurityConfig) ToPolicy() (*Policy, error) {
	builder := NewPolicyBuilder()

	// Grant default capabilities
	for _, info := range AllCapabilities() {
		builder.Grant(info.Capability)
	}

	// Block specified capabilities
	for _, capStr := range s.BlockedCapabilities {
		c, err := ParseCapability(capStr)
		if err != nil {
			return nil, fmt.Errorf("invalid blocked capability %q: %w", capStr, err)
		}
		builder.Block(c)
	}

	// Set require approval
	if s.RequireApproval != nil {
		builder.RequireApproval(*s.RequireApproval)
	}

	return builder.Build(), nil
}

// ToCSP converts security config to a CSP.
func (s *SecurityConfig) ToCSP() (*CSP, error) {
	csp := DefaultCSP()

	// Add custom deny rules
	for _, entry := range s.CSPDeny {
		if err := csp.AddDeny(entry.Pattern, entry.Reason); err != nil {
			return nil, fmt.Errorf("invalid CSP deny pattern %q: %w", entry.Pattern, err)
		}
	}

	// Add custom warn rules
	for _, entry := range s.CSPWarn {
		if err := csp.AddWarn(entry.Pattern, entry.Reason); err != nil {
			return nil, fmt.Errorf("invalid CSP warn pattern %q: %w", entry.Pattern, err)
		}
	}

	return csp, nil
}

// Validate checks the security config for errors.
func (s *SecurityConfig) Validate() error {
	// Validate blocked capabilities
	for _, capStr := range s.BlockedCapabilities {
		if _, err := ParseCapability(capStr); err != nil {
			return fmt.Errorf("invalid blocked capability %q: %w", capStr, err)
		}
	}

	// Validate CSP patterns
	for _, entry := range s.CSPDeny {
		rule := NewDenyRule(entry.Pattern, entry.Reason)
		if err := rule.Compile(); err != nil {
			return fmt.Errorf("invalid CSP deny pattern %q: %w", entry.Pattern, err)
		}
	}

	for _, entry := range s.CSPWarn {
		rule := NewWarnRule(entry.Pattern, entry.Reason)
		if err := rule.Compile(); err != nil {
			return fmt.Errorf("invalid CSP warn pattern %q: %w", entry.Pattern, err)
		}
	}

	return nil
}

// Merge merges another security config into this one.
// The other config takes precedence.
func (s *SecurityConfig) Merge(other *SecurityConfig) *SecurityConfig {
	if other == nil {
		return s
	}

	result := &SecurityConfig{
		BlockedCapabilities: make([]string, 0),
		CSPDeny:             make([]CSPEntry, 0),
		CSPWarn:             make([]CSPEntry, 0),
		RequireApproval:     s.RequireApproval,
	}

	// Merge blocked capabilities (union)
	seen := make(map[string]bool)
	for _, c := range s.BlockedCapabilities {
		if !seen[c] {
			result.BlockedCapabilities = append(result.BlockedCapabilities, c)
			seen[c] = true
		}
	}
	for _, c := range other.BlockedCapabilities {
		if !seen[c] {
			result.BlockedCapabilities = append(result.BlockedCapabilities, c)
			seen[c] = true
		}
	}

	// Merge CSP rules (append)
	result.CSPDeny = append(result.CSPDeny, s.CSPDeny...)
	result.CSPDeny = append(result.CSPDeny, other.CSPDeny...)
	result.CSPWarn = append(result.CSPWarn, s.CSPWarn...)
	result.CSPWarn = append(result.CSPWarn, other.CSPWarn...)

	// Other's require approval takes precedence
	if other.RequireApproval != nil {
		result.RequireApproval = other.RequireApproval
	}

	return result
}

// IsEmpty returns true if the config has no settings.
func (s *SecurityConfig) IsEmpty() bool {
	return len(s.BlockedCapabilities) == 0 &&
		len(s.CSPDeny) == 0 &&
		len(s.CSPWarn) == 0 &&
		s.RequireApproval == nil
}
