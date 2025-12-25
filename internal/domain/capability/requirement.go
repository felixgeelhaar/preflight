package capability

import (
	"fmt"
)

// Requirement represents a capability requirement with optional justification.
type Requirement struct {
	Capability    Capability
	Justification string
	Optional      bool
}

// NewRequirement creates a required capability.
func NewRequirement(c Capability, justification string) Requirement {
	return Requirement{
		Capability:    c,
		Justification: justification,
		Optional:      false,
	}
}

// NewOptionalRequirement creates an optional capability.
func NewOptionalRequirement(c Capability, justification string) Requirement {
	return Requirement{
		Capability:    c,
		Justification: justification,
		Optional:      true,
	}
}

// Requirements is a collection of capability requirements.
type Requirements struct {
	required []Requirement
}

// NewRequirements creates an empty requirements collection.
func NewRequirements() *Requirements {
	return &Requirements{
		required: make([]Requirement, 0),
	}
}

// Add adds a requirement.
func (r *Requirements) Add(req Requirement) {
	r.required = append(r.required, req)
}

// AddCapability adds a required capability with justification.
func (r *Requirements) AddCapability(c Capability, justification string) {
	r.Add(NewRequirement(c, justification))
}

// AddOptional adds an optional capability with justification.
func (r *Requirements) AddOptional(c Capability, justification string) {
	r.Add(NewOptionalRequirement(c, justification))
}

// All returns all requirements.
func (r *Requirements) All() []Requirement {
	result := make([]Requirement, len(r.required))
	copy(result, r.required)
	return result
}

// Required returns only required (non-optional) capabilities.
func (r *Requirements) Required() []Requirement {
	var result []Requirement
	for _, req := range r.required {
		if !req.Optional {
			result = append(result, req)
		}
	}
	return result
}

// Optional returns only optional capabilities.
func (r *Requirements) Optional() []Requirement {
	var result []Requirement
	for _, req := range r.required {
		if req.Optional {
			result = append(result, req)
		}
	}
	return result
}

// Dangerous returns requirements for dangerous capabilities.
func (r *Requirements) Dangerous() []Requirement {
	var result []Requirement
	for _, req := range r.required {
		if req.Capability.IsDangerous() {
			result = append(result, req)
		}
	}
	return result
}

// ToSet returns all required capabilities as a Set.
func (r *Requirements) ToSet() *Set {
	s := NewSet()
	for _, req := range r.required {
		if !req.Optional {
			s.Add(req.Capability)
		}
	}
	return s
}

// FullSet returns all capabilities (required + optional) as a Set.
func (r *Requirements) FullSet() *Set {
	s := NewSet()
	for _, req := range r.required {
		s.Add(req.Capability)
	}
	return s
}

// Count returns the total number of requirements.
func (r *Requirements) Count() int {
	return len(r.required)
}

// IsEmpty returns true if there are no requirements.
func (r *Requirements) IsEmpty() bool {
	return len(r.required) == 0
}

// ValidateAgainst checks if all required capabilities are allowed by the policy.
func (r *Requirements) ValidateAgainst(policy *Policy) *ValidationResult {
	result := &ValidationResult{
		Allowed: make([]Requirement, 0),
		Denied:  make([]DeniedRequirement, 0),
		Pending: make([]Requirement, 0),
	}

	for _, req := range r.required {
		err := policy.Check(req.Capability)
		switch {
		case err == nil:
			result.Allowed = append(result.Allowed, req)
		case containsError(err, ErrDangerousCapability):
			result.Pending = append(result.Pending, req)
		default:
			result.Denied = append(result.Denied, DeniedRequirement{
				Requirement: req,
				Reason:      err.Error(),
			})
		}
	}

	return result
}

// containsError checks if the error contains the target error.
func containsError(err, target error) bool {
	if err == nil {
		return false
	}
	return err.Error() == target.Error() ||
		(len(err.Error()) > len(target.Error()) &&
			err.Error()[:len(target.Error())] == target.Error()[:len(target.Error())])
}

// ValidationResult contains the result of validating requirements against a policy.
type ValidationResult struct {
	Allowed []Requirement
	Denied  []DeniedRequirement
	Pending []Requirement // Dangerous capabilities needing approval
}

// DeniedRequirement is a requirement that was denied.
type DeniedRequirement struct {
	Requirement Requirement
	Reason      string
}

// IsValid returns true if all required capabilities are allowed or pending approval.
func (v *ValidationResult) IsValid() bool {
	return len(v.Denied) == 0
}

// NeedsApproval returns true if there are capabilities pending approval.
func (v *ValidationResult) NeedsApproval() bool {
	return len(v.Pending) > 0
}

// AllDenied returns all denied requirements.
func (v *ValidationResult) AllDenied() []DeniedRequirement {
	return v.Denied
}

// Summary returns a human-readable summary.
func (v *ValidationResult) Summary() string {
	if v.IsValid() && !v.NeedsApproval() {
		return fmt.Sprintf("All %d capabilities allowed", len(v.Allowed))
	}
	if len(v.Denied) > 0 {
		return fmt.Sprintf("%d allowed, %d denied, %d pending approval",
			len(v.Allowed), len(v.Denied), len(v.Pending))
	}
	return fmt.Sprintf("%d allowed, %d pending approval",
		len(v.Allowed), len(v.Pending))
}

// ParseRequirements parses capabilities from a YAML-like structure.
func ParseRequirements(caps []string) (*Requirements, error) {
	r := NewRequirements()
	for _, s := range caps {
		c, err := ParseCapability(s)
		if err != nil {
			return nil, err
		}
		r.AddCapability(c, "")
	}
	return r, nil
}
