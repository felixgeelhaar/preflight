package catalog

import (
	"errors"
)

// Metadata errors.
var (
	ErrEmptyTitle       = errors.New("title cannot be empty")
	ErrEmptyDescription = errors.New("description cannot be empty")
)

// Metadata contains descriptive information about a preset or capability pack.
// It is an immutable value object.
type Metadata struct {
	title       string
	description string
	tradeoffs   []string
	docLinks    map[string]string
	tags        []string
}

// NewMetadata creates a new Metadata with required fields.
func NewMetadata(title, description string) (Metadata, error) {
	if title == "" {
		return Metadata{}, ErrEmptyTitle
	}

	if description == "" {
		return Metadata{}, ErrEmptyDescription
	}

	return Metadata{
		title:       title,
		description: description,
		tradeoffs:   []string{},
		docLinks:    map[string]string{},
		tags:        []string{},
	}, nil
}

// Title returns the display title.
func (m Metadata) Title() string {
	return m.title
}

// Description returns the full description.
func (m Metadata) Description() string {
	return m.description
}

// Tradeoffs returns a list of tradeoffs/considerations.
func (m Metadata) Tradeoffs() []string {
	result := make([]string, len(m.tradeoffs))
	copy(result, m.tradeoffs)
	return result
}

// DocLinks returns documentation links (name -> URL).
func (m Metadata) DocLinks() map[string]string {
	result := make(map[string]string, len(m.docLinks))
	for k, v := range m.docLinks {
		result[k] = v
	}
	return result
}

// Tags returns categorization tags.
func (m Metadata) Tags() []string {
	result := make([]string, len(m.tags))
	copy(result, m.tags)
	return result
}

// HasTag returns true if the metadata has the given tag.
func (m Metadata) HasTag(tag string) bool {
	for _, t := range m.tags {
		if t == tag {
			return true
		}
	}
	return false
}

// WithTradeoffs returns a new Metadata with tradeoffs set.
func (m Metadata) WithTradeoffs(tradeoffs []string) Metadata {
	newTradeoffs := make([]string, len(tradeoffs))
	copy(newTradeoffs, tradeoffs)

	return Metadata{
		title:       m.title,
		description: m.description,
		tradeoffs:   newTradeoffs,
		docLinks:    m.docLinks,
		tags:        m.tags,
	}
}

// WithDocLinks returns a new Metadata with doc links set.
func (m Metadata) WithDocLinks(links map[string]string) Metadata {
	newLinks := make(map[string]string, len(links))
	for k, v := range links {
		newLinks[k] = v
	}

	return Metadata{
		title:       m.title,
		description: m.description,
		tradeoffs:   m.tradeoffs,
		docLinks:    newLinks,
		tags:        m.tags,
	}
}

// WithTags returns a new Metadata with tags set.
func (m Metadata) WithTags(tags []string) Metadata {
	newTags := make([]string, len(tags))
	copy(newTags, tags)

	return Metadata{
		title:       m.title,
		description: m.description,
		tradeoffs:   m.tradeoffs,
		docLinks:    m.docLinks,
		tags:        newTags,
	}
}

// IsZero returns true if this is a zero-value Metadata.
func (m Metadata) IsZero() bool {
	return m.title == "" && m.description == ""
}

// String returns a summary string.
func (m Metadata) String() string {
	return m.title + ": " + m.description
}
