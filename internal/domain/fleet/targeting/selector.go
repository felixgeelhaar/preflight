package targeting

import (
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
)

// SelectorType indicates the type of selector.
type SelectorType string

const (
	// SelectorTypeAll selects all hosts.
	SelectorTypeAll SelectorType = "all"
	// SelectorTypeGroup selects hosts by group name.
	SelectorTypeGroup SelectorType = "group"
	// SelectorTypeTag selects hosts by tag.
	SelectorTypeTag SelectorType = "tag"
	// SelectorTypePattern selects hosts by ID pattern.
	SelectorTypePattern SelectorType = "pattern"
	// SelectorTypeHost selects a specific host by ID.
	SelectorTypeHost SelectorType = "host"
)

// Selector defines criteria for selecting hosts from an inventory.
type Selector struct {
	selectorType SelectorType
	value        string
	pattern      *Pattern
	negate       bool
}

// NewSelector creates a selector from a target string.
// Supported formats:
//   - @all or * - select all hosts
//   - @groupname - select by group
//   - tag:tagname - select by tag
//   - hostname - select by host ID (exact or pattern)
//   - !pattern - negate (exclude matching hosts)
func NewSelector(target string) (*Selector, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("selector cannot be empty")
	}

	s := &Selector{}

	// Check for negation
	if strings.HasPrefix(target, "!") {
		s.negate = true
		target = target[1:]
	}

	// Parse selector type
	switch {
	case target == "@all" || target == "*":
		s.selectorType = SelectorTypeAll
		s.value = "all"

	case strings.HasPrefix(target, "@"):
		s.selectorType = SelectorTypeGroup
		s.value = target[1:]
		if s.value == "" {
			return nil, fmt.Errorf("group name cannot be empty")
		}

	case strings.HasPrefix(target, "tag:"):
		s.selectorType = SelectorTypeTag
		s.value = target[4:]
		if s.value == "" {
			return nil, fmt.Errorf("tag name cannot be empty")
		}

	case strings.ContainsAny(target, "*?[") || strings.HasPrefix(target, "~"):
		s.selectorType = SelectorTypePattern
		s.value = target
		pattern, err := NewPattern(target)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern: %w", err)
		}
		s.pattern = pattern

	default:
		s.selectorType = SelectorTypeHost
		s.value = target
	}

	return s, nil
}

// Type returns the selector type.
func (s *Selector) Type() SelectorType {
	return s.selectorType
}

// Value returns the selector value.
func (s *Selector) Value() string {
	return s.value
}

// IsNegated returns true if this selector excludes matches.
func (s *Selector) IsNegated() bool {
	return s.negate
}

// Select returns hosts from the inventory matching this selector.
func (s *Selector) Select(inv *fleet.Inventory) []*fleet.Host {
	var hosts []*fleet.Host

	switch s.selectorType {
	case SelectorTypeAll:
		hosts = inv.AllHosts()

	case SelectorTypeGroup:
		groupName, err := fleet.NewGroupName(s.value)
		if err == nil {
			hosts = inv.HostsByGroup(groupName)
		}

	case SelectorTypeTag:
		tag, err := fleet.NewTag(s.value)
		if err == nil {
			hosts = inv.HostsByTag(tag)
		}

	case SelectorTypePattern:
		if s.pattern != nil {
			for _, host := range inv.AllHosts() {
				if s.pattern.Match(string(host.ID())) {
					hosts = append(hosts, host)
				}
			}
		}

	case SelectorTypeHost:
		hostID, err := fleet.NewHostID(s.value)
		if err == nil {
			if host, ok := inv.GetHost(hostID); ok {
				hosts = []*fleet.Host{host}
			}
		}
	}

	return hosts
}

// String returns a string representation of the selector.
func (s *Selector) String() string {
	prefix := ""
	if s.negate {
		prefix = "!"
	}

	switch s.selectorType {
	case SelectorTypeAll:
		return prefix + "@all"
	case SelectorTypeGroup:
		return prefix + "@" + s.value
	case SelectorTypeTag:
		return prefix + "tag:" + s.value
	default:
		return prefix + s.value
	}
}

// Target represents a complete targeting expression with includes and excludes.
type Target struct {
	includes []*Selector
	excludes []*Selector
}

// NewTarget creates a target from selector strings.
func NewTarget(selectors ...string) (*Target, error) {
	t := &Target{
		includes: make([]*Selector, 0),
		excludes: make([]*Selector, 0),
	}

	for _, s := range selectors {
		selector, err := NewSelector(s)
		if err != nil {
			return nil, err
		}
		if selector.IsNegated() {
			t.excludes = append(t.excludes, selector)
		} else {
			t.includes = append(t.includes, selector)
		}
	}

	// If no includes, default to all
	if len(t.includes) == 0 && len(t.excludes) > 0 {
		all, _ := NewSelector("@all")
		t.includes = append(t.includes, all)
	}

	return t, nil
}

// Select returns hosts matching the target expression.
func (t *Target) Select(inv *fleet.Inventory) []*fleet.Host {
	// Collect all included hosts
	included := make(map[fleet.HostID]*fleet.Host)
	for _, selector := range t.includes {
		for _, host := range selector.Select(inv) {
			included[host.ID()] = host
		}
	}

	// Collect all excluded host IDs
	excluded := make(map[fleet.HostID]bool)
	for _, selector := range t.excludes {
		for _, host := range selector.Select(inv) {
			excluded[host.ID()] = true
		}
	}

	// Return included minus excluded
	result := make([]*fleet.Host, 0, len(included))
	for id, host := range included {
		if !excluded[id] {
			result = append(result, host)
		}
	}

	return result
}

// Includes returns the include selectors.
func (t *Target) Includes() []*Selector {
	return t.includes
}

// Excludes returns the exclude selectors.
func (t *Target) Excludes() []*Selector {
	return t.excludes
}

// String returns a string representation of the target.
func (t *Target) String() string {
	parts := make([]string, 0, len(t.includes)+len(t.excludes))
	for _, s := range t.includes {
		parts = append(parts, s.String())
	}
	for _, s := range t.excludes {
		parts = append(parts, s.String())
	}
	return strings.Join(parts, " ")
}
