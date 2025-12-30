// Package fleet provides the fleet management domain for remote execution.
package fleet

import (
	"fmt"
	"regexp"
	"strings"
)

// Tag represents a label that can be attached to hosts for grouping and targeting.
// Tags are immutable value objects.
type Tag string

// tagPattern validates tag names: lowercase alphanumeric with hyphens, max 64 chars.
// Must start with letter, cannot end with hyphen.
var tagPattern = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,62}[a-z0-9])?$`)

// NewTag creates a new tag, validating the format.
func NewTag(name string) (Tag, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "", fmt.Errorf("tag name cannot be empty")
	}
	if !tagPattern.MatchString(name) {
		return "", fmt.Errorf("invalid tag name %q: must be lowercase alphanumeric with hyphens, 1-64 chars", name)
	}
	return Tag(name), nil
}

// MustTag creates a tag, panicking on invalid input. Use only for constants.
func MustTag(name string) Tag {
	tag, err := NewTag(name)
	if err != nil {
		panic(err)
	}
	return tag
}

// String returns the tag name.
func (t Tag) String() string {
	return string(t)
}

// Tags represents a set of tags.
type Tags []Tag

// NewTags creates a Tags set from string names.
func NewTags(names ...string) (Tags, error) {
	tags := make(Tags, 0, len(names))
	seen := make(map[Tag]bool)
	for _, name := range names {
		tag, err := NewTag(name)
		if err != nil {
			return nil, err
		}
		if !seen[tag] {
			tags = append(tags, tag)
			seen[tag] = true
		}
	}
	return tags, nil
}

// Contains checks if a tag is in the set.
func (t Tags) Contains(tag Tag) bool {
	for _, existing := range t {
		if existing == tag {
			return true
		}
	}
	return false
}

// ContainsAny checks if any of the given tags are in the set.
func (t Tags) ContainsAny(tags Tags) bool {
	for _, tag := range tags {
		if t.Contains(tag) {
			return true
		}
	}
	return false
}

// ContainsAll checks if all given tags are in the set.
func (t Tags) ContainsAll(tags Tags) bool {
	for _, tag := range tags {
		if !t.Contains(tag) {
			return false
		}
	}
	return true
}

// Strings returns the tags as a slice of strings.
func (t Tags) Strings() []string {
	result := make([]string, len(t))
	for i, tag := range t {
		result[i] = tag.String()
	}
	return result
}

// Union returns a new Tags set with all tags from both sets.
func (t Tags) Union(other Tags) Tags {
	result := make(Tags, len(t))
	copy(result, t)
	for _, tag := range other {
		if !result.Contains(tag) {
			result = append(result, tag)
		}
	}
	return result
}

// Intersect returns a new Tags set with only tags present in both sets.
func (t Tags) Intersect(other Tags) Tags {
	result := make(Tags, 0)
	for _, tag := range t {
		if other.Contains(tag) {
			result = append(result, tag)
		}
	}
	return result
}
