package fleet

import (
	"fmt"
	"regexp"
	"strings"
)

// GroupName is the identifier for a group of hosts.
type GroupName string

// groupNamePattern validates group names.
var groupNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]?$`)

// NewGroupName creates a new group name, validating the format.
func NewGroupName(name string) (GroupName, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("group name cannot be empty")
	}
	if !groupNamePattern.MatchString(name) {
		return "", fmt.Errorf("invalid group name %q: must be alphanumeric with hyphens/underscores, 1-64 chars", name)
	}
	return GroupName(name), nil
}

// String returns the group name as a string.
func (g GroupName) String() string {
	return string(g)
}

// Group represents a named collection of hosts with optional policies.
type Group struct {
	// Name is the group identifier.
	name GroupName
	// Description is a human-readable description.
	description string
	// HostPatterns are glob patterns matching host IDs.
	hostPatterns []string
	// Policies are policy names applied to this group.
	policies []string
	// Inherit lists parent groups to inherit from.
	inherit []GroupName
}

// NewGroup creates a new group with the given name.
func NewGroup(name GroupName) *Group {
	return &Group{
		name:         name,
		hostPatterns: []string{},
		policies:     []string{},
		inherit:      []GroupName{},
	}
}

// Name returns the group name.
func (g *Group) Name() GroupName {
	return g.name
}

// Description returns the group description.
func (g *Group) Description() string {
	return g.description
}

// SetDescription sets the group description.
func (g *Group) SetDescription(desc string) {
	g.description = desc
}

// HostPatterns returns the host matching patterns.
func (g *Group) HostPatterns() []string {
	result := make([]string, len(g.hostPatterns))
	copy(result, g.hostPatterns)
	return result
}

// AddHostPattern adds a host pattern to the group.
func (g *Group) AddHostPattern(pattern string) {
	for _, p := range g.hostPatterns {
		if p == pattern {
			return
		}
	}
	g.hostPatterns = append(g.hostPatterns, pattern)
}

// SetHostPatterns sets the host patterns.
func (g *Group) SetHostPatterns(patterns []string) {
	g.hostPatterns = make([]string, len(patterns))
	copy(g.hostPatterns, patterns)
}

// Policies returns the policy names.
func (g *Group) Policies() []string {
	result := make([]string, len(g.policies))
	copy(result, g.policies)
	return result
}

// AddPolicy adds a policy to the group.
func (g *Group) AddPolicy(policy string) {
	for _, p := range g.policies {
		if p == policy {
			return
		}
	}
	g.policies = append(g.policies, policy)
}

// SetPolicies sets the policies.
func (g *Group) SetPolicies(policies []string) {
	g.policies = make([]string, len(policies))
	copy(g.policies, policies)
}

// HasPolicy checks if the group has a specific policy.
func (g *Group) HasPolicy(policy string) bool {
	for _, p := range g.policies {
		if p == policy {
			return true
		}
	}
	return false
}

// Inherit returns the parent group names.
func (g *Group) Inherit() []GroupName {
	result := make([]GroupName, len(g.inherit))
	copy(result, g.inherit)
	return result
}

// AddInherit adds a parent group to inherit from.
func (g *Group) AddInherit(parent GroupName) {
	for _, p := range g.inherit {
		if p == parent {
			return
		}
	}
	g.inherit = append(g.inherit, parent)
}

// SetInherit sets the parent groups.
func (g *Group) SetInherit(parents []GroupName) {
	g.inherit = make([]GroupName, len(parents))
	copy(g.inherit, parents)
}

// GroupSummary is a read-only summary of a group.
type GroupSummary struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	HostPatterns []string `json:"host_patterns,omitempty"`
	Policies     []string `json:"policies,omitempty"`
	Inherit      []string `json:"inherit,omitempty"`
}

// Summary returns a read-only summary of the group.
func (g *Group) Summary() GroupSummary {
	inherit := make([]string, len(g.inherit))
	for i, p := range g.inherit {
		inherit[i] = p.String()
	}
	return GroupSummary{
		Name:         g.name.String(),
		Description:  g.description,
		HostPatterns: g.HostPatterns(),
		Policies:     g.Policies(),
		Inherit:      inherit,
	}
}
