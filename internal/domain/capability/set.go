package capability

import (
	"sort"
)

// Set represents a collection of capabilities.
type Set struct {
	capabilities map[string]Capability
}

// NewSet creates an empty capability set.
func NewSet() *Set {
	return &Set{
		capabilities: make(map[string]Capability),
	}
}

// NewSetFrom creates a set from a slice of capabilities.
func NewSetFrom(caps []Capability) *Set {
	s := NewSet()
	for _, c := range caps {
		s.Add(c)
	}
	return s
}

// ParseSet parses a set from string representations.
func ParseSet(strs []string) (*Set, error) {
	s := NewSet()
	for _, str := range strs {
		c, err := ParseCapability(str)
		if err != nil {
			return nil, err
		}
		s.Add(c)
	}
	return s, nil
}

// Add adds a capability to the set.
func (s *Set) Add(c Capability) {
	if !c.IsZero() {
		s.capabilities[c.String()] = c
	}
}

// Remove removes a capability from the set.
func (s *Set) Remove(c Capability) {
	delete(s.capabilities, c.String())
}

// Has checks if the set contains a capability.
func (s *Set) Has(c Capability) bool {
	_, ok := s.capabilities[c.String()]
	return ok
}

// HasAny checks if the set contains any of the given capabilities.
func (s *Set) HasAny(caps ...Capability) bool {
	for _, c := range caps {
		if s.Has(c) {
			return true
		}
	}
	return false
}

// HasAll checks if the set contains all of the given capabilities.
func (s *Set) HasAll(caps ...Capability) bool {
	for _, c := range caps {
		if !s.Has(c) {
			return false
		}
	}
	return true
}

// Matches checks if any capability in the set matches the given capability.
// Uses pattern matching (e.g., "files:*" matches "files:read").
func (s *Set) Matches(c Capability) bool {
	for _, cap := range s.capabilities {
		if cap.Matches(c) {
			return true
		}
	}
	return false
}

// List returns all capabilities as a sorted slice.
func (s *Set) List() []Capability {
	result := make([]Capability, 0, len(s.capabilities))
	for _, c := range s.capabilities {
		result = append(result, c)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}

// Strings returns all capabilities as sorted strings.
func (s *Set) Strings() []string {
	caps := s.List()
	result := make([]string, len(caps))
	for i, c := range caps {
		result[i] = c.String()
	}
	return result
}

// Count returns the number of capabilities.
func (s *Set) Count() int {
	return len(s.capabilities)
}

// IsEmpty returns true if the set has no capabilities.
func (s *Set) IsEmpty() bool {
	return len(s.capabilities) == 0
}

// DangerousCapabilities returns all dangerous capabilities in the set.
func (s *Set) DangerousCapabilities() []Capability {
	var result []Capability
	for _, c := range s.capabilities {
		if c.IsDangerous() {
			result = append(result, c)
		}
	}
	return result
}

// HasDangerous returns true if the set contains any dangerous capabilities.
func (s *Set) HasDangerous() bool {
	return len(s.DangerousCapabilities()) > 0
}

// Union returns a new set with capabilities from both sets.
func (s *Set) Union(other *Set) *Set {
	result := NewSet()
	for _, c := range s.capabilities {
		result.Add(c)
	}
	if other != nil {
		for _, c := range other.capabilities {
			result.Add(c)
		}
	}
	return result
}

// Intersection returns a new set with capabilities in both sets.
func (s *Set) Intersection(other *Set) *Set {
	result := NewSet()
	if other == nil {
		return result
	}
	for _, c := range s.capabilities {
		if other.Has(c) {
			result.Add(c)
		}
	}
	return result
}

// Difference returns a new set with capabilities in s but not in other.
func (s *Set) Difference(other *Set) *Set {
	result := NewSet()
	for _, c := range s.capabilities {
		if other == nil || !other.Has(c) {
			result.Add(c)
		}
	}
	return result
}

// ByCategory returns capabilities grouped by category.
func (s *Set) ByCategory() map[Category][]Capability {
	result := make(map[Category][]Capability)
	for _, c := range s.capabilities {
		result[c.Category()] = append(result[c.Category()], c)
	}
	return result
}

// Clone creates a copy of the set.
func (s *Set) Clone() *Set {
	result := NewSet()
	for _, c := range s.capabilities {
		result.Add(c)
	}
	return result
}
