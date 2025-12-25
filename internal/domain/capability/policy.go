package capability

import (
	"fmt"
)

// Policy defines capability access rules.
type Policy struct {
	// Granted capabilities available for use
	granted *Set

	// Blocked capabilities that are explicitly denied
	blocked *Set

	// Approved dangerous capabilities (user confirmed)
	approved *Set

	// RequireApproval when true requires approval for dangerous capabilities
	requireApproval bool
}

// NewPolicy creates a new empty policy.
func NewPolicy() *Policy {
	return &Policy{
		granted:         NewSet(),
		blocked:         NewSet(),
		approved:        NewSet(),
		requireApproval: true,
	}
}

// PolicyBuilder builds a Policy.
type PolicyBuilder struct {
	policy *Policy
}

// NewPolicyBuilder creates a new policy builder.
func NewPolicyBuilder() *PolicyBuilder {
	return &PolicyBuilder{
		policy: NewPolicy(),
	}
}

// Grant adds capabilities to the granted set.
func (b *PolicyBuilder) Grant(caps ...Capability) *PolicyBuilder {
	for _, c := range caps {
		b.policy.granted.Add(c)
	}
	return b
}

// GrantStrings parses and grants capabilities from strings.
func (b *PolicyBuilder) GrantStrings(strs ...string) *PolicyBuilder {
	for _, s := range strs {
		if c, err := ParseCapability(s); err == nil {
			b.policy.granted.Add(c)
		}
	}
	return b
}

// Block adds capabilities to the blocked set.
func (b *PolicyBuilder) Block(caps ...Capability) *PolicyBuilder {
	for _, c := range caps {
		b.policy.blocked.Add(c)
	}
	return b
}

// BlockStrings parses and blocks capabilities from strings.
func (b *PolicyBuilder) BlockStrings(strs ...string) *PolicyBuilder {
	for _, s := range strs {
		if c, err := ParseCapability(s); err == nil {
			b.policy.blocked.Add(c)
		}
	}
	return b
}

// Approve marks dangerous capabilities as approved.
func (b *PolicyBuilder) Approve(caps ...Capability) *PolicyBuilder {
	for _, c := range caps {
		b.policy.approved.Add(c)
	}
	return b
}

// RequireApproval sets whether dangerous capabilities need approval.
func (b *PolicyBuilder) RequireApproval(require bool) *PolicyBuilder {
	b.policy.requireApproval = require
	return b
}

// Build creates the policy.
func (b *PolicyBuilder) Build() *Policy {
	return b.policy
}

// Granted returns the granted capabilities.
func (p *Policy) Granted() *Set {
	return p.granted
}

// Blocked returns the blocked capabilities.
func (p *Policy) Blocked() *Set {
	return p.blocked
}

// Approved returns the approved dangerous capabilities.
func (p *Policy) Approved() *Set {
	return p.approved
}

// RequiresApproval returns true if dangerous capabilities need approval.
func (p *Policy) RequiresApproval() bool {
	return p.requireApproval
}

// Check verifies if a capability is allowed by the policy.
func (p *Policy) Check(c Capability) error {
	// Check if blocked
	if p.blocked.Matches(c) {
		return fmt.Errorf("%w: %s is blocked by policy", ErrCapabilityDenied, c)
	}

	// Check if granted
	if !p.granted.Matches(c) {
		return fmt.Errorf("%w: %s", ErrCapabilityNotGranted, c)
	}

	// Check if dangerous and needs approval
	if c.IsDangerous() && p.requireApproval && !p.approved.Has(c) {
		return fmt.Errorf("%w: %s", ErrDangerousCapability, c)
	}

	return nil
}

// CheckAll verifies if all capabilities are allowed.
func (p *Policy) CheckAll(caps ...Capability) error {
	for _, c := range caps {
		if err := p.Check(c); err != nil {
			return err
		}
	}
	return nil
}

// IsAllowed returns true if the capability is allowed.
func (p *Policy) IsAllowed(c Capability) bool {
	return p.Check(c) == nil
}

// Effective returns the effective capabilities (granted minus blocked).
func (p *Policy) Effective() *Set {
	return p.granted.Difference(p.blocked)
}

// PendingApproval returns dangerous capabilities that need approval.
func (p *Policy) PendingApproval() []Capability {
	if !p.requireApproval {
		return nil
	}

	var pending []Capability
	for _, c := range p.granted.DangerousCapabilities() {
		if !p.approved.Has(c) && !p.blocked.Has(c) {
			pending = append(pending, c)
		}
	}
	return pending
}

// NeedsApproval returns true if there are dangerous capabilities pending approval.
func (p *Policy) NeedsApproval() bool {
	return len(p.PendingApproval()) > 0
}

// ApproveAll approves all pending dangerous capabilities.
func (p *Policy) ApproveAll() {
	for _, c := range p.PendingApproval() {
		p.approved.Add(c)
	}
}

// Violation represents a policy violation.
type Violation struct {
	Capability Capability
	Reason     string
	Blocked    bool
}

// Validate checks all requested capabilities against the policy.
func (p *Policy) Validate(requested *Set) []Violation {
	var violations []Violation

	for _, c := range requested.List() {
		if p.blocked.Matches(c) {
			violations = append(violations, Violation{
				Capability: c,
				Reason:     "blocked by policy",
				Blocked:    true,
			})
		} else if !p.granted.Matches(c) {
			violations = append(violations, Violation{
				Capability: c,
				Reason:     "not granted",
				Blocked:    false,
			})
		}
	}

	return violations
}

// Summary returns a policy summary.
func (p *Policy) Summary() PolicySummary {
	effective := p.Effective()
	return PolicySummary{
		GrantedCount:   p.granted.Count(),
		BlockedCount:   p.blocked.Count(),
		EffectiveCount: effective.Count(),
		DangerousCount: len(effective.DangerousCapabilities()),
		PendingCount:   len(p.PendingApproval()),
	}
}

// PolicySummary contains policy statistics.
type PolicySummary struct {
	GrantedCount   int
	BlockedCount   int
	EffectiveCount int
	DangerousCount int
	PendingCount   int
}

// DefaultPolicy returns a default policy for safe operations.
func DefaultPolicy() *Policy {
	return NewPolicyBuilder().
		Grant(CapFilesRead, CapFilesWrite).
		Grant(CapPackagesBrew, CapPackagesApt).
		Grant(CapPackagesWinget, CapPackagesScoop, CapPackagesChoco).
		Grant(CapNetworkFetch).
		Build()
}

// FullAccessPolicy returns a policy with all capabilities granted.
func FullAccessPolicy() *Policy {
	builder := NewPolicyBuilder().RequireApproval(false)
	for _, info := range AllCapabilities() {
		builder.Grant(info.Capability)
	}
	return builder.Build()
}

// RestrictedPolicy returns a minimal policy for untrusted plugins.
func RestrictedPolicy() *Policy {
	return NewPolicyBuilder().
		Grant(CapFilesRead).
		Grant(CapNetworkFetch).
		Block(CapShellExecute).
		Block(CapSecretsRead, CapSecretsWrite).
		Block(CapSystemModify).
		Build()
}
