package compiler

import "fmt"

// DiffType represents the type of change a step will make.
type DiffType string

const (
	// DiffTypeAdd indicates a new resource will be created.
	DiffTypeAdd DiffType = "add"
	// DiffTypeRemove indicates an existing resource will be removed.
	DiffTypeRemove DiffType = "remove"
	// DiffTypeModify indicates an existing resource will be modified.
	DiffTypeModify DiffType = "modify"
	// DiffTypeNone indicates no change is needed.
	DiffTypeNone DiffType = "none"
)

// String returns the string representation of the diff type.
func (d DiffType) String() string {
	return string(d)
}

// Diff represents a planned change from a step.
type Diff struct {
	diffType DiffType
	resource string
	name     string
	oldValue string
	newValue string
}

// NewDiff creates a new Diff.
func NewDiff(diffType DiffType, resource, name, oldValue, newValue string) Diff {
	return Diff{
		diffType: diffType,
		resource: resource,
		name:     name,
		oldValue: oldValue,
		newValue: newValue,
	}
}

// Type returns the diff type.
func (d Diff) Type() DiffType {
	return d.diffType
}

// Resource returns the resource type (e.g., "package", "file").
func (d Diff) Resource() string {
	return d.resource
}

// Name returns the resource name.
func (d Diff) Name() string {
	return d.name
}

// OldValue returns the previous value (empty for add operations).
func (d Diff) OldValue() string {
	return d.oldValue
}

// NewValue returns the new value (empty for remove operations).
func (d Diff) NewValue() string {
	return d.newValue
}

// Summary returns a human-readable summary of the diff.
func (d Diff) Summary() string {
	switch d.diffType {
	case DiffTypeAdd:
		return fmt.Sprintf("+ %s %s (%s)", d.resource, d.name, d.newValue)
	case DiffTypeRemove:
		return fmt.Sprintf("- %s %s (%s)", d.resource, d.name, d.oldValue)
	case DiffTypeModify:
		return fmt.Sprintf("~ %s %s", d.resource, d.name)
	case DiffTypeNone:
		return fmt.Sprintf("  %s %s", d.resource, d.name)
	}
	return fmt.Sprintf("  %s %s", d.resource, d.name)
}

// IsEmpty returns true if this diff represents no meaningful change.
func (d Diff) IsEmpty() bool {
	// Consider both zero value ("") and explicit DiffTypeNone as empty
	return (d.diffType == DiffTypeNone || d.diffType == "") && d.resource == "" && d.name == ""
}
